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
