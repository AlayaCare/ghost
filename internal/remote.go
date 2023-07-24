package internal

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/sirupsen/logrus"
)

const FORLOOP_STOP = 100

/*
 * GoliacRemote
 * This interface is used to load the goliac organization from a Github
 * and mount it in memory
 */
type GoliacRemote interface {
	// Load from a github repository
	Load(repoconfig *config.RepositoryConfig) error

	Users() map[string]string
	TeamSlugByName() map[string]string
	Teams() map[string]*GithubTeam                           // the key is the team slug
	Repositories() map[string]*GithubRepository              // the key is the repository name
	RepositoriesByRefId() map[string]*GithubRepository       // the key is the repository id
	TeamRepositories() map[string]map[string]*GithubTeamRepo // key is team slug, second key is repo name
	RuleSets() map[string]*GithubRuleSet
	AppIds() map[string]int
}

type GoliacRemoteExecutor interface {
	GoliacRemote
	ReconciliatorExecutor
}

type GithubRepository struct {
	Name       string
	Id         int
	RefId      string
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
	client              github.GitHubClient
	users               map[string]string
	repositories        map[string]*GithubRepository
	repositoriesByRefId map[string]*GithubRepository
	teams               map[string]*GithubTeam
	teamRepos           map[string]map[string]*GithubTeamRepo
	teamSlugByName      map[string]string
	rulesets            map[string]*GithubRuleSet
	appIds              map[string]int
}

func NewGoliacRemoteImpl(client github.GitHubClient) *GoliacRemoteImpl {
	return &GoliacRemoteImpl{
		client:              client,
		users:               make(map[string]string),
		repositories:        make(map[string]*GithubRepository),
		repositoriesByRefId: make(map[string]*GithubRepository),
		teams:               make(map[string]*GithubTeam),
		teamRepos:           make(map[string]map[string]*GithubTeamRepo),
		teamSlugByName:      make(map[string]string),
		rulesets:            make(map[string]*GithubRuleSet),
		appIds:              make(map[string]int),
	}
}

func (g *GoliacRemoteImpl) RuleSets() map[string]*GithubRuleSet {
	return g.rulesets
}

func (g *GoliacRemoteImpl) AppIds() map[string]int {
	return g.appIds
}

func (g *GoliacRemoteImpl) Users() map[string]string {
	return g.users
}

func (g *GoliacRemoteImpl) TeamSlugByName() map[string]string {
	return g.teamSlugByName
}

func (g *GoliacRemoteImpl) Teams() map[string]*GithubTeam {
	return g.teams
}
func (g *GoliacRemoteImpl) Repositories() map[string]*GithubRepository {
	return g.repositories
}
func (g *GoliacRemoteImpl) RepositoriesByRefId() map[string]*GithubRepository {
	return g.repositoriesByRefId
}
func (g *GoliacRemoteImpl) TeamRepositories() map[string]map[string]*GithubTeamRepo {
	return g.teamRepos
}

const listAllOrgMembers = `
query listAllReposInOrg($orgLogin: String!, $endCursor: String) {
    organization(login: $orgLogin) {
		membersWithRole(first: 100, after: $endCursor) {
        nodes {
          login
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

type GraplQLUsers struct {
	Data struct {
		Organization struct {
			MembersWithRole struct {
				Nodes []struct {
					Login string
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				} `json:"pageInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"membersWithRole"`
		}
	}
	Errors []struct {
		Path []struct {
			Query string `json:"query"`
		} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (g *GoliacRemoteImpl) loadOrgUsers() error {
	g.users = make(map[string]string)

	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(listAllOrgMembers, variables)
		var gResult GraplQLUsers

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return err
		}
		if len(gResult.Errors) > 0 {
			return fmt.Errorf("Graphql error: %v", gResult.Errors[0].Message)
		}

		for _, c := range gResult.Data.Organization.MembersWithRole.Nodes {
			g.users[c.Login] = c.Login
		}

		hasNextPage = gResult.Data.Organization.MembersWithRole.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.MembersWithRole.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}

	return nil
}

const listAllReposInOrg = `
query listAllReposInOrg($orgLogin: String!, $endCursor: String) {
    organization(login: $orgLogin) {
      repositories(first: 100, after: $endCursor) {
        nodes {
          name
		  id
		  databaseId
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
					Id         string
					DatabaseId int
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
	Errors []struct {
		Path []struct {
			Query string `json:"query"`
		} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (g *GoliacRemoteImpl) loadRepositories() error {
	g.repositories = make(map[string]*GithubRepository)
	g.repositoriesByRefId = make(map[string]*GithubRepository)

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
		if len(gResult.Errors) > 0 {
			return fmt.Errorf("Graphql error: %v", gResult.Errors[0].Message)
		}

		for _, c := range gResult.Data.Organization.Repositories.Nodes {
			repo := &GithubRepository{
				Name:       c.Name,
				Id:         c.DatabaseId,
				RefId:      c.Id,
				IsArchived: c.IsArchived,
				IsPrivate:  c.IsPrivate,
			}
			g.repositories[c.Name] = repo
			g.repositoriesByRefId[c.Id] = repo
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
	Errors []struct {
		Path []struct {
			Query string `json:"query"`
		} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

const listAllTeamsReposInOrg = `
query listAllTeamsReposInOrg($orgLogin: String!, $teamSlug: String!, $endCursor: String) {
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
	Errors []struct {
		Path []struct {
			Query string `json:"query"`
		} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (g *GoliacRemoteImpl) loadAppIds() error {
	type Installation struct {
		TotalClount   int `json:"total_count"`
		Installations []struct {
			Id      int    `json:"id"`
			AppId   int    `json:"app_id"`
			Name    string `json:"name"`
			AppSlug string `json:"app_slug"`
		} `json:"installations"`
	}
	// https://docs.github.com/en/enterprise-cloud@latest/rest/orgs/orgs?apiVersion=2022-11-28#list-app-installations-for-an-organization
	body, err := g.client.CallRestAPI(fmt.Sprintf("/orgs/%s/installations", config.Config.GithubAppOrganization),
		"GET",
		nil)

	if err != nil {
		return fmt.Errorf("not able to list github apps: %v. %s", err, string(body))
	}

	var installations Installation
	json.Unmarshal(body, &installations)
	if err != nil {
		return fmt.Errorf("not able to list github apps: %v", err)
	}

	g.appIds = map[string]int{}
	for _, i := range installations.Installations {
		g.appIds[i.AppSlug] = i.AppId
	}

	return nil
}

func (g *GoliacRemoteImpl) Load(repoconfig *config.RepositoryConfig) error {
	err := g.loadOrgUsers()
	if err != nil {
		return err
	}

	err = g.loadRepositories()
	if err != nil {
		return err
	}

	err = g.loadTeams()
	if err != nil {
		return err
	}

	err = g.loadAppIds()
	if err != nil {
		return err
	}

	err = g.loadRulesets()
	if err != nil {
		return err
	}

	if repoconfig.GithubConcurrentThreads <= 1 {
		err = g.loadTeamReposNonConcurrently()
	} else {
		err = g.loadTeamReposConcurrently(repoconfig.GithubConcurrentThreads)
	}
	if err != nil {
		return err
	}

	logrus.Infof("Nb remote users: %d", len(g.users))
	logrus.Infof("Nb remote teams: %d", len(g.teams))
	logrus.Infof("Nb remote repositories: %d", len(g.repositories))

	return nil
}

func (g *GoliacRemoteImpl) loadTeamReposNonConcurrently() error {
	g.teamRepos = make(map[string]map[string]*GithubTeamRepo)

	for teamSlug := range g.teams {
		repos, err := g.loadTeamRepos(teamSlug)
		if err != nil {
			return err
		}
		g.teamRepos[teamSlug] = repos
	}
	return nil
}

func (g *GoliacRemoteImpl) loadTeamReposConcurrently(maxGoroutines int) error {
	g.teamRepos = make(map[string]map[string]*GithubTeamRepo)

	var wg sync.WaitGroup

	// Create buffered channels
	teamsChan := make(chan string, len(g.teams))
	errChan := make(chan error, 1) // will hold the first error
	reposChan := make(chan struct {
		teamSlug string
		repos    map[string]*GithubTeamRepo
	}, len(g.teams))

	// Create worker goroutines
	for i := 0; i < maxGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for slug := range teamsChan {
				repos, err := g.loadTeamRepos(slug)
				if err != nil {
					// Try to report the error
					select {
					case errChan <- err:
					default:
					}
					return
				}
				reposChan <- struct {
					teamSlug string
					repos    map[string]*GithubTeamRepo
				}{slug, repos}
			}
		}()
	}

	// Send teams to teamsChan
	for teamSlug := range g.teams {
		teamsChan <- teamSlug
	}
	close(teamsChan)

	// Wait for all goroutines to finish
	wg.Wait()
	close(reposChan)

	// Check if any goroutine returned an error
	select {
	case err := <-errChan:
		return err
	default:
		// No error, populate the teamRepos map
		for r := range reposChan {
			g.teamRepos[r.teamSlug] = r.repos
		}
	}

	return nil
}

func (g *GoliacRemoteImpl) loadTeamRepos(teamSlug string) (map[string]*GithubTeamRepo, error) {
	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["teamSlug"] = teamSlug
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
		if len(gResult.Errors) > 0 {
			return nil, fmt.Errorf("Graphql error: %v", gResult.Errors[0].Message)
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
query listAllTeamMembersInOrg($orgLogin: String!, $teamSlug: String!, $endCursor: String) {
    organization(login: $orgLogin) {
      team(slug: $teamSlug) {
        members(first: 100, after: $endCursor) {
          edges {
            node {
              login
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
	Errors []struct {
		Path []struct {
			Query string `json:"query"`
		} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
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
		if len(gResult.Errors) > 0 {
			return fmt.Errorf("Graphql error: %v", gResult.Errors[0].Message)
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
			if len(gResult.Errors) > 0 {
				return fmt.Errorf("Graphql error: %v", gResult.Errors[0].Message)
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

const listRulesets = `
query listRulesets ($orgLogin: String!) { 
	organization(login: $orgLogin) {
	  rulesets(first: 100) { 
		nodes {
		  databaseId
		  name
		  target
		  enforcement
		  bypassActors(first:100) {
			app:nodes {
			  actor {
				... on App {
					databaseId
					name
				}
			  }
			  bypassMode
			}
		  }
		  conditions {
			refName {
			  include
			  exclude
			}
			repositoryName {
			  exclude
			  include
			}
			repositoryId {
				repositoryIds
			}
		  }
		  rules(first:100) {
			nodes {
				parameters {
					... on PullRequestParameters {
						dismissStaleReviewsOnPush
						requireCodeOwnerReview
						requiredApprovingReviewCount
						requiredReviewThreadResolution
						requireLastPushApproval
					}
				}
				type
			}
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
`

type GithubRuleSetApp struct {
	Actor struct {
		DatabaseId int
		Name       string
	}
	BypassMode string // ALWAYS, PULL_REQUEST
}

type GithubRuleSetRuleStatusCheck struct {
	Context       string
	IntegrationId int
}

type GithubRuleSetRule struct {
	Parameters struct {
		// PullRequestParameters
		DismissStaleReviewsOnPush      bool
		RequireCodeOwnerReview         bool
		RequiredApprovingReviewCount   int
		RequiredReviewThreadResolution bool
		RequireLastPushApproval        bool

		// RequiredStatusChecksParameters
		RequiredStatusChecks             []GithubRuleSetRuleStatusCheck
		StrictRequiredStatusChecksPolicy bool
	}
	ID   int
	Type string // CREATION, UPDATE, DELETION, REQUIRED_LINEAR_HISTORY, REQUIRED_DEPLOYMENTS, REQUIRED_SIGNATURES, PULL_REQUEST, REQUIRED_STATUS_CHECKS, NON_FAST_FORWARD, COMMIT_MESSAGE_PATTERN, COMMIT_AUTHOR_EMAIL_PATTERN, COMMITTER_EMAIL_PATTERN, BRANCH_NAME_PATTERN, TAG_NAME_PATTERN
}

type GithubRuleSet struct {
	DatabaseId   int
	Name         string
	Target       string // BRANCH, TAG
	Enforcement  string // DISABLED, ACTIVE, EVALUATE
	BypassActors struct {
		App []GithubRuleSetApp
	}
	Conditions struct {
		RefName struct { // target branches
			Include []string // ~DEFAULT_BRANCH, ~ALL,
			Exclude []string
		}
		RepositoryName struct { // regex
			Include   []string
			Exclude   []string
			Protected bool
		}
		RepositoryId struct { // per repo
			RepositoryIds []string
		}
	}
	Rules struct {
		Nodes []GithubRuleSetRule
	}
}

type GraplQLRuleSets struct {
	Data struct {
		Organization struct {
			Rulesets struct {
				Nodes    []GithubRuleSet
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				} `json:"pageInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"rulesets"`
		}
	}
	Errors []struct {
		Path       []string `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (g *GoliacRemoteImpl) loadRulesets() error {
	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	rulesets := make(map[string]*GithubRuleSet)

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(listRulesets, variables)
		var gResult GraplQLRuleSets

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return err
		}
		if len(gResult.Errors) > 0 {
			return fmt.Errorf("Graphql error: %v", gResult.Errors[0].Message)
		}

		for _, c := range gResult.Data.Organization.Rulesets.Nodes {
			rulesets[c.Name] = &c
		}

		hasNextPage = gResult.Data.Organization.Rulesets.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.Rulesets.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}

	g.rulesets = rulesets
	return nil
}

func (g *GoliacRemoteImpl) prepareRuleset(ruleset *GithubRuleSet) map[string]interface{} {
	bypassActors := make([]map[string]interface{}, 0)

	for _, app := range ruleset.BypassActors.App {
		// let's find the app id based on the app slug name
		if appId, ok := g.appIds[app.Actor.Name]; ok {
			bypassActor := map[string]interface{}{
				"actor_id":    appId,
				"actor_type":  "Integration",
				"bypass_mode": "always",
			}
			bypassActors = append(bypassActors, bypassActor)
		}
	}

	repoIds := []int{}
	for _, r := range ruleset.Conditions.RepositoryId.RepositoryIds {
		if rid, ok := g.repositories[r]; ok {
			repoIds = append(repoIds, rid.Id)
		}
	}
	include := ruleset.Conditions.RefName.Include
	if include == nil {
		include = []string{}
	}
	exclude := ruleset.Conditions.RefName.Exclude
	if exclude == nil {
		exclude = []string{}
	}
	conditions := map[string]interface{}{
		"ref_name": map[string]interface{}{
			"include": include,
			"exclude": exclude,
		},
		"repository_id": map[string]interface{}{
			"repository_ids": repoIds,
		},
	}

	rules := make([]map[string]interface{}, 0)
	for _, rule := range ruleset.Rules.Nodes {
		switch rule.Type {
		case "required_signatures":
			rules = append(rules, map[string]interface{}{
				"type": "required_signatures",
			})
			break
		case "pull_request":
			rules = append(rules, map[string]interface{}{
				"type": "pull_request",
				"parameters": map[string]interface{}{
					"dismiss_stale_reviews_on_push":     rule.Parameters.DismissStaleReviewsOnPush,
					"require_code_owner_review":         rule.Parameters.RequireCodeOwnerReview,
					"required_approving_review_count":   rule.Parameters.RequiredApprovingReviewCount,
					"required_review_thread_resolution": rule.Parameters.RequiredReviewThreadResolution,
					"require_last_push_approval":        rule.Parameters.RequireLastPushApproval,
				},
			})
			break
		}
	}

	payload := map[string]interface{}{
		"name":          ruleset.Name,
		"target":        "branch",
		"enforcement":   ruleset.Enforcement,
		"bypass_actors": bypassActors,
		"conditions":    conditions,
		"rules":         rules,
	}
	return payload
}

func (g *GoliacRemoteImpl) AddRuleset(ruleset *GithubRuleSet) {
	// add ruleset
	// https://docs.github.com/en/enterprise-cloud@latest/rest/orgs/rules?apiVersion=2022-11-28#create-an-organization-repository-ruleset

	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/rulesets", config.Config.GithubAppOrganization),
		"POST",
		g.prepareRuleset(ruleset),
	)
	if err != nil {
		logrus.Errorf("failed to add ruleset to org: %v. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) UpdateRuleset(ruleset *GithubRuleSet) {
	// add ruleset
	// https://docs.github.com/en/enterprise-cloud@latest/rest/orgs/rules?apiVersion=2022-11-28#update-an-organization-repository-ruleset

	body, err := g.client.CallRestAPI(

		fmt.Sprintf("/orgs/%s/rulesets/%d", config.Config.GithubAppOrganization, ruleset.DatabaseId),
		"PUT",
		g.prepareRuleset(ruleset),
	)
	if err != nil {
		logrus.Errorf("failed to update ruleset to org: %v. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) DeleteRuleset(rulesetid int) {
	// add ruleset
	// https://docs.github.com/en/enterprise-cloud@latest/rest/orgs/rules?apiVersion=2022-11-28#delete-an-organization-repository-ruleset

	_, err := g.client.CallRestAPI(

		fmt.Sprintf("/orgs/%s/rulesets/%d", config.Config.GithubAppOrganization, rulesetid),
		"DELETE",
		nil,
	)
	if err != nil {
		logrus.Errorf("failed to remove ruleset to org: %v", err)
	}
}

func (g *GoliacRemoteImpl) AddUserToOrg(ghuserid string) {
	// add member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#create-a-team
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/memberships/%s", config.Config.GithubAppOrganization, ghuserid),
		"PUT",
		map[string]interface{}{"role": "member"},
	)
	if err != nil {
		logrus.Errorf("failed to add user to org: %v. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) RemoveUserFromOrg(ghuserid string) {
	// remove member
	// https://docs.github.com/en/rest/orgs/members?apiVersion=2022-11-28#remove-organization-membership-for-a-user
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/memberships/%s", config.Config.GithubAppOrganization, ghuserid),
		"DELETE",
		nil,
	)
	if err != nil {
		logrus.Errorf("failed to remove user from org: %v. %s", err, string(body))
	}
}

type CreateTeamResponse struct {
	Name string
	Slug string
}

func (g *GoliacRemoteImpl) CreateTeam(teamname string, description string, members []string) {
	// create team
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#create-a-team
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/teams", config.Config.GithubAppOrganization),
		"POST",
		map[string]interface{}{"name": teamname, "description": description, "privacy": "closed"},
	)
	if err != nil {
		logrus.Errorf("failed to create team: %v. %s", err, string(body))
		return
	}
	var res CreateTeamResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		logrus.Errorf("failed to create team: %v", err)
		return
	}

	// add members
	for _, member := range members {
		// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
		body, err := g.client.CallRestAPI(
			fmt.Sprintf("orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, res.Slug, member),
			"PUT",
			map[string]interface{}{"role": "member"},
		)
		if err != nil {
			logrus.Errorf("failed to create team: %v. %s", err, string(body))
			return
		}
	}
}

// role = member or maintainer (usually we use member)
func (g *GoliacRemoteImpl) UpdateTeamAddMember(teamslug string, username string, role string) {
	// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, teamslug, username),
		"PUT",
		map[string]interface{}{"role": role},
	)
	if err != nil {
		logrus.Errorf("failed to add team member: %v. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) UpdateTeamRemoveMember(teamslug string, username string) {
	// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, teamslug, username),
		"DELETE",
		nil,
	)
	if err != nil {
		logrus.Errorf("failed to remove team member: %v. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) DeleteTeam(teamslug string) {
	// delete team
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#delete-a-team
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/teams/%s", config.Config.GithubAppOrganization, teamslug),
		"DELETE",
		nil,
	)
	if err != nil {
		logrus.Errorf("failed to delete a team: %v. %s", err, string(body))
	}
}

type CreateRepositoryResponse struct {
	Id     int    `json:"id"`
	NodeId string `json:"node_id"`
}

func (g *GoliacRemoteImpl) CreateRepository(reponame string, description string, writers []string, readers []string, public bool) {
	// create repository
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#create-an-organization-repository
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/repos", config.Config.GithubAppOrganization),
		"POST",
		map[string]interface{}{"name": reponame, "description": description, "private": !public},
	)
	if err != nil {
		logrus.Errorf("failed to create repository: %v. %s", err, string(body))
		return
	}

	// get the repo id
	var resp CreateRepositoryResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logrus.Errorf("failed to read the create repository action response: %v", err)
		return
	}

	// update the repositories list
	newRepo := &GithubRepository{
		Name:       reponame,
		Id:         resp.Id,
		RefId:      resp.NodeId,
		IsArchived: false,
		IsPrivate:  !public,
	}
	g.repositories[reponame] = newRepo
	g.repositoriesByRefId[resp.NodeId] = newRepo

	// add members
	for _, reader := range readers {
		// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
		body, err := g.client.CallRestAPI(
			fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, reader, config.Config.GithubAppOrganization, reponame),
			"PUT",
			map[string]interface{}{"permission": "pull"},
		)
		if err != nil {
			logrus.Errorf("failed to create repository (and add members): %v. %s", err, string(body))
			return
		}
	}
	for _, writer := range writers {
		// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
		body, err := g.client.CallRestAPI(
			fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, writer, config.Config.GithubAppOrganization, reponame),
			"PUT",
			map[string]interface{}{"permission": "push"},
		)
		if err != nil {
			logrus.Errorf("failed to create repository (and add members): %v. %s", err, string(body))
		}
	}
}

func (g *GoliacRemoteImpl) UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string) {
	// update member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, teamslug, config.Config.GithubAppOrganization, reponame),
		"PUT",
		map[string]interface{}{"permission": permission},
	)
	if err != nil {
		logrus.Errorf("failed to add team access: %v. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) {
	// update member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, teamslug, config.Config.GithubAppOrganization, reponame),
		"PUT",
		map[string]interface{}{"permission": permission},
	)
	if err != nil {
		logrus.Errorf("failed to add team access: %v. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string) {
	// delete member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#remove-a-repository-from-a-team
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, teamslug, config.Config.GithubAppOrganization, reponame),
		"DELETE",
		nil,
	)
	if err != nil {
		logrus.Errorf("failed to remove team access: %. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) UpdateRepositoryUpdatePrivate(reponame string, private bool) {
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#update-a-repository
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("repos/%s/%s", config.Config.GithubAppOrganization, reponame),
		"PATCH",
		map[string]interface{}{"private": private},
	)
	if err != nil {
		logrus.Errorf("failed to update repository private setting: %v. %s", err, string(body))
	}
}
func (g *GoliacRemoteImpl) UpdateRepositoryUpdateArchived(reponame string, archived bool) {
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#update-a-repository
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("repos/%s/%s", config.Config.GithubAppOrganization, reponame),
		"PATCH",
		map[string]interface{}{"archived": archived},
	)
	if err != nil {
		logrus.Errorf("failed to update repository archive setting: %v. %s", err, string(body))
	}
}

func (g *GoliacRemoteImpl) DeleteRepository(reponame string) {
	// delete repo
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#delete-a-repository
	body, err := g.client.CallRestAPI(
		fmt.Sprintf("/orgs/%s/%s", config.Config.GithubAppOrganization, reponame),
		"DELETE",
		nil,
	)
	if err != nil {
		logrus.Errorf("failed to delete repository: %v. %s", err, string(body))
	}

	// update the repositories list
	if r, ok := g.repositories[reponame]; ok {
		delete(g.repositoriesByRefId, r.RefId)
		delete(g.repositories, reponame)
	}

}
func (g *GoliacRemoteImpl) Begin() {
}
func (g *GoliacRemoteImpl) Rollback(error) {
}
func (g *GoliacRemoteImpl) Commit() {
}
