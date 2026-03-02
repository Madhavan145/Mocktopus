package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"moktopus/metrics-service/internal/config"
	httpserver "moktopus/metrics-service/internal/http"
	"moktopus/metrics-service/internal/store"
)

func main() {
	cfg := config.Load()

	var st store.Store

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		pool, err := pgxpool.New(context.Background(), dbURL)
		if err != nil {
			log.Printf("WARN: failed to connect to PostgreSQL (%v), running without persistence", err)
		} else {
			if err := pool.Ping(context.Background()); err != nil {
				log.Printf("WARN: PostgreSQL ping failed (%v), running without persistence", err)
				pool.Close()
			} else {
				log.Println("Connected to PostgreSQL")
				st = store.NewPostgresStore(pool)
				defer pool.Close()
			}
		}
	} else {
		log.Println("No DATABASE_URL set, running without persistence")
	}

	handler := httpserver.NewHandler(st)
	server := httpserver.New(cfg.Port, handler)
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("metrics-service graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("metrics-service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
