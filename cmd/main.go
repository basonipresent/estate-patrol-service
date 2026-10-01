package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SawitProRecruitment/UserService/generated"
	"github.com/SawitProRecruitment/UserService/handler"
	"github.com/SawitProRecruitment/UserService/repository"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

const (
	dbPingAttempts  = 10
	shutdownTimeout = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	addr := ":" + envOr("PORT", "1323")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := openDB(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = handler.ErrorHandler
	e.Use(
		middleware.Recover(),
		middleware.RequestID(),
		middleware.Logger(),
		middleware.BodyLimit("16K"),
	)
	server := handler.NewServer(handler.NewServerOptions{
		Repository: repository.NewRepository(repository.NewRepositoryOptions{Db: db}),
	})
	generated.RegisterHandlers(e, server)

	srv := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	log.Print("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// openDB retries the first ping because Postgres may still be starting.
func openDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	backoff := 500 * time.Millisecond
	for attempt := 1; ; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return db, nil
		}
		if attempt == dbPingAttempts {
			db.Close()
			return nil, fmt.Errorf("database unreachable after %d attempts: %w", attempt, err)
		}
		log.Printf("database not ready (attempt %d/%d): %v", attempt, dbPingAttempts, err)

		select {
		case <-ctx.Done():
			db.Close()
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 5*time.Second)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
