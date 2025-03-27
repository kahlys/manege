package main

import (
	"log/slog"
	"os"

	"github.com/kahlys/manege/cmd/server/service"
)

func main() {
	server := service.NewServer()

	if err := server.Run(); err != nil {
		slog.Warn("ServerFailed", "error", err)
		os.Exit(1)
	}
}
