package github

import (
	"context"
	"errors"
	"fmt"
	"se-school/internal/config"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	repositoryVersionCacheTTL    = 10 * time.Minute
	repositoryVersionCachePrefix = "github:repo_version:"
)

type GithubService struct {
	client *github.Client
	cache  *redis.Client
}

func New(cfg *config.Github, cache *redis.Client) *GithubService {
	client := github.NewClient(nil).WithAuthToken(cfg.Token)
	return &GithubService{
		client: client,
		cache:  cache,
	}
}

func (g *GithubService) GetRepositoryVersion(ctx context.Context, owner, repositoryName string) (string, error) {
	cacheKey := buildRepoCacheKey(owner, repositoryName)

	cached, err := g.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		zap.L().Debug("cache hit for repository version",
			zap.String("owner", owner),
			zap.String("repository", repositoryName),
		)
		return cached, nil
	}

	if !errors.Is(err, redis.Nil) {
		zap.L().Warn("failed to read from redis cache, falling back to API",
			zap.Error(err),
		)
	}

	version, err := g.fetchRepositoryVersion(ctx, owner, repositoryName)
	if err != nil {
		return "", err
	}

	if cacheErr := g.cache.Set(ctx, cacheKey, version, repositoryVersionCacheTTL).Err(); cacheErr != nil {
		zap.L().Warn(
			"failed to cache repository version in redis",
			zap.Error(cacheErr),
		)
	}

	return version, nil
}

func (g *GithubService) fetchRepositoryVersion(ctx context.Context, owner, repositoryName string) (string, error) {
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
				return g.fetchRepositoryVersion(ctx, owner, repositoryName)
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		zap.L().Error("failed to get repository version", zap.Error(err))
		return "", err
	}

	return release.GetTagName(), nil
}

func buildRepoCacheKey(owner, repositoryName string) string {
	return fmt.Sprintf("%s%s/%s", repositoryVersionCachePrefix, owner, repositoryName)
}
