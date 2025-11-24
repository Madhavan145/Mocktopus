package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"moktopus/auth-service/internal/auth"
	"moktopus/auth-service/internal/config"
	"moktopus/auth-service/internal/db"
	httpserver "moktopus/auth-service/internal/http"
)

func main() {
	cfg := config.Load()

	database, err := db.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	repo := db.NewRepository(database)

	// Ensure org invite tables exist
	if err := repo.EnsureOrgTables(context.Background()); err != nil {
		log.Printf("warning: could not ensure org tables: %v", err)
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	handler := httpserver.NewHandler(repo, tokens, cfg)

	server := httpserver.New(cfg.Port, handler)
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("auth-service graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("auth-service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
