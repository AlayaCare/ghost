package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandUpdateRepositorySetTeamAccess struct {
	client     GitHubClient
	reponame   string
	teamslug   string
	permission string
}

func NewGithubCommandUpdateRepositorySetTeamAccess(client GitHubClient, reponame string, teamslug string, permission string) GithubCommand {
	return &GithubCommandUpdateRepositorySetTeamAccess{
		client:     client,
		reponame:   reponame,
		teamslug:   teamslug,
		permission: permission,
	}
}

func (g *GithubCommandUpdateRepositorySetTeamAccess) Apply() error {
	// update member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
	_, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, g.teamslug, config.Config.GithubAppOrganization, g.reponame),
		"PUT",
		map[string]interface{}{"permission": g.permission},
	)
	if err != nil {
		return err
	}
	return nil
}
