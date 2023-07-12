package config

// Config is the whole configuration of the app
var Config = struct {

	// LogrusLevel sets the logrus logging level
	LogrusLevel string `env:"GOLIAC_LOGRUS_LEVEL" envDefault:"info"`
	// LogrusFormat sets the logrus logging formatter
	// Possible values: text, json
	LogrusFormat string `env:"GOLIAC_LOGRUS_FORMAT" envDefault:"text"`

	// PrometheusEnabled - enable prometheus metrics export
	PrometheusEnabled bool `env:"GOLIAC_PROMETHEUS_ENABLED" envDefault:"false"`
	// PrometheusPath - set the path on which prometheus metrics are available to scrape
	PrometheusPath string `env:"GOLIAC_PROMETHEUS_PATH" envDefault:"/metrics"`

	GithubServer            string `env:"GOLIAC_GITHUB_SERVER" envDefault:"https://api.github.com"`
	GithubAppOrganization   string `env:"GOLIAC_GITHUB_APP_ORGANIZATION" envDefault:""`
	GithubAppID             string `env:"GOLIAC_GITHUB_APP_ID" envDefault:""`
	GithubAppPrivateKeyFile string `env:"GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE" envDefault:"github-app-private-key.pem"`
	GoliacEmail             string `env:"GOLIAC_EMAIL" envDefault:"goliac@alayacare.com"`

	// goliacGitRepository string `env:"GOLIAC_GIT_REPOSITORY" envDefault:""`
	// goliacGitBranch     string `env:"GOLIAC_GIT_BRANCH" envDefault:"main"`
}{}
