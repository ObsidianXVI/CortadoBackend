package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	filesyncpb "github.com/your-org/cortado/agent/gen/filesync/v1"
	"github.com/your-org/cortado/control-plane/internal/api"
	filesyncsvc "github.com/your-org/cortado/control-plane/internal/filesync"
	"github.com/your-org/cortado/control-plane/internal/gateway"
	"github.com/your-org/cortado/control-plane/internal/workspace"
	"google.golang.org/grpc"
)

const defaultHTTPPort = "8080"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	projectID := gcpProjectID()
	if projectID == "" {
		log.Fatal("missing GCP project id")
	}

	firestoreClient, err := newFirestoreClient(ctx, projectID)
	if err != nil {
		log.Fatalf("initialize firestore client: %v", err)
	}
	defer func() {
		if closeErr := firestoreClient.Close(); closeErr != nil {
			log.Printf("close firestore client: %v", closeErr)
		}
	}()

	workspaceNamespace := os.Getenv("CORTADO_WORKSPACE_NAMESPACE")
	clusterDNSDomain := os.Getenv("CORTADO_CLUSTER_DNS_DOMAIN")
	workspaceService, err := newWorkspaceService(ctx, projectID, firestoreClient)
	if err != nil {
		log.Fatalf("initialize workspace service: %v", err)
	}
	authService, err := newSessionService(firestoreClient)
	if err != nil {
		log.Fatalf("initialize auth service: %v", err)
	}
	defer func() {
		if closeErr := authService.Close(); closeErr != nil {
			log.Printf("close auth service: %v", closeErr)
		}
	}()
	resolver := gateway.StaticWorkspaceResolver{
		Namespace: workspaceNamespace,
		DNSDomain: clusterDNSDomain,
	}
	aiService, err := newAIService(projectID, resolver)
	if err != nil {
		log.Fatalf("initialize ai service: %v", err)
	}
	fileService := workspace.NewAgentFileService(workspace.AgentFileServiceConfig{
		WorkspaceResolver: resolver,
	})
	terminalBridge := gateway.NewTerminalBridge(gateway.TerminalBridgeConfig{
		WorkspaceResolver: resolver,
	})
	connectHandler := gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
		TerminalHandler: func(handlerCtx context.Context, session gateway.Session, frame gateway.Frame, receivedAt time.Time) error {
			if err := terminalBridge.HandleFrame(handlerCtx, session, frame, receivedAt); err != nil {
				return err
			}
			if frame.MessageType == gateway.MessageTypeData {
				if activityErr := workspaceService.RecordActivity(handlerCtx, session.WorkspaceID, receivedAt); activityErr != nil {
					log.Printf("record workspace activity workspace=%s: %v", session.WorkspaceID, activityErr)
				}
			}
			return nil
		},
		WorkspaceResolver: resolver,
	})

	go workspace.NewIdleMonitor(workspace.IdleMonitorConfig{
		IdleInspector: workspace.NewAgentIdleInspector(workspace.AgentIdleInspectorConfig{
			WorkspaceResolver: resolver,
		}),
		IdleTimeout: idleTimeoutFromEnv(),
		Service:     workspaceService,
	}).Run(ctx)

	fileSyncService, err := filesyncsvc.NewService(filesyncsvc.ServiceConfig{
		WorkspaceFiles: fileService,
	})
	if err != nil {
		log.Fatalf("initialize file sync service: %v", err)
	}
	grpcServer := grpc.NewServer()
	filesyncpb.RegisterFileSyncServiceServer(grpcServer, fileSyncService)

	httpHandler := api.NewRouter(api.RouterConfig{
		AICompletionSvc:  aiService,
		ConnectHandler:   connectHandler,
		JWKSProvider:     authService,
		SessionSvc:       authService,
		WorkspaceFileSvc: fileService,
		WorkspaceSvc:     workspaceService,
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           filesyncsvc.NewHTTPMux(grpcServer, httpHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown http server: %v", err)
		}
		grpcServer.Stop()
	}()

	log.Printf("control-plane listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}

func idleTimeoutFromEnv() time.Duration {
	value := os.Getenv("CORTADO_IDLE_TIMEOUT_MINUTES")
	if value == "" {
		return 20 * time.Minute
	}

	minutes, err := strconv.Atoi(value)
	if err != nil || minutes <= 0 {
		log.Printf("invalid CORTADO_IDLE_TIMEOUT_MINUTES=%q, using default 20", value)
		return 20 * time.Minute
	}

	return time.Duration(minutes) * time.Minute
}
