package main

import (
	"context"
	"log/slog"
	"os"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kahlys/manege/cmd/server/service"
)

func main() {
	db, err := pgxpool.New(
		context.Background(),
		"postgres://postgres:postgres@database:5432/postgres?sslmode=disable",
	)
	if err != nil {
		slog.Error("DatabaseConnectionFailed", "error", err)
		os.Exit(1)
	}

	server := service.NewServer(db)
	if err := server.Run(); err != nil {
		slog.Warn("ServerFailed", "error", err)
		os.Exit(1)
	}
}
