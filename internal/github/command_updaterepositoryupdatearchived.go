package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandUpdateRepositoryUpdateArchived struct {
	client   GitHubClient
	reponame string
	archived bool
}

func NewGithubCommandUpdateRepositoryUpdateArchived(client GitHubClient, reponame string, archived bool) GithubCommand {
	return &GithubCommandUpdateRepositoryUpdateArchived{
		client:   client,
		reponame: reponame,
		archived: archived,
	}
}

func (g *GithubCommandUpdateRepositoryUpdateArchived) Apply() error {
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#update-a-repository
	_, err := g.client.CallRestAPI(
		fmt.Sprintf("repos/%s/%s", config.Config.GithubAppOrganization, g.reponame),
		"PATCH",
		map[string]interface{}{"archived": g.archived},
	)
	if err != nil {
		return err
	}
	return nil
}
