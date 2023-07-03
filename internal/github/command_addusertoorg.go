package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandAddUserToOrg struct {
	client   GitHubClient
	ghuserid string
}

func NewGithubCommandAddUserToOrg(client GitHubClient, ghuserid string) GithubCommand {
	return &GithubCommandAddUserToOrg{
		client:   client,
		ghuserid: ghuserid,
	}
}

func (g *GithubCommandAddUserToOrg) Apply() error {
	// add member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#create-a-team
	_, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/memberships/%s", config.Config.GithubAppOrganization, g.ghuserid),
		"PUT",
		map[string]interface{}{"role": "member"},
	)
	if err != nil {
		return fmt.Errorf("failed to add user to org: %v", err)
	}

	return nil
}
