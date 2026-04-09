package github

import (
	"context"
	"fmt"
	"se-school/internal/config"

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
	release, res, err := g.client.Repositories.GetLatestRelease(ctx, owner, repositoryName)
	if err != nil {
		zap.L().Error("failed to get repository version", zap.Error(err))
		return "", err
	}
	// todo: handle rate limit errors
	if res.StatusCode != 200 {
		zap.L().Error("failed to get repository version", zap.Int("status", res.StatusCode))
		return "", fmt.Errorf("failed to get repository version, api status: %s", res.Status)
	}

	return release.GetTagName(), nil
}
