package github

import (
	"os"
	"se-school/internal/config"
	"testing"
)

const (
	testRepositoryOwner = "archlinux"
	testRepositoryName  = "linux"
)

func TestGithubIntegration(t *testing.T) {
	githubService := setupGithubService()

	t.Run("Fetch repository version", func(t *testing.T) {
		version, err := githubService.GetRepositoryVersion(t.Context(), testRepositoryOwner, testRepositoryName)
		if err != nil {
			t.Error(err)
		}
		t.Log(version)
	})

	// todo: write tests on rate-limiting logic
}

func setupGithubService() *GithubService {
	ghConfig := &config.Github{
		Token: os.Getenv("GITHUB_TOKEN"),
	}

	return New(ghConfig)
}
