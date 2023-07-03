package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

type GitHubClient interface {
	QueryGraphQLAPI(query string, variables map[string]interface{}) ([]byte, error)
	CallRestAPI(endpoint, method string, body map[string]interface{}) ([]byte, error)
	GetAccessToken() (string, error)
}

type GitHubClientImpl struct {
	gitHubServer    string
	appID           string
	installationID  int
	privateKey      []byte
	accessToken     string
	httpClient      *http.Client
	tokenExpiration time.Time
	mu              sync.Mutex
}

type AuthorizedTransport struct {
	client *GitHubClientImpl
}

func (t *AuthorizedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.client.mu.Lock()
	defer t.client.mu.Unlock()

	// Refresh the access token if necessary
	if t.client.accessToken == "" || time.Until(t.client.tokenExpiration) < 5*time.Minute {
		token, err := t.client.createJWT()
		if err != nil {
			return nil, err
		}

		accessToken, expiresAt, err := t.client.getAccessTokenForInstallation(token)
		if err != nil {
			return nil, err
		}

		t.client.accessToken = accessToken
		t.client.tokenExpiration = expiresAt
	}

	req.Header.Add("Authorization", "Bearer "+t.client.accessToken)

	return http.DefaultTransport.RoundTrip(req)
}

/**
 * NewGitHubClient
 * @param {string} githubServer usually https://api.github.com
 * @param {string} organizationName
 * @param {string} appID
 * @param {string} privateKeyFile
 * @return {GitHubClient} client
 * @return {error} error
 *
 * Example:
 * client, err := NewGitHubClient(
 * 	"https://api.github.com",
 * 	"my-org",
 * 	"12345",
 * 	"private-key.pem",
 * )
 */
func NewGitHubClientImpl(githubServer, organizationName, appID, privateKeyFile string) (GitHubClient, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}

	client := &GitHubClientImpl{
		gitHubServer: githubServer,
		appID:        appID,
		privateKey:   privateKey,
	}

	// create JWT
	token, err := client.createJWT()
	if err != nil {
		return nil, err
	}

	// retrieve all installations for the authenticated app
	installations, err := client.getInstallations(token)
	if err != nil {
		return nil, err
	}

	// find the installation ID for the given organization
	for _, installation := range installations {
		if installation.Account.Login == organizationName {
			client.installationID = installation.ID
			break
		}
	}

	if client.installationID == 0 {
		return nil, fmt.Errorf("installation not found for organization: %s", organizationName)
	}

	transport := &AuthorizedTransport{
		client: client,
	}

	httpClient := &http.Client{Transport: transport}

	client.httpClient = httpClient

	return client, nil
}

// waitRateLimix helps dealing with rate limits
// cf https://docs.github.com/en/rest/guides/best-practices-for-integrators?apiVersion=2022-11-28#dealing-with-rate-limits
func waitRateLimix(resetTimeStr string) error {
	if resetTimeStr == "" {
		return fmt.Errorf("X-RateLimit-Reset header not found")
	}

	logrus.Infof("Rate limit exceeded, waiting for %s", resetTimeStr)

	// Parse the reset time.
	resetTimeUnix, err := strconv.ParseInt(resetTimeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse X-RateLimit-Reset header: %w", err)
	}

	resetTime := time.Unix(resetTimeUnix, 0)

	// Calculate how long we need to wait.
	waitDuration := time.Until(resetTime)

	// Wait until the reset time.
	time.Sleep(waitDuration)

	return nil
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

/*
 * QueryGraphQLAPI
 * @param {string} query
 * @param {map[string]interface{}} variables
 * @return {string} responseBody
 * @return {error} error
 *
 * Example:
 * query := `
 *	query($name: String!) {
 *		user(login: $name) {
 *			name
 *			company
 *		}
 *	}
 * `
 * variables := map[string]interface{}{
 *	"name": "octocat",
 * }
 * responseBody, err := client.QueryGraphQLAPI(query, variables)
 */
func (client *GitHubClientImpl) QueryGraphQLAPI(query string, variables map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(GraphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", client.gitHubServer+"/graphql", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		// We're being rate limited. Get the reset time from the headers.
		if err := waitRateLimix(resp.Header.Get("X-RateLimit-Reset")); err != nil {
			return nil, err
		}

		// Retry the request.
		return client.QueryGraphQLAPI(query, variables)
	} else {
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return responseBody, nil
	}
}

/*
 * CallRestAPIWithBody
 * @param {string} endpoint
 * @param {string} method
 * @param {map[string]interface{}} body
 *
 * Example:
 * body := map[string]interface{}{
 *	"name": "my-repo",
 *	"private": true,
 * }
 * responseBody, err := client.CallRestAPIWithBody("orgs/my-org/repos", "POST", body)
 */
func (client *GitHubClientImpl) CallRestAPI(endpoint, method string, body map[string]interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}
	req, err := http.NewRequest(method, client.gitHubServer+"/"+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	//	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		// We're being rate limited. Get the reset time from the headers.
		if err := waitRateLimix(resp.Header.Get("X-RateLimit-Reset")); err != nil {
			return nil, err
		}

		// Retry the request.
		return client.CallRestAPI(endpoint, method, body)
	} else {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return responseBody, nil
	}
}

func (client *GitHubClientImpl) createJWT() (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(client.privateKey)
	if err != nil {
		return "", err
	}

	// create a JWT
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": int32(time.Now().Unix()),
		"exp": int32(time.Now().Add(10 * time.Minute).Unix()),
		"iss": client.appID,
	})

	// sign the JWT with the app's private key
	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

type AccessTokenResponse struct {
	Token string `json:"token"`
}

func (client *GitHubClientImpl) getAccessTokenForInstallation(jwt string) (string, time.Time, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/app/installations/%d/access_tokens", client.gitHubServer, client.installationID), nil)
	if err != nil {
		return "", time.Now(), err
	}

	req.Header.Add("Authorization", "Bearer "+jwt)
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", time.Now(), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", time.Now(), fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var accessTokenResponse AccessTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&accessTokenResponse)
	if err != nil {
		return "", time.Now(), err
	}

	return accessTokenResponse.Token, time.Now().Add(1 * time.Hour), nil
}

/*
 * GetAccessToken
 * It is used mostly to get an access token to clone a private repository
 * @return {string} accessToken
 * @return {error} error
 *
 * Example:
 * accessToken, err := client.GetAccessToken()
 * if err != nil {
 *	log.Fatal(err)
 * }
 * repo, err := git.PlainClone("/path/to/clone/repository", false, &git.CloneOptions{
 *	URL: "https://github.com/owner/repo.git",
 *	Auth: &http.BasicAuth{
 *		Username: "x-access-token",
 *		Password: accessToken,
 *	},
 */
func (client *GitHubClientImpl) GetAccessToken() (string, error) {

	if client.accessToken != "" && client.tokenExpiration.After(time.Now()) {
		return client.accessToken, nil
	}

	jwt, err := client.createJWT()
	if err != nil {
		return "", err
	}

	accessToken, expiration, err := client.getAccessTokenForInstallation(jwt)
	if err != nil {
		return "", err
	}

	client.accessToken = accessToken
	client.tokenExpiration = expiration

	return accessToken, nil
}
