package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/slugify"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

/*
 * GoliacLocal
 * This interface is used to load the goliac organization from a local directory
 * and mount it in memory
 */
type GoliacLocal interface {
	Clone(accesstoken, repositoryUrl, branch string) error
	// Load and Validate from a github repository
	LoadAndValidate() ([]error, []error)
	UpdateAndCommitCodeOwners(dryrun bool) error
	Close()

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
	repo          *git.Repository
}

func NewGoliacLocalImpl() GoliacLocal {
	return &GoliacLocalImpl{
		teams:         map[string]*entity.Team{},
		repositories:  map[string]*entity.Repository{},
		users:         map[string]*entity.User{},
		externalUsers: map[string]*entity.User{},
		repo:          nil,
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

func (g *GoliacLocalImpl) Clone(accesstoken, repositoryUrl, branch string) error {
	if g.repo != nil {
		g.Close()
	}
	// create a temp directory
	tmpDir, err := ioutil.TempDir("", "goliac")
	if err != nil {
		return err
	}

	repo, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL: repositoryUrl,
		Auth: &http.BasicAuth{
			Username: "x-access-token", // This can be anything except an empty string
			Password: accesstoken,
		},
	})
	if err != nil {
		return err
	}
	g.repo = repo

	// checkout the branch
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/remotes/origin/" + branch),
	})
	if err != nil {
		return err
	}

	return err
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

func (g *GoliacLocalImpl) codeowners_regenerate() string {
	codeowners := "# DO NOT MODIFY THIS FILE MANUALLY\n"

	teamsnames := make([]string, 0)
	for _, t := range g.teams {
		teamsnames = append(teamsnames, t.Metadata.Name)
	}
	sort.Sort(sort.StringSlice(teamsnames))

	for _, t := range teamsnames {
		codeowners += fmt.Sprintf("/org/%s @%s/%s\n", t, config.Config.GithubAppOrganization, slugify.Make(t))
	}

	return codeowners
}

/*
 * UpdateAndCommitCodeOwners will collects all teams definition to update the .github/CODEOWNERS file
 * cf https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners
 */
func (g *GoliacLocalImpl) UpdateAndCommitCodeOwners(dryrun bool) error {
	if g.repo == nil {
		return fmt.Errorf("git repository not cloned")
	}
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}

	file, err := os.Open(path.Join(w.Filesystem.Root(), ".github", "CODEOWNERS"))
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	newContent := g.codeowners_regenerate()

	if string(content) != newContent {
		logrus.Info(".github/CODEOWNERS needs to be regenerated")
		if dryrun {
			return nil
		}

		ioutil.WriteFile(path.Join(w.Filesystem.Root(), ".github", "CODEOWNERS"), []byte(newContent), 0644)

		_, err = w.Add(path.Join(".github", "CODEOWNERS"))
		if err != nil {
			return err
		}

		commit, err := w.Commit("Added example.txt", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Goliac",
				Email: config.Config.GoliacEmail,
				When:  time.Now(),
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
	return nil
}

/*
 * Load the goliac organization from Github (after the repository has been cloned)
 * - read the organization files
 * - validate the organization
 * Parameters:
 * - repositoryUrl: the URL of the repository to clone
 * - branch: the branch to checkout
 */
func (g *GoliacLocalImpl) LoadAndValidate() ([]error, []error) {
	if g.repo == nil {
		return []error{fmt.Errorf("The repository has not been cloned. Did you called .Clone()?")}, []error{}
	}

	// read the organization files
	fs := afero.NewOsFs()

	w, err := g.repo.Worktree()
	if err != nil {
		return []error{err}, []error{}
	}
	rootDir := w.Filesystem.Root()
	errs, warns := g.LoadAndValidateLocal(fs, rootDir)

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
