package github

import "context"

type GithubIntegrationMock struct {
	versionToReturn string
	errToReturn     error
}

func NewGithubIntegrationMock(versionToReturn string) *GithubIntegrationMock {
	return &GithubIntegrationMock{
		versionToReturn: versionToReturn,
	}
}

func (m *GithubIntegrationMock) GetRepositoryVersion(_ context.Context, _, _ string) (string, error) {
	return m.versionToReturn, m.errToReturn
}

func (m *GithubIntegrationMock) SetVersionToReturn(version string) {
	m.versionToReturn = version
}

func (m *GithubIntegrationMock) SetErrToReturn(err error) {
	m.errToReturn = err
}
