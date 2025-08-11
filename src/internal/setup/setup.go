package setup

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/console"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

// Installation constants.
const (
	InstallDir = "/opt/cloud-update"
	ConfigDir  = "/etc/cloud-update"
	BinaryName = "cloud-update"
)

// ServiceInstaller handles the installation of cloud-update as a system service.
type ServiceInstaller struct {
	distro     system.Distribution
	initSystem InitSystem
	fs         FileSystem
	cmd        CommandRunner
	os         OSInterface
}

// InitSystem represents the type of init system.
type InitSystem string

// Init system types.
const (
	InitSystemd  InitSystem = "systemd"
	InitOpenRC   InitSystem = "openrc"
	InitSysVInit InitSystem = "sysvinit"
	InitUpstart  InitSystem = "upstart"
	InitUnknown  InitSystem = "unknown"
)

// NewServiceInstaller creates a new service installer.
func NewServiceInstaller() *ServiceInstaller {
	return &ServiceInstaller{
		distro:     detectDistribution(),
		initSystem: detectInitSystem(),
		fs:         RealFileSystem{},
		cmd:        RealCommandRunner{},
		os:         RealOSInterface{},
	}
}

// NewServiceInstallerWithDeps creates a new service installer with injected dependencies.
func NewServiceInstallerWithDeps(fs FileSystem, cmd CommandRunner, osIface OSInterface) *ServiceInstaller {
	return &ServiceInstaller{
		distro:     detectDistribution(),
		initSystem: detectInitSystem(),
		fs:         fs,
		cmd:        cmd,
		os:         osIface,
	}
}

// Setup installs the service on the system.
func (s *ServiceInstaller) Setup() error {
	console.Println("🚀 Cloud Update Service Setup")
	console.Println(fmt.Sprintf("📦 Detected: %s with %s", s.distro, s.initSystem))

	// Check if running as root
	if s.os.Geteuid() != 0 && runtime.GOOS != "windows" {
		return fmt.Errorf("setup must be run as root (use sudo)")
	}

	// Create directories
	if err := s.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Copy binary
	if err := s.installBinary(); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}

	// Install service files
	if err := s.installService(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	// Create config file
	if err := s.createConfig(); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	// Enable service
	if err := s.enableService(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	console.Println("✅ Setup completed successfully!")
	s.printNextSteps()

	return nil
}

func (s *ServiceInstaller) createDirectories() error {
	console.Println("📁 Creating directories...")

	dirs := []struct {
		path string
		mode os.FileMode
	}{
		{InstallDir, 0o755},
		{ConfigDir, 0o700},
	}

	for _, dir := range dirs {
		if err := s.fs.MkdirAll(dir.path, dir.mode); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir.path, err)
		}
		console.Println(fmt.Sprintf("  ✓ %s", dir.path))
	}

	return nil
}

func (s *ServiceInstaller) installBinary() error {
	console.Println("📦 Installing binary...")

	// Get current executable path
	execPath, err := s.os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	destPath := filepath.Join(InstallDir, BinaryName)

	// Copy binary
	input, err := s.fs.ReadFile(execPath) //nolint:gosec // execPath comes from os.Executable()
	if err != nil {
		return fmt.Errorf("failed to read executable: %w", err)
	}

	// Binary needs executable permissions
	if err := s.fs.WriteFile(destPath, input, 0o755); err != nil { //nolint:gosec // binary must be executable
		return fmt.Errorf("failed to write binary: %w", err)
	}

	fmt.Printf("  ✓ Installed to %s\n", destPath)
	return nil
}

func (s *ServiceInstaller) installService() error {
	console.Println(fmt.Sprintf("🔧 Installing %s service...", s.initSystem))

	switch s.initSystem {
	case InitSystemd:
		return s.installSystemdService()
	case InitOpenRC:
		return s.installOpenRCService()
	case InitSysVInit:
		return s.installSysVInitService()
	default:
		return fmt.Errorf("unsupported init system: %s", s.initSystem)
	}
}

func (s *ServiceInstaller) installSystemdService() error {
	servicePath := "/etc/systemd/system/cloud-update.service"

	// Systemd service files need 0644 permissions
	if err := s.fs.WriteFile(servicePath, []byte(SystemdService), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write systemd service: %w", err)
	}

	// Reload systemd
	if err := s.cmd.Run("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Printf("  ✓ Installed systemd service to %s\n", servicePath)
	return nil
}

func (s *ServiceInstaller) installOpenRCService() error {
	servicePath := "/etc/init.d/cloud-update"

	// Init scripts need executable permissions
	if err := s.fs.WriteFile(servicePath, []byte(OpenRCScript), 0o755); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write OpenRC script: %w", err)
	}

	fmt.Printf("  ✓ Installed OpenRC service to %s\n", servicePath)
	return nil
}

func (s *ServiceInstaller) installSysVInitService() error {
	servicePath := "/etc/init.d/cloud-update"

	// Init scripts need executable permissions
	if err := s.fs.WriteFile(servicePath, []byte(SysVInitScript), 0o755); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write SysVInit script: %w", err)
	}

	// Update rc.d if available
	if _, err := s.cmd.LookPath("update-rc.d"); err == nil {
		if err := s.cmd.Run("update-rc.d", "cloud-update", "defaults"); err != nil {
			fmt.Printf("  ⚠️  Failed to update rc.d: %v\n", err)
		}
	}

	fmt.Printf("  ✓ Installed SysVInit service to %s\n", servicePath)
	return nil
}

func (s *ServiceInstaller) createConfig() error {
	console.Println("⚙️  Creating configuration...")

	configPath := filepath.Join(ConfigDir, "config.env")

	// Check if config already exists
	if _, err := s.fs.Stat(configPath); err == nil {
		fmt.Printf("  ⚠️  Config already exists at %s\n", configPath)
		return nil
	}

	// Generate secret
	secret, err := generateSecret()
	if err != nil {
		return fmt.Errorf("failed to generate secret: %w", err)
	}

	config := fmt.Sprintf(`# Cloud Update Configuration
# Generated by setup command

# Port on which the HTTP server will listen
CLOUD_UPDATE_PORT=9999

# Secret key for webhook signature validation (HMAC SHA256)
CLOUD_UPDATE_SECRET=%s

# Log level (debug, info, warn, error)
CLOUD_UPDATE_LOG_LEVEL=info
`, secret)

	if err := s.fs.WriteFile(configPath, []byte(config), 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Set ownership to root
	if err := s.fs.Chown(configPath, 0, 0); err != nil {
		return fmt.Errorf("failed to set config ownership: %w", err)
	}

	fmt.Printf("  ✓ Created config at %s\n", configPath)
	fmt.Printf("  🔑 Generated secret and saved to config file\n")
	fmt.Printf("     View it with: sudo cat %s\n", configPath)

	return nil
}

func (s *ServiceInstaller) enableService() error {
	console.Println("🔌 Enabling service...")

	switch s.initSystem {
	case InitSystemd:
		if err := s.cmd.Run("systemctl", "enable", "cloud-update"); err != nil {
			return fmt.Errorf("failed to enable service: %w", err)
		}
	case InitOpenRC:
		if err := s.cmd.Run("rc-update", "add", "cloud-update", "default"); err != nil {
			return fmt.Errorf("failed to enable service: %w", err)
		}
	case InitSysVInit:
		if _, err := s.cmd.LookPath("update-rc.d"); err == nil {
			if err := s.cmd.Run("update-rc.d", "cloud-update", "enable"); err != nil {
				return fmt.Errorf("failed to enable service: %w", err)
			}
		} else {
			fmt.Println("  ⚠️  Please manually enable the service")
			return nil
		}
	default:
		fmt.Printf("  ⚠️  Cannot auto-enable for %s\n", s.initSystem)
		return nil
	}

	fmt.Println("  ✓ Service enabled")
	return nil
}

func (s *ServiceInstaller) printNextSteps() {
	console.Println("\n📋 Next steps:")
	fmt.Printf("1. Review configuration: %s\n", filepath.Join(ConfigDir, "config.env"))

	switch s.initSystem {
	case InitSystemd:
		fmt.Println("2. Start service: sudo systemctl start cloud-update")
		fmt.Println("3. Check status: sudo systemctl status cloud-update")
		fmt.Println("4. View logs: sudo journalctl -u cloud-update -f")
	case InitOpenRC:
		fmt.Println("2. Start service: sudo rc-service cloud-update start")
		fmt.Println("3. Check status: sudo rc-service cloud-update status")
		fmt.Println("4. View logs: tail -f /var/log/cloud-update.log")
	case InitSysVInit:
		fmt.Println("2. Start service: sudo service cloud-update start")
		fmt.Println("3. Check status: sudo service cloud-update status")
		fmt.Println("4. View logs: tail -f /var/log/syslog | grep cloud-update")
	}
}

// Uninstall removes the service from the system.
func (s *ServiceInstaller) Uninstall() error {
	console.Println("🗑️  Uninstalling Cloud Update Service")

	// Stop service
	s.stopService()

	// Disable service
	s.disableService()

	// Remove service files
	s.removeServiceFiles()

	// Remove binary
	if err := s.fs.RemoveAll(InstallDir); err != nil {
		fmt.Printf("  ⚠️  Failed to remove %s: %v\n", InstallDir, err)
	} else {
		fmt.Printf("  ✓ Removed %s\n", InstallDir)
	}

	// Ask about config
	fmt.Printf("\n❓ Remove configuration directory %s? (y/N): ", ConfigDir)
	var response string
	if _, err := s.os.Scanln(&response); err != nil {
		// Default to "no" if scan fails
		response = "n"
	}
	if strings.EqualFold(response, "y") {
		if err := s.fs.RemoveAll(ConfigDir); err != nil {
			fmt.Printf("  ⚠️  Failed to remove %s: %v\n", ConfigDir, err)
		} else {
			fmt.Printf("  ✓ Removed %s\n", ConfigDir)
		}
	}

	console.Println("✅ Uninstall completed")
	return nil
}

func (s *ServiceInstaller) stopService() {
	console.Println("⏹️  Stopping service...")

	switch s.initSystem {
	case InitSystemd:
		if err := s.cmd.Run("systemctl", "stop", "cloud-update"); err != nil {
			fmt.Printf("  ⚠️  Failed to stop service: %v\n", err)
		} else {
			fmt.Println("  ✓ Service stopped")
		}
	case InitOpenRC:
		if err := s.cmd.Run("rc-service", "cloud-update", "stop"); err != nil {
			fmt.Printf("  ⚠️  Failed to stop service: %v\n", err)
		} else {
			fmt.Println("  ✓ Service stopped")
		}
	case InitSysVInit:
		if err := s.cmd.Run("service", "cloud-update", "stop"); err != nil {
			fmt.Printf("  ⚠️  Failed to stop service: %v\n", err)
		} else {
			fmt.Println("  ✓ Service stopped")
		}
	default:
		return
	}
}

func (s *ServiceInstaller) disableService() {
	console.Println("🔌 Disabling service...")

	switch s.initSystem {
	case InitSystemd:
		if err := s.cmd.Run("systemctl", "disable", "cloud-update"); err != nil {
			fmt.Printf("  ⚠️  Failed to disable service: %v\n", err)
		} else {
			fmt.Println("  ✓ Service disabled")
		}
	case InitOpenRC:
		if err := s.cmd.Run("rc-update", "del", "cloud-update"); err != nil {
			fmt.Printf("  ⚠️  Failed to disable service: %v\n", err)
		} else {
			fmt.Println("  ✓ Service disabled")
		}
	case InitSysVInit:
		if _, err := s.cmd.LookPath("update-rc.d"); err == nil {
			if err := s.cmd.Run("update-rc.d", "-f", "cloud-update", "remove"); err != nil {
				fmt.Printf("  ⚠️  Failed to disable service: %v\n", err)
			} else {
				fmt.Println("  ✓ Service disabled")
			}
		}
	}
}

func (s *ServiceInstaller) removeServiceFiles() {
	console.Println("📄 Removing service files...")

	var servicePath string
	switch s.initSystem {
	case InitSystemd:
		servicePath = "/etc/systemd/system/cloud-update.service"
		defer func() {
			if err := s.cmd.Run("systemctl", "daemon-reload"); err != nil {
				fmt.Printf("  ⚠️  Warning: failed to reload systemd: %v\n", err)
			}
		}()
	case InitOpenRC, InitSysVInit:
		servicePath = "/etc/init.d/cloud-update"
	}

	if servicePath != "" {
		if err := s.fs.Remove(servicePath); err != nil {
			fmt.Printf("  ⚠️  Failed to remove %s: %v\n", servicePath, err)
		} else {
			fmt.Printf("  ✓ Removed %s\n", servicePath)
		}
	}
}

func detectDistribution() system.Distribution {
	executor := system.NewSystemExecutor()
	return executor.DetectDistribution()
}

func detectInitSystem() InitSystem {
	fs := RealFileSystem{}
	cmd := RealCommandRunner{}
	return detectInitSystemWithDeps(fs, cmd)
}

func detectInitSystemWithDeps(fs FileSystem, cmd CommandRunner) InitSystem {
	// Check for systemd
	if _, err := fs.Stat("/run/systemd/system"); err == nil {
		return InitSystemd
	}

	// Check for OpenRC
	if _, err := cmd.LookPath("openrc"); err == nil {
		return InitOpenRC
	}

	// Check for Upstart
	if _, err := fs.Stat("/etc/init"); err == nil {
		if _, err := cmd.LookPath("initctl"); err == nil {
			return InitUpstart
		}
	}

	// Check for SysVInit
	if _, err := fs.Stat("/etc/init.d"); err == nil {
		return InitSysVInit
	}

	return InitUnknown
}

func generateSecret() (string, error) {
	cmd := RealCommandRunner{}
	return generateSecretWithDeps(cmd)
}

func generateSecretWithDeps(cmd CommandRunner) (string, error) {
	output, err := cmd.Output("openssl", "rand", "-hex", "32")
	if err == nil && len(output) > 0 {
		trimmed := strings.TrimSpace(string(output))
		// Check if the output is exactly 64 characters (32 bytes in hex)
		if len(trimmed) == 64 {
			return trimmed, nil
		}
	}

	// Fallback to Go's crypto/rand
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure random secret: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}
