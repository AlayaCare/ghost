package internal

import (
	"testing"

	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/slugify"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type GoliacLocalMock struct {
	teams map[string]*entity.Team
	repos map[string]*entity.Repository
}

func (m *GoliacLocalMock) LoadAndValidate(accesstoken, repositoryUrl, branch string) ([]error, []error) {
	return nil, nil
}
func (m *GoliacLocalMock) LoadAndValidateLocal(fs afero.Fs, path string) ([]error, []error) {
	return nil, nil
}
func (m *GoliacLocalMock) Teams() map[string]*entity.Team {
	return m.teams
}
func (m *GoliacLocalMock) Repositories() map[string]*entity.Repository {
	return m.repos
}
func (m *GoliacLocalMock) Users() map[string]*entity.User {
	return nil
}
func (m *GoliacLocalMock) ExternalUsers() map[string]*entity.User {
	return nil
}

type GoliacRemoteMock struct {
	teams      map[string]*GithubTeam
	repos      map[string]*GithubRepository
	teamsrepos map[string]map[string]*GithubTeamRepo
}

func (m *GoliacRemoteMock) Load() error {
	return nil
}
func (m *GoliacRemoteMock) TeamSlugByName() map[string]string {
	slugs := make(map[string]string)
	for k := range m.teams {
		slugs[k] = slugify.Make(k)
	}
	return slugs
}
func (m *GoliacRemoteMock) Teams() map[string]*GithubTeam {
	return m.teams
}
func (m *GoliacRemoteMock) Repositories() map[string]*GithubRepository {
	return m.repos
}
func (m *GoliacRemoteMock) TeamRepositories() map[string]map[string]*GithubTeamRepo {
	return m.teamsrepos
}

type ReconciliatorListenerRecorder struct {
	TeamsCreated      map[string][]string
	TeamMemberAdded   map[string][]string
	TeamMemberRemoved map[string][]string
	TeamDeleted       map[string]bool

	RepositoryCreated          map[string]bool
	RepositoryTeamAdded        map[string][]string
	RepositoryTeamUpdated      map[string][]string
	RepositoryTeamRemoved      map[string][]string
	RepositoriesDeleted        map[string]bool
	RepositoriesUpdatePrivate  map[string]bool
	RepositoriesUpdateArchived map[string]bool
}

func NewReconciliatorListenerRecorder() *ReconciliatorListenerRecorder {
	r := ReconciliatorListenerRecorder{
		TeamsCreated:               make(map[string][]string),
		TeamMemberAdded:            make(map[string][]string),
		TeamMemberRemoved:          make(map[string][]string),
		TeamDeleted:                make(map[string]bool),
		RepositoryCreated:          make(map[string]bool),
		RepositoryTeamAdded:        make(map[string][]string),
		RepositoryTeamUpdated:      make(map[string][]string),
		RepositoryTeamRemoved:      make(map[string][]string),
		RepositoriesDeleted:        make(map[string]bool),
		RepositoriesUpdatePrivate:  make(map[string]bool),
		RepositoriesUpdateArchived: make(map[string]bool),
	}
	return &r
}
func (r *ReconciliatorListenerRecorder) CreateTeam(teamname string, description string, members []string) {
	r.TeamsCreated[teamname] = append(r.TeamsCreated[teamname], members...)
}
func (r *ReconciliatorListenerRecorder) UpdateTeamAddMember(teamslug string, username string, role string) {
	r.TeamMemberAdded[teamslug] = append(r.TeamMemberAdded[teamslug], username)
}
func (r *ReconciliatorListenerRecorder) UpdateTeamRemoveMember(teamslug string, username string) {
	r.TeamMemberRemoved[teamslug] = append(r.TeamMemberRemoved[teamslug], username)
}
func (r *ReconciliatorListenerRecorder) DeleteTeam(teamslug string) {
	r.TeamDeleted[teamslug] = true
}
func (r *ReconciliatorListenerRecorder) CreateRepository(reponame string, descrition string, writers []string, readers []string, public bool) {
	r.RepositoryCreated[reponame] = true
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string) {
	r.RepositoryTeamAdded[reponame] = append(r.RepositoryTeamAdded[reponame], teamslug)
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) {
	r.RepositoryTeamUpdated[reponame] = append(r.RepositoryTeamUpdated[reponame], teamslug)
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string) {
	r.RepositoryTeamRemoved[reponame] = append(r.RepositoryTeamRemoved[reponame], teamslug)
}
func (r *ReconciliatorListenerRecorder) DeleteRepository(reponame string) {
	r.RepositoriesDeleted[reponame] = true
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryUpdatePrivate(reponame string, private bool) {
	r.RepositoriesUpdatePrivate[reponame] = true
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryUpdateArchived(reponame string, archived bool) {
	r.RepositoriesUpdateArchived[reponame] = true
}
func (r *ReconciliatorListenerRecorder) Begin() {
}
func (r *ReconciliatorListenerRecorder) Rollback(error) {
}
func (r *ReconciliatorListenerRecorder) Commit() {
}

func TestReconciliation(t *testing.T) {

	t.Run("happy path: new team", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		newTeam := &entity.Team{}
		newTeam.Metadata.Name = "new"
		newTeam.Data.Owners = []string{"new_owner"}
		newTeam.Data.Members = []string{"new_member"}
		local.teams["new"] = newTeam

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}

		r.Reconciliate(&local, &remote, false)

		// 2 members created
		assert.Equal(t, 2, len(recorder.TeamsCreated["new"]))
	})

	t.Run("happy path: existing team with new members", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		existingTeam := &entity.Team{}
		existingTeam.Metadata.Name = "existing"
		existingTeam.Data.Owners = []string{"existing_owner", "existing_owner2"}
		existingTeam.Data.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing

		r.Reconciliate(&local, &remote, false)

		// 1 members added
		assert.Equal(t, 0, len(recorder.TeamsCreated))
		assert.Equal(t, 1, len(recorder.TeamMemberAdded["existing"]))
	})

	t.Run("happy path: removed team", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		removing := &GithubTeam{
			Name:    "removing",
			Slug:    "removing",
			Members: []string{"existing_owner", "existing_owner"},
		}
		remote.teams["removing"] = removing

		r.Reconciliate(&local, &remote, false)

		// 1 team deleted
		assert.Equal(t, 1, len(recorder.TeamDeleted))
	})

	t.Run("happy path: new repo without owner", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		newRepo := &entity.Repository{}
		newRepo.Metadata.Name = "new"
		newRepo.Data.Readers = []string{}
		newRepo.Data.Writers = []string{}
		local.repos["new"] = newRepo

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}

		r.Reconciliate(&local, &remote, false)

		// 1 repo created
		assert.Equal(t, 1, len(recorder.RepositoryCreated))
	})

	t.Run("happy path: new repo with owner", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		newRepo := &entity.Repository{}
		newRepo.Metadata.Name = "new"
		newRepo.Data.Readers = []string{}
		newRepo.Data.Writers = []string{}
		owner := "existing"
		newRepo.Owner = &owner
		local.repos["new"] = newRepo

		existingTeam := &entity.Team{}
		existingTeam.Metadata.Name = "existing"
		existingTeam.Data.Owners = []string{"existing_owner"}
		existingTeam.Data.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing

		r.Reconciliate(&local, &remote, false)

		// 1 repo created
		assert.Equal(t, 1, len(recorder.RepositoryCreated))
	})

	t.Run("happy path: existing repo with new owner (from read to write)", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Metadata.Name = "myrepo"
		lRepo.Data.Readers = []string{}
		lRepo.Data.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Metadata.Name = "existing"
		existingTeam.Data.Owners = []string{"existing_owner"}
		existingTeam.Data.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name: "myrepo",
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "READ",
		}

		r.Reconciliate(&local, &remote, false)

		// 1 team updated
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 1, len(recorder.RepositoryTeamUpdated))
	})

	t.Run("happy path: add a team to an existing repo", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Metadata.Name = "myrepo"
		lRepo.Data.Readers = []string{"reader"}
		lRepo.Data.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Metadata.Name = "existing"
		existingTeam.Data.Owners = []string{"existing_owner"}
		existingTeam.Data.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		readerTeam := &entity.Team{}
		readerTeam.Metadata.Name = "reader"
		readerTeam.Data.Owners = []string{"existing_owner"}
		readerTeam.Data.Members = []string{"existing_member"}
		local.teams["reader"] = readerTeam

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		reader := &GithubTeam{
			Name:    "reader",
			Slug:    "reader",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		remote.teams["reader"] = reader
		rRepo := GithubRepository{
			Name: "myrepo",
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		r.Reconciliate(&local, &remote, false)

		// 1 team added
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 1, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
	})

	t.Run("happy path: remove a team from an existing repo", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Metadata.Name = "myrepo"
		lRepo.Data.Readers = []string{}
		lRepo.Data.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Metadata.Name = "existing"
		existingTeam.Data.Owners = []string{"existing_owner"}
		existingTeam.Data.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		readerTeam := &entity.Team{}
		readerTeam.Metadata.Name = "reader"
		readerTeam.Data.Owners = []string{"existing_owner"}
		readerTeam.Data.Members = []string{"existing_member"}
		local.teams["reader"] = readerTeam

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		reader := &GithubTeam{
			Name:    "reader",
			Slug:    "reader",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		remote.teams["reader"] = reader
		rRepo := GithubRepository{
			Name: "myrepo",
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}
		remote.teamsrepos["reader"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["reader"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "READ",
		}

		r.Reconciliate(&local, &remote, false)

		// 1 team removed
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 1, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
	})

	t.Run("happy path: remove a team member", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Metadata.Name = "myrepo"
		lRepo.Data.Readers = []string{}
		lRepo.Data.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Metadata.Name = "existing"
		existingTeam.Data.Owners = []string{"existing_owner"}
		existingTeam.Data.Members = []string{}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name: "myrepo",
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		r.Reconciliate(&local, &remote, false)

		// 1 member removed
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
		assert.Equal(t, 1, len(recorder.TeamMemberRemoved))
	})

	t.Run("happy path: add a team AND add it to an existing repo", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Metadata.Name = "myrepo"
		lRepo.Data.Readers = []string{"reader"}
		lRepo.Data.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Metadata.Name = "existing"
		existingTeam.Data.Owners = []string{"existing_owner"}
		existingTeam.Data.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		readerTeam := &entity.Team{}
		readerTeam.Metadata.Name = "reader"
		readerTeam.Data.Owners = []string{"existing_owner"}
		readerTeam.Data.Members = []string{"existing_member"}
		local.teams["reader"] = readerTeam

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name: "myrepo",
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		r.Reconciliate(&local, &remote, false)

		// 1 repo updated
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 1, len(recorder.RepositoryTeamAdded))
	})

	t.Run("happy path: removed repo", func(t *testing.T) {
		r := NewGoliacReconciliatorImpl()
		recorder := NewReconciliatorListenerRecorder()
		r.AddListener(recorder)

		local := GoliacLocalMock{
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		remote := GoliacRemoteMock{
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
		}
		removing := &GithubRepository{
			Name: "removing",
		}
		remote.repos["removing"] = removing

		r.Reconciliate(&local, &remote, false)

		// 1 repo deleted
		assert.Equal(t, 1, len(recorder.RepositoriesDeleted))
	})

}
