package setup

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

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
			errorContains: "failed to enable service",
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
