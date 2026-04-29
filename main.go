package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

		altered := strings.ToUpper(body.Input)
		fmt.Fprintf(w, altered)
	})

	fmt.Println("Server starting on :8080...")

	http.ListenAndServe(":8080", nil)
}