package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandUpdateTeamAddMember struct {
	client   GitHubClient
	teamslug string
	member   string
	role     string
}

func NewGithubCommandUpdateTeamAddMember(client GitHubClient, teamslug string, member string, role string) GithubCommand {
	return &GithubCommandUpdateTeamAddMember{
		client:   client,
		teamslug: teamslug,
		member:   member,
		role:     role,
	}
}

func (g *GithubCommandUpdateTeamAddMember) Apply() error {
	// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
	_, err := g.client.CallRestAPI(
		fmt.Sprintf("orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, g.teamslug, g.member),
		"PUT",
		map[string]interface{}{"role": g.role},
	)
	if err != nil {
		return err
	}
	return nil
}
