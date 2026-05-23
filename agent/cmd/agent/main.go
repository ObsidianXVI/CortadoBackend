package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	ptymanager "github.com/your-org/cortado/agent/internal/pty"
	agentserver "github.com/your-org/cortado/agent/internal/server"
	"github.com/your-org/cortado/agent/internal/usage"
	"google.golang.org/grpc"
)

const defaultGRPCPort = "9090"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := os.Getenv("CORTADO_AGENT_GRPC_PORT")
	if port == "" {
		port = defaultGRPCPort
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	usageTracker, err := usage.NewTrackerFromEnv(ctx)
	if err != nil {
		log.Fatalf("initialize usage tracker: %v", err)
	}
	defer func() {
		if closeErr := usageTracker.Close(); closeErr != nil {
			log.Printf("close usage tracker: %v", closeErr)
		}
	}()
	if err := usageTracker.ReplayPending(ctx); err != nil {
		log.Printf("replay usage WAL: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterWorkspaceAgentServiceServer(
		grpcServer,
		agentserver.NewAgentServer(&ptymanager.Manager{}, usageTracker),
	)

	go func() {
		<-ctx.Done()

		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
		case <-time.After(15 * time.Second):
			grpcServer.Stop()
		}
	}()

	log.Printf("cortado-agent listening on :%s", port)
	if err := grpcServer.Serve(listener); err != nil && ctx.Err() == nil {
		log.Fatalf("serve: %v", err)
	}
}
