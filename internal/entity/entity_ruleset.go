package entity

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type RuleSetParameters struct {
	// PullRequestParameters
	DismissStaleReviewsOnPush      bool `yaml:"dismissStaleReviewsOnPush"`
	RequireCodeOwnerReview         bool `yaml:"requireCodeOwnerReview"`
	RequiredApprovingReviewCount   int  `yaml:"requiredApprovingReviewCount"`
	RequiredReviewThreadResolution bool `yaml:"requiredReviewThreadResolution"`
	RequireLastPushApproval        bool `yaml:"requireLastPushApproval"`

	// RequiredStatusChecksParameters
	RequiredStatusChecks             []string `yaml:"requiredStatusChecks"`
	StrictRequiredStatusChecksPolicy bool     `yaml:"strictRequiredStatusChecksPolicy"`
}

func CompareRulesetParameters(ruletype string, left RuleSetParameters, right RuleSetParameters) bool {
	switch ruletype {
	case "required_signatures":
		return true
	case "pull_request":
		if left.DismissStaleReviewsOnPush != right.DismissStaleReviewsOnPush {
			return false
		}
		if left.RequireCodeOwnerReview != right.RequireCodeOwnerReview {
			return false
		}
		if left.RequiredApprovingReviewCount != right.RequiredApprovingReviewCount {
			return false
		}
		if left.RequiredReviewThreadResolution != right.RequiredReviewThreadResolution {
			return false
		}
		if left.RequireLastPushApproval != right.RequireLastPushApproval {
			return false
		}
		return true
	case "required_status_checks":
		if res, _, _ := StringArrayEquivalent(left.RequiredStatusChecks, right.RequiredStatusChecks); res == false {
			return false
		}
		if left.StrictRequiredStatusChecksPolicy != right.StrictRequiredStatusChecksPolicy {
			return false
		}
		return true
	}
	return false
}

/*
 * Ruleset are applied per repos based on the goliac configuration file (pattern x ruleset name)
 */
type RuleSet struct {
	Entity `yaml:",inline"`
	// Target // branch, tag
	Enforcement string // disabled, active, evaluate
	BypassApps  []struct {
		AppName string
		Mode    string // always, pull_request
	}
	On struct {
		Include []string // ~DEFAULT_BRANCH, ~ALL, branch_name, ...
		Exclude []string //  branch_name, ...
	}

	Rules []struct {
		Ruletype   string // required_signatures, pull_request, required_status_checks...
		Parameters RuleSetParameters
	} `yaml:"rules"`
}

/*
 * NewRuleSet reads a file and returns a RuleSet object
 * The next step is to validate the RuleSet object using the Validate method
 */
func NewRuleSet(fs afero.Fs, filename string) (*RuleSet, error) {
	filecontent, err := afero.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	ruleset := RuleSet{}
	err = yaml.Unmarshal(filecontent, &ruleset)
	if err != nil {
		return nil, err
	}

	return &ruleset, nil
}

/**
 * ReadRuleSetDirectory reads all the files in the dirname directory and returns
 * - a map of RuleSet objects
 * - a slice of errors that must stop the validation process
 * - a slice of warning that must not stop the validation process
 */
func ReadRuleSetDirectory(fs afero.Fs, dirname string) (map[string]*RuleSet, []error, []Warning) {
	errors := []error{}
	warning := []Warning{}
	rulesets := make(map[string]*RuleSet)

	exist, err := afero.Exists(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return rulesets, errors, warning
	}
	if exist == false {
		return rulesets, errors, warning
	}

	// Parse all the rulesets in the dirname directory
	entries, err := afero.ReadDir(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return rulesets, errors, warning
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ruleset, err := NewRuleSet(fs, filepath.Join(dirname, e.Name()))
		if err != nil {
			errors = append(errors, err)
		} else {
			err := ruleset.Validate(filepath.Join(dirname, e.Name()))
			if err != nil {
				errors = append(errors, err)
			} else {
				rulesets[ruleset.Metadata.Name] = ruleset
			}

		}
	}
	return rulesets, errors, warning
}

func (r *RuleSet) Validate(filename string) error {

	if r.ApiVersion != "v1" {
		return fmt.Errorf("invalid apiVersion: %s for ruleset filename %s", r.ApiVersion, filename)
	}

	if r.Kind != "Ruleset" {
		return fmt.Errorf("invalid kind: %s for ruleset filename %s", r.Kind, filename)
	}

	if r.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is empty for ruleset filename %s", filename)
	}

	filename = filepath.Base(filename)
	if r.Metadata.Name != filename[:len(filename)-len(filepath.Ext(filename))] {
		return fmt.Errorf("invalid metadata.name: %s for ruleset filename %s", r.Metadata.Name, filename)
	}

	for _, rule := range r.Rules {
		if rule.Ruletype != "required_signatures" && rule.Ruletype != "pull_request" {
			return fmt.Errorf("invalid rulettype: %s for ruleset filename %s", rule.Ruletype, filename)
		}
	}

	if r.Enforcement != "disable" && r.Enforcement != "active" && r.Enforcement != "evaluate" {
		return fmt.Errorf("invalid enforcement: %s for ruleset filename %s", r.Enforcement, filename)
	}

	for _, ba := range r.BypassApps {
		if ba.Mode != "always" && ba.Mode != "pull_request" {
			return fmt.Errorf("invalid mode: %s for bypassapp %s in ruleset filename %s", ba.Mode, ba.AppName, filename)
		}
	}
	for _, on := range r.On.Include {
		if on[0] == '~' && (on != "~DEFAULT_BRANCH" && on != "~ALL") {
			return fmt.Errorf("invalid include: %s in ruleset filename %s", on, filename)
		}
	}

	return nil
}
