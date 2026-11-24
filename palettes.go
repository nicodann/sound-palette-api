package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Palette struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Prompt string `json:"prompt"`
	Palette []interface{} `json:"palette"`
	CreatedAt time.Time `json:"created_at"`
}

func palettesHandler(db *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Name	string	`json:"name"`
				Palette []interface{} `json:"palette"`
				Prompt string `json:"prompt"`
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

			var palette Palette

			err = db.QueryRow(context.Background(), "INSERT INTO palettes (user_id, name, prompt, palette) VALUES ($1, $2, $3, $4) RETURNING id, name, prompt, palette, created_at", userId, body.Name, body.Prompt, body.Palette).Scan(&palette.ID, &palette.Name, &palette.Prompt, &palette.Palette, &palette.CreatedAt)

			if err != nil {
				http.Error(w, "failed to save palette", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(palette)
		} else if r.Method == http.MethodGet {
			userId, err := extractUserID(r, jwtSecret)

			if err != nil {
				http.Error(w, "error parsing user ID", http.StatusUnauthorized)
				return
			}

			rows, err := db.Query(context.Background(), "SELECT id, name, prompt, palette, created_at FROM palettes WHERE user_id = $1 ORDER BY created_at DESC", userId)

			if err != nil {
				http.Error(w, "failed to fetch palettes", http.StatusInternalServerError)
				return
			}

			defer rows.Close()

			palettes := []Palette{}

			for rows.Next() {
				var p Palette
				err := rows.Scan(&p.ID, &p.Name, &p.Prompt, &p.Palette, &p.CreatedAt)
				if err != nil {
					http.Error(w, "failes to scan palette", http.StatusInternalServerError)
					return
				}
				palettes = append(palettes, p)
			}

			if err := rows.Err(); err != nil {
				http.Error(w, "failed to read palettes", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(palettes)
		} else {
			http.Error(w, "GET or POST required", http.StatusMethodNotAllowed)
			return
		}
	}
}

func deletePaletteHandler(db *pgxpool.Pool, jwtSecret string) http.HandlerFunc {

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

		result, err := db.Exec(context.Background(), "DELETE FROM palettes WHERE user_id = $1 AND id = $2", userId, id)

		if err != nil {
			log.Printf("delete palette %s for use %s: %v", id, userId, err)
			http.Error(w, "failed to delete palette", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected() == 0 {
			http.Error(w, "palette not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)


	}
}