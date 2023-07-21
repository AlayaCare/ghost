package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/usersync"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func createBasicStructure(fs afero.Fs, path string) error {
	// Create a fake repository
	err := fs.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, filepath.Join(path, "goliac.yaml"), []byte(`
`), 0644)
	// Create users
	err = fs.MkdirAll(filepath.Join(path, "users/org"), 0755)
	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, filepath.Join(path, "users/org/user1.yaml"), []byte(`
apiVersion: v1
kind: User
metadata:
  name: user1
data:
  githubID: github1
`), 0644)
	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, filepath.Join(path, "users/org/user2.yaml"), []byte(`
apiVersion: v1
kind: User
metadata:
  name: user2
data:
  githubID: github2
`), 0644)
	if err != nil {
		return err
	}

	// Create teams
	err = fs.MkdirAll(filepath.Join(path, "teams/team1"), 0755)
	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, filepath.Join(path, "teams/team1/team.yaml"), []byte(`
apiVersion: v1
kind: Team
metadata:
  name: team1
data:
  owners:
  - user1
  - user2
`), 0644)
	if err != nil {
		return err
	}

	// Create repositories
	err = afero.WriteFile(fs, filepath.Join(path, "projects/repo1.yaml"), []byte(`
apiVersion: v1
kind: Repository
metadata:
  name: repo1
`), 0644)
	if err != nil {
		return err
	}
	return nil
}

func TestRepository(t *testing.T) {

	// happy path
	t.Run("happy path: local directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createBasicStructure(fs, "/tmp/goliac")
		g := NewGoliacLocalImpl()
		errs, warns := g.LoadAndValidateLocal(fs, "/tmp/goliac")

		assert.Equal(t, 0, len(errs))
		assert.Equal(t, 0, len(warns))
	})

	t.Run("happy path: local repository", func(t *testing.T) {
		tmpDirectory, err := ioutil.TempDir("", "goliac")
		assert.Nil(t, err)
		defer os.RemoveAll(tmpDirectory)

		// Initializes a new repository
		r, err := git.PlainInit(tmpDirectory, false)
		assert.Nil(t, err)

		fs := afero.NewOsFs()
		createBasicStructure(fs, tmpDirectory)
		w, err := r.Worktree()
		assert.Nil(t, err)
		_, err = w.Add(".")
		assert.Nil(t, err)

		_, err = w.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "goliac",
				Email: "goliac@alayacare.com",
				When:  time.Now(),
			},
		})
		assert.Nil(t, err)

		// Verify the commit
		_, err = r.Head()
		assert.Nil(t, err)

		g := &GoliacLocalImpl{
			teams:         map[string]*entity.Team{},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			repo:          r,
		}

		errs, warns := g.LoadAndValidate()

		assert.Equal(t, 0, len(errs))
		assert.Equal(t, 0, len(warns))
	})
}

type ScrambleUserSync struct {
}

func (p *ScrambleUserSync) UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error) {
	users := make(map[string]*entity.User)

	// added
	foobar := &entity.User{}
	foobar.ApiVersion = "v1"
	foobar.Kind = "User"
	foobar.Metadata.Name = "foobar"
	foobar.Data.GithubID = "foobar"
	users["foobar"] = foobar

	// updated
	user1 := &entity.User{}
	user1.ApiVersion = "v1"
	user1.Kind = "User"
	user1.Metadata.Name = "user1"
	user1.Data.GithubID = "user1"
	users["user1"] = foobar

	return users, nil
}

type ErroreUserSync struct {
}

func (p *ErroreUserSync) UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error) {
	return nil, fmt.Errorf("unknown error")
}

func TestSyncUsersViaUserPlugin(t *testing.T) {
	t.Run("happy path: noop", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createBasicStructure(fs, "/tmp/goliac")

		removed, added, err := syncUsersViaUserPlugin(&config.RepositoryConfig{}, fs, &usersync.UserSyncPluginNoop{
			Fs: fs,
		}, "/tmp/goliac")

		assert.Nil(t, err)
		assert.Equal(t, 0, len(removed))
		assert.Equal(t, 0, len(added))
	})
	t.Run("happy path: replcae with foobar", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createBasicStructure(fs, "/tmp/goliac")

		removed, added, err := syncUsersViaUserPlugin(&config.RepositoryConfig{}, fs, &ScrambleUserSync{}, "/tmp/goliac")

		assert.Nil(t, err)
		assert.Equal(t, 1, len(removed))
		assert.Equal(t, 2, len(added))
		assert.Equal(t, "/tmp/goliac/users/org/user1.yaml", added[0])
		assert.Equal(t, "/tmp/goliac/users/org/foobar.yaml", added[1])
	})
	t.Run("not happy path: dealing with usersync error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createBasicStructure(fs, "/tmp/goliac")

		_, _, err := syncUsersViaUserPlugin(&config.RepositoryConfig{}, fs, &ErroreUserSync{}, "/tmp/goliac")

		assert.NotNil(t, err)
	})
}
