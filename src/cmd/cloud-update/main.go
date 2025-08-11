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
	if handleSpecialCommands() {
		return
	}

	runServer()
}

// handleSpecialCommands processes version, help, setup, and uninstall commands.
// Returns true if a special command was handled and the program should exit.
func handleSpecialCommands() bool {
	var (
		showVersion  = flag.Bool("version", false, "Show version information")
		showHelp     = flag.Bool("help", false, "Show help")
		runSetup     = flag.Bool("setup", false, "Install service on the system")
		runUninstall = flag.Bool("uninstall", false, "Uninstall service from the system")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		return true
	}

	if *showVersion {
		console.Println(version.GetFullVersion())
		return true
	}

	if *runSetup {
		handleSetup()
		return true
	}

	if *runUninstall {
		handleUninstall()
		return true
	}

	return false
}

// Variable to allow testing of os.Exit.
var osExit = os.Exit

// handleSetup runs the service installation.
func handleSetup() {
	installer := setup.NewServiceInstaller()
	if err := installer.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
		osExit(1)
	}
}

// handleUninstall runs the service uninstallation.
func handleUninstall() {
	installer := setup.NewServiceInstaller()
	if err := installer.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "Uninstall failed: %v\n", err)
		osExit(1)
	}
}

// runServer starts the main server.
func runServer() {
	cfg := config.Load()
	initializeLogger(cfg)
	defer logger.Close()

	components := initializeComponents(cfg)
	defer components.cleanup()

	setupHTTPRoutes(components)
	server := createHTTPServer(cfg, components.tlsConfig)
	startGracefulShutdown(server)
	startServer(server, components.tlsConfig)
}

// serverComponents holds all initialized server components.
type serverComponents struct {
	authenticator  security.Authenticator
	workerPool     *worker.Pool
	actionService  service.ActionService
	rateLimiter    *ratelimit.RateLimiter
	healthHandler  *handler.HealthHandler
	webhookHandler *handler.WebhookHandlerWithPool
	tlsConfig      *config.TLSConfig
}

// cleanup performs cleanup for all components.
func (c *serverComponents) cleanup() {
	if err := c.workerPool.Shutdown(30 * time.Second); err != nil {
		logger.Errorf("Failed to shutdown worker pool: %v", err)
	}
}

// initializeLogger sets up the logging system.
func initializeLogger(cfg *config.Config) {
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
}

// initializeComponents creates and initializes all server components.
func initializeComponents(cfg *config.Config) *serverComponents {
	authenticator, authErr := security.NewHMACAuthenticator(cfg.Secret)
	if authErr != nil {
		logger.Fatalf("Failed to initialize authenticator: %v", authErr)
	}

	workerPool := worker.NewPool(10, 100) // 10 workers, 100 task backlog
	systemExecutor := system.NewSystemExecutor()
	actionService := service.NewActionService(systemExecutor)
	rateLimiter := ratelimit.NewRateLimiter(ratelimit.DefaultConfig())
	healthHandler := handler.NewHealthHandler()
	webhookHandler := handler.NewWebhookHandlerWithPool(actionService, authenticator, workerPool)

	// Start cleanup goroutine for old jobs
	go webhookHandler.Cleanup()

	// Load and validate TLS configuration
	tlsConfig := config.LoadTLSConfig()
	if err := tlsConfig.Validate(); err != nil {
		logger.Warnf("TLS configuration error: %v", err)
		logger.Info("Starting without TLS (HTTP only)")
	}

	return &serverComponents{
		authenticator:  authenticator,
		workerPool:     workerPool,
		actionService:  actionService,
		rateLimiter:    rateLimiter,
		healthHandler:  healthHandler,
		webhookHandler: webhookHandler,
		tlsConfig:      tlsConfig,
	}
}

// setupHTTPRoutes configures all HTTP route handlers.
func setupHTTPRoutes(components *serverComponents) {
	http.HandleFunc("/health", components.healthHandler.HandleHealth)
	http.HandleFunc("/webhook", components.rateLimiter.MiddlewareFunc(components.webhookHandler.HandleWebhook))
	http.HandleFunc("/job/status", components.webhookHandler.HandleJobStatus)
}

// createHTTPServer creates and configures the HTTP server.
func createHTTPServer(cfg *config.Config, tlsConfig *config.TLSConfig) *http.Server {
	addr := fmt.Sprintf(":%s", cfg.Port)
	logServerInfo(cfg, addr, tlsConfig)

	var serverTLSConfig *tls.Config
	if tlsConfig.Enabled && !tlsConfig.Auto {
		var err error
		serverTLSConfig, err = tlsConfig.GetTLSConfig()
		if err != nil {
			logger.Fatalf("Failed to configure TLS: %v", err)
		}
	}

	return &http.Server{
		Addr:              addr,
		Handler:           nil, // uses DefaultServeMux
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 16, // 64KB
		TLSConfig:         serverTLSConfig,
	}
}

// logServerInfo logs server startup information.
func logServerInfo(cfg *config.Config, addr string, tlsConfig *config.TLSConfig) {
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
}

// startGracefulShutdown sets up graceful shutdown handling.
func startGracefulShutdown(server *http.Server) {
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
}

// startServer starts the HTTP/HTTPS server.
func startServer(server *http.Server, tlsConfig *config.TLSConfig) {
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
