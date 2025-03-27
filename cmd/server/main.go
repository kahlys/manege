package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	_ "github.com/lib/pq"

	"github.com/kahlys/manege/cmd/server/service"
)

func main() {
	connStr := "postgres://postgres:postgres@database:5432/postgres?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		slog.Error("DatabaseConnectionFailed", "error", err)
		os.Exit(1)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		slog.Error("MigrationFailed", "error", err)
		os.Exit(1)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file:///migrations",
		"postgres",
		driver,
	)
	if err != nil {
		slog.Error("MigrationFailed", "error", err)
		os.Exit(1)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		slog.Error("MigrationFailed", "error", err)
		os.Exit(1)
	}

	// Create a listener for PostgreSQL notifications
	listener := pq.NewListener(connStr, 10, 30, func(event pq.ListenerEventType, err error) {
		if err != nil {
			slog.Error("ListenerError", "error", err)
			os.Exit(1)
		}
	})
	defer listener.Close()

	err = listener.Listen("config_changes")
	if err != nil {
		slog.Error("ListenerError", "error", err)
		os.Exit(1)
	}

	go func() {
		for {
			notification := <-listener.Notify
			if notification != nil {
				fmt.Printf("Received notification: %s\n", notification.Extra)
			}
		}
	}()

	time.Sleep(3 * time.Second)

	// Insert a record into the config table
	insertQuery := `INSERT INTO config (name, is_active) VALUES ($1, $2)`
	_, err = db.Exec(insertQuery, "Example Config", true)
	if err != nil {
		log.Fatalf("Failed to insert record: %v", err)
	}
	fmt.Println("Record inserted successfully.")

	// Select all records from the config table
	selectQuery := `SELECT id, name, is_active FROM config`
	rows, err := db.Query(selectQuery)
	if err != nil {
		log.Fatalf("Failed to select records: %v", err)
	}
	defer rows.Close()

	// Print all records
	fmt.Println("Config Table Records:")
	for rows.Next() {
		var id int
		var name string
		var isActive bool
		if err := rows.Scan(&id, &name, &isActive); err != nil {
			log.Fatalf("Failed to scan record: %v", err)
		}
		fmt.Printf("ID: %d, Name: %s, IsActive: %t\n", id, name, isActive)
	}
	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating over rows: %v", err)
	}

	server := service.NewServer()
	if err := server.Run(); err != nil {
		slog.Warn("ServerFailed", "error", err)
		os.Exit(1)
	}
}
