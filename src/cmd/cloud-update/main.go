// Package main provides the entry point for the Cloud Update service.
package main

import (
	"context"
	"crypto/tls"
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
	"github.com/kodflow/cloud-update/src/internal/infrastructure/console"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/logger"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/ratelimit"
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
		console.Println(version.GetFullVersion())
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
	authenticator, authErr := security.NewHMACAuthenticator(cfg.Secret)
	if authErr != nil {
		logger.Fatalf("Failed to initialize authenticator: %v", authErr)
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

	// Initialize rate limiter
	rateLimiter := ratelimit.NewRateLimiter(ratelimit.DefaultConfig())

	// Initialize handlers with worker pool support
	healthHandler := handler.NewHealthHandler()
	webhookHandler := handler.NewWebhookHandlerWithPool(actionService, authenticator, workerPool)

	// Start cleanup goroutine for old jobs
	go webhookHandler.Cleanup()

	// Setup HTTP routes with rate limiting on webhook endpoint
	http.HandleFunc("/health", healthHandler.HandleHealth)
	http.HandleFunc("/webhook", rateLimiter.MiddlewareFunc(webhookHandler.HandleWebhook))
	http.HandleFunc("/job/status", webhookHandler.HandleJobStatus)

	// Load TLS configuration
	tlsConfig := config.LoadTLSConfig()
	if err := tlsConfig.Validate(); err != nil {
		logger.Warnf("TLS configuration error: %v", err)
		logger.Info("Starting without TLS (HTTP only)")
	}

	// Start server with proper timeouts
	addr := fmt.Sprintf(":%s", cfg.Port)
	protocol := "HTTP"
	if tlsConfig.Enabled {
		protocol = "HTTPS"
	}

	logger.Infof("Starting Cloud Update service on %s (%s)", addr, protocol)
	logger.Infof("Version: %s", version.GetFullVersion())
	logger.Infof("Log level: %s", cfg.LogLevel)
	logger.Infof("Log file: /var/log/cloud-update/cloud-update.log")
	logger.Infof("Rate limiting: %d req/s, burst: %d", 10, 20)
	logger.Infof("Worker pool: 10 workers, 100 task backlog")

	// Configure TLS if enabled
	var serverTLSConfig *tls.Config
	if tlsConfig.Enabled && !tlsConfig.Auto {
		var err error
		serverTLSConfig, err = tlsConfig.GetTLSConfig()
		if err != nil {
			logger.Fatalf("Failed to configure TLS: %v", err)
		}
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           nil, // uses DefaultServeMux
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 16, // 64KB
		TLSConfig:         serverTLSConfig,
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

	// Start server
	var err error
	if tlsConfig.Enabled {
		if tlsConfig.Auto {
			logger.Info("Automatic TLS certificate management not yet implemented")
			logger.Info("Starting with HTTP only")
			err = server.ListenAndServe()
		} else {
			logger.Infof("Starting HTTPS server with certificates from %s", tlsConfig.CertFile)
			err = server.ListenAndServeTLS(tlsConfig.CertFile, tlsConfig.KeyFile)
		}
	} else {
		err = server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Failed to start server: %v", err)
	}

	logger.Info("Cloud Update service stopped")
}

func printHelp() {
	console.Println("Cloud Update Service")
	console.Println()
	console.Println("Usage:")
	console.Println("  cloud-update [options]")
	console.Println()
	console.Println("Options:")
	console.Println("  --version     Show version information")
	console.Println("  --help        Show this help message")
	console.Println("  --setup       Install service on the system")
	console.Println("  --uninstall   Uninstall service from the system")
	console.Println()
	console.Println("Environment Variables:")
	console.Println("  CLOUD_UPDATE_PORT       Port to listen on (default: 9999)")
	console.Println("  CLOUD_UPDATE_SECRET     HMAC secret for webhook authentication (required)")
	console.Println("  CLOUD_UPDATE_LOG_LEVEL  Log level: debug, info, warn, error (default: info)")
	console.Println()
	console.Println("Service Control:")
	console.Println("  systemctl start cloud-update    # Start service")
	console.Println("  systemctl stop cloud-update     # Stop service")
	console.Println("  systemctl status cloud-update   # Check status")
	console.Println("  journalctl -u cloud-update -f   # View logs")
}
