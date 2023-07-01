package github

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
)

type GithubCommandUpdateRepositoryUpdatePrivate struct {
	client   GitHubClient
	reponame string
	private  bool
}

func NewGithubCommandUpdateRepositoryUpdatePrivate(client GitHubClient, reponame string, private bool) GithubCommand {
	return &GithubCommandUpdateRepositoryUpdatePrivate{
		client:   client,
		reponame: reponame,
		private:  private,
	}
}

func (g *GithubCommandUpdateRepositoryUpdatePrivate) Apply() error {
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#update-a-repository
	_, err := g.client.CallRestAPI(
		fmt.Sprintf("repos/%s/%s", config.Config.GithubAppOrganization, g.reponame),
		"PATCH",
		map[string]interface{}{"private": g.private},
	)
	if err != nil {
		return err
	}
	return nil
}
