package github

import "context"

type GithubIntegrationMock struct {
	versionToReturn string
}

func NewGithubIntegrationMock(versionToReturn string) *GithubIntegrationMock {
	return &GithubIntegrationMock{
		versionToReturn: versionToReturn,
	}
}

func (m *GithubIntegrationMock) GetRepositoryVersion(_ context.Context, _, _ string) (string, error) {
	return m.versionToReturn, nil
}

func (m *GithubIntegrationMock) SetVersionToReturn(version string) {
	m.versionToReturn = version
}
