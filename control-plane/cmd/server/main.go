package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-org/cortado/control-plane/internal/api"
	"github.com/your-org/cortado/control-plane/internal/gateway"
)

const defaultHTTPPort = "8080"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	workspaceNamespace := os.Getenv("CORTADO_WORKSPACE_NAMESPACE")
	clusterDNSDomain := os.Getenv("CORTADO_CLUSTER_DNS_DOMAIN")
	workspaceService, err := newWorkspaceService(ctx)
	if err != nil {
		log.Fatalf("initialize workspace service: %v", err)
	}

	server := &http.Server{
		Addr: ":" + port,
		Handler: api.NewRouter(api.RouterConfig{
			ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
				WorkspaceResolver: gateway.StaticWorkspaceResolver{
					Namespace: workspaceNamespace,
					DNSDomain: clusterDNSDomain,
				},
			}),
			WorkspaceSvc: workspaceService,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown http server: %v", err)
		}
	}()

	log.Printf("control-plane listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}
