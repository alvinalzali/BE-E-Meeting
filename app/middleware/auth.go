package middleware

import (
	"net/http"
	"os"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// RoleAuthMiddleware mengecek apakah user punya akses (Role)
func RoleAuthMiddleware(requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			// Ambil Secret Key dari ENV
			jwtSecret := []byte(os.Getenv("secret_key"))

			// Ambil Authorization Header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized"})
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Authorization header"})
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Parsing JWT
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid token"})
			}

			// Simpan token ke context agar bisa dipakai di Handler nanti
			c.Set("user", token)

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid token claims"})
			}

			// Ambil role dari claims
			var userRoles []string
			if rolesClaimSlice, ok := claims["role"].([]interface{}); ok {
				for _, roleInterface := range rolesClaimSlice {
					if role, ok := roleInterface.(string); ok {
						userRoles = append(userRoles, role)
					}
				}
			} else if roleClaimString, ok := claims["role"].(string); ok {
				userRoles = append(userRoles, roleClaimString)
			} else {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Role claim missing"})
			}

			// Cek apakah role user cocok dengan requiredRoles
			for _, requiredRole := range requiredRoles {
				for _, userRole := range userRoles {
					if requiredRole == userRole {
						return next(c)
					}
				}
			}

			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized"})
		}
	}
}
