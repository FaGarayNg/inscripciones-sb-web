package config

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Credentials representa las credenciales del administrador
type Credentials struct {
	Username string
	Password string
}

// Tipos de token: un token de alumno nunca debe servir en /api/admin y un
// token de administrador nunca debe servir para responder invitaciones.
const (
	TokenTypeAdmin   = "admin"
	TokenTypeStudent = "student"
)

// Claims representa los claims del JWT
type Claims struct {
	Username  string `json:"username,omitempty"`
	Type      string `json:"type"`
	StudentID string `json:"studentId,omitempty"`
	jwt.RegisteredClaims
}

// GetAdminCredentials obtiene las credenciales del administrador desde variables de entorno
func GetAdminCredentials() (*Credentials, error) {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")

	if username == "" || password == "" {
		return nil, errors.New("credenciales de administrador no configuradas")
	}

	return &Credentials{
		Username: username,
		Password: password,
	}, nil
}

// VerifyPassword verifica si la contraseña coincide con el hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken genera un JWT token
func GenerateToken(username string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET no configurado")
	}

	claims := &Claims{
		Username: username,
		Type:     TokenTypeAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateStudentToken genera un token para la identidad del ingresante.
// No tiene username: se identifica por el ID del alumno en los claims.
func GenerateStudentToken(studentID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET no configurado")
	}

	claims := &Claims{
		Type:      TokenTypeStudent,
		StudentID: studentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken valida un JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET no configurado")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Los tokens se emiten con HS256: no aceptamos otro método de firma.
		if token.Method.Alg() != "HS256" {
			return nil, errors.New("algoritmo de firma no permitido")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("token inválido")
	}

	return claims, nil
}
