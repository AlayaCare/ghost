package internal

import (
	"strings"

	"github.com/sirupsen/logrus"
)

/*
 * GoliacReconciliator is here to sync the local state to the remote state
 */
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
	rremote := NewMutableGoliacRemoteImpl(remote)
	r.Begin(dryrun)
	err := r.reconciliateTeams(local, rremote, dryrun)
	if err != nil {
		r.Rollback(dryrun, err)
		return err
	}

	err = r.reconciliateRepositories(local, rremote, dryrun)
	if err != nil {
		r.Rollback(dryrun, err)
		return err
	}

	r.Commit(dryrun)

	return nil
}

/*
 * This function sync teams and team's members
 */
func (r *GoliacReconciliatorImpl) reconciliateTeams(local GoliacLocal, remote *MutableGoliacRemoteImpl, dryrun bool) error {
	ghTeams := remote.Teams()

	rTeams := make(map[string]*GithubTeam)
	for k, v := range ghTeams {
		rTeams[k] = v
	}

	for _, lTeam := range local.Teams() {
		slugTeam, ok := remote.TeamSlugByName()[lTeam.Metadata.Name]
		if !ok {
			// deal with non existing team
			members := make([]string, 0)
			members = append(members, lTeam.Data.Members...)
			members = append(members, lTeam.Data.Owners...)
			// CREATE team
			r.CreateTeam(dryrun, remote, lTeam.Metadata.Name, lTeam.Metadata.Name, members)
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
					// REMOVE team member
					r.UpdateTeamRemoveMember(dryrun, remote, slugTeam, m)
				} else {
					delete(localMembers, m)
				}
			}
			for m := range localMembers {
				// ADD team member
				r.UpdateTeamAddMember(dryrun, remote, slugTeam, m, "member")
			}
			delete(rTeams, slugTeam)
		}
	}

	// remaining (GH) teams (aka not found locally)
	for _, rTeam := range rTeams {
		// DELETE team
		r.DeleteTeam(dryrun, remote, rTeam.Slug)
	}
	return nil
}

/*
 * This function sync repositories and team's repositories permissions
 */
func (r *GoliacReconciliatorImpl) reconciliateRepositories(local GoliacLocal, remote *MutableGoliacRemoteImpl, dryrun bool) error {
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
		if rRepo, ok := ghRepos[reponame]; !ok {
			// deal with non existing repo
			writers := make([]string, 0)
			for _, w := range lRepo.Data.Writers {
				writers = append(writers, remote.TeamSlugByName()[w])
			}
			if lRepo.Owner != nil {
				writers = append(writers, remote.TeamSlugByName()[*lRepo.Owner])
			}
			readers := make([]string, 0)
			for _, r := range lRepo.Data.Readers {
				readers = append(readers, remote.TeamSlugByName()[r])
			}
			// CREATE team
			r.CreateRepository(dryrun, remote, reponame, reponame, writers, readers, lRepo.Data.IsPublic)
		} else {
			// deal with existing repo

			// reconciliate repositories teams members

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

			// let's get remote teams per type (read/write)
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

			// let's check if a reader was put in a writer
			for teamSlug, teamName := range remoteReadMembers {
				if _, ok := localReadMembers[teamName]; !ok {
					if _, ok := localWriteMembers[teamName]; ok {
						r.UpdateRepositoryUpdateTeamAccess(dryrun, remote, reponame, teamSlug, "WRITE")
						remoteWriteMembers[teamSlug] = teamName
						delete(remoteReadMembers, teamSlug)
					}
				}
			}
			for teamSlug, teamName := range remoteWriteMembers {
				if _, ok := localWriteMembers[teamName]; !ok {
					if _, ok := localReadMembers[teamName]; ok { // not an update
						r.UpdateRepositoryUpdateTeamAccess(dryrun, remote, reponame, teamSlug, "READ")
						remoteReadMembers[teamSlug] = teamName
						delete(remoteWriteMembers, teamSlug)
					}
				}
			}

			// let's deal with readers
			for teamSlug, teamName := range remoteReadMembers {
				if _, ok := localReadMembers[teamName]; !ok {

					// REMOVE team member
					r.UpdateRepositoryRemoveTeamAccess(dryrun, remote, reponame, teamSlug)
				} else {
					// we will keep in localReadMembers members not found (and to be deleted)
					delete(localReadMembers, teamName)
				}
			}
			for m := range localReadMembers {
				// ADD team member
				if teamSlug, ok := remote.TeamSlugByName()[m]; ok {
					r.UpdateRepositoryAddTeamAccess(dryrun, remote, reponame, teamSlug, "READ")
				}
			}

			// let's deal with writers
			for teamSlug, teamName := range remoteWriteMembers {
				if _, ok := localWriteMembers[teamName]; !ok {
					// REMOVE team member
					r.UpdateRepositoryRemoveTeamAccess(dryrun, remote, reponame, teamSlug)
				} else {
					// we will keep in localWriteMembers members not found (and to be deleted)
					delete(localWriteMembers, teamName)
				}
			}
			for m := range localWriteMembers {
				if teamSlug, ok := remote.TeamSlugByName()[m]; ok {
					// ADD team member
					r.UpdateRepositoryAddTeamAccess(dryrun, remote, reponame, teamSlug, "WRITE")
				}
			}

			// reconciliate repositories public/private
			if lRepo.Data.IsPublic == rRepo.IsPrivate {
				// UPDATE private repository
				r.UpdateRepositoryUpdatePrivate(dryrun, remote, reponame, !lRepo.Data.IsPublic)
			}

			// reconciliate repositories archived
			if lRepo.Data.IsArchived != rRepo.IsArchived {
				// UPDATE archived repository
				r.UpdateRepositoryUpdateArchived(dryrun, remote, reponame, lRepo.Data.IsArchived)
			}

			delete(rRepos, reponame)
		}
	}

	// remaining (GH) teams (aka not found locally)
	for reponame := range rRepos {
		// DELETE team
		r.DeleteRepository(dryrun, remote, reponame)
	}
	return nil
}

func (r *GoliacReconciliatorImpl) CreateTeam(dryrun bool, remote *MutableGoliacRemoteImpl, teamname string, description string, members []string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "create_team"}).Infof("teamname: %s, members: %s", teamname, strings.Join(members, ","))
	remote.CreateTeam(teamname, description, members)
	if !dryrun {
		for _, l := range r.listeners {
			l.CreateTeam(teamname, description, members)
		}
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamAddMember(dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, username string, role string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_add_member"}).Infof("teamslug: %s, username: %s, role: %s", teamslug, username, role)
	remote.UpdateTeamAddMember(teamslug, username, "member")
	if !dryrun {
		for _, l := range r.listeners {
			l.UpdateTeamAddMember(teamslug, username, "member")
		}
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamRemoveMember(dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, username string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_remove_member"}).Infof("teamslug: %s, username: %s", teamslug, username)
	remote.UpdateTeamRemoveMember(teamslug, username)
	if !dryrun {
		for _, l := range r.listeners {
			l.UpdateTeamRemoveMember(teamslug, username)
		}
	}
}
func (r *GoliacReconciliatorImpl) DeleteTeam(dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_team"}).Infof("teamslug: %s", teamslug)
	remote.DeleteTeam(teamslug)
	if !dryrun {
		for _, l := range r.listeners {
			l.DeleteTeam(teamslug)
		}
	}
}
func (r *GoliacReconciliatorImpl) CreateRepository(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, descrition string, writers []string, readers []string, public bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "create_repository"}).Infof("repositoryname: %s, readers: %s, writers: %s, public: %v", strings.Join(readers, ","), strings.Join(writers, ","), reponame, public)
	remote.CreateRepository(reponame, reponame, writers, readers, public)
	if !dryrun {
		for _, l := range r.listeners {
			l.CreateRepository(reponame, reponame, writers, readers, public)
		}
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryAddTeamAccess(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_add_team"}).Infof("repositoryname: %s, teamslug: %s, permission: %s", reponame, teamslug, permission)
	remote.UpdateRepositoryAddTeamAccess(reponame, teamslug, permission)
	if !dryrun {
		for _, l := range r.listeners {
			l.UpdateRepositoryAddTeamAccess(reponame, teamslug, permission)
		}
	}
}

func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdateTeamAccess(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_team"}).Infof("repositoryname: %s, teamslug:%s, permission: %s", reponame, teamslug, permission)
	remote.UpdateRepositoryUpdateTeamAccess(reponame, teamslug, permission)
	if !dryrun {
		for _, l := range r.listeners {
			l.UpdateRepositoryUpdateTeamAccess(reponame, teamslug, permission)
		}
	}

}
func (r *GoliacReconciliatorImpl) UpdateRepositoryRemoveTeamAccess(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_remove_team"}).Infof("repositoryname: %s, teamslug:%s", reponame, teamslug)
	remote.UpdateRepositoryRemoveTeamAccess(reponame, teamslug)
	if !dryrun {
		for _, l := range r.listeners {
			l.UpdateRepositoryRemoveTeamAccess(reponame, teamslug)
		}
	}
}
func (r *GoliacReconciliatorImpl) DeleteRepository(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository"}).Infof("repositoryname: %s", reponame)
	remote.DeleteRepository(reponame)
	if !dryrun {
		for _, l := range r.listeners {
			l.DeleteRepository(reponame)
		}
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdatePrivate(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, private bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_private"}).Infof("repositoryname: %s private:%v", reponame, private)
	remote.UpdateRepositoryUpdatePrivate(reponame, private)
	if !dryrun {
		for _, l := range r.listeners {
			l.UpdateRepositoryUpdatePrivate(reponame, private)
		}
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdateArchived(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, archived bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_private"}).Infof("repositoryname: %s archived:%v", reponame, archived)
	remote.UpdateRepositoryUpdateArchived(reponame, archived)
	if !dryrun {
		for _, l := range r.listeners {
			l.UpdateRepositoryUpdateArchived(reponame, archived)
		}
	}
}
func (r *GoliacReconciliatorImpl) Begin(dryrun bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Infof("reconciliation begin")
	if !dryrun {
		for _, l := range r.listeners {
			l.Begin()
		}
	}
}
func (r *GoliacReconciliatorImpl) Rollback(dryrun bool, err error) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Infof("reconciliation rollback")
	if !dryrun {
		for _, l := range r.listeners {
			l.Rollback(err)
		}
	}
}
func (r *GoliacReconciliatorImpl) Commit(dryrun bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Infof("reconciliation commit")
	if !dryrun {
		for _, l := range r.listeners {
			l.Commit()
		}
	}
}
