package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"moktopus/cloud-catalog-service/internal/config"
	httpserver "moktopus/cloud-catalog-service/internal/http"
)

func main() {
	cfg := config.Load()
	handler := httpserver.NewHandler()
	server := httpserver.New(cfg.Port, handler)

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("cloud-catalog-service graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("cloud-catalog-service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
