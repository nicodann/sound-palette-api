package main

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID string `json:"user_id"`
    jwt.RegisteredClaims
}

func extractUserID(r *http.Request, secret string) (string, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return "", err
	}
	tokenString := cookie.Value

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}
	return claims.UserID, nil
}