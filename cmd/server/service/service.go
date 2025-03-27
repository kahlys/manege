package service

import (
	"log/slog"
	"net"
	"time"

	devicev1 "github.com/kahlys/manege/cmd/server/service/device/v1"
	devicev2 "github.com/kahlys/manege/cmd/server/service/device/v2"
	pb "github.com/kahlys/manege/internal/proto"
	"google.golang.org/grpc"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (a *Server) Run() error {
	if err := a.runDeviceManagement(); err != nil {
		return err
	}

	return nil
}

func (a *Server) runDeviceManagement() error {
	slog.Info("ServerStarts", "module", "device", "port", ":50051")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}
	grpcSrv := grpc.NewServer()

	deviceSvc := devicev1.NewDeviceService()
	pb.RegisterNotificationServiceServer(grpcSrv, deviceSvc)
	pb.RegisterNotificationServiceV2Server(grpcSrv, devicev2.NewDeviceService())

	go func() {
		for {
			time.Sleep(10 * time.Second)
			slog.Info("SendNotification")
			deviceSvc.NotifyClients()
		}
	}()

	return grpcSrv.Serve(lis)
}
