package repositories

import (
	"database/sql"
	"time"

	"BE-E-Meeting/app/entities"
)

// UserRepository adalah 'Daftar Menu' apa saja yang bisa dilakukan ke tabel users
type UserRepository interface {
	Create(user entities.User, avatarURL string) error               // Tambah parameter avatarURL
	GetByUsername(username string) (entities.GetUser, string, error) // Return GetUser + Hash
	GetByEmail(email string) (entities.GetUser, string, error)       // Return GetUser + Hash
	GetByID(id int) (entities.GetUser, error)
}

// Implementasi sql DB
type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// 1. Create (Register)
func (r *userRepository) Create(user entities.User, avatarURL string) error {
	// Status kita set default 'active' di sini sesuai logic main.go sebelumnya
	status := "active"

	sqlStatement := `INSERT INTO users (username, email, password_hash, name, status, avatar_url, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`

	_, err := r.db.Exec(sqlStatement, user.Username, user.Email, user.Password, user.Name, status, avatarURL)
	return err
}

// 2. GetByUsername (Login)
func (r *userRepository) GetByUsername(username string) (entities.GetUser, string, error) {
	var user entities.GetUser
	var passwordHash string

	// Kita scan ke struct GetUser agar mendapatkan ID dan Role untuk Token
	// Perhatikan urutan scan harus sama dengan urutan SELECT
	sqlStatement := `SELECT id, username, email, name, role, avatar_url, password_hash FROM users WHERE username=$1`

	err := r.db.QueryRow(sqlStatement, username).Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.Avatar_url,
		&passwordHash, // Hash kita pisah karena tidak ada di struct GetUser
	)

	return user, passwordHash, err
}

// 3. GetByEmail (Login)
func (r *userRepository) GetByEmail(email string) (entities.GetUser, string, error) {
	var user entities.GetUser
	var passwordHash string

	sqlStatement := `SELECT id, username, email, name, role, avatar_url, password_hash FROM users WHERE email=$1`

	err := r.db.QueryRow(sqlStatement, email).Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.Avatar_url,
		&passwordHash,
	)

	return user, passwordHash, err
}

// 4. GetByID (Profile)
func (r *userRepository) GetByID(id int) (entities.GetUser, error) {
	var user entities.GetUser

	// Kita ambil created_at sebagai time.Time dulu agar aman, lalu convert ke string
	// atau biarkan driver sql convert ke string jika kompatibel.
	// Di sini saya asumsikan driver pq bisa scan timestamp ke string langsung.

	sqlStatement := `SELECT id, username, email, name, avatar_url, lang, role, status, created_at, updated_at FROM users WHERE id=$1`

	err := r.db.QueryRow(sqlStatement, id).Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Avatar_url,
		&user.Lang,
		&user.Role,
		&user.Status,
		&user.Created_at,
		&user.Updated_at,
	)

	// Handle format tanggal jika user.Created_at kosong atau formatnya aneh (Optional logic)
	if user.Created_at == "" {
		user.Created_at = time.Now().Format(time.RFC3339)
	}

	return user, err
}
