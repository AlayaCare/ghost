package internal

import (
	"fmt"
	"strings"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/slugify"
	"github.com/sirupsen/logrus"
)

/*
 * GoliacReconciliator is here to sync the local state to the remote state
 */
type GoliacReconciliator interface {
	Reconciliate(local GoliacLocal, remote GoliacRemote, teamreponame string, dryrun bool) error
}

type GoliacReconciliatorImpl struct {
	executor   ReconciliatorExecutor
	repoconfig *config.RepositoryConfig
}

func NewGoliacReconciliatorImpl(executor ReconciliatorExecutor, repoconfig *config.RepositoryConfig) GoliacReconciliator {
	return &GoliacReconciliatorImpl{
		executor:   executor,
		repoconfig: repoconfig,
	}
}

func (r *GoliacReconciliatorImpl) Reconciliate(local GoliacLocal, remote GoliacRemote, teamsreponame string, dryrun bool) error {
	rremote := NewMutableGoliacRemoteImpl(remote)
	r.Begin(dryrun)
	err := r.reconciliateUsers(local, rremote, dryrun)
	if err != nil {
		r.Rollback(dryrun, err)
		return err
	}

	err = r.reconciliateTeams(local, rremote, dryrun)
	if err != nil {
		r.Rollback(dryrun, err)
		return err
	}

	err = r.reconciliateRepositories(local, rremote, teamsreponame, dryrun)
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
func (r *GoliacReconciliatorImpl) reconciliateUsers(local GoliacLocal, remote *MutableGoliacRemoteImpl, dryrun bool) error {
	ghUsers := remote.Users()

	rUsers := make(map[string]string)
	for _, u := range ghUsers {
		rUsers[u] = u
	}

	for _, lUser := range local.Users() {
		user, ok := rUsers[lUser.Data.GithubID]

		if !ok {
			// deal with non existing remote user
			r.AddUserToOrg(dryrun, remote, user)
		} else {
			delete(rUsers, user)
		}
	}

	// remaining (GH) users (aka not found locally)
	for _, rUser := range rUsers {
		// DELETE User
		r.RemoveUserFromOrg(dryrun, remote, rUser)
	}
	return nil
}

/*
 * Compare 2 string arrays to see if they contains the same elements
 * Returns
 * - true if both are the same
 * - left only
 * - rigght only
 */
func StringArrayEquivalent(a, b []string) (bool, []string, []string) {
	leftOnly := []string{}
	rightOnly := []string{}
	lefts := make(map[string]bool)
	for _, m := range a {
		lefts[m] = true
	}

	rights := make(map[string]bool)
	for _, m := range b {
		rights[m] = true
	}

	result := true

	if len(lefts) != len(rights) {
		result = false
	}

	for r, _ := range rights {
		if _, ok := lefts[r]; !ok {
			leftOnly = append(leftOnly, r)
			result = false
		}
	}
	for l, _ := range lefts {
		if _, ok := rights[l]; !ok {
			rightOnly = append(rightOnly, l)
			result = false
		}
	}
	return result, leftOnly, rightOnly
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

	// prepare the teams we want (regular and "-owners")
	slugTeams := make(map[string]*GithubTeam)
	for teamname, teamvalue := range local.Teams() {
		members := []string{}
		members = append(members, teamvalue.Data.Members...)
		members = append(members, teamvalue.Data.Owners...)

		teamslug := slugify.Make(teamname)
		slugTeams[teamslug] = &GithubTeam{
			Name:    teamname,
			Slug:    teamslug,
			Members: members,
		}

		// owners
		slugTeams[teamslug+"-owners"] = &GithubTeam{
			Name:    teamname + "-owners",
			Slug:    teamslug + "-owners",
			Members: teamvalue.Data.Owners,
		}
	}

	// now we compare local (slugTeams) and remote (rTeams)

	compareTeam := func(lTeam *GithubTeam, rTeam *GithubTeam) bool {
		res, _, _ := StringArrayEquivalent(lTeam.Members, rTeam.Members)
		return res
	}

	onAdded := func(key string, lTeam *GithubTeam, rTeam *GithubTeam) {
		members := make([]string, 0)
		for _, m := range lTeam.Members {
			if ghuserid, ok := local.Users()[m]; ok {
				members = append(members, ghuserid.Data.GithubID)
			}
		}
		// CREATE team
		r.CreateTeam(dryrun, remote, lTeam.Slug, lTeam.Name, members)
	}

	onRemoved := func(key string, lTeam *GithubTeam, rTeam *GithubTeam) {
		// DELETE team
		r.DeleteTeam(dryrun, remote, rTeam.Slug)
	}

	onChanged := func(slugTeam string, lTeam *GithubTeam, rTeam *GithubTeam) {
		localMembers := make(map[string]bool)
		for _, m := range lTeam.Members {
			if ghuserid, ok := local.Users()[m]; ok {
				localMembers[ghuserid.Data.GithubID] = true
			}
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
	}

	CompareEntities(slugTeams, rTeams, compareTeam, onAdded, onRemoved, onChanged)

	return nil
}

type GithubRepoComparable struct {
	IsPublic   bool
	IsArchived bool
	Writers    []string
	Readers    []string
}

/*
 * This function sync repositories and team's repositories permissions
 */
func (r *GoliacReconciliatorImpl) reconciliateRepositories(local GoliacLocal, remote *MutableGoliacRemoteImpl, teamsreponame string, dryrun bool) error {
	ghRepos := remote.Repositories()
	rRepos := make(map[string]*GithubRepoComparable)
	for k, v := range ghRepos {
		rRepos[k] = &GithubRepoComparable{
			IsPublic:   !v.IsPrivate,
			IsArchived: v.IsArchived,
			Writers:    []string{},
			Readers:    []string{},
		}
	}

	// on the remote object, I have teams->repos, and I need repos->teams
	for t, repos := range remote.TeamRepositories() {
		for r, p := range repos {
			if rr, ok := rRepos[r]; ok {
				if p.Permission == "ADMIN" || p.Permission == "WRITE" {
					rr.Writers = append(rr.Writers, t)
				} else {
					rr.Readers = append(rr.Readers, t)
				}
			}
		}
	}

	lRepos := make(map[string]*GithubRepoComparable)
	for reponame, lRepo := range local.Repositories() {
		writers := make([]string, 0)
		for _, w := range lRepo.Data.Writers {
			writers = append(writers, slugify.Make(w))
		}
		// add the team owner's name ;-)
		if lRepo.Owner != nil {
			writers = append(writers, slugify.Make(*lRepo.Owner))
		}
		readers := make([]string, 0)
		for _, r := range lRepo.Data.Readers {
			readers = append(readers, slugify.Make(r))
		}

		// special case for the Goliac "teams" repo
		if reponame == teamsreponame {
			for teamname := range local.Teams() {
				writers = append(writers, slugify.Make(teamname)+"-owners")
			}
		}

		lRepos[reponame] = &GithubRepoComparable{
			IsPublic:   lRepo.Data.IsPublic,
			IsArchived: lRepo.Data.IsArchived,
			Readers:    readers,
			Writers:    writers,
		}
	}

	// now we compare local (slugTeams) and remote (rTeams)

	compareRepos := func(lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) bool {
		if lRepo.IsArchived != rRepo.IsArchived {
			return false
		}
		if lRepo.IsPublic != rRepo.IsPublic {
			return false
		}

		if res, _, _ := StringArrayEquivalent(lRepo.Readers, rRepo.Readers); res == false {
			return false
		}

		if res, _, _ := StringArrayEquivalent(lRepo.Writers, rRepo.Writers); res == false {
			return false
		}
		return true
	}

	onAdded := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		// CREATE repository
		r.CreateRepository(dryrun, remote, reponame, reponame, lRepo.Writers, lRepo.Readers, lRepo.IsPublic)
	}

	onRemoved := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		r.DeleteRepository(dryrun, remote, reponame)
	}

	onChanged := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		// reconciliate repositories public/private
		if lRepo.IsPublic != rRepo.IsPublic {
			// UPDATE private repository
			r.UpdateRepositoryUpdatePrivate(dryrun, remote, reponame, !lRepo.IsPublic)
		}

		// reconciliate repositories archived
		if lRepo.IsArchived != rRepo.IsArchived {
			// UPDATE archived repository
			r.UpdateRepositoryUpdateArchived(dryrun, remote, reponame, lRepo.IsArchived)
		}

		fmt.Println(lRepo.Readers, rRepo.Readers)
		if res, readToRemove, readToAdd := StringArrayEquivalent(lRepo.Readers, rRepo.Readers); res == false {
			for _, teamSlug := range readToAdd {
				r.UpdateRepositoryAddTeamAccess(dryrun, remote, reponame, teamSlug, "pull")
			}
			for _, teamSlug := range readToRemove {
				r.UpdateRepositoryRemoveTeamAccess(dryrun, remote, reponame, teamSlug)
			}
		}

		if res, writeToRemove, writeToAdd := StringArrayEquivalent(lRepo.Writers, rRepo.Writers); res == false {
			for _, teamSlug := range writeToAdd {
				r.UpdateRepositoryAddTeamAccess(dryrun, remote, reponame, teamSlug, "push")
			}
			for _, teamSlug := range writeToRemove {
				r.UpdateRepositoryRemoveTeamAccess(dryrun, remote, reponame, teamSlug)
			}
		}

	}

	CompareEntities(lRepos, rRepos, compareRepos, onAdded, onRemoved, onChanged)

	return nil
}

func (r *GoliacReconciliatorImpl) AddUserToOrg(dryrun bool, remote *MutableGoliacRemoteImpl, ghuserid string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_user_to_org"}).Infof("ghusername: %s", ghuserid)
	remote.AddUserToOrg(ghuserid)
	if !dryrun && r.executor != nil {
		r.executor.AddUserToOrg(ghuserid)
	}
}

func (r *GoliacReconciliatorImpl) RemoveUserFromOrg(dryrun bool, remote *MutableGoliacRemoteImpl, ghuserid string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "remove_user_from_org"}).Infof("ghusername: %s", ghuserid)
	remote.RemoveUserFromOrg(ghuserid)
	if !dryrun && r.executor != nil {
		if r.repoconfig.DestructiveOperations.AllowDestructiveUsers {
			r.executor.RemoveUserFromOrg(ghuserid)
		}
	}
}

func (r *GoliacReconciliatorImpl) CreateTeam(dryrun bool, remote *MutableGoliacRemoteImpl, teamname string, description string, members []string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "create_team"}).Infof("teamname: %s, members: %s", teamname, strings.Join(members, ","))
	remote.CreateTeam(teamname, description, members)
	if !dryrun && r.executor != nil {
		r.executor.CreateTeam(teamname, description, members)
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamAddMember(dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, username string, role string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_add_member"}).Infof("teamslug: %s, username: %s, role: %s", teamslug, username, role)
	remote.UpdateTeamAddMember(teamslug, username, "member")
	if !dryrun && r.executor != nil {
		r.executor.UpdateTeamAddMember(teamslug, username, "member")
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamRemoveMember(dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, username string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_remove_member"}).Infof("teamslug: %s, username: %s", teamslug, username)
	remote.UpdateTeamRemoveMember(teamslug, username)
	if !dryrun && r.executor != nil {
		r.executor.UpdateTeamRemoveMember(teamslug, username)
	}
}
func (r *GoliacReconciliatorImpl) DeleteTeam(dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_team"}).Infof("teamslug: %s", teamslug)
	remote.DeleteTeam(teamslug)
	if !dryrun && r.executor != nil {
		if r.repoconfig.DestructiveOperations.AllowDestructiveTeams {
			r.executor.DeleteTeam(teamslug)
		}
	}
}
func (r *GoliacReconciliatorImpl) CreateRepository(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, descrition string, writers []string, readers []string, public bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "create_repository"}).Infof("repositoryname: %s, readers: %s, writers: %s, public: %v", reponame, strings.Join(readers, ","), strings.Join(writers, ","), public)
	remote.CreateRepository(reponame, reponame, writers, readers, public)
	if !dryrun && r.executor != nil {
		r.executor.CreateRepository(reponame, reponame, writers, readers, public)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryAddTeamAccess(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_add_team"}).Infof("repositoryname: %s, teamslug: %s, permission: %s", reponame, teamslug, permission)
	remote.UpdateRepositoryAddTeamAccess(reponame, teamslug, permission)
	if !dryrun && r.executor != nil {
		r.executor.UpdateRepositoryAddTeamAccess(reponame, teamslug, permission)
	}
}

func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdateTeamAccess(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_team"}).Infof("repositoryname: %s, teamslug:%s, permission: %s", reponame, teamslug, permission)
	remote.UpdateRepositoryUpdateTeamAccess(reponame, teamslug, permission)
	if !dryrun && r.executor != nil {
		r.executor.UpdateRepositoryUpdateTeamAccess(reponame, teamslug, permission)
	}

}
func (r *GoliacReconciliatorImpl) UpdateRepositoryRemoveTeamAccess(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_remove_team"}).Infof("repositoryname: %s, teamslug:%s", reponame, teamslug)
	remote.UpdateRepositoryRemoveTeamAccess(reponame, teamslug)
	if !dryrun && r.executor != nil {
		r.executor.UpdateRepositoryRemoveTeamAccess(reponame, teamslug)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRepository(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository"}).Infof("repositoryname: %s", reponame)
	remote.DeleteRepository(reponame)
	if !dryrun && r.executor != nil {
		if r.repoconfig.DestructiveOperations.AllowDestructiveRepositories {
			r.executor.DeleteRepository(reponame)
		}
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdatePrivate(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, private bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_private"}).Infof("repositoryname: %s private:%v", reponame, private)
	remote.UpdateRepositoryUpdatePrivate(reponame, private)
	if !dryrun && r.executor != nil {
		r.executor.UpdateRepositoryUpdatePrivate(reponame, private)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdateArchived(dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, archived bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_archived"}).Infof("repositoryname: %s archived:%v", reponame, archived)
	remote.UpdateRepositoryUpdateArchived(reponame, archived)
	if !dryrun && r.executor != nil {
		r.executor.UpdateRepositoryUpdateArchived(reponame, archived)
	}
}
func (r *GoliacReconciliatorImpl) Begin(dryrun bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Infof("reconciliation begin")
	if !dryrun && r.executor != nil {
		r.executor.Begin()
	}
}
func (r *GoliacReconciliatorImpl) Rollback(dryrun bool, err error) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Infof("reconciliation rollback")
	if !dryrun && r.executor != nil {
		r.executor.Rollback(err)
	}
}
func (r *GoliacReconciliatorImpl) Commit(dryrun bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Infof("reconciliation commit")
	if !dryrun && r.executor != nil {
		r.executor.Commit()
	}
}
