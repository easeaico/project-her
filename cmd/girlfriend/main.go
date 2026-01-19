// Package main is the entry point for the Legacy Code Hunter agent.
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

// main is the entry point for the Legacy Code Hunter agent application.
func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 优雅关闭处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n正在关闭...")
		cancel()
		// 给一个短暂的时间让程序优雅关闭，然后强制退出
		// 这是因为 launcher 可能在阻塞等待 stdin 输入，context 取消无法中断
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	// 初始化数据库连接
	store, err := memory.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	// 初始化嵌入服务
	embedder, err := memory.NewEmbedder(ctx, cfg.APIKey)
	if err != nil {
		log.Fatalf("failed to create embedder service: %v", err)
	}

	// 创建记忆服务
	memoryService := memory.NewService(embedder, store)

	// 初始化Agent
	llmAgent, err := internal.NewHunterAgent(ctx, embedder, store, &cfg)
	if err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// 启动launcher
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
