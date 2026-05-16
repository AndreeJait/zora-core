package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/AndreeJait/go-utility/v2/gracefulw"
	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/zora-core/port/outbound"
	_ "github.com/AndreeJait/zora-core/docs"
	docs "github.com/AndreeJait/zora-core/docs"
	"github.com/AndreeJait/zora-core/usecase"
)

// @title Zora Core API
// @version 1.0
// @description Agent orchestration service for the Zora platform.
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

func main() {
	engineFlag := flag.String("engine", "", "HTTP engine: echo|gin|mux (overrides config file)")
	configFlag := flag.String("config", "files/config/app.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// CLI flag overrides config file
	if *engineFlag != "" {
		cfg.HTTP.Engine = *engineFlag
	}

	// Override swagger host from config port
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.App.HTTPPort)

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

	logw.Infof("Starting %s with engine: %s", cfg.App.Name, cfg.HTTP.Engine)

	// Wire all dependencies
	handler, cleanup, err := wire(cfg)
	if err != nil {
		logw.Errorf("failed to wire dependencies: %v", err)
		os.Exit(1)
	}

	// Extract dependencies for startup
	var (
		settingRepo outbound.SettingRepository
		adminRepo   outbound.AdminRepository
	)
	if err := digContainer.Invoke(func(
		sr outbound.SettingRepository,
		ar outbound.AdminRepository,
	) {
		settingRepo = sr
		adminRepo = ar
	}); err != nil {
		logw.Errorf("failed to extract dependencies: %v", err)
		os.Exit(1)
	}

	// Seed default settings
	usecase.SeedDefaults(context.Background(), settingRepo)

	// Seed config-based admins into database
	usecase.SeedAdmins(context.Background(), adminRepo, cfg)

	// Start server with graceful shutdown
	addr := fmt.Sprintf(":%d", cfg.App.HTTPPort)
	srv := &http.Server{Addr: addr, Handler: handler}

	gracefulw.Register("http-server", srv.Shutdown)
	gracefulw.Register("dependencies", cleanup)

	logw.Infof("HTTP server listening on %s", addr)
	gracefulw.Start(srv.ListenAndServe, cfg.Graceful.ShutdownTimeout)
}