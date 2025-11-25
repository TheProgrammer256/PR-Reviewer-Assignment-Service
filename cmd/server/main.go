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

	openapi "github.com/TheProgrammer256/PR-Reviewer-Assignment-Service/go"

	"github.com/avito/pr-reviewer-assignment-service/internal/config"
	"github.com/avito/pr-reviewer-assignment-service/internal/db"
	"github.com/avito/pr-reviewer-assignment-service/internal/server"
	"github.com/avito/pr-reviewer-assignment-service/internal/service"
	"github.com/avito/pr-reviewer-assignment-service/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to init database pool: %v", err)
	}
	defer pool.Close()

	if err := db.EnsureSchema(ctx, pool); err != nil {
		log.Fatalf("failed to ensure schema: %v", err)
	}

	repo := storage.NewRepository(pool)
	apiService := service.New(repo)

	pullRequestsController := openapi.NewPullRequestsAPIController(
		apiService,
		openapi.WithPullRequestsAPIErrorHandler(server.ErrorHandler),
	)
	teamsController := openapi.NewTeamsAPIController(
		apiService,
		openapi.WithTeamsAPIErrorHandler(server.ErrorHandler),
	)
	usersController := openapi.NewUsersAPIController(
		apiService,
		openapi.WithUsersAPIErrorHandler(server.ErrorHandler),
	)

	router := openapi.NewRouter(pullRequestsController, teamsController, usersController)

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown error: %v", err)
		}
	}()

	log.Printf("PR Reviewer Assignment Service listening on %s", cfg.Addr())
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
}
