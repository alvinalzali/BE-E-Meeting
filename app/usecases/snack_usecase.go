package usecases

import (
	"BE-E-MEETING/app/models"
	"BE-E-MEETING/app/repositories"
)

type SnackUsecase interface {
	GetSnacks() ([]models.Snack, error)
}

type snackUsecase struct {
	snackRepo repositories.SnackRepository
}

func NewSnackUsecase(snackRepo repositories.SnackRepository) SnackUsecase {
	return &snackUsecase{snackRepo: snackRepo}
}

func (u *snackUsecase) GetSnacks() ([]models.Snack, error) {
	return u.snackRepo.GetSnacks()
}
