package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandUpdateRepositoryRemoveTeamAccess struct {
	client   GitHubClient
	reponame string
	teamslug string
}

func NewGithubCommandUpdateRepositoryRemoveTeamAccess(client GitHubClient, reponame string, teamslug string) GithubCommand {
	return &GithubCommandUpdateRepositorySetTeamAccess{
		client:   client,
		reponame: reponame,
		teamslug: teamslug,
	}
}

func (g *GithubCommandUpdateRepositoryRemoveTeamAccess) Apply() error {
	// delete member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#remove-a-repository-from-a-team
	_, err := g.client.CallRestAPI(
		fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, g.teamslug, config.Config.GithubAppOrganization, g.reponame),
		"DELETE",
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}
