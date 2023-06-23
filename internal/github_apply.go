package internal

import "github.com/sirupsen/logrus"

type GithubApplyCommand interface {
	Apply() error
}

type GithubApplyListener struct {
	commands []GithubApplyCommand
}

func NewGithubApplyListener() *GithubApplyListener {
	gal := GithubApplyListener{
		commands: make([]GithubApplyCommand, 0),
	}
	return &gal
}

func (g *GithubApplyListener) CreateTeam(teamname string, description string, members []string) {

}
func (g *GithubApplyListener) UpdateTeamAddMember(teamslug string, username string, role string) {

}
func (g *GithubApplyListener) UpdateTeamRemoveMember(teamslug string, username string) {

}
func (g *GithubApplyListener) DeleteTeam(teamslug string) {

}

func (g *GithubApplyListener) CreateRepository(reponame string, descrition string, writers []string, readers []string, public bool) {

}
func (g *GithubApplyListener) UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string) {

}

func (g *GithubApplyListener) UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) {

}
func (g *GithubApplyListener) UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string) {

}
func (g *GithubApplyListener) DeleteRepository(reponame string) {
	// NOOP: we dont want to delete repositories
}
func (g *GithubApplyListener) Begin() {
	g.commands = make([]GithubApplyCommand, 0)
}
func (g *GithubApplyListener) Rollback(error) {
	g.commands = make([]GithubApplyCommand, 0)
}
func (g *GithubApplyListener) Commit() {
	for _, c := range g.commands {
		err := c.Apply()
		if err != nil {
			logrus.Error(err)
		}
	}
	g.commands = make([]GithubApplyCommand, 0)
}
