package internal

import (
	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/sirupsen/logrus"
)

/*
 * GithubApplyListener will collects all commands to apply
 * if there the number of changes to apply is not too big, it will apply on the `Commit()`
 * Usage:
 * gal := NewGithubApplyListener(client)
 * gal.Begin()
 * gal.Create...
 * gal.Update...
 * ...
 * gal.Commit()
 */
type GithubApplyListener struct {
	client   github.GitHubClient
	commands []github.GithubCommand
}

func NewGithubApplyListener(client github.GitHubClient) *GithubApplyListener {
	gal := GithubApplyListener{
		client:   client,
		commands: make([]github.GithubCommand, 0),
	}
	return &gal
}

func (g *GithubApplyListener) CreateTeam(teamname string, description string, members []string) {
	g.commands = append(g.commands, github.NewGithubCommandCreateTeam(g.client, teamname, description, members))
}

// role = member or maintainer (usually we use member)
func (g *GithubApplyListener) UpdateTeamAddMember(teamslug string, username string, role string) {
	g.commands = append(g.commands, github.NewGithubCommandUpdateTeamAddMember(g.client, teamslug, username, role))
}
func (g *GithubApplyListener) UpdateTeamRemoveMember(teamslug string, username string) {
	g.commands = append(g.commands, github.NewGithubCommandUpdateTeamRemoveMember(g.client, teamslug, username))
}
func (g *GithubApplyListener) DeleteTeam(teamslug string) {
	// NOOP: we dont want to delete teams
	//g.commands = append(g.commands, github.NewGithubCommandDeleteTeam(g.client, teamslug))
}

func (g *GithubApplyListener) CreateRepository(reponame string, description string, writers []string, readers []string, public bool) {
	g.commands = append(g.commands, github.NewGithubCommandCreateRepository(g.client, reponame, description, writers, readers, public))
}
func (g *GithubApplyListener) UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string) {
	g.commands = append(g.commands, github.NewGithubCommandUpdateRepositorySetTeamAccess(g.client, reponame, teamslug, permission))
}

func (g *GithubApplyListener) UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) {
	g.commands = append(g.commands, github.NewGithubCommandUpdateRepositorySetTeamAccess(g.client, reponame, teamslug, permission))
}
func (g *GithubApplyListener) UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string) {
	g.commands = append(g.commands, github.NewGithubCommandUpdateRepositoryRemoveTeamAccess(g.client, reponame, teamslug))
}
func (g *GithubApplyListener) UpdateRepositoryUpdatePrivate(reponame string, private bool) {
	g.commands = append(g.commands, github.NewGithubCommandUpdateRepositoryUpdatePrivate(g.client, reponame, private))
}
func (g *GithubApplyListener) UpdateRepositoryUpdateArchived(reponame string, archived bool) {
	g.commands = append(g.commands, github.NewGithubCommandUpdateRepositoryUpdateArchived(g.client, reponame, archived))
}
func (g *GithubApplyListener) DeleteRepository(reponame string) {
	// NOOP: we dont want to delete repositories
	//g.commands = append(g.commands,github.NewGithubCommandDeleteRepository(g.client,reponame))
}
func (g *GithubApplyListener) Begin() {
	g.commands = make([]github.GithubCommand, 0)
}
func (g *GithubApplyListener) Rollback(error) {
	g.commands = make([]github.GithubCommand, 0)
}
func (g *GithubApplyListener) Commit() {
	if len(g.commands) > config.Config.MaxChangesetsPerBatch {
		logrus.Errorf("More than %d changesets to apply (total of %d), this is suspicious. Aborting", config.Config.MaxChangesetsPerBatch, len(g.commands))
		return
	}
	for _, c := range g.commands {
		err := c.Apply()
		if err != nil {
			logrus.Error(err)
		}
	}
	g.commands = make([]github.GithubCommand, 0)
}
