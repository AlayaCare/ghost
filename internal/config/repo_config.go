package config

import (
	"gopkg.in/yaml.v3"
)

type RepositoryConfig struct {
	AdminTeam           string `yaml:"admin_team"`
	EveryoneTeamEnabled bool   `yaml:"everyone_team_enabled"`

	Rulesets []struct {
		Pattern string
		Ruleset string
	}
	MaxChangesets           int `yaml:"max_changesets"`
	GithubConcurrentThreads int `yaml:"github_concurrent_threads"`
	UserSync                struct {
		Plugin string `yaml:"plugin"`
		Path   string `yaml:"path"`
	}
	DestructiveOperations struct {
		AllowDestructiveRepositories bool `yaml:"repositories"`
		AllowDestructiveTeams        bool `yaml:"teams"`
		AllowDestructiveUsers        bool `yaml:"users"`
		AllowDestructiveRulesets     bool `yaml:"rulesets"`
	} `yaml:"destructive_operations"`
}

// set default values
func (rc *RepositoryConfig) UnmarshalYAML(value *yaml.Node) error {
	type myStructAlias RepositoryConfig // Create a new alias type to avoid recursion
	x := &myStructAlias{}
	x.AdminTeam = "admin"
	x.MaxChangesets = 50
	x.GithubConcurrentThreads = 4
	x.UserSync.Plugin = "noop"

	if err := value.Decode(x); err != nil {
		return err
	}

	*rc = RepositoryConfig(*x)
	return nil
}
