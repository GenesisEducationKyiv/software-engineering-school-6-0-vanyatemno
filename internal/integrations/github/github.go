package github

import (
	"context"
	"errors"
	"se-school/internal/config"
	"time"

	"github.com/google/go-github/v84/github"
	"go.uber.org/zap"
)

type GithubService struct {
	client *github.Client
}

func New(cfg *config.Github) *GithubService {
	client := github.NewClient(nil).WithAuthToken(cfg.Token)
	return &GithubService{
		client: client,
	}
}

func (g *GithubService) GetRepositoryVersion(ctx context.Context, owner, repositoryName string) (string, error) {
	release, _, err := g.client.Repositories.GetLatestRelease(ctx, owner, repositoryName)
	if err != nil {
		if rateLimitErr, ok := errors.AsType[*github.RateLimitError](err); ok {
			zap.L().Warn(
				"github integration just hit rate limit",
				zap.Time("rate limit reset time", rateLimitErr.Rate.GetReset().Time),
			)
			waitDuration := time.Until(rateLimitErr.Rate.GetReset().Time)
			select {
			case <-time.After(waitDuration):
				return g.GetRepositoryVersion(ctx, owner, repositoryName)
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		zap.L().Error("failed to get repository version", zap.Error(err))
		return "", err
	}

	return release.GetTagName(), nil
}
