package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	internalagent "github.com/easeaico/project-her/internal/agent"
	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/memory"
	"github.com/easeaico/project-her/internal/models"
	"github.com/easeaico/project-her/internal/repository"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/session/database"
	"google.golang.org/genai"
	"gorm.io/driver/postgres"
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

	store, err := repository.NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	embedder, err := memory.NewEmbedder(ctx, cfg.GoogleAPIKey, cfg.EmbeddingModel)
	if err != nil {
		log.Fatalf("failed to create embedder service: %v", err)
	}

	summarizerModel, err := models.NewGrokModel(ctx, cfg.LLMModel, &genai.ClientConfig{
		APIKey: cfg.XAIAPIKey,
	})
	if err != nil {
		log.Fatalf("failed to create summarizer model: %v", err)
	}

	summarizer, err := internalagent.NewMemorySummarizer(ctx, summarizerModel)
	if err != nil {
		log.Fatalf("failed to create memory summarizer: %v", err)
	}

	memoryService := memory.NewService(embedder, store.Memories, store.ChatHistories, summarizer, cfg.TopK, cfg.SimilarityThreshold)
	sessionService, err := database.NewSessionService(postgres.Open(cfg.DatabaseURL))
	if err != nil {
		log.Fatalf("failed to create session service: %v", err)
	}
	// err = database.AutoMigrate(sessionService)
	// if err != nil {
	// 	log.Fatalf("failed to migrate session service: %v", err)
	// }

	llmAgent, err := internalagent.NewRolePlayAgent(ctx, &cfg, store.Characters)
	if err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	launcherConfig := &launcher.Config{
		SessionService: sessionService,
		MemoryService:  memoryService,
		AgentLoader:    agent.NewSingleLoader(llmAgent),
	}

	l := full.NewLauncher()
	if err := l.Execute(ctx, launcherConfig, os.Args[1:]); err != nil {
		if err != context.Canceled && err != context.DeadlineExceeded {
			log.Fatalf("Failed to run agent: %v\n\n%s", err, l.CommandLineSyntax())
		}
	}
}
