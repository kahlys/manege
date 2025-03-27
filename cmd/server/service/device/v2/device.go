package device

import (
	"log/slog"
	"sync"

	pb "github.com/kahlys/manege/internal/proto"
)

type DeviceService struct {
	pb.UnimplementedNotificationServiceV2Server
	clients map[string]pb.NotificationServiceV2_NotificationStreamServer
	mu      sync.Mutex
}

func NewDeviceService() *DeviceService {
	return &DeviceService{
		clients: make(map[string]pb.NotificationServiceV2_NotificationStreamServer),
	}
}

func (ds *DeviceService) NotificationStream(req *pb.NotificationRequestV2, stream pb.NotificationServiceV2_NotificationStreamServer) error {
	ctx := stream.Context()

	slog.Info("ClientConnectedV2", "email", req.ClientEmail, "name", req.ClientName)

	clientID := req.ClientEmail
	ds.mu.Lock()
	ds.clients[clientID] = stream
	ds.mu.Unlock()

	defer func() {
		ds.mu.Lock()
		delete(ds.clients, clientID)
		ds.mu.Unlock()
		slog.Info("ClientDisconnectedV2", "email", clientID)
	}()

	<-ctx.Done()
	return ctx.Err()
}

func (ds *DeviceService) NotifyClients() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	for clientID, clientStream := range ds.clients {
		resp := &pb.NotificationResponseV2{Message: "Update available (v2)"}
		if err := clientStream.Send(resp); err != nil {
			slog.Warn("BroadcastFailedV2", "client", clientID, "error", err)
		}
	}
}
