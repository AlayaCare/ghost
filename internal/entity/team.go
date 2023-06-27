package entity

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Team struct {
	Entity `yaml:",inline"`
	Data   struct {
		Owners  []string `yaml:"owners"`
		Members []string `yaml:"members"`
	} `yaml:"data"`
}

/*
 * NewTeam reads a file and returns a Team object
 * The next step is to validate the Team object using the Validate method
 */
func NewTeam(fs afero.Fs, filename string) (*Team, error) {
	filecontent, err := afero.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	team := &Team{}
	err = yaml.Unmarshal(filecontent, team)
	if err != nil {
		return nil, err
	}

	return team, nil
}

/**
 * ReadTeamDirectory reads all the files in the dirname directory and returns
 * - a map of Team objects
 * - a slice of errors that must stop the vlidation process
 * - a slice of warning that must not stop the validation process
 */
func ReadTeamDirectory(fs afero.Fs, dirname string, users map[string]*User) (map[string]*Team, []error, []error) {
	errors := []error{}
	warning := []error{}
	teams := make(map[string]*Team)

	exist, err := afero.Exists(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return teams, errors, warning
	}
	if exist == false {
		return teams, errors, warning
	}

	// Parse all the users in the dirname directory
	entries, err := afero.ReadDir(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return nil, errors, warning
	}

	for _, e := range entries {
		if e.IsDir() {
			team, err := NewTeam(fs, filepath.Join(dirname, e.Name(), "team.yaml"))
			if err != nil {
				errors = append(errors, err)
			} else {
				err, warns := team.Validate(filepath.Join(dirname, e.Name()), users)
				warning = append(warning, warns...)
				if err != nil {
					errors = append(errors, err)
				} else {
					teams[team.Metadata.Name] = team
				}
			}
		}
	}
	return teams, errors, warning
}

func (t *Team) Validate(dirname string, users map[string]*User) (error, []error) {
	warnings := []error{}

	if t.ApiVersion != "v1" {
		return fmt.Errorf("invalid apiVersion: %s for team filename %s/team.yaml", t.ApiVersion, dirname), warnings
	}

	if t.Kind != "Team" {
		return fmt.Errorf("invalid kind: %s for team filename %s/team.yaml", t.Kind, dirname), warnings
	}

	if t.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is empty for team filename %s", dirname), warnings
	}

	if strings.HasSuffix(t.Metadata.Name, "-owners") {
		return fmt.Errorf("metadata.name cannot finish with '-owners' for team filename %s. It is a reserved suffix", dirname), warnings
	}

	teamname := filepath.Base(dirname)
	if t.Metadata.Name != teamname {
		return fmt.Errorf("invalid metadata.name: %s for team filename %s/team.yaml", t.Metadata.Name, dirname), warnings
	}

	for _, owner := range t.Data.Owners {
		if _, ok := users[owner]; !ok {
			return fmt.Errorf("invalid owner: %s doesn't exist in team filename %s/team.yaml", owner, dirname), warnings
		}
	}

	for _, member := range t.Data.Members {
		if _, ok := users[member]; !ok {
			return fmt.Errorf("invalid member: %s doesn't exist in team filename %s/team.yaml", member, dirname), warnings
		}
	}

	// warnings

	if len(t.Data.Owners) < 2 {
		warnings = append(warnings, fmt.Errorf("no enough owner for team filename %s/team.yaml", dirname))
	}

	return nil, warnings
}
