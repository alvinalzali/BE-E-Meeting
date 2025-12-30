package usecases

import (
	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/repositories"
)

type SnackUsecase interface {
	GetAll() ([]entities.Snack, error)
}

type snackUsecase struct {
	snackRepo repositories.SnackRepository
}

func NewSnackUsecase(snackRepo repositories.SnackRepository) SnackUsecase {
	return &snackUsecase{snackRepo: snackRepo}
}

func (u *snackUsecase) GetAll() ([]entities.Snack, error) {
	return u.snackRepo.GetAll()
}
