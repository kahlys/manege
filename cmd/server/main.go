package main

import (
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	pb "github.com/kahlys/manege/internal/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedNotificationServiceServer
	clients map[string]pb.NotificationService_NotificationStreamServer
	mu      sync.Mutex
}

type serverV2 struct {
	pb.UnimplementedNotificationServiceV2Server
	clients map[string]pb.NotificationServiceV2_NotificationStreamServer
	mu      sync.Mutex
}

func (s *server) NotificationStream(req *pb.NotificationRequest, stream pb.NotificationService_NotificationStreamServer) error {
	ctx := stream.Context()

	slog.Info("ClientConnected", "email", req.ClientEmail)

	clientEmail := req.ClientEmail
	s.mu.Lock()
	s.clients[clientEmail] = stream
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientEmail)
		s.mu.Unlock()
		slog.Info("ClientDisconnected", "email", clientEmail)
	}()

	<-ctx.Done()
	return ctx.Err()
}

func (s *server) notifyClients() {
	message := "Update available"
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("BroadcastMessage", "message", message)
	for clientID, clientStream := range s.clients {
		resp := &pb.NotificationResponse{Message: message}
		if err := clientStream.Send(resp); err != nil {
			slog.Warn("BroadcastFailed", "client", clientID, "error", err)
		}
	}
}

func (s *serverV2) NotificationStream(req *pb.NotificationRequestV2, stream pb.NotificationServiceV2_NotificationStreamServer) error {
	ctx := stream.Context()

	slog.Info("ClientConnectedV2", "email", req.ClientEmail, "name", req.ClientName)

	clientID := req.ClientEmail
	s.mu.Lock()
	s.clients[clientID] = stream
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
		slog.Info("ClientDisconnectedV2", "email", clientID)
	}()

	<-ctx.Done()
	return ctx.Err()
}

func (s *serverV2) notifyClients() {
	message := "Update available (v2)"
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("BroadcastMessageV2", "message", message)
	for clientID, clientStream := range s.clients {
		resp := &pb.NotificationResponseV2{Message: message}
		if err := clientStream.Send(resp); err != nil {
			slog.Warn("BroadcastFailedV2", "client", clientID, "error", err)
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("ServerFailed", "error", err)
		os.Exit(1)
	}
	s := grpc.NewServer()

	// Register v1 service
	srvV1 := &server{
		clients: make(map[string]pb.NotificationService_NotificationStreamServer),
	}
	pb.RegisterNotificationServiceServer(s, srvV1)

	// Register v2 service
	srvV2 := &serverV2{
		clients: make(map[string]pb.NotificationServiceV2_NotificationStreamServer),
	}
	pb.RegisterNotificationServiceV2Server(s, srvV2)

	slog.Info("ServerStart", "port", ":50051")

	// Notify clients periodically for both v1 and v2
	go func() {
		for {
			time.Sleep(10 * time.Second)
			srvV1.notifyClients()
			srvV2.notifyClients()
		}
	}()

	if err := s.Serve(lis); err != nil {
		slog.Error("ServerFailed", "error", err)
		os.Exit(1)
	}
}
