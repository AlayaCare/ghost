package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandRemoveUserFromOrg struct {
	client   GitHubClient
	ghuserid string
}

func NewGithubCommandRemoveUserFromOrg(client GitHubClient, ghuserid string) GithubCommand {
	return &GithubCommandAddUserToOrg{
		client:   client,
		ghuserid: ghuserid,
	}
}

func (g *GithubCommandRemoveUserFromOrg) Apply() error {
	// add member
	// https://docs.github.com/en/rest/orgs/members?apiVersion=2022-11-28#remove-organization-membership-for-a-user
	_, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/memberships/%s", config.Config.GithubAppOrganization, g.ghuserid),
		"DELETE",
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to remove user from org: %v", err)
	}

	return nil
}
