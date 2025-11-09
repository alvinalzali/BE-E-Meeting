package entities

import "github.com/golang-jwt/jwt/v5"

type Login struct {
	//login using username or email
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type User struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	// password harus ada angka, huruf besar, huruf kecil, dan simbol
	Password string `json:"password" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

type GetUser struct {
	Created_at string `json:"createdAt"`
	Email      string `json:"email"`
	Id         string `json:"id"`
	Avatar_url string `json:"imageURL"`
	Lang       string `json:"language"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Updated_at string `json:"updatedAt"`
	Username   string `json:"username"`
	Name       string `json:"name"`
}

type UpdateUser struct {
	Email      string `json:"email" validate:"omitempty,email"`
	Avatar_url string `json:"imageURL" validate:"omitempty,url"`
	Lang       string `json:"language" validate:"omitempty,oneof=en id"`
	Role       string `json:"role" validate:"omitempty,oneof=admin user"`
	Status     string `json:"status" validate:"omitempty,oneof=active inactive"`
	Username   string `json:"username" validate:"omitempty"`
	Name       string `json:"name" validate:"omitempty"`
	Updated_at string `json:"updatedAt"`
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type ResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type PasswordConfirmReset struct {
	ConfirmPassword string `json:"confirm_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required"`
}
