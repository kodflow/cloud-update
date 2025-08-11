package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test", "nested", "dir")

	err := CreateDirectory(testPath, 0755)
	if err != nil {
		t.Fatalf("CreateDirectory() error = %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("Directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}
}

func TestWriteFileWithMode(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testData := []byte("test content")

	err := WriteFileWithMode(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("WriteFileWithMode() error = %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("File content = %q, want %q", string(data), string(testData))
	}

	// Check permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// Check at least the basic permissions (may vary by OS)
	mode := info.Mode()
	if mode&0400 == 0 {
		t.Error("File should be readable")
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "source.txt")
	dstFile := filepath.Join(tempDir, "dest.txt")
	testData := []byte("source content")

	// Create source file
	if err := os.WriteFile(srcFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	err := CopyFile(srcFile, dstFile, 0755)
	if err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	// Verify destination
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Copied content = %q, want %q", string(data), string(testData))
	}
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "nonexistent.txt")
	dstFile := filepath.Join(tempDir, "dest.txt")

	err := CopyFile(srcFile, dstFile, 0755)
	if err == nil {
		t.Error("CopyFile() should fail with nonexistent source")
	}
}

func TestGenerateRandomSecret(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int // expected hex string length
	}{
		{"16 bytes", 16, 32},
		{"32 bytes", 32, 64},
		{"64 bytes", 64, 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := GenerateRandomSecret(tt.length)
			if err != nil {
				t.Fatalf("GenerateRandomSecret() error = %v", err)
			}

			if len(secret) != tt.want {
				t.Errorf("Secret length = %d, want %d", len(secret), tt.want)
			}

			// Test uniqueness
			secret2, err := GenerateRandomSecret(tt.length)
			if err != nil {
				t.Fatalf("GenerateRandomSecret() error = %v", err)
			}

			if secret == secret2 {
				t.Error("GenerateRandomSecret() should return unique values")
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "exists.txt")
	nonExistentFile := filepath.Join(tempDir, "notexists.txt")

	// Create a file
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing file", existingFile, true},
		{"non-existent file", nonExistentFile, false},
		{"empty path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExists(tt.path); got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveFileIfExists(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		setupFile bool
		wantErr   bool
	}{
		{"remove existing file", true, false},
		{"remove non-existent file", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test.txt")

			if tt.setupFile {
				if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			err := RemoveFileIfExists(testFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveFileIfExists() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify file is gone
			if FileExists(testFile) {
				t.Error("File should not exist after removal")
			}
		})
	}
}

func TestGetServiceFilePath(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		want       string
		wantErr    bool
	}{
		{"systemd", InitSystemd, "/etc/systemd/system/cloud-update.service", false},
		{"openrc", InitOpenRC, "/etc/init.d/cloud-update", false},
		{"sysvinit", InitSysVInit, "/etc/init.d/cloud-update", false},
		{"unknown", InitUnknown, "", true},
		{"upstart", InitUpstart, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetServiceFilePath(tt.initSystem)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceFilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetServiceFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetServiceCommand(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		action     string
		wantCmd    string
		wantArgs   []string
		wantErr    bool
	}{
		// Systemd tests
		{"systemd enable", InitSystemd, "enable", "systemctl", []string{"enable", "cloud-update"}, false},
		{"systemd start", InitSystemd, "start", "systemctl", []string{"start", "cloud-update"}, false},
		{"systemd stop", InitSystemd, "stop", "systemctl", []string{"stop", "cloud-update"}, false},
		{"systemd restart", InitSystemd, "restart", "systemctl", []string{"restart", "cloud-update"}, false},

		// OpenRC tests
		{"openrc enable", InitOpenRC, "enable", "rc-update", []string{"add", "cloud-update", "default"}, false},
		{"openrc disable", InitOpenRC, "disable", "rc-update", []string{"del", "cloud-update"}, false},
		{"openrc start", InitOpenRC, "start", "rc-service", []string{"cloud-update", "start"}, false},
		{"openrc stop", InitOpenRC, "stop", "rc-service", []string{"cloud-update", "stop"}, false},
		{"openrc invalid", InitOpenRC, "invalid", "", nil, true},

		// SysVInit tests
		{"sysvinit enable", InitSysVInit, "enable", "update-rc.d", []string{"cloud-update", "defaults"}, false},
		{"sysvinit disable", InitSysVInit, "disable", "update-rc.d", []string{"-f", "cloud-update", "remove"}, false},
		{"sysvinit start", InitSysVInit, "start", "service", []string{"cloud-update", "start"}, false},
		{"sysvinit stop", InitSysVInit, "stop", "service", []string{"cloud-update", "stop"}, false},
		{"sysvinit invalid", InitSysVInit, "invalid", "", nil, true},

		// Unknown init system
		{"unknown system", InitUnknown, "start", "", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args, err := GetServiceCommand(tt.initSystem, tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if cmd != tt.wantCmd {
				t.Errorf("GetServiceCommand() cmd = %v, want %v", cmd, tt.wantCmd)
			}
			if !equalSlices(args, tt.wantArgs) {
				t.Errorf("GetServiceCommand() args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestBuildConfigContent(t *testing.T) {
	secret := "test-secret-123"
	content := BuildConfigContent(secret)

	// Check that content contains expected elements
	expectedStrings := []string{
		"# Cloud Update Configuration",
		"webhook_secret: \"test-secret-123\"",
		"host: \"0.0.0.0\"",
		"port: 9999",
		"level: \"info\"",
		"/var/log/cloud-update.log",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(content, expected) {
			t.Errorf("Config content missing %q", expected)
		}
	}
}

func TestGetBinaryPath(t *testing.T) {
	path := GetBinaryPath()
	expected := filepath.Join(InstallDir, BinaryName)

	if path != expected {
		t.Errorf("GetBinaryPath() = %v, want %v", path, expected)
	}
}

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	expected := filepath.Join(ConfigDir, "config.yaml")

	if path != expected {
		t.Errorf("GetConfigPath() = %v, want %v", path, expected)
	}
}

// Helper function to compare slices.
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
