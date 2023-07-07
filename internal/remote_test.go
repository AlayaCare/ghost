package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

func GetGithubGraphqlSchema() (string, error) {
	response, err := http.Get("https://docs.github.com/public/schema.docs.graphql")
	if err != nil {
		return "", err
	} else {
		defer response.Body.Close()
		content, err := ioutil.ReadAll(response.Body)
		return string(content), err
	}
}

type MockGithubClient struct {
	cursorValue    string
	cursorPosition int
}

type GraphQLResult struct {
	Data map[string]interface{} `mapstructure:"data" json:"data"`
}

func extractVariable(name string, args ast.ArgumentList, variables map[string]interface{}) string {
	value := ""
	for _, a := range args {
		if a.Name == name {
			value = a.Value.String()
			if len(value) > 0 && value[0] == '$' {
				if variables[value[1:]] == nil {
					value = ""
				} else {
					value = variables[value[1:]].(string)
				}
			}
			return value
		}
	}
	return value
}

func hasChild(childname string, children ast.SelectionSet) (bool, *ast.Field) {
	for _, c := range children {
		switch s := c.(type) {
		case *ast.Field:
			if s.Name == childname {
				return true, s
			}
			//		case *ast.InlineFragment:
			//		case *ast.FragmentSpread:
		}
	}
	return false, nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

/*
 * Returns:
 * - nodes
 * - hasNextPage
 * - endCursor
 * - totalCount
 */
func (m *MockGithubClient) reposNodes(first, after string, args ast.ArgumentList, children ast.SelectionSet, variables map[string]interface{}, maxToFake int) ([]map[string]interface{}, bool, *string, int) {
	data := make([]map[string]interface{}, 0)

	iFirst, err := strconv.Atoi(first)
	if err != nil {
		iFirst = 0
	}
	iAfter := 0
	if after == m.cursorValue {
		iAfter = m.cursorPosition
	}

	searchName, _ := hasChild("name", children)
	searchArchived, _ := hasChild("isArchived", children)
	searchPrivate, _ := hasChild("isPrivate", children)

	index := iAfter
	totalCount := 0
	hasNext := true
	var endCursor *string
	r := (randStringRunes(12))
	endCursor = &r
	for totalCount = 0; totalCount < iFirst; totalCount++ {
		block := make(map[string]interface{})
		if searchName {
			block["name"] = fmt.Sprintf("repo_%d", index)
		}
		if searchArchived {
			block["isArchived"] = index%3 == 0 // let's pretend each 3 repo is an archive repo
		}
		if searchPrivate {
			block["isPrivate"] = index%10 == 0 // let's pretend each 10 repo is a private repo
		}
		index++
		if index > maxToFake { // let's pretend we have maxToFake repos
			hasNext = false
			endCursor = nil
			break
		}
		data = append(data, block)
	}

	m.cursorValue = r
	m.cursorPosition = index

	return data, hasNext, endCursor, totalCount
}

/*
 * Returns:
 * - nodes
 * - hasNextPage
 * - endCursor
 * - totalCount
 */
func (m *MockGithubClient) reposEdges(first, after string, args ast.ArgumentList, children ast.SelectionSet, variables map[string]interface{}, maxToFake int) ([]map[string]interface{}, bool, *string, int) {
	data := make([]map[string]interface{}, 0)

	iFirst, err := strconv.Atoi(first)
	if err != nil {
		iFirst = 0
	}
	iAfter := 0
	if after == m.cursorValue {
		iAfter = m.cursorPosition
	}

	searchPermission, _ := hasChild("permission", children)
	searchNode, nodeField := hasChild("node", children)

	index := iAfter
	totalCount := 0
	hasNext := true
	var endCursor *string
	r := (randStringRunes(12))
	endCursor = &r
	for totalCount = 0; totalCount < iFirst; totalCount++ {
		block := make(map[string]interface{})
		if searchPermission {
			block["permission"] = "push"
		}
		if searchNode {
			node := make(map[string]interface{})
			if c, _ := hasChild("name", nodeField.SelectionSet); c {
				node["name"] = fmt.Sprintf("repo_%d", index)
			}
			block["node"] = node
		}
		index++
		if index > maxToFake { // let's pretend we have maxToFake repos
			hasNext = false
			endCursor = nil
			break
		}
		data = append(data, block)
	}

	m.cursorValue = r
	m.cursorPosition = index

	return data, hasNext, endCursor, totalCount
}

func (m *MockGithubClient) repositories(args ast.ArgumentList, children ast.SelectionSet, variables map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	first := extractVariable("first", args, variables)
	after := extractVariable("after", args, variables)

	var hasNextPage bool
	var endCursor *string
	var totalCount int
	if c, s := hasChild("nodes", children); c {
		data["nodes"], hasNextPage, endCursor, totalCount = m.reposNodes(first, after, s.Arguments, s.SelectionSet, variables, 133)
	}
	if c, _ := hasChild("pageInfo", children); c {
		block := make(map[string]interface{})
		block["hasNextPage"] = hasNextPage
		if endCursor == nil {
			block["endCursor"] = nil
		} else {
			block["endCursor"] = *endCursor
		}

		data["pageInfo"] = block
	}
	if c, _ := hasChild("totalCount", children); c {
		data["totalCount"] = totalCount
	}

	return data
}

/*
 * Returns:
 * - nodes
 * - hasNextPage
 * - endCursor
 * - totalCount
 */
func (m *MockGithubClient) teamsNodes(first, after string, args ast.ArgumentList, children ast.SelectionSet, variables map[string]interface{}) ([]map[string]interface{}, bool, *string, int) {
	data := make([]map[string]interface{}, 0)

	iFirst, err := strconv.Atoi(first)
	if err != nil {
		iFirst = 0
	}
	iAfter := 0
	if after == m.cursorValue {
		iAfter = m.cursorPosition
	}

	searchName, _ := hasChild("name", children)
	searchSlug, _ := hasChild("slug", children)

	index := iAfter
	totalCount := 0
	hasNext := true
	var endCursor *string
	r := (randStringRunes(12))
	endCursor = &r
	for totalCount = 0; totalCount < iFirst; totalCount++ {
		block := make(map[string]interface{})
		if searchName {
			block["name"] = fmt.Sprintf("team_%d", index)
		}
		if searchSlug {
			block["slug"] = fmt.Sprintf("slug-%d", index)
		}
		index++
		if index > 122 { // let's pretend we have 133 teams
			hasNext = false
			endCursor = nil
			break
		}
		data = append(data, block)
	}

	m.cursorValue = r
	m.cursorPosition = index

	return data, hasNext, endCursor, totalCount
}

func (m *MockGithubClient) teams(args ast.ArgumentList, children ast.SelectionSet, variables map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	first := extractVariable("first", args, variables)
	after := extractVariable("after", args, variables)

	var hasNextPage bool
	var endCursor *string
	var totalCount int
	if c, s := hasChild("nodes", children); c {
		data["nodes"], hasNextPage, endCursor, totalCount = m.teamsNodes(first, after, s.Arguments, s.SelectionSet, variables)
	}
	if c, _ := hasChild("pageInfo", children); c {
		block := make(map[string]interface{})
		block["hasNextPage"] = hasNextPage
		if endCursor == nil {
			block["endCursor"] = nil
		} else {
			block["endCursor"] = *endCursor
		}

		data["pageInfo"] = block
	}
	if c, _ := hasChild("totalCount", children); c {
		data["totalCount"] = totalCount
	}

	return data
}

func (m *MockGithubClient) teamrepositories(args ast.ArgumentList, children ast.SelectionSet, variables map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	first := extractVariable("first", args, variables)
	after := extractVariable("after", args, variables)

	var hasNextPage bool
	var endCursor *string
	var totalCount int
	if c, s := hasChild("edges", children); c {
		data["edges"], hasNextPage, endCursor, totalCount = m.reposEdges(first, after, s.Arguments, s.SelectionSet, variables, 2)
	}
	if c, _ := hasChild("pageInfo", children); c {
		block := make(map[string]interface{})
		block["hasNextPage"] = hasNextPage
		if endCursor == nil {
			block["endCursor"] = nil
		} else {
			block["endCursor"] = *endCursor
		}

		data["pageInfo"] = block
	}
	if c, _ := hasChild("totalCount", children); c {
		data["totalCount"] = totalCount
	}

	return data
}

func (m *MockGithubClient) team(args ast.ArgumentList, children ast.SelectionSet, variables map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	//	slug := extractVariable("slug", args, variables)

	if c, s := hasChild("repositories", children); c {
		data["repositories"] = m.teamrepositories(s.Arguments, s.SelectionSet, variables)
	}

	return data
}
func (m *MockGithubClient) organization(args ast.ArgumentList, children ast.SelectionSet, variables map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	//login := extractVariable("login", args, variables)

	if c, s := hasChild("repositories", children); c {
		data["repositories"] = m.repositories(s.Arguments, s.SelectionSet, variables)
	}
	if c, s := hasChild("teams", children); c {
		data["teams"] = m.teams(s.Arguments, s.SelectionSet, variables)
	}
	if c, s := hasChild("team", children); c {
		data["team"] = m.team(s.Arguments, s.SelectionSet, variables)
	}
	return data
}

func (m *MockGithubClient) QueryGraphQLAPI(query string, variables map[string]interface{}) ([]byte, error) {

	doc, err := parser.ParseQuery(&ast.Source{Input: query})

	// Check for parsing error.
	if err != nil {
		return nil, err
	}

	result := GraphQLResult{
		Data: make(map[string]interface{}),
	}

	// Print the parsed query document.
	for _, op := range doc.Operations {
		if op.Operation == "query" {
			if c, s := hasChild("organization", op.SelectionSet); c {
				result.Data["organization"] = m.organization(s.Arguments, s.SelectionSet, variables)
			}
		}
	}

	j, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (m *MockGithubClient) CallRestAPI(endpoint, method string, body map[string]interface{}) ([]byte, error) {
	return nil, nil
}
func (m *MockGithubClient) GetAccessToken() (string, error) {
	return "", nil
}

func TestRemoteRepository(t *testing.T) {

	// happy path
	t.Run("happy path: load remote repositories", func(t *testing.T) {
		client := MockGithubClient{}

		remote := NewGoliacRemoteImpl(&client)

		remoteImpl := remote.(*GoliacRemoteImpl)

		err := remoteImpl.loadRepositories()
		assert.Nil(t, err)
		assert.Equal(t, 133, len(remoteImpl.repositories))
		assert.Equal(t, false, remoteImpl.repositories["repo_1"].IsArchived)
		assert.Equal(t, true, remoteImpl.repositories["repo_3"].IsArchived)
		assert.Equal(t, false, remoteImpl.repositories["repo_1"].IsPrivate)
		assert.Equal(t, true, remoteImpl.repositories["repo_10"].IsPrivate)
	})
	t.Run("happy path: load remote teams", func(t *testing.T) {
		client := MockGithubClient{}

		remote := NewGoliacRemoteImpl(&client)

		remoteImpl := remote.(*GoliacRemoteImpl)
		err := remoteImpl.loadTeams()
		assert.Nil(t, err)
		assert.Equal(t, 122, len(remoteImpl.teams))
		assert.Equal(t, "team_1", remoteImpl.teams["slug-1"].Name)
	})

	t.Run("happy path: load remote team's repos", func(t *testing.T) {
		client := MockGithubClient{}

		remote := NewGoliacRemoteImpl(&client)

		remoteImpl := remote.(*GoliacRemoteImpl)
		repos, err := remoteImpl.loadTeamRepos("team-1")
		assert.Nil(t, err)
		assert.Equal(t, 2, len(repos))
		assert.Equal(t, "push", repos["repo_0"].Permission)
	})

	t.Run("happy path: load remote teams and team's repos", func(t *testing.T) {
		client := MockGithubClient{}

		remote := NewGoliacRemoteImpl(&client)

		remoteImpl := remote.(*GoliacRemoteImpl)
		err := remoteImpl.Load()
		assert.Nil(t, err)
		assert.Equal(t, 122, len(remoteImpl.teams))
		assert.Equal(t, 2, len(remoteImpl.teamRepos["slug-1"]))
	})
}
