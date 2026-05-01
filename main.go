package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you!")
	})

	http.HandleFunc("/ai-query", func(w http.ResponseWriter, r *http.Request) {
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

		// AI

		client := anthropic.NewClient() // sdk picks up ANTHROPIC_API_KEY variable automatically from env

		prompt := "Convert the following phrase into 5 to 8 words that capture it's rhythm, tempo and energy. Only the adjectives, your response should literally be 5, space-seperated words on one line:"

		message, err := client.Messages.New(context.TODO(),	anthropic.MessageNewParams{
				Model: anthropic.ModelClaudeHaiku4_5,
				MaxTokens: 100,
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(
						anthropic.NewTextBlock(prompt + " " + body.Input),
					),
				},
		})

		if err != nil {
			http.Error(w, "AI request failed", http.StatusInternalServerError)
		}
		
		fmt.Fprintf(w, message.Content[0].Text)
	})

	fmt.Println("Server starting on :8080...")

	http.ListenAndServe(":8080", nil)
}