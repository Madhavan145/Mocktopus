package main

import (
	"errors"
	"log"
	"net/http"

	"moktopus/chaos-service/internal/config"
	httpserver "moktopus/chaos-service/internal/http"
	"moktopus/chaos-service/internal/store"
)

func main() {
	cfg := config.Load()
	repo := store.NewInMemoryStore()
	handler := httpserver.NewHandler(repo)
	server := httpserver.New(cfg.Port, handler)

	log.Printf("chaos-service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
