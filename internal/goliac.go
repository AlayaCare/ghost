package internal

import (
	"fmt"
	"strings"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/github"
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

	// List Repsotiories that are managed by goliac
	ListManagedRepositories() ([]*entity.Repository, error)
}

type GoliacImpl struct {
	local        GoliacLocal
	githubClient github.GitHubClient
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

	return &GoliacImpl{
		local:        NewGoliacLocalImpl(),
		githubClient: githubClient,
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
			return fmt.Errorf("unable to clone: %v", err)
		}
		errs, warns = g.local.LoadAndValidate()
	} else {
		// Local
		fs := afero.NewOsFs()
		g.local.LoadAndValidateLocal(fs, repositoryUrl)
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
	return nil
}

// List Repsotiories that are managed by goliac
func (g *GoliacImpl) ListManagedRepositories() ([]*entity.Repository, error) {
	return nil, nil
}
