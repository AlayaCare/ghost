package internal

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Alayacare/goliac/internal/entity"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/afero"
)

/*
 * GoliacLocal
 * This interface is used to load the goliac organization from a local directory
 * and mount it in memory
 */
type GoliacLocal interface {
	// Load and Validate from a github repository
	LoadAndValidate(accesstoken, repositoryUrl, branch string) ([]error, []error)
	// Load and Validate from a local directory
	LoadAndValidateLocal(fs afero.Fs, path string) ([]error, []error)
	Teams() map[string]*entity.Team
	Repositories() map[string]*entity.Repository
	Users() map[string]*entity.User
	ExternalUsers() map[string]*entity.User
}

type GoliacLocalImpl struct {
	teams         map[string]*entity.Team
	repositories  map[string]*entity.Repository
	users         map[string]*entity.User
	externalUsers map[string]*entity.User
}

func NewGoliacLocalImpl() GoliacLocal {
	return &GoliacLocalImpl{
		teams:         map[string]*entity.Team{},
		repositories:  map[string]*entity.Repository{},
		users:         map[string]*entity.User{},
		externalUsers: map[string]*entity.User{},
	}
}

func (g *GoliacLocalImpl) Teams() map[string]*entity.Team {
	return g.teams
}

func (g *GoliacLocalImpl) Repositories() map[string]*entity.Repository {
	return g.repositories
}

func (g *GoliacLocalImpl) Users() map[string]*entity.User {
	return g.users
}

func (g *GoliacLocalImpl) ExternalUsers() map[string]*entity.User {
	return g.externalUsers
}

/*
 * Load the goliac organization from Github
 * - clone the repository
 * - read the organization files
 * - validate the organization
 * Parameters:
 * - repositoryUrl: the URL of the repository to clone
 * - branch: the branch to checkout
 */
func (g *GoliacLocalImpl) LoadAndValidate(accesstoken, repositoryUrl, branch string) ([]error, []error) {
	// create a temp directory
	tmpDir, err := ioutil.TempDir("", "goliac")
	if err != nil {
		return []error{err}, []error{}
	}
	defer os.RemoveAll(tmpDir) // clean up

	repo, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL: repositoryUrl,
		Auth: &http.BasicAuth{
			Username: "x-access-token", // This can be anything except an empty string
			Password: accesstoken,
		},
	})
	if err != nil {
		return []error{err}, []error{}
	}

	// checkout the branch
	w, err := repo.Worktree()
	if err != nil {
		return []error{err}, []error{}
	}
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/remotes/origin/" + branch),
	})
	if err != nil {
		return []error{err}, []error{}
	}

	// read the organization files

	fs := afero.NewOsFs()

	errs, warns := g.LoadAndValidateLocal(fs, tmpDir)

	return errs, warns
}

/**
 * readOrganization reads all the organization files and returns
 * - a slice of errors that must stop the vlidation process
 * - a slice of warning that must not stop the validation process
 */
func (g *GoliacLocalImpl) LoadAndValidateLocal(fs afero.Fs, orgDirectory string) ([]error, []error) {
	errors := []error{}
	warnings := []error{}

	// Parse all the users in the <orgDirectory>/protected-users directory
	protectedUsers, errs, warns := entity.ReadUserDirectory(fs, filepath.Join(orgDirectory, "users", "protected"))
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.users = protectedUsers

	// Parse all the users in the <orgDirectory>/org-users directory
	orgUsers, errs, warns := entity.ReadUserDirectory(fs, filepath.Join(orgDirectory, "users", "org"))
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)

	// not users? not good
	if orgUsers == nil {
		return errors, warnings
	}

	for k, v := range orgUsers {
		g.users[k] = v
	}

	// Parse all the users in the <orgDirectory>/external-users directory
	externalUsers, errs, warns := entity.ReadUserDirectory(fs, filepath.Join(orgDirectory, "users", "external"))
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.externalUsers = externalUsers

	// Parse all the teams in the <orgDirectory>/teams directory
	teams, errs, warns := entity.ReadTeamDirectory(fs, filepath.Join(orgDirectory, "teams"), g.users)
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.teams = teams

	// Parse all repositories in the <orgDirectory>/teams/<teamname> directories
	repos, errs, warns := entity.ReadRepositories(fs, filepath.Join(orgDirectory, "archived"), filepath.Join(orgDirectory, "teams"), g.teams, g.externalUsers)
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.repositories = repos

	return errors, warnings
}
