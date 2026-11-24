package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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

func registerHandler(db *pgxpool.Pool, jwtSecret string ) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			Email string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if body.Email == "" || body.Password == "" {
			http.Error(w, "Missing input", http.StatusBadRequest)
			return
		}

		if len(body.Password) < 8 {
			http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		var email string
		err := db.QueryRow(context.Background(), "SELECT email FROM users WHERE email=$1", body.Email).Scan(&email)

		if err == nil {
			http.Error(w, "email already exists", http.StatusConflict)
			return
		}
		
		hashedPassowrd, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)

		if err != nil {
			http.Error(w, "Password failed to hash", http.StatusInternalServerError)
			return
		}

		var id string

		err = db.QueryRow(context.Background(), "INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id", body.Email, hashedPassowrd).Scan(&id)

		if err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		claims := Claims{
			UserID: id,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},

		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		ss, err := token.SignedString([]byte(jwtSecret))

		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}


		w.Header().Set("Content-Type", "application/json")

		http.SetCookie(w, &http.Cookie{
			Name: "token",
			Value: ss,
			HttpOnly: true,
			Path: "/",
		})

		json.NewEncoder(w).Encode(map[string]string{"id": id, "email": body.Email})
		
	}

} 

func loginHandler(db *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
	
		var body struct {
			Email string `json:"email"`
			Password string `json:"password"`
		}
	
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
	
		if body.Email == "" || body.Password == "" {
			http.Error(w, "Missing input", http.StatusBadRequest)
			return
		}
	
		var id string
		var passwordDb string
		err := db.QueryRow(context.Background(), "select id, password from users where email=$1", body.Email).Scan(&id, &passwordDb)
		if err != nil {
			http.Error(w, "Email not found", http.StatusUnauthorized)
			return
		}
	
		err = bcrypt.CompareHashAndPassword( []byte(passwordDb), []byte(body.Password))
	
		if err != nil {
			http.Error(w, "Password is incorrect", http.StatusUnauthorized)
			return
		}
	
		claims := Claims{
			UserID: id,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
	
		}
	
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		ss, err := token.SignedString([]byte(jwtSecret))
	
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		
		http.SetCookie(w, &http.Cookie{
			Name: "token",
			Value: ss,
			HttpOnly: true,
			Path: "/",
		})

		json.NewEncoder(w).Encode(map[string]string{"id": id, "email": body.Email})		
	}
}

func meHandler(db *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			userId, err := extractUserID(r, jwtSecret)

			if err != nil {
				http.Error(w, "error parsing user ID", http.StatusUnauthorized)
				return
			}

			var email string
			err = db.QueryRow(context.Background(), "SELECT email FROM users WHERE id = $1", userId).Scan(&email)

			if err != nil {
				http.Error(w, "failed to Authenicate", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"id": userId, "email": email})

		} else {
			http.Error(w, "GET required", http.StatusMethodNotAllowed)
		}
	}
}

func logoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
		
		http.SetCookie(w, &http.Cookie{
			Name: "token",
			Value: "",
			HttpOnly: true,
			Path: "/",
			Expires: time.Unix(0, 0),
		})

		w.WriteHeader(http.StatusNoContent)

		} else {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
		}
	}
}