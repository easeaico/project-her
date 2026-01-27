// Package main boots the Project Her platform service and wires application dependencies.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	internalagent "github.com/easeaico/project-her/internal/agent"
	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/memory"
	"github.com/easeaico/project-her/internal/storage"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/session/database"
	"gorm.io/driver/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	slog.Info("slog logger initialized", "level", "debug")

	cfg := config.Load()
	slog.Info("configuration loaded", "chat_model", cfg.ChatModel, "memory_model", cfg.MemoryModel, "image_model", cfg.ImageModel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := storage.NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	memoryService := memory.NewService(ctx, &cfg, store.Memories, store.ChatHistories)

	sessionService, err := database.NewSessionService(postgres.Open(cfg.DatabaseURL))
	if err != nil {
		log.Fatalf("failed to create session service: %v", err)
	}

	llmAgent, err := internalagent.NewRolePlayAgent(ctx, &cfg, store.Characters, sessionService, memoryService)
	if err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	launcherConfig := &launcher.Config{
		SessionService: sessionService,
		MemoryService:  memoryService,
		AgentLoader:    agent.NewSingleLoader(llmAgent),
	}

	l := full.NewLauncher()
	errCh := make(chan error, 1)
	go func() {
		slog.Info("launcher starting")
		errCh <- l.Execute(ctx, launcherConfig, os.Args[1:])
	}()

	var execErr error
	select {
	case execErr = <-errCh:
	case <-ctx.Done():
		fmt.Println("\n正在关闭...")
	}

	if execErr != nil {
		if execErr != context.Canceled && execErr != context.DeadlineExceeded {
			log.Fatalf("Failed to run agent: %v\n\n%s", execErr, l.CommandLineSyntax())
		}
	}

	fmt.Println("Agent shutdown complete")
}
