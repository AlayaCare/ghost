package internal

import (
	"github.com/Alayacare/goliac/internal/config"
	"github.com/sirupsen/logrus"
)

/**
 * Each command/mutation we want to perform will be isloated into a GithubCommand
 * object, so we can regroup all of them to apply (or cancel) them in batch
 */
type GithubCommand interface {
	Apply()
}

/*
 * GithubBatchExecutor will collects all commands to apply
 * if there the number of changes to apply is not too big, it will apply on the `Commit()`
 * Usage:
 * gal := NewGithubBatchExecutor(client)
 * gal.Begin()
 * gal.Create...
 * gal.Update...
 * ...
 * gal.Commit()
 */
type GithubBatchExecutor struct {
	client   ReconciliatorExecutor
	commands []GithubCommand
}

func NewGithubBatchExecutor(client ReconciliatorExecutor) *GithubBatchExecutor {
	gal := GithubBatchExecutor{
		client:   client,
		commands: make([]GithubCommand, 0),
	}
	return &gal
}

func (g *GithubBatchExecutor) AddUserToOrg(ghuserid string) {
	g.commands = append(g.commands, &GithubCommandAddUserToOrg{
		client:   g.client,
		ghuserid: ghuserid,
	})
}

func (g *GithubBatchExecutor) RemoveUserFromOrg(ghuserid string) {
	g.commands = append(g.commands, &GithubCommandAddUserToOrg{
		client:   g.client,
		ghuserid: ghuserid,
	})
}

func (g *GithubBatchExecutor) CreateTeam(teamname string, description string, members []string) {
	g.commands = append(g.commands, &GithubCommandCreateTeam{
		client:      g.client,
		teamname:    teamname,
		description: description,
		members:     members,
	})
}

// role = member or maintainer (usually we use member)
func (g *GithubBatchExecutor) UpdateTeamAddMember(teamslug string, username string, role string) {
	g.commands = append(g.commands, &GithubCommandUpdateTeamAddMember{
		client:   g.client,
		teamslug: teamslug,
		member:   username,
		role:     role,
	})
}

func (g *GithubBatchExecutor) UpdateTeamRemoveMember(teamslug string, username string) {
	g.commands = append(g.commands, &GithubCommandUpdateTeamRemoveMember{
		client:   g.client,
		teamslug: teamslug,
		member:   username,
	})
}

func (g *GithubBatchExecutor) DeleteTeam(teamslug string) {
	g.commands = append(g.commands, &GithubCommandDeleteTeam{
		client:   g.client,
		teamslug: teamslug,
	})
}

func (g *GithubBatchExecutor) CreateRepository(reponame string, description string, writers []string, readers []string, public bool) {
	g.commands = append(g.commands, &GithubCommandCreateRepository{
		client:      g.client,
		reponame:    reponame,
		description: description,
		readers:     readers,
		writers:     writers,
		public:      public,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryAddTeamAccess{
		client:     g.client,
		reponame:   reponame,
		teamslug:   teamslug,
		permission: permission,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryUpdateTeamAccess{
		client:     g.client,
		reponame:   reponame,
		teamslug:   teamslug,
		permission: permission,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryRemoveTeamAccess{
		client:   g.client,
		reponame: reponame,
		teamslug: teamslug,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryUpdatePrivate(reponame string, private bool) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryUpdatePrivate{
		client:   g.client,
		reponame: reponame,
		private:  private,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryUpdateArchived(reponame string, archived bool) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryUpdateArchived{
		client:   g.client,
		reponame: reponame,
		archived: archived,
	})
}

func (g *GithubBatchExecutor) DeleteRepository(reponame string) {
	g.commands = append(g.commands, &GithubCommandDeleteRepository{
		client:   g.client,
		reponame: reponame,
	})
}

func (g *GithubBatchExecutor) Begin() {
	g.commands = make([]GithubCommand, 0)
}
func (g *GithubBatchExecutor) Rollback(error) {
	g.commands = make([]GithubCommand, 0)
}
func (g *GithubBatchExecutor) Commit() {
	if len(g.commands) > config.Config.MaxChangesetsPerBatch {
		logrus.Errorf("More than %d changesets to apply (total of %d), this is suspicious. Aborting", config.Config.MaxChangesetsPerBatch, len(g.commands))
		return
	}
	for _, c := range g.commands {
		c.Apply()
	}
	g.commands = make([]GithubCommand, 0)
}

type GithubCommandAddUserToOrg struct {
	client   ReconciliatorExecutor
	ghuserid string
}

func (g *GithubCommandAddUserToOrg) Apply() {
	g.client.AddUserToOrg(g.ghuserid)
}

type GithubCommandCreateRepository struct {
	client      ReconciliatorExecutor
	reponame    string
	description string
	writers     []string
	readers     []string
	public      bool
}

func (g *GithubCommandCreateRepository) Apply() {
	g.client.CreateRepository(g.reponame, g.description, g.writers, g.readers, g.public)
}

type GithubCommandCreateTeam struct {
	client      ReconciliatorExecutor
	teamname    string
	description string
	members     []string
}

func (g *GithubCommandCreateTeam) Apply() {
	g.client.CreateTeam(g.teamname, g.description, g.members)
}

type GithubCommandDeleteRepository struct {
	client   ReconciliatorExecutor
	reponame string
}

func (g *GithubCommandDeleteRepository) Apply() {
	g.client.DeleteRepository(g.reponame)
}

type GithubCommandDeleteTeam struct {
	client   ReconciliatorExecutor
	teamslug string
}

func (g *GithubCommandDeleteTeam) Apply() {
	g.client.DeleteTeam(g.teamslug)
}

type GithubCommandRemoveUserFromOrg struct {
	client   ReconciliatorExecutor
	ghuserid string
}

func (g *GithubCommandRemoveUserFromOrg) Apply() {
	g.client.RemoveUserFromOrg(g.ghuserid)
}

type GithubCommandUpdateRepositoryRemoveTeamAccess struct {
	client   ReconciliatorExecutor
	reponame string
	teamslug string
}

func (g *GithubCommandUpdateRepositoryRemoveTeamAccess) Apply() {
	g.client.UpdateRepositoryRemoveTeamAccess(g.reponame, g.teamslug)
}

type GithubCommandUpdateRepositoryAddTeamAccess struct {
	client     ReconciliatorExecutor
	reponame   string
	teamslug   string
	permission string
}

func (g *GithubCommandUpdateRepositoryAddTeamAccess) Apply() {
	g.client.UpdateRepositoryAddTeamAccess(g.reponame, g.teamslug, g.permission)
}

type GithubCommandUpdateRepositoryUpdateTeamAccess struct {
	client     ReconciliatorExecutor
	reponame   string
	teamslug   string
	permission string
}

func (g *GithubCommandUpdateRepositoryUpdateTeamAccess) Apply() {
	g.client.UpdateRepositoryUpdateTeamAccess(g.reponame, g.teamslug, g.permission)
}

type GithubCommandUpdateRepositoryUpdateArchived struct {
	client   ReconciliatorExecutor
	reponame string
	archived bool
}

func (g *GithubCommandUpdateRepositoryUpdateArchived) Apply() {
	g.client.UpdateRepositoryUpdateArchived(g.reponame, g.archived)
}

type GithubCommandUpdateRepositoryUpdatePrivate struct {
	client   ReconciliatorExecutor
	reponame string
	private  bool
}

func (g *GithubCommandUpdateRepositoryUpdatePrivate) Apply() {
	g.client.UpdateRepositoryUpdatePrivate(g.reponame, g.private)
}

type GithubCommandUpdateTeamAddMember struct {
	client   ReconciliatorExecutor
	teamslug string
	member   string
	role     string
}

func (g *GithubCommandUpdateTeamAddMember) Apply() {
	g.client.UpdateTeamAddMember(g.teamslug, g.member, g.role)
}

type GithubCommandUpdateTeamRemoveMember struct {
	client   ReconciliatorExecutor
	teamslug string
	member   string
}

func (g *GithubCommandUpdateTeamRemoveMember) Apply() {
	g.client.UpdateTeamRemoveMember(g.teamslug, g.member)
}
