package github

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestQueryGraphQLAPI(t *testing.T) {
	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"user": {"name": "octocat"}}}`))
	}))
	defer testServer.Close()

	// Replace the httpClient with a mock
	client := &GitHubClientImpl{
		httpClient: &http.Client{
			Transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return http.Get(testServer.URL) // Use the test server's URL
				},
			},
		},
	}

	// Call the function and check the result
	query := `query { user(login: "octocat") { name } }`
	result, err := client.QueryGraphQLAPI(query, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(string(result), "octocat") {
		t.Errorf("expected 'octocat' in the result, got %s", result)
	}
}
