package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandUpdateTeamRemoveMember struct {
	client   GitHubClient
	teamslug string
	member   string
}

func NewGithubCommandUpdateTeamRemoveMember(client GitHubClient, teamslug string, member string) GithubCommand {
	return &GithubCommandUpdateTeamAddMember{
		client:   client,
		teamslug: teamslug,
		member:   member,
	}
}

func (g *GithubCommandUpdateTeamRemoveMember) Apply() error {
	// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
	_, err := g.client.CallRestAPIWithBody(
		fmt.Sprintf("orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, g.teamslug, g.member),
		"DELETE",
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}
