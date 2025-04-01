package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	pb "github.com/kahlys/manege/internal/proto"
	"google.golang.org/grpc"

	devicev1 "github.com/kahlys/manege/cmd/server/service/device/v1"
	devicev2 "github.com/kahlys/manege/cmd/server/service/device/v2"
)

type Server struct {
	dmAddr    string
	adminAddr string

	db *pgxpool.Pool
}

func NewServer(db *pgxpool.Pool) *Server {
	return &Server{
		dmAddr:    ":50051",
		adminAddr: ":8080",

		db: db,
	}
}

func (a *Server) Run() error {
	version, err := a.migrate()
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	slog.Info("ServerStarts", "dmAddr", a.dmAddr, "adminAddr", a.adminAddr, "dbschema", version)

	if err := a.runDeviceManagement(); err != nil {
		return err
	}

	return nil
}

func (a *Server) migrate() (uint, error) {
	driver, err := postgres.WithInstance(stdlib.OpenDBFromPool(a.db), &postgres.Config{})
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
		return 0, err
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return 0, err
	}

	version, _, err := m.Version()
	if err != nil {
		return 0, err
	}

	return version, nil
}

func (a *Server) runDeviceManagement() error {
	lis, err := net.Listen("tcp", a.dmAddr)
	if err != nil {
		return err
	}
	grpcSrv := grpc.NewServer()

	deviceSvc := devicev1.NewDeviceService()
	pb.RegisterNotificationServiceServer(grpcSrv, deviceSvc)
	pb.RegisterNotificationServiceV2Server(grpcSrv, devicev2.NewDeviceService())

	go func() {
		conn, err := a.db.Acquire(context.Background())
		if err != nil {
			slog.Error("DatabaseListenFailed", "error", err)
			os.Exit(1)
		}
		defer conn.Release()

		_, err = conn.Exec(context.Background(), "LISTEN config_changes")
		if err != nil {
			slog.Error("DatabaseListenFailed", "error", err)
			os.Exit(1)
		}

		for {
			_, err := conn.Conn().WaitForNotification(context.Background())
			if err != nil {
				slog.Error("DatabaseListenFailed", "error", err)
				os.Exit(1)
			}
			deviceSvc.NotifyClients()
		}
	}()

	go a.runRestAPI()

	return grpcSrv.Serve(lis)
}

type Client struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

func (a *Server) runRestAPI() error {
	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows, err := a.db.Query(context.Background(), "SELECT id, name, is_active FROM config")
			if err != nil {
				slog.Error("DatabaseQueryError", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			configs := []Client{}
			for rows.Next() {
				var id int
				var name string
				var isActive bool
				if err := rows.Scan(&id, &name, &isActive); err != nil {
					slog.Error("RowScanError", "error", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				configs = append(configs, Client{ID: id, Name: name, IsActive: isActive})
			}

			if err := rows.Err(); err != nil {
				slog.Error("RowIterationError", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(configs); err != nil {
				slog.Error("JSONEncodeError", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

		case http.MethodPost:
			var payload Client
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				slog.Error("JSONDecodeError", "error", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			_, err := a.db.Exec(
				context.Background(),
				"UPDATE config SET is_active = $1, name = $2 WHERE id = $3",
				payload.IsActive,
				payload.Name,
				payload.ID,
			)
			if err != nil {
				slog.Error("DatabaseUpdateError", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			slog.Info("ConfigUpdated", "name", payload.Name, "is_active", payload.IsActive)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	return http.ListenAndServe(a.adminAddr, nil)
}
