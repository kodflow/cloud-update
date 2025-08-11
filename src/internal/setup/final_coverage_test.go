package setup

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

// Test Setup() method failure paths to reach 100% coverage.
func TestServiceInstaller_Setup_FailurePaths(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockFileSystem, *MockCommandRunner, *MockOSInterface)
		expectError   bool
		errorContains string
	}{
		{
			name: "installService fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("/test/cloud-update", nil)
				fs.WriteFile("/test/cloud-update", []byte("binary content"), 0755)
				// Make service installation fail
				serviceErr := fmt.Errorf("service write failed")
				fs.SetShouldFail("WriteFile", "/etc/systemd/system/cloud-update.service", serviceErr)
			},
			expectError:   true,
			errorContains: "failed to install service",
		},
		{
			name: "createConfig fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("/test/cloud-update", nil)
				fs.WriteFile("/test/cloud-update", []byte("binary content"), 0755)
				// Make config creation fail
				configPath := ConfigDir + "/config.env"
				fs.SetShouldFail("WriteFile", configPath, fmt.Errorf("config write failed"))
			},
			expectError:   true,
			errorContains: "failed to create config",
		},
		{
			name: "enableService fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetEuid(0)
				osIface.SetExecutable("/test/cloud-update", nil)
				fs.WriteFile("/test/cloud-update", []byte("binary content"), 0755)
				key := "deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678"
				cmd.SetOutput("openssl", []byte(key))
				// Make service enabling fail
				cmd.SetShouldFail("systemctl", fmt.Errorf("enable failed"))
			},
			expectError:   true,
			errorContains: "failed to install service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// Test createConfig secret generation failure edge case.
func TestServiceInstaller_createConfig_SecretGenerationError(t *testing.T) {
	fs := NewMockFileSystem()
	cmd := NewMockCommandRunner()

	// Set up to fail secret generation in a way that would cause the function to error
	// The generateSecret function has a fallback, so we need to test the specific error path
	cmd.SetShouldFail("openssl", fmt.Errorf("openssl failed"))

	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
		fs:         fs,
		cmd:        cmd,
		os:         NewMockOSInterface(),
	}

	// Call createConfig - it should succeed due to crypto/rand fallback
	err := installer.createConfig()
	if err != nil {
		t.Errorf("createConfig() should succeed with crypto/rand fallback: %v", err)
	}
}

// Test GenerateRandomSecret with different sizes and potential error conditions.
func TestGenerateRandomSecret_Coverage(t *testing.T) {
	// Test the function that might have uncovered error paths
	tests := []struct {
		name    string
		size    int
		wantLen int
		wantErr bool
	}{
		{"size 1", 1, 2, false},
		{"size 16", 16, 32, false},
		{"size 32", 32, 64, false},
		{"size 64", 64, 128, false},
		{"size 0", 0, 0, false}, // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomSecret(tt.size)

			if tt.wantErr && err == nil {
				t.Errorf("GenerateRandomSecret(%d) expected error but got none", tt.size)
			} else if !tt.wantErr && err != nil {
				t.Errorf("GenerateRandomSecret(%d) unexpected error: %v", tt.size, err)
			}

			if len(result) != tt.wantLen {
				t.Errorf("GenerateRandomSecret(%d) length = %d, want %d", tt.size, len(result), tt.wantLen)
			}

			// Verify it's valid hex (if non-empty)
			if len(result) > 0 {
				for _, char := range result {
					if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
						t.Errorf("Result contains invalid hex character: %c", char)
						break
					}
				}
			}
		})
	}
}

// Test generateSecretWithDeps edge cases to improve coverage.
func TestGenerateSecretWithDeps_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockCommandRunner)
		expectError bool
	}{
		{
			name: "openssl returns short output",
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte("short"))
			},
			expectError: false, // Should fallback to crypto/rand
		},
		{
			name: "openssl returns exactly right length",
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"))
			},
			expectError: false,
		},
		{
			name: "openssl returns with whitespace",
			setupMocks: func(cmd *MockCommandRunner) {
				key := "  deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678  \n\t"
				cmd.SetOutput("openssl", []byte(key))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewMockCommandRunner()
			tt.setupMocks(cmd)

			result, err := generateSecretWithDeps(cmd)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if result == "" {
					t.Error("Expected non-empty result")
				}
				if len(result) != 64 {
					t.Errorf("Result length = %d, want 64", len(result))
				}
			}
		})
	}
}

// Test specific helper functions to ensure full coverage.
func TestHelperFunctions_Coverage(t *testing.T) {
	// Test GenerateRandomSecret with size 0
	result, err := GenerateRandomSecret(0)
	if err != nil {
		t.Errorf("GenerateRandomSecret(0) failed: %v", err)
	}
	if result != "" {
		t.Errorf("GenerateRandomSecret(0) = %q, want empty string", result)
	}

	// Test with very small size
	result, err = GenerateRandomSecret(1)
	if err != nil {
		t.Errorf("GenerateRandomSecret(1) failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("GenerateRandomSecret(1) length = %d, want 2", len(result))
	}

	// Test with larger sizes to cover different paths
	for _, size := range []int{8, 16, 32, 128} {
		result, err := GenerateRandomSecret(size)
		if err != nil {
			t.Errorf("GenerateRandomSecret(%d) failed: %v", size, err)
		}
		expectedLen := size * 2
		if len(result) != expectedLen {
			t.Errorf("GenerateRandomSecret(%d) length = %d, want %d", size, len(result), expectedLen)
		}
	}
}
