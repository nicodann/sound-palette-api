package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

func aiQueryHandler(jwtSecret string, anthropicClient anthropic.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		// Don't require login b/c demo. users can try before logging in.

		// _, err := extractUserID(r, jwtSecret)

		// if err != nil {
		// 		http.Error(w, "login required", http.StatusUnauthorized)
		// 		return
		// 	}

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

		if len([]rune(body.Input)) > 500 {
			http.Error(w, "Input must be 500 characters or fewer", http.StatusBadRequest)
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

		prompt := `Convert the following phrase into 5 words that capture it's rhythm, tempo and energy. Also one colour, in hexcode format, that corresponds to each word. Return ONLY a valid JSON Array, no other text, no markdown. format: [{"adjective": "word", "colour": "#hexcode"}]. For the colours avoid bold primaries unless explicity stated by the user.`

		message, err := anthropicClient.Messages.New(r.Context(), anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeHaiku4_5,
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

		var parsed []paletteColour
		if err := json.Unmarshal([]byte(text), &parsed); err != nil {
			fmt.Println("JSON parse failed:", err)
			http.Error(w, "failed to parseAI response", http.StatusInternalServerError)
			return
		}

		if err := validatePalette(parsed); err != nil {
			log.Printf("AI returned an invalid palette: %v; raw: %s", err, text)
			http.Error(w, "Couldn't generate a valid palette, please try again", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(parsed)

	}
}
