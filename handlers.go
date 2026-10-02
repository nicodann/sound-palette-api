package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)


func registerHandler(conn *pgx.Conn, jwtSecret string) http.HandlerFunc {
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

		var email string
		err := conn.QueryRow(context.Background(), "select email from users where email=$1", body.Email).Scan(&email)
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
		err = conn.QueryRow(context.Background(), "INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id", body.Email, hashedPassowrd).Scan(&id)

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
		// json.NewEncoder(w).Encode(map[string]string{"token":ss})
	}

} 

func loginHandler(conn *pgx.Conn, jwtSecret string) http.HandlerFunc {
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
		err := conn.QueryRow(context.Background(), "select id, password from users where email=$1", body.Email).Scan(&id, &passwordDb)
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
		// json.NewEncoder(w).Encode(map[string]string{"token":ss})
		http.SetCookie(w, &http.Cookie{
			Name: "token",
			Value: ss,
			HttpOnly: true,
			Path: "/",
		})
	}

}

func meHandler(conn *pgx.Conn, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			userId, err := extractUserID(r, jwtSecret)

			if err != nil {
				http.Error(w, "error parsing user ID", http.StatusUnauthorized)
				return
			}

			var email string
			err = conn.QueryRow(context.Background(), "SELECT email FROM users WHERE id = $1", userId).Scan(&email)

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

func aiQueryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
	
		var body struct {
			Input string `json:"input"`
		}
	
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
	
		if body.Input == "" {
			http.Error(w, "Missing input", http.StatusBadRequest)
			return
		}
	
		// MOCK
	
		if os.Getenv("MOCK_AI") == "true" {
			mock := `[{"adjective":"dreamy","colour":"#A8D8EA"},{"adjective":"soft","colour":"#AA96DA"},{"adjective":"calm","colour":"#FCBAD3"},{"adjective":"gentle","colour":"#FFFFD2"},{"adjective":"slow","colour":"#B5EAD7"}]`
			var parsed []interface{}
			json.Unmarshal([]byte(mock), &parsed)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(parsed)
			fmt.Println("Sending mock data")
			return
		}
	
		// AI
	
		client := anthropic.NewClient() // sdk picks up ANTHROPIC_API_KEY variable automatically from env
	
		prompt := `Convert the following phrase into 5 words that capture it's rhythm, tempo and energy. Also one colour, in hexcode format, that corresponds to each word. Return ONLY a valid JSON Array, no other text, no markdown. format: [{"adjective": "word", "colour": "#hexcode"}]. For the colours avoid bold primaries unless explicity stated by the user.`
	
		message, err := client.Messages.New(context.TODO(),	anthropic.MessageNewParams{
				Model: anthropic.ModelClaudeHaiku4_5,
				MaxTokens: 300,
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(
						anthropic.NewTextBlock(prompt + " " + body.Input),
					),
				},
		})
	
		if err != nil {
			fmt.Println("AI request failed:", err)
			http.Error(w, "AI request failed", http.StatusInternalServerError)
			return
		}
		
		fmt.Println("Claude raw response:", message.Content[0].Text)
		text := message.Content[0].Text
		text = strings.TrimSpace(text)
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	
		var parsed []interface{}
		if err := json.Unmarshal([]byte(text), &parsed); err != nil {
			fmt.Println("JSON parse failed:", err)
			http.Error(w, "failed to parseAI response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(parsed)
		
	}
}

func palettesHandler(conn *pgx.Conn, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Name	string	`json:"name"`
				Palette []interface{} `json:"palette"`
			}

			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}

			userId, err := extractUserID(r, jwtSecret)

			if err != nil {
				http.Error(w, "error parsing user ID", http.StatusUnauthorized)
				return
			}

			_, err = conn.Exec(context.Background(), "INSERT INTO palettes (user_id, name, palette) VALUES ($1, $2, $3)", userId, body.Name, body.Palette)

			if err != nil {
				http.Error(w, "failed to save palette", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		} else if r.Method == http.MethodGet {
			userId, err := extractUserID(r, jwtSecret)

			if err != nil {
				http.Error(w, "error parsing user ID", http.StatusUnauthorized)
				return
			}

			rows, err := conn.Query(context.Background(), "SELECT id, name, palette, created_at FROM palettes WHERE user_id = $1", userId)

			if err != nil {
				http.Error(w, "failed to fetch palettes", http.StatusInternalServerError)
				return
			}

			defer rows.Close()

			type Palette struct {
				ID string `json:"id"`
				Name string `json:"name"`
				Palette []interface{} `json:"palette"`
				CreatedAt time.Time `json:"created_at"`
			}

			var palettes []Palette

			for rows.Next() {
				var p Palette
				err := rows.Scan(&p.ID, &p.Name, &p.Palette, &p.CreatedAt)
				if err != nil {
					http.Error(w, "failes to scan palette", http.StatusInternalServerError)
					return
				}
				palettes = append(palettes, p)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(palettes)
		} else {
			http.Error(w, "GET or POST required", http.StatusMethodNotAllowed)
			return
		}
	}
}

func deletePaletteHandler(conn *pgx.Conn, jwtSecret string) http.HandlerFunc {

	return func(w http.ResponseWriter, r * http.Request) {
		if r.Method != http.MethodDelete {
		http.Error(w, "DELETE required", http.StatusMethodNotAllowed)
		return
		}

		userId, err := extractUserID(r, jwtSecret)

		if err != nil {
			http.Error(w, "error parsing user ID", http.StatusUnauthorized)
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/palettes/")

		_, err = conn.Exec(context.Background(), "DELETE FROM palettes WHERE user_id = $1 AND id = $2", userId, id)

		if err != nil {
			http.Error(w, "failed to delete palette", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)


	}
}