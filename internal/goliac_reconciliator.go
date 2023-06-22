package internal

type GoliacReconciliator interface {
	AddListener(ReconciliatorListener)
	Reconciliate(local GoliacLocal, remote GoliacRemote, dryrun bool) error
}

type GoliacReconciliatorImpl struct {
	listeners []ReconciliatorListener
}

func NewGoliacReconciliatorImpl() GoliacReconciliator {
	return &GoliacReconciliatorImpl{
		listeners: make([]ReconciliatorListener, 0),
	}
}

func (r *GoliacReconciliatorImpl) AddListener(l ReconciliatorListener) {
	r.listeners = append(r.listeners, l)
}

func (r *GoliacReconciliatorImpl) Reconciliate(local GoliacLocal, remote GoliacRemote, dryrun bool) error {
	err := r.reconciliateTeams(local, remote, dryrun)
	if err != nil {
		return err
	}

	err = r.reconciliateRepositories(local, remote, dryrun)
	if err != nil {
		return err
	}
	return nil
}

/*
 * This function sync teams and team's members
 */
func (r *GoliacReconciliatorImpl) reconciliateTeams(local GoliacLocal, remote GoliacRemote, dryrun bool) error {
	ghTeams := remote.Teams()

	rTeams := make(map[string]*GithubTeam)
	for k, v := range ghTeams {
		rTeams[k] = v
	}

	for _, lTeam := range local.Teams() {
		slugTeam, ok := remote.TeamSlugByName(lTeam.Metadata.Name)
		if !ok {
			// deal with non existing team
			members := make([]string, 0)
			members = append(members, lTeam.Data.Members...)
			members = append(members, lTeam.Data.Owners...)
			for _, l := range r.listeners {
				// CREATE team
				l.CreateTeam(lTeam.Metadata.Name, lTeam.Metadata.Name, members)
			}
		} else {
			// deal with existing team
			rTeam := ghTeams[slugTeam]

			localMembers := make(map[string]bool)
			for _, m := range lTeam.Data.Members {
				localMembers[m] = true
			}
			for _, m := range lTeam.Data.Owners {
				localMembers[m] = true
			}

			for _, m := range rTeam.Members {
				if _, ok := localMembers[m]; !ok {
					for _, l := range r.listeners {
						// REMOVE team member
						l.UpdateTeamRemoveMember(slugTeam, m)
					}
				} else {
					delete(localMembers, m)
				}
			}
			for m, _ := range localMembers {
				for _, l := range r.listeners {
					// ADD team member
					l.UpdateTeamAddMember(slugTeam, m, "member")
				}
			}
			delete(rTeams, slugTeam)
		}
	}

	// remaining (GH) teams (aka not found locally)
	for _, rTeam := range rTeams {
		for _, l := range r.listeners {
			// DELETE team
			l.DeleteTeam(rTeam.Slug)
		}
	}
	return nil
}

/*
 * This function sync repositories and team's repositories permissions
 */
func (r *GoliacReconciliatorImpl) reconciliateRepositories(local GoliacLocal, remote GoliacRemote, dryrun bool) error {
	ghRepos := remote.Repositories()
	rRepos := make(map[string]*GithubRepository)
	for k, v := range ghRepos {
		rRepos[k] = v
	}

	// on the remote object, I have teams->repos, and I need repos->teams
	teamsPerRepos := make(map[string]map[string]*GithubTeamRepo)
	for t, repos := range remote.TeamRepositories() {
		for r, p := range repos {
			if _, ok := teamsPerRepos[r]; !ok {
				teamsPerRepos[r] = make(map[string]*GithubTeamRepo)
			}
			teamsPerRepos[r][t] = p
		}
	}

	for reponame, lRepo := range local.Repositories() {
		if _, ok := ghRepos[reponame]; !ok {
			// deal with non existing repo
			writers := make([]string, 0)
			writers = append(writers, lRepo.Data.Writers...)
			if lRepo.Owner != nil {
				writers = append(writers, *lRepo.Owner)
			}
			for _, l := range r.listeners {
				// CREATE team
				l.CreateRepository(reponame, reponame, writers, lRepo.Data.Readers, lRepo.Data.IsPublic)
			}
		} else {
			// deal with existing repo

			localReadMembers := make(map[string]bool)
			for _, m := range lRepo.Data.Readers {
				localReadMembers[m] = true
			}
			localWriteMembers := make(map[string]bool)
			for _, m := range lRepo.Data.Writers {
				localWriteMembers[m] = true
			}
			if lRepo.Owner != nil {
				localWriteMembers[*lRepo.Owner] = true
			}

			// let's get reomte teams per type (read/write)
			remoteReadMembers := make(map[string]string)
			remoteWriteMembers := make(map[string]string)
			for slugteam, permissions := range teamsPerRepos[reponame] {
				team := remote.Teams()[slugteam]
				if permissions.Permission == "READ" {
					remoteReadMembers[slugteam] = team.Name
				} else {
					remoteWriteMembers[slugteam] = team.Name
				}
			}

			// let's deal with readers
			for teamSlug, teamName := range remoteReadMembers {
				if _, ok := localReadMembers[teamName]; !ok {
					for _, l := range r.listeners {
						// REMOVE team member
						l.UpdateRepositoryRemoveTeamAccess(reponame, teamSlug)
					}
				} else {
					delete(localReadMembers, teamName)
				}
			}
			for m, _ := range localReadMembers {
				for _, l := range r.listeners {
					// ADD team member
					if teamSlug, ok := remote.TeamSlugByName(m); ok {
						l.UpdateRepositoryAddTeamAccess(reponame, teamSlug, "READ")
					}
				}
			}

			// let's deal with writers
			for teamSlug, teamName := range remoteWriteMembers {
				if _, ok := localWriteMembers[teamName]; !ok {
					for _, l := range r.listeners {
						// REMOVE team member
						l.UpdateRepositoryRemoveTeamAccess(reponame, teamSlug)
					}
				} else {
					delete(localWriteMembers, teamName)
				}
			}
			for m, _ := range localWriteMembers {
				for _, l := range r.listeners {
					// ADD team member
					if teamSlug, ok := remote.TeamSlugByName(m); ok {
						l.UpdateRepositoryAddTeamAccess(reponame, teamSlug, "WRITE")
					}
				}
			}

			delete(rRepos, reponame)
		}
	}

	// remaining (GH) teams (aka not found locally)
	for reponame, _ := range rRepos {
		for _, l := range r.listeners {
			// DELETE team
			l.DeleteRepository(reponame)
		}
	}
	return nil
}
