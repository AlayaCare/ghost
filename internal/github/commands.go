package github

type GithubCommand interface {
	Apply() error
}
