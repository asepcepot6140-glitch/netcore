package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

func signingKey() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes")
	}
	return []byte(secret), nil
}

func ValidateJWTSecret() error {
	_, err := signingKey()
	return err
}

func GenerateAccessToken(userID uint) (string, error) {
	secret, err := signingKey()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID: userID,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(secret)
}

func GenerateRefreshToken(userID uint) (string, error) {
	secret, err := signingKey()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID: userID,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(secret)
}

func ParseToken(tokenString string) (*Claims, error) {
	secret, err := signingKey()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("invalid signing method")
			}

			return secret, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
