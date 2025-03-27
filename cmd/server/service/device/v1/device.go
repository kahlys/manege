package device

import (
	"log/slog"
	"sync"

	pb "github.com/kahlys/manege/internal/proto"
)

type DeviceService struct {
	pb.UnimplementedNotificationServiceServer
	clients map[string]pb.NotificationService_NotificationStreamServer
	mu      sync.Mutex
}

func NewDeviceService() *DeviceService {
	return &DeviceService{
		clients: make(map[string]pb.NotificationService_NotificationStreamServer),
	}
}

func (ds *DeviceService) NotificationStream(req *pb.NotificationRequest, stream pb.NotificationService_NotificationStreamServer) error {
	ctx := stream.Context()

	slog.Info("ClientConnected", "email", req.ClientEmail)

	clientEmail := req.ClientEmail
	ds.mu.Lock()
	ds.clients[clientEmail] = stream
	ds.mu.Unlock()

	defer func() {
		ds.mu.Lock()
		delete(ds.clients, clientEmail)
		ds.mu.Unlock()
		slog.Info("ClientDisconnected", "email", clientEmail)
	}()

	<-ctx.Done()
	return ctx.Err()
}

func (ds *DeviceService) NotifyClients() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	for clientID, clientStream := range ds.clients {
		resp := &pb.NotificationResponse{Message: "Update available"}
		if err := clientStream.Send(resp); err != nil {
			slog.Warn("BroadcastFailed", "client", clientID, "error", err)
		}
	}
}
