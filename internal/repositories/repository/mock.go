package repository

import (
	"se-school/internal/models"
)

type RepositoriesRepositoryMock struct {
	Repositories map[uint]*models.Repository

	GetByIDErr      error
	GetAllErr       error
	FindErr         error
	CreateErr       error
	FindOrCreateErr error
	UpdateTagErr    error
	DeleteErr       error

	UpdateTagCalls []UpdateTagCall
}

type UpdateTagCall struct {
	ID  uint
	Tag string
}

func NewRepositoriesRepositoryMock() *RepositoriesRepositoryMock {
	return &RepositoriesRepositoryMock{
		Repositories: make(map[uint]*models.Repository),
	}
}

func (m *RepositoriesRepositoryMock) GetByID(id uint) (*models.Repository, error) {
	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}
	repo, ok := m.Repositories[id]
	if !ok {
		return nil, m.GetByIDErr
	}
	return repo, nil
}

func (m *RepositoriesRepositoryMock) GetAll() ([]*models.Repository, error) {
	if m.GetAllErr != nil {
		return nil, m.GetAllErr
	}
	result := make([]*models.Repository, 0, len(m.Repositories))
	for _, repo := range m.Repositories {
		result = append(result, repo)
	}
	return result, nil
}

func (m *RepositoriesRepositoryMock) Find(repo *models.Repository) (*models.Repository, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	for _, r := range m.Repositories {
		if r.Owner == repo.Owner && r.Name == repo.Name {
			return r, nil
		}
	}
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	return nil, models.ErrNotFound
}

func (m *RepositoriesRepositoryMock) Create(repo *models.Repository) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.Repositories[repo.ID] = repo
	return nil
}

func (m *RepositoriesRepositoryMock) FindOrCreate(repo *models.Repository) (*models.Repository, error) {
	if m.FindOrCreateErr != nil {
		return nil, m.FindOrCreateErr
	}
	m.Repositories[repo.ID] = repo
	return repo, nil
}

func (m *RepositoriesRepositoryMock) UpdateTag(id uint, tag string) (*models.Repository, error) {
	m.UpdateTagCalls = append(m.UpdateTagCalls, UpdateTagCall{ID: id, Tag: tag})
	if m.UpdateTagErr != nil {
		return nil, m.UpdateTagErr
	}
	repo, ok := m.Repositories[id]
	if !ok {
		return nil, m.UpdateTagErr
	}
	repo.Version = tag
	return repo, nil
}

func (m *RepositoriesRepositoryMock) Delete(repo *models.Repository) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	delete(m.Repositories, repo.ID)
	return nil
}
