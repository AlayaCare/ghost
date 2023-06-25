package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandDeleteRepository struct {
	client   GitHubClient
	reponame string
}

func NewGithubCommandDeleteRepository(client GitHubClient, reponame string) GithubCommand {
	return &GithubCommandDeleteRepository{
		client:   client,
		reponame: reponame,
	}
}

func (g *GithubCommandDeleteRepository) Apply() error {
	// delete repo
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#delete-a-repository
	_, err := g.client.CallRestAPIWithBody(
		fmt.Sprintf("/orgs/%s/%s", config.Config.GithubAppOrganization, g.reponame),
		"DELETE",
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}
