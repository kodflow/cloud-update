// Package main provides the entry point for the Cloud Update service.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kodflow/cloud-update/src/internal/application/handler"
	"github.com/kodflow/cloud-update/src/internal/domain/service"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/config"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/logger"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/security"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/worker"
	"github.com/kodflow/cloud-update/src/internal/setup"
	"github.com/kodflow/cloud-update/src/internal/version"
)

func main() {
	var (
		showVersion  = flag.Bool("version", false, "Show version information")
		showHelp     = flag.Bool("help", false, "Show help")
		runSetup     = flag.Bool("setup", false, "Install service on the system")
		runUninstall = flag.Bool("uninstall", false, "Uninstall service from the system")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Println(version.GetFullVersion())
		os.Exit(0)
	}

	if *runSetup {
		installer := setup.NewServiceInstaller()
		if err := installer.Setup(); err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *runUninstall {
		installer := setup.NewServiceInstaller()
		if err := installer.Uninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "Uninstall failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logCfg := logger.Config{
		Level:      cfg.LogLevel,
		FilePath:   cfg.LogFilePath,
		MaxSize:    10 * 1024 * 1024, // 10MB
		MaxBackups: 5,
	}
	if err := logger.Initialize(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		// Continue with stdout logging
	}
	defer logger.Close()

	// Initialize components
	authenticator, err := security.NewHMACAuthenticator(cfg.Secret)
	if err != nil {
		logger.Fatalf("Failed to initialize authenticator: %v", err)
	}
	// Initialize worker pool for async processing
	workerPool := worker.NewPool(10, 100) // 10 workers, 100 task backlog
	defer func() {
		if err := workerPool.Shutdown(30 * time.Second); err != nil {
			logger.Errorf("Failed to shutdown worker pool: %v", err)
		}
	}()

	systemExecutor := system.NewSystemExecutor()
	actionService := service.NewActionService(systemExecutor)

	// Initialize handlers with status tracking
	healthHandler := handler.NewHealthHandler()
	webhookHandler := handler.NewWebhookHandlerWithStatus(actionService, authenticator)

	// Start cleanup goroutine for old jobs
	go webhookHandler.Cleanup()

	// Setup HTTP routes
	http.HandleFunc("/health", healthHandler.HandleHealth)
	http.HandleFunc("/webhook", webhookHandler.HandleWebhook)
	http.HandleFunc("/job/status", webhookHandler.HandleJobStatus)

	// Start server with proper timeouts
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Infof("Starting Cloud Update service on %s", addr)
	logger.Infof("Version: %s", version.GetFullVersion())
	logger.Infof("Log level: %s", cfg.LogLevel)
	logger.Infof("Log file: /var/log/cloud-update/cloud-update.log")

	server := &http.Server{
		Addr:              addr,
		Handler:           nil, // uses DefaultServeMux
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 16, // 64KB
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Errorf("Server shutdown error: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Server failed: %v", err)
	}
}

func printHelp() {
	fmt.Println("Cloud Update Service")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cloud-update [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --version     Show version information")
	fmt.Println("  --help        Show this help message")
	fmt.Println("  --setup       Install service on the system")
	fmt.Println("  --uninstall   Uninstall service from the system")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CLOUD_UPDATE_PORT       Port to listen on (default: 9999)")
	fmt.Println("  CLOUD_UPDATE_SECRET     HMAC secret for webhook authentication (required)")
	fmt.Println("  CLOUD_UPDATE_LOG_LEVEL  Log level: debug, info, warn, error (default: info)")
	fmt.Println()
	fmt.Println("Service Control:")
	fmt.Println("  systemctl start cloud-update    # Start service")
	fmt.Println("  systemctl stop cloud-update     # Stop service")
	fmt.Println("  systemctl status cloud-update   # Check status")
	fmt.Println("  journalctl -u cloud-update -f   # View logs")
}
