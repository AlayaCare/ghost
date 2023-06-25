package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandCreateRepository struct {
	client      GitHubClient
	reponame    string
	description string
	writers     []string
	readers     []string
	public      bool
}

func NewGithubCommandCreateRepository(client GitHubClient, reponame string, description string, writers []string, readers []string, public bool) GithubCommand {
	return &GithubCommandCreateRepository{
		client:      client,
		reponame:    reponame,
		description: description,
		readers:     readers,
		writers:     writers,
		public:      public,
	}
}

func (g *GithubCommandCreateRepository) Apply() error {
	// create team
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#create-a-team
	_, err := g.client.CallRestAPIWithBody(
		fmt.Sprintf("/orgs/%s/repos", config.Config.GithubAppOrganization),
		"POST",
		map[string]interface{}{"name": g.reponame, "description": g.description, "private": !g.public},
	)
	if err != nil {
		return err
	}

	// add members
	for _, reader := range g.readers {
		// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
		_, err := g.client.CallRestAPIWithBody(
			fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, reader, config.Config.GithubAppOrganization, g.reponame),
			"PUT",
			map[string]interface{}{"permission": "pull"},
		)
		if err != nil {
			return err
		}
	}
	for _, writer := range g.writers {
		// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
		_, err := g.client.CallRestAPIWithBody(
			fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, writer, config.Config.GithubAppOrganization, g.reponame),
			"PUT",
			map[string]interface{}{"permission": "push"},
		)
		if err != nil {
			return err
		}
	}
	return nil
}
