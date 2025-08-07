// Package main provides the entry point for the Cloud Update service.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kodflow/cloud-update/src/internal/application/handler"
	"github.com/kodflow/cloud-update/src/internal/domain/service"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/config"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/security"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
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

	// Initialize components
	authenticator := security.NewHMACAuthenticator(cfg.Secret)
	systemExecutor := system.NewSystemExecutor()
	actionService := service.NewActionService(systemExecutor)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler()
	webhookHandler := handler.NewWebhookHandler(actionService, authenticator)

	// Setup HTTP routes
	http.HandleFunc("/health", healthHandler.HandleHealth)
	http.HandleFunc("/webhook", webhookHandler.HandleWebhook)

	// Start server with proper timeouts
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting Cloud Update service on %s", addr)
	log.Printf("Version: %s", version.GetFullVersion())
	log.Printf("Log level: %s", cfg.LogLevel)

	server := &http.Server{
		Addr:         addr,
		Handler:      nil, // uses DefaultServeMux
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
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
