package internal

import (
	"fmt"
	"strings"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/Alayacare/goliac/internal/usersync"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

/*
 * Goliac is the main interface of the application.
 * It is used to load and validate a goliac repository and apply it to github.
 */
type Goliac interface {
	// Git clone (if repositoryUrl is https://...), load and validate a goliac repository
	LoadAndValidateGoliacOrganization(repositoryUrl, branch string) error

	// You need to call LoadAndValidategoliacOrganization before calling this function
	ApplyToGithub(dryrun bool, teamreponame string, branch string) error

	// You dont need to call LoadAndValidategoliacOrganization before calling this function
	UsersUpdate(repositoryUrl, branch string) error

	// to close the clone git repository (if you called LoadAndValidateGoliacOrganization)
	Close()
}

type GoliacImpl struct {
	local        GoliacLocal
	remote       GoliacRemoteExecutor
	githubClient github.GitHubClient
	repoconfig   *config.RepositoryConfig
}

func NewGoliacImpl() (Goliac, error) {
	githubClient, err := github.NewGitHubClientImpl(
		config.Config.GithubServer,
		config.Config.GithubAppOrganization,
		config.Config.GithubAppID,
		config.Config.GithubAppPrivateKeyFile,
	)

	if err != nil {
		return nil, err
	}

	remote := NewGoliacRemoteImpl(githubClient)

	return &GoliacImpl{
		local:        NewGoliacLocalImpl(),
		githubClient: githubClient,
		remote:       remote,
		repoconfig:   &config.RepositoryConfig{},
	}, nil
}

func (g *GoliacImpl) LoadAndValidateGoliacOrganization(repositoryUrl, branch string) error {
	errs := []error{}
	warns := []entity.Warning{}
	if strings.HasPrefix(repositoryUrl, "https://") || strings.HasPrefix(repositoryUrl, "git@") {
		accessToken, err := g.githubClient.GetAccessToken()
		if err != nil {
			return err
		}

		err = g.local.Clone(accessToken, repositoryUrl, branch)
		if err != nil {
			return fmt.Errorf("unable to clone: %v", err)
		}
		err, repoconfig := g.local.LoadRepoConfig()
		if err != nil {
			return fmt.Errorf("unable to read goliac.yaml config file: %v", err)
		}
		g.repoconfig = repoconfig

		errs, warns = g.local.LoadAndValidate()
	} else {
		// Local
		fs := afero.NewOsFs()
		errs, warns = g.local.LoadAndValidateLocal(fs, repositoryUrl)
	}

	for _, warn := range warns {
		logrus.Warn(warn)
	}
	if errs != nil && len(errs) != 0 {
		for _, err := range errs {
			logrus.Error(err)
		}
		return fmt.Errorf("Not able to load and validate the goliac organization: see logs")
	}

	return nil
}

func (g *GoliacImpl) ApplyToGithub(dryrun bool, teamreponame string, branch string) error {
	err := g.remote.Load(g.repoconfig)
	if err != nil {
		return fmt.Errorf("Error when fetching data from Github: %v", err)
	}

	ga := NewGithubBatchExecutor(g.remote, g.repoconfig.MaxChangesets)
	reconciliator := NewGoliacReconciliatorImpl(ga, g.repoconfig)

	err = reconciliator.Reconciliate(g.local, g.remote, teamreponame, dryrun)
	if err != nil {
		return fmt.Errorf("Error when reconciliating: %v", err)
	}
	accessToken, err := g.githubClient.GetAccessToken()
	if err != nil {
		return err
	}
	err = g.local.UpdateAndCommitCodeOwners(g.repoconfig, dryrun, accessToken, branch)
	if err != nil {
		return fmt.Errorf("Error when updating and commiting: %v", err)
	}
	return nil
}

func (g *GoliacImpl) UsersUpdate(repositoryUrl, branch string) error {
	accessToken, err := g.githubClient.GetAccessToken()
	if err != nil {
		return err
	}

	err = g.local.Clone(accessToken, repositoryUrl, branch)
	if err != nil {
		return err
	}

	err, repoconfig := g.local.LoadRepoConfig()
	if err != nil {
		return fmt.Errorf("unable to read goliac.yaml config file: %v", err)
	}

	userplugin, found := usersync.GetUserSyncPlugin(g.repoconfig.UserSync.Plugin)
	if found == false {
		return fmt.Errorf("User Sync Plugin %s not found", g.repoconfig.UserSync.Plugin)
	}

	err = g.local.SyncUsersAndTeams(repoconfig, userplugin, false)
	return err
}

func (g *GoliacImpl) Close() {
	g.local.Close()
}
