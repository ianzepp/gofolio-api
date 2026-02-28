package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/handler"
)

func main() {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/v1/health", handler.HandleGetHealth)

	// Info
	mux.HandleFunc("GET /api/v1/info", handler.HandleGetInfo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3333"
	}

	log.Printf("api-go listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
