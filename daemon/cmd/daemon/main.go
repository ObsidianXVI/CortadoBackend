package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/your-org/cortado/daemon/internal/app"
	"github.com/your-org/cortado/daemon/internal/config"
	"github.com/your-org/cortado/daemon/internal/state"
	"github.com/your-org/cortado/daemon/internal/version"
	"github.com/your-org/cortado/daemon/internal/watch"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("load daemon config: %v", err)
	}

	store, err := state.Open(cfg.StatePath)
	if err != nil {
		log.Fatalf("open daemon state: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			log.Printf("close daemon state: %v", closeErr)
		}
	}()

	logger := log.New(os.Stderr, "", log.LstdFlags)
	conflictBroadcaster := app.NewConflictBroadcaster()
	if len(cfg.WatchRoots) > 0 {
		manager, err := watch.NewManager(watch.ManagerConfig{
			Logger:     logger,
			Roots:      cfg.WatchRoots,
			StateStore: store,
		})
		if err != nil {
			log.Fatalf("initialize daemon watcher: %v", err)
		}

		go func() {
			if err := manager.Run(ctx); err != nil && ctx.Err() == nil {
				logger.Printf("daemon watcher stopped: %v", err)
			}
		}()
		go func() {
			for event := range manager.Events() {
				logger.Printf(
					"watched file event type=%s path=%s checksum=%s",
					event.Type,
					event.Path,
					event.Checksum,
				)
			}
		}()
		go func() {
			for warning := range manager.Warnings() {
				logger.Printf("watch warning: %s", warning)
			}
		}()
	}

	server, err := app.NewServer(app.ServerConfig{
		ConflictBroadcaster: conflictBroadcaster,
		ListenAddr:          cfg.ListenAddr,
		Logger:              logger,
		StateStore:          store,
		Version:             version.Info(),
	})
	if err != nil {
		log.Fatalf("initialize daemon server: %v", err)
	}

	logger.Printf("cortado-daemon listening on %s", cfg.ListenAddr)
	if err := server.Run(ctx); err != nil {
		log.Fatalf("run daemon server: %v", err)
	}
}
