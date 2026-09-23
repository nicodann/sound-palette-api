package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "*"
	}

	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
      w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}

	aiQueryHandler := func(w http.ResponseWriter, r *http.Request) {
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
		// RETURN JUST NewTextBlock
		// fmt.Fprintf(w, message.Content[0].Text)
		// json.NewEncoder(w).Encode(map[string]string{"result": message.Content[0].Text})
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you!")
	})

	http.HandleFunc("/ai-query", corsMiddleware(aiQueryHandler))

	fmt.Println("Server starting on :8080...")

	http.ListenAndServe(":8080", nil)
}