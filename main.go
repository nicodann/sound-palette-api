package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	jwtSecret := os.Getenv("JWT_SECRET")
	
	// PGX

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// END PGX

	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "*"
	}

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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you!")
	})

	http.HandleFunc("/ai-query", corsMiddleware(aiQueryHandler()))

	http.HandleFunc("/auth/register", corsMiddleware(registerHandler(pool, jwtSecret)))

	http.HandleFunc("/auth/login", corsMiddleware(loginHandler(pool, jwtSecret)))

	http.HandleFunc("/auth/me", corsMiddleware(meHandler(pool, jwtSecret)))

	http.HandleFunc("/auth/logout", corsMiddleware(logoutHandler()))

	http.HandleFunc("/palettes", corsMiddleware(palettesHandler(pool, jwtSecret)))

	http.HandleFunc("/palettes/", corsMiddleware(deletePaletteHandler(pool, jwtSecret)))

	fmt.Println("Server starting on :8080...")

	log.Fatal(http.ListenAndServe(":8080", nil))
}