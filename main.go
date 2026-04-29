package main

import (
	"fmt"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you!")
	})

	http.HandleFunc("/ai-query", func(w http.ResponseWriter, r *http.Request) {
		input := r.URL.Query().Get("input")
		if input == "" {
			http.Error(w, "Missing 'input' query paramter", http.StatusBadRequest)
			return
		}
		altered := strings.ToUpper(input)
		fmt.Fprintf(w, altered)
	})

	fmt.Println("Server starting on :8080...")

	http.ListenAndServe(":8080", nil)
}