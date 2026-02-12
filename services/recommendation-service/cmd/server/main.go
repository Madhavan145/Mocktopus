package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"moktopus/recommendation-service/internal/config"
	httpserver "moktopus/recommendation-service/internal/http"
	"moktopus/recommendation-service/internal/llm"
)

func main() {
	cfg := config.Load()

	// Initialize LLM registry (providers are configured at runtime via API)
	httpserver.LLMRegistry = llm.NewRegistry()

	handler := httpserver.NewHandler()
	server := httpserver.New(cfg.Port, handler)
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("recommendation-service graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("recommendation-service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
