package repository

import "se-school/internal/models"

func (r *Repository) Create(repository *models.Repository) error {
	return r.db.Create(repository).Error
}

func (r *Repository) UpdateTag(id uint, tag string) (*models.Repository, error) {
	repository, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	repository.Version = tag

	return repository, r.db.Save(repository).Error
}

func (r *Repository) Delete(repository *models.Repository) error {
	repository, err := r.GetByID(repository.ID)
	if err != nil {
		return err
	}

	return r.db.Delete(repository).Error
}

func (r *Repository) FindOrCreate(repository *models.Repository) (*models.Repository, error) {
	err := r.db.FirstOrCreate(repository).Error
	if err != nil {
		return nil, err
	}

	return repository, nil
}
