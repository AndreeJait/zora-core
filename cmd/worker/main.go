package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/AndreeJait/go-utility/v2/brokerw"
	"github.com/AndreeJait/go-utility/v2/gracefulw"
	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/zora-core/port/outbound"
	"github.com/AndreeJait/zora-core/usecase"
)

func main() {
	configFlag := flag.String("config", "files/config/app.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logw.Init(&logw.LogConfig{
		Level:       cfg.Log.Level,
		Format:      cfg.Log.Format,
		WriteToFile: cfg.Log.WriteToFile,
		FilePath:    cfg.Log.FilePath,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}

	logw.Infof("Starting %s worker", cfg.App.Name)

	// Wire all dependencies
	cleanup, err := wireWorker(cfg)
	if err != nil {
		logw.Errorf("failed to wire dependencies: %v", err)
		os.Exit(1)
	}

	// Extract dependencies from dig container
	var (
		settingRepo   outbound.SettingRepository
		adminRepo     outbound.AdminRepository
		taskRepo      outbound.TaskRepository
		dispatcher    *usecase.TaskDispatcher
		taskHandler   *usecase.TaskHandler
		stepHandler   *usecase.GraphStepHandler
		retrySweeper  *usecase.RetrySweeper
		consumer      brokerw.Consumer
	)
	if err := digContainer.Invoke(func(
		sr outbound.SettingRepository,
		ar outbound.AdminRepository,
		tr outbound.TaskRepository,
		d *usecase.TaskDispatcher,
		th *usecase.TaskHandler,
		sh *usecase.GraphStepHandler,
		rs *usecase.RetrySweeper,
		c brokerw.Consumer,
	) {
		settingRepo = sr
		adminRepo = ar
		taskRepo = tr
		dispatcher = d
		taskHandler = th
		stepHandler = sh
		retrySweeper = rs
		consumer = c
	}); err != nil {
		logw.Errorf("failed to extract worker dependencies: %v", err)
		os.Exit(1)
	}

	// Seed default settings
	usecase.SeedDefaults(context.Background(), settingRepo)

	// Seed config-based admins into database
	usecase.SeedAdmins(context.Background(), adminRepo, cfg)

	// Recover stuck tasks from previous crash
	usecase.RecoverStuckTasks(context.Background(), taskRepo, dispatcher)

	// Start NSQ consumers
	ctx, cancel := context.WithCancel(context.Background())

	if err := consumer.Consume(ctx, usecase.TopicTask, taskHandler.HandleTask); err != nil {
		logw.Errorf("failed to start NSQ consumer for %s: %v", usecase.TopicTask, err)
		cancel()
		os.Exit(1)
	}
	logw.Infof("NSQ consumer started for topic: %s", usecase.TopicTask)

	if err := consumer.Consume(ctx, usecase.TopicGraphStep, stepHandler.HandleGraphStep); err != nil {
		logw.Errorf("failed to start NSQ consumer for %s: %v", usecase.TopicGraphStep, err)
		cancel()
		os.Exit(1)
	}
	logw.Infof("NSQ consumer started for topic: %s", usecase.TopicGraphStep)

	// Start retry sweeper as goroutine ticker (1 min interval)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := retrySweeper.Sweep(ctx); err != nil {
					logw.Errorf("retry sweep failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	logw.Info("Retry sweeper started (1m interval)")

	// Graceful shutdown
	gracefulw.Register("nsq-consumer", func(ctx context.Context) error {
		cancel()
		return consumer.Close()
	})
	gracefulw.Register("nsq-producer", func(ctx context.Context) error {
		return dispatcher.Close()
	})
	gracefulw.Register("dependencies", cleanup)

	logw.Info("Worker ready")
	gracefulw.Start(func() error { select {} }, cfg.Graceful.ShutdownTimeout)
}