package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
)

type paletteColour struct {
	Adjective string `json:"adjective"`
	Colour    string `json:"colour"`
}

type Palette struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Prompt    string          `json:"prompt"`
	Palette   []paletteColour `json:"palette"`
	CreatedAt time.Time       `json:"created_at"`
}

var hexColour = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validatePalette(p []paletteColour) error {
	if len(p) != 5 {
		return errors.New("palette must contain 5 colours")
	}
	for _, c := range p {
		if c.Adjective == "" || utf8.RuneCountInString(c.Adjective) > 30 {
			return errors.New("each adjective must be 1–30 characters")
		}
		if !hexColour.MatchString(c.Colour) {
			return errors.New("each colour must be a hex code like #A8D8EA")
		}
	}
	return nil
}

func palettesHandler(db *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Name    string          `json:"name"`
				Palette []paletteColour `json:"palette"`
				Prompt  string          `json:"prompt"`
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

			if utf8.RuneCountInString(body.Name) > 50 {
				http.Error(w, "Name must be 50 chars or fewer", http.StatusBadRequest)
				return
			}
			if utf8.RuneCountInString(body.Prompt) > 500 {
				http.Error(w, "Prompt must be 500 chars or fewer", http.StatusBadRequest)
				return
			}

			if err := validatePalette(body.Palette); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
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

	return func(w http.ResponseWriter, r *http.Request) {
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
