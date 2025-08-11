package setup

import (
	"errors"
	"testing"
)

// Test detectInitSystemWithDeps function.
func TestDetectInitSystemWithDeps(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockFileSystem, *MockCommandRunner)
		expected   InitSystem
	}{
		{
			name: "detect systemd",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// systemd is detected by presence of /run/systemd/system
				fs.WriteFile("/run/systemd/system/dummy", []byte(""), 0644)
			},
			expected: InitSystemd,
		},
		{
			name: "detect openrc",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// systemd not present
				fs.SetStatError("/run/systemd/system", errors.New("not found"))
				// openrc is detected by presence of openrc command
				cmd.SetLookupPath("openrc", "/sbin/openrc")
			},
			expected: InitOpenRC,
		},
		{
			name: "detect upstart",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// systemd not present
				fs.SetStatError("/run/systemd/system", errors.New("not found"))
				// openrc not present
				cmd.SetLookupError("openrc", errors.New("not found"))
				// upstart is detected by presence of /etc/init and initctl command
				fs.WriteFile("/etc/init/dummy", []byte(""), 0644)
				cmd.SetLookupPath("initctl", "/sbin/initctl")
			},
			expected: InitUpstart,
		},
		{
			name: "detect sysvinit",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// systemd not present
				fs.SetStatError("/run/systemd/system", errors.New("not found"))
				// openrc not present
				cmd.SetLookupError("openrc", errors.New("not found"))
				// upstart not present (no /etc/init)
				fs.SetStatError("/etc/init", errors.New("not found"))
				// sysvinit is detected by presence of /etc/init.d
				fs.WriteFile("/etc/init.d/dummy", []byte(""), 0755)
			},
			expected: InitSysVInit,
		},
		{
			name: "upstart without initctl",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// systemd not present
				fs.SetStatError("/run/systemd/system", errors.New("not found"))
				// openrc not present
				cmd.SetLookupError("openrc", errors.New("not found"))
				// /etc/init present but no initctl command
				fs.WriteFile("/etc/init/dummy", []byte(""), 0644)
				cmd.SetLookupError("initctl", errors.New("not found"))
				// sysvinit is detected by presence of /etc/init.d
				fs.WriteFile("/etc/init.d/dummy", []byte(""), 0755)
			},
			expected: InitSysVInit, // Falls through to sysvinit
		},
		{
			name: "unknown init system",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// systemd not present
				fs.SetStatError("/run/systemd/system", errors.New("not found"))
				// openrc not present
				cmd.SetLookupError("openrc", errors.New("not found"))
				// upstart not present
				fs.SetStatError("/etc/init", errors.New("not found"))
				// sysvinit not present
				fs.SetStatError("/etc/init.d", errors.New("not found"))
			},
			expected: InitUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			cmd := NewMockCommandRunner()
			tt.setupMocks(fs, cmd)

			result := detectInitSystemWithDeps(fs, cmd)

			if result != tt.expected {
				t.Errorf("detectInitSystemWithDeps() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test detectInitSystem (wrapper function).
func TestDetectInitSystem_Wrapper(t *testing.T) {
	// This tests the wrapper function that uses real implementations
	result := detectInitSystem()

	// Should return one of the valid init systems
	validSystems := []InitSystem{
		InitSystemd, InitOpenRC, InitSysVInit, InitUpstart, InitUnknown,
	}

	found := false
	for _, valid := range validSystems {
		if result == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("detectInitSystem() = %v, want one of %v", result, validSystems)
	}
}

// validateSecretError checks if the error state matches expectations.
func validateSecretError(t *testing.T, expectError bool, err error) {
	if expectError && err == nil {
		t.Errorf("Expected error but got none")
	} else if !expectError && err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

// validateSecretResult validates the secret result when no error is expected.
func validateSecretResult(t *testing.T, result string, validateLen bool, expectedLen int) {
	if result == "" {
		t.Errorf("Expected non-empty secret")
		return
	}

	if validateLen && len(result) != expectedLen {
		t.Errorf("Expected secret length %d, got %d", expectedLen, len(result))
	}

	validateHexCharacters(t, result)
}

// validateHexCharacters checks if the secret contains only valid hex characters.
func validateHexCharacters(t *testing.T, secret string) {
	for _, char := range secret {
		if char < '0' || (char > '9' && char < 'a') || char > 'f' {
			t.Errorf("Secret contains invalid hex character: %c", char)
			break
		}
	}
}

// Test generateSecretWithDeps function.
func TestGenerateSecretWithDeps(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockCommandRunner)
		expectError bool
		validateLen bool
		expectedLen int
	}{
		{
			name: "openssl success",
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte("deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678"))
			},
			expectError: false,
			validateLen: true,
			expectedLen: 64,
		},
		{
			name: "openssl fails, fallback to crypto/rand",
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("openssl", errors.New("openssl not found"))
			},
			expectError: false,
			validateLen: true,
			expectedLen: 64,
		},
		{
			name: "openssl returns empty output, fallback to crypto/rand",
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte(""))
			},
			expectError: false,
			validateLen: true,
			expectedLen: 64,
		},
		{
			name: "openssl returns with newline",
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetOutput("openssl", []byte("deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678\n"))
			},
			expectError: false,
			validateLen: true,
			expectedLen: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewMockCommandRunner()
			tt.setupMocks(cmd)

			result, err := generateSecretWithDeps(cmd)

			validateSecretError(t, tt.expectError, err)

			if !tt.expectError {
				validateSecretResult(t, result, tt.validateLen, tt.expectedLen)
			}
		})
	}
}

// Test generateSecret (wrapper function).
func TestGenerateSecret_Wrapper(t *testing.T) {
	// This tests the wrapper function that uses real implementations
	result, err := generateSecret()

	validateSecretError(t, false, err)

	if err == nil {
		validateSecretResult(t, result, true, 64)
	}
}

// Test edge cases for init system detection.
func TestDetectInitSystemWithDeps_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockFileSystem, *MockCommandRunner)
		expected   InitSystem
	}{
		{
			name: "systemd directory exists but is empty",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// Create directory but it should still be detected
				fs.MkdirAll("/run/systemd/system", 0755)
			},
			expected: InitSystemd,
		},
		{
			name: "multiple init systems present, systemd takes precedence",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// All systems present, systemd should win
				fs.WriteFile("/run/systemd/system/dummy", []byte(""), 0644)
				cmd.SetLookupPath("openrc", "/sbin/openrc")
				fs.WriteFile("/etc/init/dummy", []byte(""), 0644)
				cmd.SetLookupPath("initctl", "/sbin/initctl")
				fs.WriteFile("/etc/init.d/dummy", []byte(""), 0755)
			},
			expected: InitSystemd,
		},
		{
			name: "openrc takes precedence over upstart and sysvinit",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// systemd not present
				fs.SetStatError("/run/systemd/system", errors.New("not found"))
				// openrc present
				cmd.SetLookupPath("openrc", "/sbin/openrc")
				// upstart also present
				fs.WriteFile("/etc/init/dummy", []byte(""), 0644)
				cmd.SetLookupPath("initctl", "/sbin/initctl")
				// sysvinit also present
				fs.WriteFile("/etc/init.d/dummy", []byte(""), 0755)
			},
			expected: InitOpenRC,
		},
		{
			name: "upstart takes precedence over sysvinit",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				// systemd not present
				fs.SetStatError("/run/systemd/system", errors.New("not found"))
				// openrc not present
				cmd.SetLookupError("openrc", errors.New("not found"))
				// upstart present
				fs.WriteFile("/etc/init/dummy", []byte(""), 0644)
				cmd.SetLookupPath("initctl", "/sbin/initctl")
				// sysvinit also present
				fs.WriteFile("/etc/init.d/dummy", []byte(""), 0755)
			},
			expected: InitUpstart,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			cmd := NewMockCommandRunner()
			tt.setupMocks(fs, cmd)

			result := detectInitSystemWithDeps(fs, cmd)

			if result != tt.expected {
				t.Errorf("detectInitSystemWithDeps() = %v, want %v", result, tt.expected)
			}
		})
	}
}
