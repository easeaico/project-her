package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	internal "github.com/easeaico/adk-memory-agent/internal/agent"
	"github.com/easeaico/adk-memory-agent/internal/config"
	"github.com/easeaico/adk-memory-agent/internal/memory"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n正在关闭...")
		cancel()
		// launcher 可能阻塞等待 stdin，给它短暂时间退出
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	store, err := memory.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	embedder, err := memory.NewEmbedder(ctx, cfg.GoogleAPIKey, cfg.EmbeddingModel)
	if err != nil {
		log.Fatalf("failed to create embedder service: %v", err)
	}

	memoryService := memory.NewService(embedder, store.Store, cfg.TopK, cfg.SimilarityThreshold)

	llmAgent, err := internal.NewGirlfriendAgent(ctx, embedder, store.Store, &cfg, memoryService)
	if err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	launcherConfig := &launcher.Config{
		MemoryService: memoryService,
		AgentLoader:   agent.NewSingleLoader(llmAgent),
	}
	l := full.NewLauncher()

	if err := l.Execute(ctx, launcherConfig, os.Args[1:]); err != nil {
		if err != context.Canceled && err != context.DeadlineExceeded {
			log.Fatalf("Failed to run agent: %v\n\n%s", err, l.CommandLineSyntax())
		}
	}
}
