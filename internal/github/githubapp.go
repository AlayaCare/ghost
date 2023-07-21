package github

import (
	"encoding/json"
	"net/http"
)

type Installation struct {
	ID      int    `json:"id"`
	AppId   int    `json:"app_id"`
	AppSlug string `json:"app_slug"`
	Account struct {
		Login string `json:"login"`
	} `json:"account"`
}

func (client *GitHubClientImpl) getInstallations(jwt string) ([]Installation, error) {
	req, err := http.NewRequest("GET", client.gitHubServer+"/app/installations", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+jwt)
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var installations []Installation
	err = json.NewDecoder(resp.Body).Decode(&installations)
	if err != nil {
		return nil, err
	}

	return installations, nil
}
