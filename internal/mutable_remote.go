package internal

import "github.com/Alayacare/goliac/internal/slugify"

/*
 * MutableGoliacRemoteImpl is used by GoliacReconciliatorImpl to update
 * the internal status of Github representation before appyling it for real
 * (or running in drymode)
 */
type MutableGoliacRemoteImpl struct {
	repositories   map[string]*GithubRepository
	teams          map[string]*GithubTeam
	teamRepos      map[string]map[string]*GithubTeamRepo
	teamSlugByName map[string]string
}

func NewMutableGoliacRemoteImpl(remote GoliacRemote) *MutableGoliacRemoteImpl {
	rTeamSlugByName := make(map[string]string)
	for k, v := range remote.TeamSlugByName() {
		rTeamSlugByName[k] = v
	}
	rTeams := make(map[string]*GithubTeam)
	for k, v := range remote.Teams() {
		ght := *v
		rTeams[k] = &ght
	}

	rRepositories := make(map[string]*GithubRepository)
	for k, v := range remote.Repositories() {
		ghr := *v
		rRepositories[k] = &ghr
	}

	rTeamRepositories := make(map[string]map[string]*GithubTeamRepo)
	for k1, v1 := range remote.TeamRepositories() {
		repos := make(map[string]*GithubTeamRepo)
		for k2, v2 := range v1 {
			gtr := *v2
			repos[k2] = &gtr
		}
		rTeamRepositories[k1] = repos
	}
	return &MutableGoliacRemoteImpl{
		repositories:   rRepositories,
		teams:          rTeams,
		teamRepos:      rTeamRepositories,
		teamSlugByName: rTeamSlugByName,
	}
}

func (m *MutableGoliacRemoteImpl) TeamSlugByName() map[string]string {
	return m.teamSlugByName
}

func (m *MutableGoliacRemoteImpl) Teams() map[string]*GithubTeam {
	return m.teams
}
func (m *MutableGoliacRemoteImpl) Repositories() map[string]*GithubRepository {
	return m.repositories
}
func (m *MutableGoliacRemoteImpl) TeamRepositories() map[string]map[string]*GithubTeamRepo {
	return m.teamRepos
}

// LISTENER

func (m *MutableGoliacRemoteImpl) CreateTeam(teamname string, description string, members []string) {
	teamslug := slugify.Make(teamname)
	t := GithubTeam{
		Name:    teamname,
		Slug:    teamslug,
		Members: members,
	}
	m.teams[teamslug] = &t
	m.teamSlugByName[teamname] = teamslug
}
func (m *MutableGoliacRemoteImpl) UpdateTeamAddMember(teamslug string, username string, role string) {
	if t, ok := m.teams[teamslug]; ok {
		t.Members = append(t.Members, username)
	}
}
func (m *MutableGoliacRemoteImpl) UpdateTeamRemoveMember(teamslug string, username string) {
	if t, ok := m.teams[teamslug]; ok {
		for i, m := range t.Members {
			if m == username {
				t.Members = append(t.Members[:i], t.Members[i+1:]...)
				return
			}
		}
	}

}
func (m *MutableGoliacRemoteImpl) DeleteTeam(teamslug string) {
	teamname := m.teams[teamslug].Name
	delete(m.teams, teamslug)
	delete(m.teamSlugByName, teamname)
	delete(m.teamRepos, teamslug)
}
func (m *MutableGoliacRemoteImpl) CreateRepository(reponame string, descrition string, writers []string, readers []string, public bool) {
	r := GithubRepository{
		Name: reponame,
	}
	m.repositories[reponame] = &r
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string) {
	if tr, ok := m.teamRepos[teamslug]; ok {
		tr[reponame] = &GithubTeamRepo{
			Name:       reponame,
			Permission: permission,
		}
	}
}

//func (m *MutableGoliacRemoteImpl) UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) {
//	if tr, ok := m.teamRepos[teamslug]; ok {
//		if r, ok := tr[reponame]; ok {
//			r.Permission = permission
//		}
//	}
//}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string) {
	if tr, ok := m.teamRepos[teamslug]; ok {
		delete(tr, reponame)
	}
}
func (m *MutableGoliacRemoteImpl) DeleteRepository(reponame string) {
	delete(m.repositories, reponame)
}
