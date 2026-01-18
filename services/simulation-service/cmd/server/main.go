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

	"moktopus/simulation-service/internal/config"
	"moktopus/simulation-service/internal/engine"
	"moktopus/simulation-service/internal/eventbus"
	grpcserver "moktopus/simulation-service/internal/grpc"
	httpserver "moktopus/simulation-service/internal/http"
	"moktopus/simulation-service/internal/store"
)

func main() {
	cfg := config.Load()
	simEngine := engine.New()

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

	pub := eventbus.NewPublisher()
	defer pub.Close()

	handler := httpserver.NewHandler(simEngine, st, pub)
	server := httpserver.New(cfg.Port, handler)

	// Start gRPC server for inter-service communication.
	grpcSrv := grpcserver.NewServer(cfg.GRPCPort, simEngine, st)
	go func() {
		if err := grpcSrv.Serve(); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		grpcSrv.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("simulation-service graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("simulation-service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
