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
	ApplyToGithub(dryrun bool) error

	// You dont need to call LoadAndValidategoliacOrganization before calling this function
	UsersUpdate(repositoryUrl, branch string) error

	// to close the clone git repository (if you called LoadAndValidateGoliacOrganization)
	Close()
}

type GoliacImpl struct {
	local         GoliacLocal
	remote        GoliacRemote
	githubClient  github.GitHubClient
	reconciliator GoliacReconciliator
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

	reconciliator := NewGoliacReconciliatorImpl()
	ga := NewGithubApplyListener(githubClient)
	reconciliator.AddListener(ga)

	return &GoliacImpl{
		local:         NewGoliacLocalImpl(),
		githubClient:  githubClient,
		remote:        NewGoliacRemoteImpl(githubClient),
		reconciliator: reconciliator,
	}, nil
}

func (g *GoliacImpl) LoadAndValidateGoliacOrganization(repositoryUrl, branch string) error {
	errs := []error{}
	warns := []entity.Warning{}
	if strings.HasPrefix(repositoryUrl, "https://") {
		accessToken, err := g.githubClient.GetAccessToken()
		if err != nil {
			return err
		}

		err = g.local.Clone(accessToken, repositoryUrl, branch)
		if err != nil {
			return err
		}
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

func (g *GoliacImpl) ApplyToGithub(dryrun bool) error {
	err := g.reconciliator.Reconciliate(g.local, g.remote, dryrun)
	if err != nil {
		return err
	}
	err = g.local.UpdateAndCommitCodeOwners(dryrun)
	return err
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

	userplugin, found := usersync.GetUserSyncPlugin(config.Config.UserSyncPlugin)
	if found == false {
		return fmt.Errorf("User Sync Plugin %s not found", config.Config.UserSyncPlugin)
	}

	err = g.local.SyncUsersAndTeams(userplugin, false)
	return err
}

func (g *GoliacImpl) Close() {
	g.local.Close()
}
