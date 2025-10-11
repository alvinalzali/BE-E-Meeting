package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	_ "github.com/lib/pq"
)

type CustomValdator struct {
	validator *validator.Validate
}

func (cv *CustomValdator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

type Login struct {
	//login using username or email
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type User struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

var db *sql.DB
var JwtSecret []byte

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	dbHost := os.Getenv("db_host")
	dbPort, _ := strconv.Atoi(os.Getenv("db_port"))
	dbUser := os.Getenv("db_user")
	dbPassword := os.Getenv("db_password")
	dbName := os.Getenv("db_name")

	db = connectDB(dbUser, dbPassword, dbName, dbHost, dbPort)

	e := echo.New()

	e.Validator = &CustomValdator{validator: validator.New()}

	e.POST("/login", login)
	e.POST("/register", registerUser)

	e.Logger.Fatal(e.Start(":8080"))

}

func connectDB(username, password, dbname, host string, port int) *sql.DB {
	// connect to db
	connSt := "host=" + host + " port=" + strconv.Itoa(port) + " user=" + username + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	db, err := sql.Open("postgres", connSt)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to DB " + dbname + " successfully on port" + strconv.Itoa(port))
	return db
}

func login(c echo.Context) error {
	var loginData Login

	if err := c.Bind(&loginData); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid Input"})
	}

	//cek username apakah email atau username
	var sqlStatement string
	if isEmail(loginData.Username) {
		sqlStatement = `SELECT username, password_hash FROM users WHERE email=$1`
	} else {
		sqlStatement = `SELECT username, password_hash FROM users WHERE username=$1`
	}

	var storedUsername, storedPasswordHash string
	err := db.QueryRow(sqlStatement, loginData.Username).Scan(&storedUsername, &storedPasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Credentials"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database Error"})
	}

	// hash the provided password and compare with stored hash
	hashedInputPassword := hashPassword(loginData.Password)
	if hashedInputPassword != storedPasswordHash {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Credentials"})
	}

	// generate JWT token
	token, err := generateToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Token Generation Failed"})
	}

	return c.JSON(http.StatusOK, echo.Map{"token": token})
}

func isEmail(input string) bool {
	// simple check for email format
	for _, char := range input {
		if char == '@' {
			return true
		}
	}
	return false
}

func generateToken() (string, error) {
	JwtSecret = []byte(os.Getenv("secret_key"))
	jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "alvin",
		"role":     "admin",
		"exp":      time.Now().Add(60 * time.Minute).Unix(),
	})

	tokenString, err := jwt.SignedString(JwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func registerUser(c echo.Context) error {
	var newUser User
	if err := c.Bind(&newUser); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid Input"})
	}

	if err := c.Validate(&newUser); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Cek " + err.Error()})
	}

	//insert variable default, Enum status, role, lang
	status := "active"

	// hash password
	hashedPassword := hashPassword(newUser.Password)

	// insert to db
	sqlStatement := `INSERT INTO users (username, email, password_hash, name, status) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(sqlStatement, newUser.Username, newUser.Email, hashedPassword, newUser.Name, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "User registered successfully!"})
}

func hashPassword(password string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
