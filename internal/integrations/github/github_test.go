package github

import (
	"os"
	"se-school/internal/config"
	"testing"

	"github.com/redis/go-redis/v9"
)

const (
	testRepositoryOwner = "archlinux"
	testRepositoryName  = "linux"
)

func TestGithubIntegration(t *testing.T) {
	if os.Getenv("IS_LOCAL") != "true" {
		t.Skip()
	}
	githubService := setupGithubService(t)

	t.Run("Fetch repository version", func(t *testing.T) {
		version, err := githubService.GetRepositoryVersion(t.Context(), testRepositoryOwner, testRepositoryName)
		if err != nil {
			t.Error(err)
		}
		t.Log(version)
	})

	// todo: write tests on rate-limiting logic
}

func setupGithubService(t *testing.T) *GithubService {
	t.Helper()

	ghConfig := &config.Github{
		Token: os.Getenv("GITHUB_TOKEN"),
	}

	redisAddr := os.Getenv("REDIS_ADDRESS")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	return New(ghConfig, redisClient)
}
