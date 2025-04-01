package main

import (
	"context"
	"flag"
	"io"
	"log"
	"log/slog"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/kahlys/manege/internal/proto"
)

func main() {
	email := flag.String("email", "user", "Client email")
	flag.Parse()

	conn, err := grpc.NewClient(
		"server:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("ClientFailed", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewNotificationServiceClient(conn)
	stream, err := client.NotificationStream(
		context.Background(),
		&pb.NotificationRequest{ClientEmail: *email},
	)
	if err != nil {
		log.Fatalf("could not open stream: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("ClientNotificationFailed", "error", err)
			os.Exit(1)
		}
		slog.Info("ClientNotificationReceived", "message", resp.GetMessage())
	}
}
