package repositories

import (
	"BE-E-MEETING/app/models"
	"database/sql"
)

type SnackRepository interface {
	GetSnacks() ([]models.Snack, error)
}

type snackRepository struct {
	db *sql.DB
}

func NewSnackRepository(db *sql.DB) SnackRepository {
	return &snackRepository{db: db}
}

func (r *snackRepository) GetSnacks() ([]models.Snack, error) {
	rows, err := r.db.Query(`SELECT id, name, unit, price, category FROM snacks ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snacks []models.Snack
	for rows.Next() {
		var s models.Snack
		if err := rows.Scan(&s.ID, &s.Name, &s.Unit, &s.Price, &s.Category); err != nil {
			return nil, err
		}
		snacks = append(snacks, s)
	}
	return snacks, nil
}
