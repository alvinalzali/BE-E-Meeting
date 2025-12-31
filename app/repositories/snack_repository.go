package repositories

import (
	"BE-E-Meeting/app/entities"
	"database/sql"
)

type SnackRepository interface {
	GetAll() ([]entities.Snack, error)
	GetByID(id int) (entities.Snack, error)
}

type snackRepository struct {
	db *sql.DB
}

func NewSnackRepository(db *sql.DB) SnackRepository {
	return &snackRepository{db: db}
}

func (r *snackRepository) GetAll() ([]entities.Snack, error) {
	rows, err := r.db.Query(`SELECT id, name, unit, price, category FROM snacks ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snacks []entities.Snack
	for rows.Next() {
		var s entities.Snack
		if err := rows.Scan(&s.ID, &s.Name, &s.Unit, &s.Price, &s.Category); err != nil {
			return nil, err
		}
		snacks = append(snacks, s)
	}
	return snacks, nil
}

func (r *snackRepository) GetByID(id int) (entities.Snack, error) {
	var s entities.Snack
	err := r.db.QueryRow("SELECT id, name, unit, price, category FROM snacks WHERE id=$1", id).Scan(&s.ID, &s.Name, &s.Unit, &s.Price, &s.Category)
	return s, err
}
