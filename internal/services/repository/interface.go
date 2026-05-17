package repository

type RepositoriesService interface {
	CheckRepoTagAndAlert(id uint) error
	CheckAllReposTagAndAlert() error
}
