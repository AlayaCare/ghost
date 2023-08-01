package entity

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func fixtureCreateRuleSet(t *testing.T, fs afero.Fs) {
	fs.Mkdir("rulesets", 0755)
	err := afero.WriteFile(fs, "rulesets/ruleset1.yaml", []byte(`
apiVersion: v1
kind: Ruleset
metadata:
  name: ruleset1
enforcement: evaluate
bypassapps:
  - appname: goliac-project-app
    mode: always
on:
  include: 
  - "~DEFAULT_BRANCH"

rules:
  - ruletype: pull_request
    parameters:
      requiredApprovingReviewCount: 1
`), 0644)
	assert.Nil(t, err)

	err = afero.WriteFile(fs, "rulesets/ruleset2.yaml", []byte(`
apiVersion: v1
kind: Ruleset
metadata:
  name: ruleset2
enforcement: evaluate
bypassapps:
  - appname: goliac-project-app
    mode: always
on:
  include: 
  - "~DEFAULT_BRANCH"

rules:
  - ruletype: required_status_checks
    parameters:
      requiredStatusChecks:
      - circleCI check
      - jenkins check
`), 0644)
	assert.Nil(t, err)
}

func TestRuleset(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := afero.NewMemMapFs()
		fixtureCreateRuleSet(t, fs)

		rulesets, errs, warns := ReadRuleSetDirectory(fs, "rulesets")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, rulesets)
		assert.Equal(t, 2, len(rulesets))

	})
}

func TestRulesetParametersComparison(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := afero.NewMemMapFs()
		fixtureCreateRuleSet(t, fs)

		rulesets, errs, warns := ReadRuleSetDirectory(fs, "rulesets")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, rulesets)

		res := CompareRulesetParameters(rulesets["ruleset1"].Rules[0].Ruletype, rulesets["ruleset1"].Rules[0].Parameters, rulesets["ruleset2"].Rules[0].Parameters)
		assert.False(t, res)

		res = CompareRulesetParameters(rulesets["ruleset1"].Rules[0].Ruletype, rulesets["ruleset1"].Rules[0].Parameters, rulesets["ruleset1"].Rules[0].Parameters)
		assert.True(t, res)

		res = CompareRulesetParameters(rulesets["ruleset2"].Rules[0].Ruletype, rulesets["ruleset2"].Rules[0].Parameters, rulesets["ruleset2"].Rules[0].Parameters)
		assert.True(t, res)
	})
}
