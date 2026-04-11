package github

import "context"

type GithubIntegration interface {
	GetRepositoryVersion(ctx context.Context, owner, repositoryName string) (string, error)
}
