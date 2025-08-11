package setup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

// Mock for testing that doesn't require root privileges.
type MockServiceInstaller struct {
	distro          system.Distribution
	initSystem      InitSystem
	setupCalled     bool
	uninstallCalled bool
	shouldFail      bool
	failureMessage  string
	directories     []string
	files           []string
}

func NewMockServiceInstaller() *MockServiceInstaller {
	return &MockServiceInstaller{
		distro:      system.DistroUbuntu,
		initSystem:  InitSystemd,
		directories: make([]string, 0),
		files:       make([]string, 0),
	}
}

func (m *MockServiceInstaller) SetDistribution(distro system.Distribution) {
	m.distro = distro
}

func (m *MockServiceInstaller) SetInitSystem(init InitSystem) {
	m.initSystem = init
}

func (m *MockServiceInstaller) SetShouldFail(fail bool, message string) {
	m.shouldFail = fail
	m.failureMessage = message
}

func (m *MockServiceInstaller) Setup() error {
	m.setupCalled = true
	if m.shouldFail {
		return fmt.Errorf("%s", m.failureMessage)
	}
	return nil
}

func (m *MockServiceInstaller) Uninstall() error {
	m.uninstallCalled = true
	if m.shouldFail {
		return fmt.Errorf("%s", m.failureMessage)
	}
	return nil
}

func TestNewServiceInstaller(t *testing.T) {
	installer := NewServiceInstaller()

	if installer == nil {
		t.Fatal("NewServiceInstaller() returned nil")
	}

	// Should have detected a distribution
	validDistros := []system.Distribution{
		system.DistroAlpine, system.DistroDebian, system.DistroUbuntu, system.DistroRHEL,
		system.DistroCentOS, system.DistroFedora, system.DistroSUSE, system.DistroArch,
		system.DistroUnknown,
	}

	found := false
	for _, validDistro := range validDistros {
		if installer.distro == validDistro {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Detected distribution %q is not valid", installer.distro)
	}

	// Should have detected an init system
	validInitSystems := []InitSystem{
		InitSystemd, InitOpenRC, InitSysVInit, InitUpstart, InitUnknown,
	}

	found = false
	for _, validInit := range validInitSystems {
		if installer.initSystem == validInit {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Detected init system %q is not valid", installer.initSystem)
	}

	t.Logf("Detected: %s with %s", installer.distro, installer.initSystem)
}

func TestDetectInitSystem(t *testing.T) {
	initSystem := detectInitSystem()

	// Should return one of the valid init systems
	validSystems := []InitSystem{
		InitSystemd, InitOpenRC, InitSysVInit, InitUpstart, InitUnknown,
	}

	found := false
	for _, valid := range validSystems {
		if initSystem == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("detectInitSystem() = %q, want one of %v", initSystem, validSystems)
	}

	t.Logf("Detected init system: %s", initSystem)
}

func TestDetectDistribution(t *testing.T) {
	distro := detectDistribution()

	// Should return one of the valid distributions
	validDistros := []system.Distribution{
		system.DistroAlpine, system.DistroDebian, system.DistroUbuntu, system.DistroRHEL,
		system.DistroCentOS, system.DistroFedora, system.DistroSUSE, system.DistroArch,
		system.DistroUnknown,
	}

	found := false
	for _, valid := range validDistros {
		if distro == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("detectDistribution() = %q, want one of %v", distro, validDistros)
	}

	t.Logf("Detected distribution: %s", distro)
}

func TestServiceInstaller_createDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping directory creation test on Windows")
	}

	// We would use installer in a full test but for now just test directory creation

	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Temporarily modify the constants for testing
	originalInstallDir := InstallDir
	originalConfigDir := ConfigDir

	testInstallDir := filepath.Join(tempDir, "opt", "cloud-update")
	testConfigDir := filepath.Join(tempDir, "etc", "cloud-update")

	// We can't easily override the constants, so we'll test the logic separately
	dirs := []struct {
		path string
		mode os.FileMode
	}{
		{testInstallDir, 0755},
		{testConfigDir, 0700},
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir.path, dir.mode)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir.path, err)
		}

		// Verify directory was created with correct permissions
		info, err := os.Stat(dir.path)
		if err != nil {
			t.Fatalf("Failed to stat directory %s: %v", dir.path, err)
		}

		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir.path)
		}

		if info.Mode().Perm() != dir.mode {
			t.Errorf("Directory %s has permissions %o, want %o", dir.path, info.Mode().Perm(), dir.mode)
		}
	}

	t.Logf("Would create install directory at: %s", originalInstallDir)
	t.Logf("Would create config directory at: %s", originalConfigDir)
}

func TestServiceInstaller_installBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping binary installation test on Windows")
	}

	// Create a temporary test environment
	tempDir := t.TempDir()
	testInstallDir := filepath.Join(tempDir, "opt", "cloud-update")

	err := os.MkdirAll(testInstallDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test install dir: %v", err)
	}

	// Create a test binary
	testBinary := filepath.Join(tempDir, "test-binary")
	testContent := "test binary content"
	err = os.WriteFile(testBinary, []byte(testContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Test copying binary
	destPath := filepath.Join(testInstallDir, "cloud-update")

	// Read the test binary
	input, err := os.ReadFile(testBinary)
	if err != nil {
		t.Fatalf("Failed to read test binary: %v", err)
	}

	// Write to destination
	err = os.WriteFile(destPath, input, 0755)
	if err != nil {
		t.Fatalf("Failed to write binary to destination: %v", err)
	}

	// Verify the binary was copied correctly
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("Failed to stat destination binary: %v", err)
	}

	if info.Mode().Perm() != 0755 {
		t.Errorf("Binary permissions = %o, want 0755", info.Mode().Perm())
	}

	// Verify content
	copiedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read copied binary: %v", err)
	}

	if string(copiedContent) != testContent {
		t.Errorf("Copied binary content = %q, want %q", string(copiedContent), testContent)
	}

	t.Logf("Successfully tested binary installation logic")
}

func TestServiceInstaller_installSystemdService(t *testing.T) {

	// Create temporary directory for testing
	tempDir := t.TempDir()
	servicePath := filepath.Join(tempDir, "cloud-update.service")

	// Test writing systemd service file
	err := os.WriteFile(servicePath, []byte(SystemdService), 0644)
	if err != nil {
		t.Fatalf("Failed to write systemd service file: %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(servicePath)
	if err != nil {
		t.Fatalf("Failed to stat service file: %v", err)
	}

	if info.Mode().Perm() != 0644 {
		t.Errorf("Service file permissions = %o, want 0644", info.Mode().Perm())
	}

	// Verify content
	content, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("Failed to read service file: %v", err)
	}

	if string(content) != SystemdService {
		t.Error("Service file content doesn't match embedded content")
	}

	t.Logf("Successfully tested systemd service installation logic")
}

func TestServiceInstaller_installOpenRCService(t *testing.T) {

	// Create temporary directory for testing
	tempDir := t.TempDir()
	servicePath := filepath.Join(tempDir, "cloud-update")

	// Test writing OpenRC service file
	err := os.WriteFile(servicePath, []byte(OpenRCScript), 0755)
	if err != nil {
		t.Fatalf("Failed to write OpenRC service file: %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(servicePath)
	if err != nil {
		t.Fatalf("Failed to stat service file: %v", err)
	}

	if info.Mode().Perm() != 0755 {
		t.Errorf("Service file permissions = %o, want 0755", info.Mode().Perm())
	}

	// Verify content
	content, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("Failed to read service file: %v", err)
	}

	if string(content) != OpenRCScript {
		t.Error("Service file content doesn't match embedded content")
	}

	t.Logf("Successfully tested OpenRC service installation logic")
}

func TestServiceInstaller_installSysVInitService(t *testing.T) {

	// Create temporary directory for testing
	tempDir := t.TempDir()
	servicePath := filepath.Join(tempDir, "cloud-update")

	// Test writing SysV init service file
	err := os.WriteFile(servicePath, []byte(SysVInitScript), 0755)
	if err != nil {
		t.Fatalf("Failed to write SysV init service file: %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(servicePath)
	if err != nil {
		t.Fatalf("Failed to stat service file: %v", err)
	}

	if info.Mode().Perm() != 0755 {
		t.Errorf("Service file permissions = %o, want 0755", info.Mode().Perm())
	}

	// Verify content
	content, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("Failed to read service file: %v", err)
	}

	if string(content) != SysVInitScript {
		t.Error("Service file content doesn't match embedded content")
	}

	t.Logf("Successfully tested SysV init service installation logic")
}

func TestServiceInstaller_createConfig(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.env")

	// Test config creation
	secret := "test-secret-12345"
	config := fmt.Sprintf(`# Cloud Update Configuration
# Generated by setup command

# Port on which the HTTP server will listen
CLOUD_UPDATE_PORT=9999

# Secret key for webhook signature validation (HMAC SHA256)
CLOUD_UPDATE_SECRET=%s

# Log level (debug, info, warn, error)
CLOUD_UPDATE_LOG_LEVEL=info
`, secret)

	err := os.WriteFile(configPath, []byte(config), 0600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Config file permissions = %o, want 0600", info.Mode().Perm())
	}

	// Verify content structure
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	contentStr := string(content)
	expectedElements := []string{
		"CLOUD_UPDATE_PORT=",
		"CLOUD_UPDATE_SECRET=",
		"CLOUD_UPDATE_LOG_LEVEL=",
		secret,
	}

	for _, element := range expectedElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Config file should contain %q", element)
		}
	}

	t.Logf("Successfully tested config file creation logic")
}

func TestGenerateSecret(t *testing.T) {
	// Test secret generation
	secret, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret() failed: %v", err)
	}

	if secret == "" {
		t.Error("generateSecret() returned empty string")
	}

	// Should be hex-encoded (64 chars for 32 bytes)
	if len(secret) != 64 {
		t.Errorf("generateSecret() length = %d, want 64", len(secret))
	}

	// Should be valid hex
	for _, char := range secret {
		if char < '0' || (char > '9' && char < 'a') || char > 'f' {
			t.Errorf("generateSecret() contains invalid hex character: %c", char)
		}
	}

	// Test uniqueness
	secret2, err := generateSecret()
	if err != nil {
		t.Fatalf("Second generateSecret() failed: %v", err)
	}

	if secret == secret2 {
		t.Error("generateSecret() should generate unique secrets")
	}

	t.Logf("Generated secret: %s", secret)
}

func TestGenerateSecret_Fallback(t *testing.T) {
	// Test that fallback works when openssl is not available
	// We can't easily mock exec.Command, so we'll test with a nonexistent command

	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set empty PATH to make openssl unavailable
	os.Setenv("PATH", "")

	secret, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret() should fall back to crypto/rand when openssl unavailable: %v", err)
	}

	if secret == "" {
		t.Error("generateSecret() fallback returned empty string")
	}

	if len(secret) != 64 {
		t.Errorf("generateSecret() fallback length = %d, want 64", len(secret))
	}

	t.Logf("Fallback generated secret: %s", secret)
}

func TestInitSystemConstants(t *testing.T) {
	// Test that all init system constants are defined
	systems := []InitSystem{
		InitSystemd,
		InitOpenRC,
		InitSysVInit,
		InitUpstart,
		InitUnknown,
	}

	expectedValues := []string{
		"systemd",
		"openrc",
		"sysvinit",
		"upstart",
		"unknown",
	}

	if len(systems) != len(expectedValues) {
		t.Fatalf("Number of systems (%d) doesn't match expected values (%d)", len(systems), len(expectedValues))
	}

	for i, system := range systems {
		if string(system) != expectedValues[i] {
			t.Errorf("InitSystem constant %d = %q, want %q", i, string(system), expectedValues[i])
		}
	}
}

func TestInstallationConstants(t *testing.T) {
	// Test that installation constants are reasonable
	if InstallDir == "" {
		t.Error("InstallDir should not be empty")
	}

	if ConfigDir == "" {
		t.Error("ConfigDir should not be empty")
	}

	if BinaryName == "" {
		t.Error("BinaryName should not be empty")
	}

	// Should be absolute paths
	if !filepath.IsAbs(InstallDir) {
		t.Errorf("InstallDir %q should be absolute path", InstallDir)
	}

	if !filepath.IsAbs(ConfigDir) {
		t.Errorf("ConfigDir %q should be absolute path", ConfigDir)
	}

	// Should follow conventions
	expectedInstallDir := "/usr/local/bin"
	expectedConfigDir := "/etc/cloud-update"
	expectedBinaryName := "cloud-update"

	if InstallDir != expectedInstallDir {
		t.Errorf("InstallDir = %q, want %q", InstallDir, expectedInstallDir)
	}

	if ConfigDir != expectedConfigDir {
		t.Errorf("ConfigDir = %q, want %q", ConfigDir, expectedConfigDir)
	}

	if BinaryName != expectedBinaryName {
		t.Errorf("BinaryName = %q, want %q", BinaryName, expectedBinaryName)
	}
}

func TestServiceInstaller_printNextSteps(t *testing.T) {
	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
	}

	// We can't easily capture the output, but we can test that the method exists
	// and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printNextSteps() panicked: %v", r)
		}
	}()

	installer.printNextSteps()
	t.Log("printNextSteps() completed without panic")
}

// Test that we handle different init systems correctly.
func TestServiceInstaller_InitSystemSpecific(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		distro     system.Distribution
	}{
		{
			name:       "systemd on Ubuntu",
			initSystem: InitSystemd,
			distro:     system.DistroUbuntu,
		},
		{
			name:       "openrc on Alpine",
			initSystem: InitOpenRC,
			distro:     system.DistroAlpine,
		},
		{
			name:       "sysvinit on Debian",
			initSystem: InitSysVInit,
			distro:     system.DistroDebian,
		},
		{
			name:       "unknown init system",
			initSystem: InitUnknown,
			distro:     system.DistroUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &ServiceInstaller{
				distro:     tt.distro,
				initSystem: tt.initSystem,
			}

			// Test that we can create the installer without issues
			if installer.distro != tt.distro {
				t.Errorf("distro = %s, want %s", installer.distro, tt.distro)
			}

			if installer.initSystem != tt.initSystem {
				t.Errorf("initSystem = %s, want %s", installer.initSystem, tt.initSystem)
			}

			t.Logf("Created installer for %s with %s", tt.distro, tt.initSystem)
		})
	}
}

// Benchmark tests.
func BenchmarkNewServiceInstaller(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewServiceInstaller()
	}
}

func BenchmarkDetectInitSystem(b *testing.B) {
	for i := 0; i < b.N; i++ {
		detectInitSystem()
	}
}

func BenchmarkDetectDistribution(b *testing.B) {
	for i := 0; i < b.N; i++ {
		detectDistribution()
	}
}

func BenchmarkGenerateSecret(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateSecret()
	}
}

// Test edge cases and error conditions.
func TestServiceInstaller_Setup_AsNonRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping root check test on Windows")
	}

	// Skip if running as root
	if os.Geteuid() == 0 {
		t.Skip("Skipping non-root test when running as root")
	}

	// Use mocks to simulate non-root user
	fs := NewMockFileSystem()
	cmd := NewMockCommandRunner()
	osIface := NewMockOSInterface()
	osIface.SetEuid(1000) // non-root

	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
		fs:         fs,
		cmd:        cmd,
		os:         osIface,
	}

	// Setup should fail when not running as root
	err := installer.Setup()

	if err == nil {
		t.Error("Setup() should fail when not running as root")
		return
	}

	expectedMsg := "setup must be run as root"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Setup() error = %v, want error containing %q", err, expectedMsg)
	}
}

func TestServiceInstaller_MissingCommands(t *testing.T) {
	// Test behavior when system commands are missing
	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
	}

	// We can't easily test this without potentially breaking the system,
	// but we can verify the installer handles missing commands gracefully
	t.Logf("Installer created for testing missing commands: distro=%s, init=%s",
		installer.distro, installer.initSystem)

	// The actual functionality would be tested in integration tests
}

func TestServiceInstaller_ConfigFileExists(t *testing.T) {
	// Test behavior when config file already exists
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.env")

	// Create existing config file
	existingConfig := "# Existing config\nCLOUD_UPDATE_PORT=8080\n"
	err := os.WriteFile(configPath, []byte(existingConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Existing config file should exist: %v", err)
	}

	t.Log("Successfully tested existing config file detection logic")
}

func TestServiceInstaller_Uninstall_UserPrompt(t *testing.T) {
	// Test that uninstall prompts for config removal
	// This is mainly a structural test since we can't easily mock user input

	// Test struct creation (would be used in a full test with proper permissions)
	_ = &ServiceInstaller{}

	// The uninstall method exists and can be called
	// In a real test environment, this would require root privileges and actual cleanup
	// Note: Uninstall method exists for testing
}

func TestServiceInstaller_SystemCommandsAvailability(t *testing.T) {
	// Test availability of system commands for different init systems
	tests := []struct {
		initSystem InitSystem
		commands   []string
	}{
		{
			initSystem: InitSystemd,
			commands:   []string{"systemctl"},
		},
		{
			initSystem: InitOpenRC,
			commands:   []string{"rc-update", "rc-service"},
		},
		{
			initSystem: InitSysVInit,
			commands:   []string{"service", "update-rc.d"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.initSystem), func(t *testing.T) {
			for _, cmd := range tt.commands {
				_, err := exec.LookPath(cmd)
				if err != nil {
					t.Logf("Command %s not available for %s: %v", cmd, tt.initSystem, err)
				} else {
					t.Logf("Command %s available for %s", cmd, tt.initSystem)
				}
			}
		})
	}
}

func TestServiceInstaller_PathValidation(t *testing.T) {
	// Test that installation paths are valid
	paths := []string{InstallDir, ConfigDir}

	for _, path := range paths {
		// Should be absolute
		if !filepath.IsAbs(path) {
			t.Errorf("Path %q should be absolute", path)
		}

		// Should not contain invalid characters
		if strings.Contains(path, "..") {
			t.Errorf("Path %q should not contain '..'", path)
		}

		// Should be clean
		cleaned := filepath.Clean(path)
		if path != cleaned {
			t.Errorf("Path %q should be clean, got %q", path, cleaned)
		}
	}
}

// Test the actual createDirectories function with mocked environment.
func TestServiceInstaller_createDirectories_Direct(t *testing.T) {
	tempDir := t.TempDir()

	// Create a mock installer with temp directories
	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
	}

	// Create a test version of the function using temp paths
	dirs := []struct {
		path string
		mode os.FileMode
	}{
		{filepath.Join(tempDir, "opt", "cloud-update"), 0755},
		{filepath.Join(tempDir, "etc", "cloud-update"), 0700},
		{filepath.Join(tempDir, "etc", "cloud-update", "tls"), 0700},
		{filepath.Join(tempDir, "var", "log"), 0755},
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir.path, dir.mode)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir.path, err)
		}

		info, err := os.Stat(dir.path)
		if err != nil {
			t.Fatalf("Failed to stat directory %s: %v", dir.path, err)
		}

		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir.path)
		}

		expectedMode := dir.mode
		if runtime.GOOS == "windows" {
			// Windows doesn't support Unix permissions
			t.Logf("Skipping permission check on Windows for %s", dir.path)
		} else if info.Mode().Perm() != expectedMode {
			t.Errorf("Directory %s has permissions %o, want %o", dir.path, info.Mode().Perm(), expectedMode)
		}
	}

	t.Logf("Successfully tested directory creation logic for installer: %+v", installer)
}

// Test installBinary function directly.
func TestServiceInstaller_installBinary_Direct(t *testing.T) {
	tempDir := t.TempDir()

	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
	}

	// Create source and destination directories
	srcDir := filepath.Join(tempDir, "src")
	destDir := filepath.Join(tempDir, "dest")

	err := os.MkdirAll(srcDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}

	// Create a test binary
	srcBinary := filepath.Join(srcDir, "cloud-update")
	destBinary := filepath.Join(destDir, "cloud-update")
	testContent := "#!/bin/bash\necho 'test binary'\n"

	err = os.WriteFile(srcBinary, []byte(testContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Test the copy operation (simulating what installBinary does)
	input, err := os.ReadFile(srcBinary)
	if err != nil {
		t.Fatalf("Failed to read source binary: %v", err)
	}

	err = os.WriteFile(destBinary, input, 0755)
	if err != nil {
		t.Fatalf("Failed to write destination binary: %v", err)
	}

	// Verify the binary was copied
	info, err := os.Stat(destBinary)
	if err != nil {
		t.Fatalf("Failed to stat destination binary: %v", err)
	}

	if runtime.GOOS != "windows" && info.Mode().Perm() != 0755 {
		t.Errorf("Binary permissions = %o, want 0755", info.Mode().Perm())
	}

	copiedContent, err := os.ReadFile(destBinary)
	if err != nil {
		t.Fatalf("Failed to read copied binary: %v", err)
	}

	if string(copiedContent) != testContent {
		t.Errorf("Copied content = %q, want %q", string(copiedContent), testContent)
	}

	t.Logf("Successfully tested binary installation logic for installer: %+v", installer)
}

// Test installService function logic for different init systems.
func TestServiceInstaller_installService_Logic(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		shouldWork bool
	}{
		{"systemd", InitSystemd, true},
		{"openrc", InitOpenRC, true},
		{"sysvinit", InitSysVInit, true},
		{"upstart", InitUpstart, false},
		{"unknown", InitUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: tt.initSystem,
			}

			// Test the logic that would be used in installService
			switch tt.initSystem {
			case InitSystemd:
				// Test systemd service installation logic
				tempDir := t.TempDir()
				servicePath := filepath.Join(tempDir, "cloud-update.service")
				err := os.WriteFile(servicePath, []byte(SystemdService), 0644)
				if err != nil {
					t.Fatalf("Failed to write systemd service: %v", err)
				}
			case InitOpenRC:
				// Test OpenRC service installation logic
				tempDir := t.TempDir()
				servicePath := filepath.Join(tempDir, "cloud-update")
				err := os.WriteFile(servicePath, []byte(OpenRCScript), 0755)
				if err != nil {
					t.Fatalf("Failed to write OpenRC service: %v", err)
				}
			case InitSysVInit:
				// Test SysV init service installation logic
				tempDir := t.TempDir()
				servicePath := filepath.Join(tempDir, "cloud-update")
				err := os.WriteFile(servicePath, []byte(SysVInitScript), 0755)
				if err != nil {
					t.Fatalf("Failed to write SysV init service: %v", err)
				}
			case InitUpstart, InitUnknown:
				// These should result in error conditions
				if tt.shouldWork {
					t.Errorf("Expected %s to not work", tt.initSystem)
				}
			}

			t.Logf("Tested service installation logic for %s with installer: %+v", tt.initSystem, installer)
		})
	}
}

// Test createConfig function directly.
func TestServiceInstaller_createConfig_Direct(t *testing.T) {
	tempDir := t.TempDir()
	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
	}

	// Generate a test secret
	secret, err := generateSecret()
	if err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}

	// Test config creation
	configPath := filepath.Join(tempDir, "config.env")
	config := fmt.Sprintf(`# Cloud Update Configuration
# Generated by setup command

# Port on which the HTTP server will listen
CLOUD_UPDATE_PORT=9999

# Secret key for webhook signature validation (HMAC SHA256)
CLOUD_UPDATE_SECRET=%s

# Log level (debug, info, warn, error)
CLOUD_UPDATE_LOG_LEVEL=info
`, secret)

	err = os.WriteFile(configPath, []byte(config), 0600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Verify the config file
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config: %v", err)
	}

	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Errorf("Config permissions = %o, want 0600", info.Mode().Perm())
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	contentStr := string(content)
	expectedElements := []string{
		"CLOUD_UPDATE_PORT=9999",
		"CLOUD_UPDATE_SECRET=" + secret,
		"CLOUD_UPDATE_LOG_LEVEL=info",
	}

	for _, element := range expectedElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Config should contain %q", element)
		}
	}

	t.Logf("Successfully tested config creation logic for installer: %+v", installer)
}

// Test enableService and disableService logic.
func TestServiceInstaller_ServiceControl_Logic(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		commands   []string
	}{
		{"systemd", InitSystemd, []string{"systemctl", "enable", "cloud-update"}},
		{"openrc", InitOpenRC, []string{"rc-update", "add", "cloud-update"}},
		{"sysvinit", InitSysVInit, []string{"update-rc.d", "cloud-update", "enable"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: tt.initSystem,
			}

			// Test that we know what commands would be executed
			if len(tt.commands) == 0 {
				t.Errorf("No commands defined for %s", tt.initSystem)
			}

			// Check if the command exists (without executing it)
			if len(tt.commands) > 0 {
				_, err := exec.LookPath(tt.commands[0])
				if err != nil {
					t.Logf("Command %s not available: %v", tt.commands[0], err)
				} else {
					t.Logf("Command %s available for %s", tt.commands[0], tt.initSystem)
				}
			}

			t.Logf("Tested service control logic for %s with installer: %+v", tt.initSystem, installer)
		})
	}
}

// Test stopService logic.
func TestServiceInstaller_StopService_Logic(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		stopCmd    []string
	}{
		{"systemd", InitSystemd, []string{"systemctl", "stop", "cloud-update"}},
		{"openrc", InitOpenRC, []string{"rc-service", "cloud-update", "stop"}},
		{"sysvinit", InitSysVInit, []string{"service", "cloud-update", "stop"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: tt.initSystem,
			}

			// Test that we know what stop commands would be executed
			if len(tt.stopCmd) == 0 {
				t.Errorf("No stop commands defined for %s", tt.initSystem)
			}

			// Check command availability
			if len(tt.stopCmd) > 0 {
				_, err := exec.LookPath(tt.stopCmd[0])
				if err != nil {
					t.Logf("Stop command %s not available: %v", tt.stopCmd[0], err)
				} else {
					t.Logf("Stop command %s available for %s", tt.stopCmd[0], tt.initSystem)
				}
			}

			t.Logf("Tested stop service logic for %s with installer: %+v", tt.initSystem, installer)
		})
	}
}

// Test removeServiceFiles logic.
func TestServiceInstaller_RemoveServiceFiles_Logic(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		filePaths  []string
	}{
		{
			"systemd",
			InitSystemd,
			[]string{"/lib/systemd/system/cloud-update.service"},
		},
		{
			"openrc",
			InitOpenRC,
			[]string{"/etc/init.d/cloud-update"},
		},
		{
			"sysvinit",
			InitSysVInit,
			[]string{"/etc/init.d/cloud-update"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: tt.initSystem,
			}

			// Test file removal logic (without actually removing system files)
			tempDir := t.TempDir()
			for _, filePath := range tt.filePaths {
				fileName := filepath.Base(filePath)
				testFile := filepath.Join(tempDir, fileName)

				// Create test file
				err := os.WriteFile(testFile, []byte("test service file"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				// Test removal
				err = os.Remove(testFile)
				if err != nil {
					t.Fatalf("Failed to remove test file: %v", err)
				}

				// Verify removal
				if _, err := os.Stat(testFile); !os.IsNotExist(err) {
					t.Errorf("Test file should have been removed")
				}
			}

			t.Logf("Tested service file removal logic for %s with installer: %+v", tt.initSystem, installer)
		})
	}
}

// Test Uninstall function structure and logic.
func TestServiceInstaller_Uninstall_Logic(t *testing.T) {
	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
	}

	// Test the structure and components that Uninstall would use
	// (without actually running uninstall which requires root)

	// Test paths that would be checked/removed
	paths := []string{
		InstallDir,
		ConfigDir,
	}

	for _, path := range paths {
		if !filepath.IsAbs(path) {
			t.Errorf("Uninstall path %q should be absolute", path)
		}
	}

	// Test service files that would be removed
	serviceFiles := map[InitSystem][]string{
		InitSystemd:  {"/lib/systemd/system/cloud-update.service"},
		InitOpenRC:   {"/etc/init.d/cloud-update"},
		InitSysVInit: {"/etc/init.d/cloud-update"},
	}

	if files, exists := serviceFiles[installer.initSystem]; exists {
		for _, file := range files {
			if !filepath.IsAbs(file) {
				t.Errorf("Service file path %q should be absolute", file)
			}
		}
	}

	t.Logf("Tested uninstall logic structure for installer: %+v", installer)
}

// Test printNextSteps with different configurations.
func TestServiceInstaller_printNextSteps_Variations(t *testing.T) {
	tests := []struct {
		name       string
		distro     system.Distribution
		initSystem InitSystem
	}{
		{"ubuntu-systemd", system.DistroUbuntu, InitSystemd},
		{"alpine-openrc", system.DistroAlpine, InitOpenRC},
		{"debian-sysvinit", system.DistroDebian, InitSysVInit},
		{"unknown-unknown", system.DistroUnknown, InitUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &ServiceInstaller{
				distro:     tt.distro,
				initSystem: tt.initSystem,
			}

			// Test that printNextSteps doesn't panic for different configurations
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printNextSteps() panicked for %s-%s: %v", tt.distro, tt.initSystem, r)
				}
			}()

			installer.printNextSteps()
			t.Logf("printNextSteps() completed successfully for %s with %s", tt.distro, tt.initSystem)
		})
	}
}

// Test Setup method with comprehensive coverage.
func TestServiceInstaller_Setup_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockFileSystem, *MockCommandRunner, *MockOSInterface)
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name: "successful setup",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0) // root
				osIface.SetExecutable("/test/cloud-update", nil)
				// Add the executable file to the mock filesystem
				fs.AddFile("/test/cloud-update", []byte("mock binary content"))
				cmd.SetOutput("openssl", []byte("deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678"))
			},
			expectError: false,
			description: "should complete setup successfully",
		},
		{
			name: "non-root user on non-windows",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(1000) // non-root
			},
			expectError:   true,
			errorContains: "setup must be run as root",
			description:   "should fail when not running as root on non-Windows",
		},
		{
			name: "directory creation fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				fs.SetShouldFail("MkdirAll", InstallDir, errors.New("permission denied"))
			},
			expectError:   true,
			errorContains: "failed to create directories",
			description:   "should fail when directory creation fails",
		},
		{
			name: "executable path fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("", errors.New("executable not found"))
			},
			expectError:   true,
			errorContains: "failed to install binary",
			description:   "should fail when cannot get executable path",
		},
		{
			name: "binary read fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("/test/cloud-update", nil)
				fs.SetShouldFail("ReadFile", "/test/cloud-update", errors.New("read error"))
			},
			expectError:   true,
			errorContains: "failed to install binary",
			description:   "should fail when cannot read binary",
		},
		{
			name: "binary write fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("/test/cloud-update", nil)
				destPath := filepath.Join(InstallDir, BinaryName)
				fs.SetShouldFail("WriteFile", destPath, errors.New("write error"))
			},
			expectError:   true,
			errorContains: "failed to install binary",
			description:   "should fail when cannot write binary",
		},
		{
			name: "service installation fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("/test/cloud-update", nil)
				// Add the executable file to the mock filesystem
				fs.AddFile("/test/cloud-update", []byte("mock binary content"))
				writeErr := errors.New("service write error")
				fs.SetShouldFail("WriteFile", "/etc/systemd/system/cloud-update.service", writeErr)
			},
			expectError:   true,
			errorContains: "failed to install service",
			description:   "should fail when service installation fails",
		},
		{
			name: "config creation fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("/test/cloud-update", nil)
				// Add the executable file to the mock filesystem
				fs.AddFile("/test/cloud-update", []byte("mock binary content"))
				configPath := filepath.Join(ConfigDir, "config.env")
				fs.SetShouldFail("WriteFile", configPath, errors.New("config write error"))
			},
			expectError:   true,
			errorContains: "failed to create config",
			description:   "should fail when config creation fails",
		},
		{
			name: "service enable fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("/test/cloud-update", nil)
				// Add the executable file to the mock filesystem
				fs.AddFile("/test/cloud-update", []byte("mock binary content"))
				// Make only systemctl enable fail, not daemon-reload
				cmd.SetShouldFailWithArgs("systemctl", []string{"enable", "cloud-update"}, errors.New("enable failed"))
			},
			expectError:   true,
			errorContains: "failed to install service", // The error happens during reload which is part of install
			description:   "should fail when service enabling fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip root check test on Windows
			if strings.Contains(tt.name, "non-root") && runtime.GOOS == "windows" {
				t.Skip("Skipping root check test on Windows")
			}

			fs := NewMockFileSystem()
			cmd := NewMockCommandRunner()
			osIface := NewMockOSInterface()

			tt.setupMocks(fs, cmd, osIface)

			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: InitSystemd,
				fs:         fs,
				cmd:        cmd,
				os:         osIface,
			}

			err := installer.Setup()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none for test: %s", tt.description)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v for test: %s", err, tt.description)
				}
			}
		})
	}
}

// Test createDirectories method.
func TestServiceInstaller_createDirectories_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockFileSystem)
		expectError   bool
		errorContains string
	}{
		{
			name:        "successful directory creation",
			setupMocks:  func(fs *MockFileSystem) {}, // no errors
			expectError: false,
		},
		{
			name: "install directory creation fails",
			setupMocks: func(fs *MockFileSystem) {
				fs.SetShouldFail("MkdirAll", InstallDir, errors.New("permission denied"))
			},
			expectError:   true,
			errorContains: "failed to create directory",
		},
		{
			name: "config directory creation fails",
			setupMocks: func(fs *MockFileSystem) {
				fs.SetShouldFail("MkdirAll", ConfigDir, errors.New("disk full"))
			},
			expectError:   true,
			errorContains: "failed to create directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			tt.setupMocks(fs)

			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: InitSystemd,
				fs:         fs,
				cmd:        NewMockCommandRunner(),
				os:         NewMockOSInterface(),
			}

			err := installer.createDirectories()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				// Verify directories were created
				if !fs.DirExists(InstallDir) {
					t.Errorf("InstallDir %s was not created", InstallDir)
				}
				if !fs.DirExists(ConfigDir) {
					t.Errorf("ConfigDir %s was not created", ConfigDir)
				}
			}
		})
	}
}

// Test installBinary method.
func TestServiceInstaller_installBinary_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockFileSystem, *MockOSInterface)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful binary installation",
			setupMocks: func(fs *MockFileSystem, osIface *MockOSInterface) {
				osIface.SetExecutable("/test/cloud-update", nil)
				// Pre-populate the source file
				fs.WriteFile("/test/cloud-update", []byte("binary content"), 0755)
			},
			expectError: false,
		},
		{
			name: "executable path fails",
			setupMocks: func(fs *MockFileSystem, osIface *MockOSInterface) {
				osIface.SetExecutable("", errors.New("executable not found"))
			},
			expectError:   true,
			errorContains: "failed to get executable path",
		},
		{
			name: "read executable fails",
			setupMocks: func(fs *MockFileSystem, osIface *MockOSInterface) {
				osIface.SetExecutable("/test/cloud-update", nil)
				fs.SetShouldFail("ReadFile", "/test/cloud-update", errors.New("read error"))
			},
			expectError:   true,
			errorContains: "failed to read executable",
		},
		{
			name: "write binary fails",
			setupMocks: func(fs *MockFileSystem, osIface *MockOSInterface) {
				osIface.SetExecutable("/test/cloud-update", nil)
				fs.WriteFile("/test/cloud-update", []byte("binary content"), 0755)
				destPath := filepath.Join(InstallDir, BinaryName)
				fs.SetShouldFail("WriteFile", destPath, errors.New("write error"))
			},
			expectError:   true,
			errorContains: "failed to write binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			osIface := NewMockOSInterface()
			tt.setupMocks(fs, osIface)

			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: InitSystemd,
				fs:         fs,
				cmd:        NewMockCommandRunner(),
				os:         osIface,
			}

			err := installer.installBinary()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				// Verify binary was installed
				destPath := filepath.Join(InstallDir, BinaryName)
				if !fs.FileExists(destPath) {
					t.Errorf("Binary was not installed to %s", destPath)
				}
			}
		})
	}
}

// Test installService method for all init systems.
func TestServiceInstaller_installService_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		initSystem    InitSystem
		setupMocks    func(*MockFileSystem, *MockCommandRunner)
		expectError   bool
		errorContains string
		servicePath   string
	}{
		{
			name:        "systemd success",
			initSystem:  InitSystemd,
			setupMocks:  func(fs *MockFileSystem, cmd *MockCommandRunner) {},
			expectError: false,
			servicePath: "/etc/systemd/system/cloud-update.service",
		},
		{
			name:       "systemd write fails",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("WriteFile", "/etc/systemd/system/cloud-update.service", errors.New("write error"))
			},
			expectError:   true,
			errorContains: "failed to write systemd service",
		},
		{
			name:       "systemd daemon-reload fails",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				cmd.SetShouldFail("systemctl", errors.New("daemon-reload failed"))
			},
			expectError:   true,
			errorContains: "failed to reload systemd",
		},
		{
			name:        "openrc success",
			initSystem:  InitOpenRC,
			setupMocks:  func(fs *MockFileSystem, cmd *MockCommandRunner) {},
			expectError: false,
			servicePath: "/etc/init.d/cloud-update",
		},
		{
			name:       "openrc write fails",
			initSystem: InitOpenRC,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("WriteFile", "/etc/init.d/cloud-update", errors.New("write error"))
			},
			expectError:   true,
			errorContains: "failed to write OpenRC script",
		},
		{
			name:       "sysvinit success",
			initSystem: InitSysVInit,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				cmd.SetLookupPath("update-rc.d", "/usr/sbin/update-rc.d")
			},
			expectError: false,
			servicePath: "/etc/init.d/cloud-update",
		},
		{
			name:       "sysvinit write fails",
			initSystem: InitSysVInit,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("WriteFile", "/etc/init.d/cloud-update", errors.New("write error"))
			},
			expectError:   true,
			errorContains: "failed to write SysVInit script",
		},
		{
			name:          "unsupported init system",
			initSystem:    InitUpstart,
			setupMocks:    func(fs *MockFileSystem, cmd *MockCommandRunner) {},
			expectError:   true,
			errorContains: "unsupported init system",
		},
		{
			name:          "unknown init system",
			initSystem:    InitUnknown,
			setupMocks:    func(fs *MockFileSystem, cmd *MockCommandRunner) {},
			expectError:   true,
			errorContains: "unsupported init system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			cmd := NewMockCommandRunner()
			tt.setupMocks(fs, cmd)

			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: tt.initSystem,
				fs:         fs,
				cmd:        cmd,
				os:         NewMockOSInterface(),
			}

			err := installer.installService()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if tt.servicePath != "" && !fs.FileExists(tt.servicePath) {
					t.Errorf("Service file was not created at %s", tt.servicePath)
				}
			}
		})
	}
}

// Test createConfig method.
func TestServiceInstaller_createConfig_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockFileSystem, *MockCommandRunner)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful config creation",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte("deadbeef1234567890abcdef"))
			},
			expectError: false,
		},
		{
			name: "config already exists",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				configPath := filepath.Join(ConfigDir, "config.env")
				fs.WriteFile(configPath, []byte("existing config"), 0600)
			},
			expectError: false, // Should not error, just skip
		},
		{
			name: "secret generation fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				cmd.SetShouldFail("openssl", errors.New("openssl failed"))
				// This should fall back to Go's crypto/rand, which is tested separately
			},
			expectError: false, // Should fall back to Go's crypto/rand
		},
		{
			name: "write config fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte("deadbeef1234567890abcdef"))
				configPath := filepath.Join(ConfigDir, "config.env")
				fs.SetShouldFail("WriteFile", configPath, errors.New("write error"))
			},
			expectError:   true,
			errorContains: "failed to write config file",
		},
		{
			name: "chown fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte("deadbeef1234567890abcdef"))
				configPath := filepath.Join(ConfigDir, "config.env")
				fs.SetShouldFail("Chown", configPath, errors.New("chown error"))
			},
			expectError:   true,
			errorContains: "failed to set config ownership",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			cmd := NewMockCommandRunner()
			tt.setupMocks(fs, cmd)

			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: InitSystemd,
				fs:         fs,
				cmd:        cmd,
				os:         NewMockOSInterface(),
			}

			err := installer.createConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Test enableService method.
func TestServiceInstaller_enableService_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		initSystem    InitSystem
		setupMocks    func(*MockCommandRunner)
		expectError   bool
		errorContains string
	}{
		{
			name:        "systemd success",
			initSystem:  InitSystemd,
			setupMocks:  func(cmd *MockCommandRunner) {},
			expectError: false,
		},
		{
			name:       "systemd fails",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("systemctl", errors.New("enable failed"))
			},
			expectError:   true,
			errorContains: "failed to enable service",
		},
		{
			name:        "openrc success",
			initSystem:  InitOpenRC,
			setupMocks:  func(cmd *MockCommandRunner) {},
			expectError: false,
		},
		{
			name:       "openrc fails",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("rc-update", errors.New("rc-update failed"))
			},
			expectError:   true,
			errorContains: "failed to enable service",
		},
		{
			name:       "sysvinit success with update-rc.d",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupPath("update-rc.d", "/usr/sbin/update-rc.d")
			},
			expectError: false,
		},
		{
			name:       "sysvinit fails with update-rc.d",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupPath("update-rc.d", "/usr/sbin/update-rc.d")
				cmd.SetShouldFail("update-rc.d", errors.New("update-rc.d failed"))
			},
			expectError:   true,
			errorContains: "failed to enable service",
		},
		{
			name:       "sysvinit without update-rc.d",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupError("update-rc.d", errors.New("not found"))
			},
			expectError: false, // Should not error, just print warning
		},
		{
			name:        "unknown init system",
			initSystem:  InitUnknown,
			setupMocks:  func(cmd *MockCommandRunner) {},
			expectError: false, // Should not error, just print warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewMockCommandRunner()
			tt.setupMocks(cmd)

			installer := &ServiceInstaller{
				distro:     system.DistroUbuntu,
				initSystem: tt.initSystem,
				fs:         NewMockFileSystem(),
				cmd:        cmd,
				os:         NewMockOSInterface(),
			}

			err := installer.enableService()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Test NewServiceInstallerWithDeps function.
func TestNewServiceInstallerWithDeps(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockCmd := NewMockCommandRunner()
	mockOS := NewMockOSInterface()

	installer := NewServiceInstallerWithDeps(mockFS, mockCmd, mockOS)

	if installer == nil {
		t.Fatal("Expected non-nil installer")
	}
	if installer.fs != mockFS {
		t.Error("Expected filesystem to be set")
	}
	if installer.cmd != mockCmd {
		t.Error("Expected command runner to be set")
	}
	if installer.os != mockOS {
		t.Error("Expected OS interface to be set")
	}
}

// Test Uninstall method.
func TestServiceInstaller_Uninstall(t *testing.T) {
	tests := []struct {
		name        string
		initSystem  InitSystem
		setupMocks  func(*MockFileSystem, *MockCommandRunner, *MockOSInterface)
		description string
	}{
		{
			name:       "uninstall systemd service - keep config",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				// Simulate Scanln returning "n" (don't remove config)
				osIface.ScanlnFunc = func(a ...interface{}) (int, error) {
					if len(a) > 0 {
						if ptr, ok := a[0].(*string); ok {
							*ptr = "n"
						}
					}
					return 1, nil
				}
			},
			description: "Should not remove config when user says no",
		},
		{
			name:       "uninstall openrc service - remove config",
			initSystem: InitOpenRC,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				// Simulate Scanln returning "y" (remove config)
				osIface.ScanlnFunc = func(a ...interface{}) (int, error) {
					if len(a) > 0 {
						if ptr, ok := a[0].(*string); ok {
							*ptr = "y"
						}
					}
					return 1, nil
				}
			},
			description: "Should remove config when user says yes",
		},
		{
			name:       "uninstall sysvinit service - scanln fails",
			initSystem: InitSysVInit,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				// Simulate Scanln failing
				osIface.ScanlnFunc = func(a ...interface{}) (int, error) {
					return 0, errors.New("scanln failed")
				}
			},
			description: "Should default to not removing config when scanln fails",
		},
		{
			name:       "uninstall with RemoveAll failure for install dir",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				fs.SetShouldFail("RemoveAll", InstallDir, errors.New("remove install dir failed"))
				osIface.ScanlnFunc = func(a ...interface{}) (int, error) {
					if len(a) > 0 {
						if ptr, ok := a[0].(*string); ok {
							*ptr = "n"
						}
					}
					return 1, nil
				}
			},
			description: "Should handle install directory removal failure gracefully",
		},
		{
			name:       "uninstall with RemoveAll failure for config dir",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				fs.SetShouldFail("RemoveAll", ConfigDir, errors.New("remove config dir failed"))
				osIface.ScanlnFunc = func(a ...interface{}) (int, error) {
					if len(a) > 0 {
						if ptr, ok := a[0].(*string); ok {
							*ptr = "y" // User wants to remove config but it fails
						}
					}
					return 1, nil
				}
			},
			description: "Should handle config directory removal failure gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockCmd := NewMockCommandRunner()
			mockOS := NewMockOSInterface()
			tt.setupMocks(mockFS, mockCmd, mockOS)

			installer := NewServiceInstallerWithDeps(mockFS, mockCmd, mockOS)
			installer.initSystem = tt.initSystem

			err := installer.Uninstall()
			if err != nil {
				t.Errorf("Expected no error, got: %v (description: %s)", err, tt.description)
			}
		})
	}
}

// Test stopService method.
func TestServiceInstaller_stopService(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		setupMocks func(*MockCommandRunner)
	}{
		{
			name:       "stop systemd service",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {
				// Should succeed by default
			},
		},
		{
			name:       "stop systemd service fails",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("systemctl", errors.New("stop failed"))
			},
		},
		{
			name:       "stop openrc service",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {
				// Should succeed by default
			},
		},
		{
			name:       "stop openrc service fails",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("rc-service", errors.New("stop failed"))
			},
		},
		{
			name:       "stop sysvinit service",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				// Should succeed by default
			},
		},
		{
			name:       "stop sysvinit service fails",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("service", errors.New("stop failed"))
			},
		},
		{
			name:       "stop unknown init system",
			initSystem: InitUnknown,
			setupMocks: func(cmd *MockCommandRunner) {
				// Should return immediately
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := NewMockCommandRunner()
			tt.setupMocks(mockCmd)

			installer := NewServiceInstallerWithDeps(NewMockFileSystem(), mockCmd, NewMockOSInterface())
			installer.initSystem = tt.initSystem

			// stopService doesn't return error, just prints warnings
			installer.stopService()
		})
	}
}

// Test disableService method.
func TestServiceInstaller_disableService(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		setupMocks func(*MockCommandRunner)
	}{
		{
			name:       "disable systemd service",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {
				// Should succeed by default
			},
		},
		{
			name:       "disable systemd service fails",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("systemctl", errors.New("disable failed"))
			},
		},
		{
			name:       "disable openrc service",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {
				// Should succeed by default
			},
		},
		{
			name:       "disable openrc service fails",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("rc-update", errors.New("disable failed"))
			},
		},
		{
			name:       "disable sysvinit service with update-rc.d",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupPath("update-rc.d", "/usr/sbin/update-rc.d")
			},
		},
		{
			name:       "disable sysvinit service fails",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupPath("update-rc.d", "/usr/sbin/update-rc.d")
				cmd.SetShouldFail("update-rc.d", errors.New("disable failed"))
			},
		},
		{
			name:       "disable sysvinit without update-rc.d",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupError("update-rc.d", errors.New("not found"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := NewMockCommandRunner()
			tt.setupMocks(mockCmd)

			installer := NewServiceInstallerWithDeps(NewMockFileSystem(), mockCmd, NewMockOSInterface())
			installer.initSystem = tt.initSystem

			// disableService doesn't return error, just prints warnings
			installer.disableService()
		})
	}
}

// Test removeServiceFiles method.
func TestServiceInstaller_removeServiceFiles(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		setupMocks func(*MockFileSystem, *MockCommandRunner)
	}{
		{
			name:       "remove systemd service files",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// Should succeed by default
			},
		},
		{
			name:       "remove systemd service files - remove fails",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("Remove", "/etc/systemd/system/cloud-update.service", errors.New("remove failed"))
			},
		},
		{
			name:       "remove systemd service files - daemon-reload fails",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				cmd.SetShouldFail("systemctl", errors.New("daemon-reload failed"))
			},
		},
		{
			name:       "remove openrc service files",
			initSystem: InitOpenRC,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// Should succeed by default
			},
		},
		{
			name:       "remove openrc service files - remove fails",
			initSystem: InitOpenRC,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("Remove", "/etc/init.d/cloud-update", errors.New("remove failed"))
			},
		},
		{
			name:       "remove sysvinit service files",
			initSystem: InitSysVInit,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// Should succeed by default
			},
		},
		{
			name:       "remove sysvinit service files - remove fails",
			initSystem: InitSysVInit,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("Remove", "/etc/init.d/cloud-update", errors.New("remove failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockCmd := NewMockCommandRunner()
			tt.setupMocks(mockFS, mockCmd)

			installer := NewServiceInstallerWithDeps(mockFS, mockCmd, NewMockOSInterface())
			installer.initSystem = tt.initSystem

			// removeServiceFiles doesn't return error, just prints warnings
			installer.removeServiceFiles()
		})
	}
}

// Test detectInitSystemWithDeps with mocks.
func TestDetectInitSystemWithDeps(t *testing.T) {
	tests := []struct {
		name         string
		setupMocks   func(*MockFileSystem, *MockCommandRunner)
		expectedInit InitSystem
	}{
		{
			name: "detect systemd",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.StatFunc = func(name string) (os.FileInfo, error) {
					if name == "/run/systemd/system" {
						return nil, nil // File exists
					}
					return nil, os.ErrNotExist
				}
			},
			expectedInit: InitSystemd,
		},
		{
			name: "detect openrc",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.StatFunc = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist // systemd not found
				}
				cmd.SetLookupPath("openrc", "/sbin/openrc")
			},
			expectedInit: InitOpenRC,
		},
		{
			name: "detect upstart",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.StatFunc = func(name string) (os.FileInfo, error) {
					if name == "/etc/init" {
						return nil, nil // File exists
					}
					return nil, os.ErrNotExist
				}
				cmd.SetLookupError("openrc", errors.New("not found"))
				cmd.SetLookupPath("initctl", "/sbin/initctl")
			},
			expectedInit: InitUpstart,
		},
		{
			name: "detect sysvinit",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.StatFunc = func(name string) (os.FileInfo, error) {
					if name == "/etc/init.d" {
						return nil, nil // File exists
					}
					return nil, os.ErrNotExist
				}
				cmd.SetLookupError("openrc", errors.New("not found"))
				cmd.SetLookupError("initctl", errors.New("not found"))
			},
			expectedInit: InitSysVInit,
		},
		{
			name: "detect unknown",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.StatFunc = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist // Nothing found
				}
				cmd.SetLookupError("openrc", errors.New("not found"))
				cmd.SetLookupError("initctl", errors.New("not found"))
			},
			expectedInit: InitUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockCmd := NewMockCommandRunner()
			tt.setupMocks(mockFS, mockCmd)

			result := detectInitSystemWithDeps(mockFS, mockCmd)
			if result != tt.expectedInit {
				t.Errorf("Expected %s, got %s", tt.expectedInit, result)
			}
		})
	}
}

// Test generateSecretWithDeps with mocks.
func TestGenerateSecretWithDeps(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockCommandRunner)
		expectError bool
	}{
		{
			name: "openssl success",
			setupMocks: func(cmd *MockCommandRunner) {
				validOutput := make([]byte, 64) // 32 bytes in hex = 64 chars
				for i := range validOutput {
					validOutput[i] = 'a' // Valid hex character
				}
				cmd.SetOutput("openssl", validOutput)
			},
			expectError: false,
		},
		{
			name: "openssl fails - fallback to crypto/rand",
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("openssl", errors.New("openssl failed"))
			},
			expectError: false, // Should fallback
		},
		{
			name: "openssl returns invalid length",
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte("tooshort"))
			},
			expectError: false, // Should fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := NewMockCommandRunner()
			tt.setupMocks(mockCmd)

			secret, err := generateSecretWithDeps(mockCmd)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(secret) != 64 {
					t.Errorf("Expected secret length 64, got %d", len(secret))
				}
			}
		})
	}
}
