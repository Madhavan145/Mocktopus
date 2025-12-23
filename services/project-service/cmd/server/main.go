package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"moktopus/project-service/internal/config"
	httpserver "moktopus/project-service/internal/http"
	"moktopus/project-service/internal/store"
)

func main() {
	cfg := config.Load()

	var repo store.Store

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		pool, err := pgxpool.New(context.Background(), dbURL)
		if err != nil {
			log.Printf("WARN: failed to connect to PostgreSQL (%v), falling back to in-memory", err)
			repo = store.NewInMemoryStore()
		} else {
			if err := pool.Ping(context.Background()); err != nil {
				log.Printf("WARN: PostgreSQL ping failed (%v), falling back to in-memory", err)
				pool.Close()
				repo = store.NewInMemoryStore()
			} else {
				log.Println("Connected to PostgreSQL")
				repo = store.NewPostgresStore(pool)
				defer pool.Close()
			}
		}
	} else {
		log.Println("No DATABASE_URL set, using in-memory store")
		repo = store.NewInMemoryStore()
	}

	handler := httpserver.NewHandler(repo)
	server := httpserver.New(cfg.Port, handler)

	log.Printf("project-service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
