package internal

import (
	"encoding/json"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/github"
)

const FORLOOP_STOP = 100

/*
 * GoliacRemote
 * This interface is used to load the goliac organization from a Github
 * and mount it in memory
 */
type GoliacRemote interface {
	// Load from a github repository
	Load() error

	TeamSlugByName(name string) (string, bool)
	Teams() map[string]*GithubTeam                           // the key is the team slug
	Repositories() map[string]*GithubRepository              // the key is the repository name
	TeamRepositories() map[string]map[string]*GithubTeamRepo // key is team slug, second key is repo name
}

type GithubRepository struct {
	Name       string
	IsArchived bool
	IsPrivate  bool
}

type GithubTeam struct {
	Name    string
	Slug    string
	Members []string // user login
}

type GithubTeamRepo struct {
	Name       string // repository name
	Permission string // possible values: ADMIN, MAINTAIN, WRITE, TRIAGE, READ
}

type GoliacRemoteImpl struct {
	client         github.GitHubClient
	repositories   map[string]*GithubRepository
	teams          map[string]*GithubTeam
	teamRepos      map[string]map[string]*GithubTeamRepo
	teamSlugByName map[string]string
}

func NewGoliacRemoteImpl(client github.GitHubClient) GoliacRemote {
	return &GoliacRemoteImpl{
		client:         client,
		repositories:   make(map[string]*GithubRepository),
		teams:          make(map[string]*GithubTeam),
		teamRepos:      make(map[string]map[string]*GithubTeamRepo),
		teamSlugByName: make(map[string]string),
	}
}

func (g *GoliacRemoteImpl) TeamSlugByName(name string) (string, bool) {
	v, ok := g.teamSlugByName[name]
	return v, ok
}

func (g *GoliacRemoteImpl) Teams() map[string]*GithubTeam {
	return g.teams
}
func (g *GoliacRemoteImpl) Repositories() map[string]*GithubRepository {
	return g.repositories
}
func (g *GoliacRemoteImpl) TeamRepositories() map[string]map[string]*GithubTeamRepo {
	return g.TeamRepositories()
}

const listAllReposInOrg = `
query listAllReposInOrg($orgLogin: String!, $endCursor: String) {
    organization(login: $orgLogin) {
      repositories(first: 100, after: $endCursor) {
        nodes {
          name
          isArchived
          isPrivate
        }
        pageInfo {
          hasNextPage
          endCursor
        }
        totalCount
      }
    }
  }
`

type GraplQLRepositories struct {
	Data struct {
		Organization struct {
			Repositories struct {
				Nodes []struct {
					Name       string
					IsArchived bool
					IsPrivate  bool
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				} `json:"pageInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"repositories"`
		}
	}
}

func (g *GoliacRemoteImpl) loadRepositories() error {
	g.repositories = make(map[string]*GithubRepository)

	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(listAllReposInOrg, variables)
		var gResult GraplQLRepositories

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return err
		}

		for _, c := range gResult.Data.Organization.Repositories.Nodes {
			g.repositories[c.Name] = &GithubRepository{
				Name:       c.Name,
				IsArchived: c.IsArchived,
				IsPrivate:  c.IsPrivate,
			}
		}

		hasNextPage = gResult.Data.Organization.Repositories.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.Repositories.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}

	return nil
}

const listAllTeamsInOrg = `
query listAllTeamsInOrg($orgLogin: String!, $endCursor: String) {
    organization(login: $orgLogin) {
      teams(first: 100, after: $endCursor) {
        nodes {
          name
          slug
        }
        pageInfo {
          hasNextPage
          endCursor
        }
        totalCount
      }
    }
  }
`

type GraplQLTeams struct {
	Data struct {
		Organization struct {
			Teams struct {
				Nodes []struct {
					Name string
					Slug string
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				} `json:"pageInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"teams"`
		}
	}
}

const listAllTeamsReposInOrg = `
query listAllTeamsReposInOrg($orgLogin: String!, $teamSlug: String, $endCursor: String) {
  organization(login: $orgLogin) {
    team(slug: $teamSlug) {
       repositories(first: 100, after: $endCursor) {
        edges {
          permission
          node {
            name
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
        totalCount
      }
    }
  }
}
`

type GraplQLTeamsRepos struct {
	Data struct {
		Organization struct {
			Team struct {
				Repository struct {
					Edges []struct {
						Permission string
						Node       struct {
							Name string
						}
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					} `json:"pageInfo"`
					TotalCount int `json:"totalCount"`
				} `json:"repositories"`
			} `json:"team"`
		}
	}
}

func (g *GoliacRemoteImpl) Load() error {
	err := g.loadRepositories()
	if err != nil {
		return err
	}

	err = g.loadTeams()
	if err != nil {
		return err
	}

	g.teamRepos = make(map[string]map[string]*GithubTeamRepo)

	for teamSlug, _ := range g.teams {
		repos, err := g.loadTeamRepos(teamSlug)
		if err != nil {
			return err
		}
		g.teamRepos[teamSlug] = repos
	}
	return nil
}

func (g *GoliacRemoteImpl) loadTeamRepos(teamSlug string) (map[string]*GithubTeamRepo, error) {
	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["slug"] = teamSlug
	variables["endCursor"] = nil

	repos := make(map[string]*GithubTeamRepo)

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(listAllTeamsReposInOrg, variables)
		var gResult GraplQLTeamsRepos

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return nil, err
		}

		for _, c := range gResult.Data.Organization.Team.Repository.Edges {
			repos[c.Node.Name] = &GithubTeamRepo{
				Name:       c.Node.Name,
				Permission: c.Permission,
			}
		}

		hasNextPage = gResult.Data.Organization.Team.Repository.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.Team.Repository.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}
	return repos, nil
}

const listAllTeamMembersInOrg = `
query listAllTeamMembersInOrg($orgLogin: String!, $teamSlug: String, $endCursor: String) {
    organization(login: $orgLogin) {
      team(slug: $teamSlug) {
        members(first: 100, after: $endCursor) {
          edges {
            node {
              login
              name
            }
          }
          pageInfo {
            hasNextPage
            endCursor
          }
          totalCount
        }
      }
    }
  }
`

type GraplQLTeamMembers struct {
	Data struct {
		Organization struct {
			Team struct {
				Members struct {
					Edges []struct {
						Node struct {
							Login string
							Name  string
						}
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					} `json:"pageInfo"`
					TotalCount int `json:"totalCount"`
				} `json:"members"`
			} `json:"team"`
		}
	}
}

func (g *GoliacRemoteImpl) loadTeams() error {
	g.teams = make(map[string]*GithubTeam)
	g.teamSlugByName = make(map[string]string)

	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(listAllTeamsInOrg, variables)
		var gResult GraplQLTeams

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return err
		}

		for _, c := range gResult.Data.Organization.Teams.Nodes {
			g.teams[c.Slug] = &GithubTeam{
				Name: c.Name,
				Slug: c.Slug,
			}
			g.teamSlugByName[c.Name] = c.Slug
		}

		hasNextPage = gResult.Data.Organization.Teams.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.Teams.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}

	// load team's members
	for _, t := range g.teams {
		variables["orgLogin"] = config.Config.GithubAppOrganization
		variables["endCursor"] = nil
		variables["teamSlug"] = t.Slug

		hasNextPage := true
		count := 0
		for hasNextPage {
			data, err := g.client.QueryGraphQLAPI(listAllTeamMembersInOrg, variables)
			var gResult GraplQLTeamMembers

			// parse first page
			err = json.Unmarshal(data, &gResult)
			if err != nil {
				return err
			}

			for _, c := range gResult.Data.Organization.Team.Members.Edges {
				t.Members = append(t.Members, c.Node.Login)
			}

			hasNextPage = gResult.Data.Organization.Team.Members.PageInfo.HasNextPage
			variables["endCursor"] = gResult.Data.Organization.Team.Members.PageInfo.EndCursor

			count++
			// sanity check to avoid loops
			if count > FORLOOP_STOP {
				break
			}
		}
	}

	return nil
}
