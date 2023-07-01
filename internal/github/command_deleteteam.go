package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandDeleteTeam struct {
	client   GitHubClient
	teamslug string
}

func NewGithubCommandDeleteTeam(client GitHubClient, teamslug string) GithubCommand {
	return &GithubCommandDeleteTeam{
		client:   client,
		teamslug: teamslug,
	}
}

func (g *GithubCommandDeleteTeam) Apply() error {
	// delete team
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#delete-a-team
	_, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/teams/%s", config.Config.GithubAppOrganization, g.teamslug),
		"DELETE",
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}
