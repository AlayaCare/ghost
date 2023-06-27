package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/Alayacare/goliac/internal/entity"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

	GetCodeOwnersFileContent() ([]byte, error)
	SaveCodeOwnersFileContent([]byte) error
	Close()
}

type GoliacLocalImpl struct {
	teams         map[string]*entity.Team
	repositories  map[string]*entity.Repository
	users         map[string]*entity.User
	externalUsers map[string]*entity.User
	repo          *git.Repository
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

func (g *GoliacLocalImpl) Close() {
	if g.repo != nil {
		w, err := g.repo.Worktree()
		if err == nil {
			os.RemoveAll(w.Filesystem.Root())
		}
	}
	g.repo = nil
}

func (g *GoliacLocalImpl) GetCodeOwnersFileContent() ([]byte, error) {
	if g.repo == nil {
		return nil, fmt.Errorf("git repository not cloned")
	}
	w, err := g.repo.Worktree()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path.Join(w.Filesystem.Root(), ".github", "CODEOWNERS"))
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return content, nil

}
func (g *GoliacLocalImpl) SaveCodeOwnersFileContent(content []byte) error {
	if g.repo == nil {
		return fmt.Errorf("git repository not cloned")
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}

	ioutil.WriteFile(path.Join(w.Filesystem.Root(), ".github", "CODEOWNERS"), content, 0644)

	_, err = w.Add(path.Join(".github", "CODEOWNERS"))
	if err != nil {
		return err
	}

	commit, err := w.Commit("Added example.txt", &git.CommitOptions{
		Author: &object.Signature{
			Name: "Goliac",
			//			Email: "goliac@alayacare.com",
			When: time.Now(),
		},
	})

	if err != nil {
		return err
	}

	_, err = g.repo.CommitObject(commit)
	if err != nil {
		return err
	}

	err = g.repo.Push(&git.PushOptions{})

	return err
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
	g.repo = repo

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
