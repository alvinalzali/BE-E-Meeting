package database

import (
	"BE-E-MEETING/config"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"strconv"
)

type postgres struct {
	db *sql.DB
}

func NewPostgresDatabase(cfg *config.Config) (Database, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	fmt.Println("Connected to DB " + cfg.Database.DBName + " successfully on port" + strconv.Itoa(cfg.Database.Port))

	return &postgres{db: db}, nil
}

func (p *postgres) GetDB() *sql.DB {
	return p.db
}

func (p *postgres) Close() error {
	return p.db.Close()
}
