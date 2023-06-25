package github

import (
	"encoding/json"
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandCreateTeam struct {
	client      GitHubClient
	teamname    string
	description string
	members     []string
}

func NewGithubCommandCreateTeam(client GitHubClient, teamname string, description string, members []string) GithubCommand {
	return &GithubCommandCreateTeam{
		client:      client,
		teamname:    teamname,
		description: description,
		members:     members,
	}
}

type CreateTeamResponse struct {
	Name string
	Slug string
}

func (g *GithubCommandCreateTeam) Apply() error {
	// create team
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#create-a-team
	body, err := g.client.CallRestAPIWithBody(
		fmt.Sprintf("/orgs/%s/teams", config.Config.GithubAppOrganization),
		"POST",
		map[string]interface{}{"name": g.teamname, "description": g.description, "privacy": "closed"},
	)
	if err != nil {
		return err
	}
	var res CreateTeamResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return err
	}

	// add members
	for _, member := range g.members {
		// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
		_, err := g.client.CallRestAPIWithBody(
			fmt.Sprintf("orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, res.Slug, member),
			"PUT",
			map[string]interface{}{"role": "member"},
		)
		if err != nil {
			return err
		}
	}
	return nil
}
