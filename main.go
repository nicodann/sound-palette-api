package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 chars")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		log.Fatal("ALLOWED_ORIGIN must be set")
	}

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" && os.Getenv("MOCK_AI") != "true" {
		log.Fatal("ANTHROPIC_API_KEY must be set")
	}

	anthropicClient := anthropic.NewClient(option.WithAPIKey(anthropicKey))

	secureCookies := os.Getenv("COOKIE_SECURE") == "true"

	// PGX

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// END PGX

	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}

	aiLimiter := newAIRateLimiter()

	srv := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you!")
	})

	http.HandleFunc("/ai-query", corsMiddleware(aiLimiter.middleware(aiQueryHandler(jwtSecret, anthropicClient))))

	http.HandleFunc("/auth/register", corsMiddleware(registerHandler(pool, jwtSecret, secureCookies)))

	http.HandleFunc("/auth/login", corsMiddleware(loginHandler(pool, jwtSecret, secureCookies)))

	http.HandleFunc("/auth/me", corsMiddleware(meHandler(pool, jwtSecret)))

	http.HandleFunc("/auth/logout", corsMiddleware(logoutHandler(secureCookies)))

	http.HandleFunc("/palettes", corsMiddleware(palettesHandler(pool, jwtSecret)))

	http.HandleFunc("/palettes/", corsMiddleware(deletePaletteHandler(pool, jwtSecret)))

	fmt.Println("Server starting on :" + port + "...")

	log.Fatal(srv.ListenAndServe())
}
