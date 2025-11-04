package repositories

import (
	"BE-E-MEETING/app/models"
	"database/sql"
)

type UserRepository interface {
	GetUserByUsernameOrEmail(username string) (string, string, error)
	GetUserRole(username string) (string, error)
	GetUserID(username string) (string, error)
	RegisterUser(user models.User, hashedPassword, status string) error
	UpdateUserPassword(hashedPassword, id string) error
	GetUserEmail(email string) (string, error)
	GetUserByID(id int) (models.GetUser, error)
	UpdateUser(id int, user models.UpdateUser) error
	CheckUsernameExists(username string, id int) (bool, error)
	CheckEmailExists(email string, id int) (bool, error)
	GetCurrentUser(id int) (models.UpdateUser, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUserByUsernameOrEmail(username string) (string, string, error) {
	var sqlStatement string
	if isEmail(username) {
		sqlStatement = `SELECT username, password_hash FROM users WHERE email=$1`
	} else {
		sqlStatement = `SELECT username, password_hash FROM users WHERE username=$1`
	}
	var storedUsername, storedPasswordHash string
	err := r.db.QueryRow(sqlStatement, username).Scan(&storedUsername, &storedPasswordHash)
	return storedUsername, storedPasswordHash, err
}

func (r *userRepository) GetUserRole(username string) (string, error) {
	var role string
	err := r.db.QueryRow(`SELECT role FROM users WHERE username=$1`, username).Scan(&role)
	return role, err
}

func (r *userRepository) GetUserID(username string) (string, error) {
	var userID string
	err := r.db.QueryRow(`SELECT id FROM users WHERE username=$1`, username).Scan(&userID)
	return userID, err
}

func (r *userRepository) RegisterUser(user models.User, hashedPassword, status string) error {
	sqlStatement := `INSERT INTO users (username, email, password_hash, name, status) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(sqlStatement, user.Username, user.Email, hashedPassword, user.Name, status)
	return err
}

func (r *userRepository) UpdateUserPassword(hashedPassword, id string) error {
	sqlStatement := `UPDATE users SET password_hash=$1 WHERE id=$2`
	_, err := r.db.Exec(sqlStatement, hashedPassword, id)
	return err
}

func (r *userRepository) GetUserEmail(email string) (string, error) {
	var storedEmail string
	err := r.db.QueryRow(`SELECT email FROM users WHERE email=$1`, email).Scan(&storedEmail)
	return storedEmail, err
}

func (r *userRepository) GetUserByID(id int) (models.GetUser, error) {
	var user models.GetUser
	sqlStatement := `SELECT id, username, email, name, avatar_url, lang, role, status, created_at, updated_at FROM users WHERE id=$1`
	err := r.db.QueryRow(sqlStatement, id).Scan(
		&user.Id, &user.Username, &user.Email, &user.Name,
		&user.AvatarURL, &user.Lang, &user.Role, &user.Status,
		&user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

func (r *userRepository) UpdateUser(id int, user models.UpdateUser) error {
	sqlStatement := `
		UPDATE users
		SET username=$1, email=$2, name=$3, avatar_url=$4,
			lang=$5, role=$6, status=$7, updated_at=$8
		WHERE id=$9
	`
	_, err := r.db.Exec(sqlStatement,
		user.Username, user.Email, user.Name, user.AvatarURL,
		user.Lang, user.Role, user.Status, user.UpdatedAt, id,
	)
	return err
}

func (r *userRepository) CheckUsernameExists(username string, id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE username=$1 AND id<>$2)`, username, id).Scan(&exists)
	return exists, err
}

func (r *userRepository) CheckEmailExists(email string, id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email=$1 AND id<>$2)`, email, id).Scan(&exists)
	return exists, err
}

func (r *userRepository) GetCurrentUser(id int) (models.UpdateUser, error) {
	var currentUser models.UpdateUser
	query := `SELECT username, email, avatar_url FROM users WHERE id=$1`
	err := r.db.QueryRow(query, id).Scan(&currentUser.Username, &currentUser.Email, &currentUser.AvatarURL)
	return currentUser, err
}

func isEmail(input string) bool {
	for _, char := range input {
		if char == '@' {
			return true
		}
	}
	return false
}
