package setup

import (
	"errors"
	"strings"
	"testing"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

// Test Uninstall method.
func TestServiceInstaller_Uninstall_Comprehensive(t *testing.T) {
	tests := []struct {
		name         string
		setupMocks   func(*MockFileSystem, *MockCommandRunner, *MockOSInterface)
		userResponse string
		expectError  bool
	}{
		{
			name: "successful uninstall with config removal",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetScanInput("y", nil)
			},
			userResponse: "y",
			expectError:  false,
		},
		{
			name: "successful uninstall without config removal",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetScanInput("n", nil)
			},
			userResponse: "n",
			expectError:  false,
		},
		{
			name: "binary removal fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				fs.SetShouldFail("RemoveAll", InstallDir, errors.New("removal failed"))
				osIface.SetScanInput("n", nil)
			},
			userResponse: "n",
			expectError:  false, // Uninstall doesn't fail for removal errors, just warns
		},
		{
			name: "config removal fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetScanInput("y", nil)
				fs.SetShouldFail("RemoveAll", ConfigDir, errors.New("removal failed"))
			},
			userResponse: "y",
			expectError:  false, // Uninstall doesn't fail for removal errors, just warns
		},
		{
			name: "scan input fails",
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner, osIface *MockOSInterface) {
				osIface.SetScanInput("", errors.New("scan failed"))
			},
			userResponse: "",
			expectError:  false, // Defaults to "n"
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

			err := installer.Uninstall()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}

			// Verify scan was called
			if osIface.GetScanCallCount() < 1 {
				t.Errorf("Expected scan to be called at least once")
			}
		})
	}
}

// Test stopService method.
func TestServiceInstaller_stopService_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		setupMocks func(*MockCommandRunner)
	}{
		{
			name:       "systemd success",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {},
		},
		{
			name:       "systemd fails",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("systemctl", errors.New("stop failed"))
			},
		},
		{
			name:       "openrc success",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {},
		},
		{
			name:       "openrc fails",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("rc-service", errors.New("stop failed"))
			},
		},
		{
			name:       "sysvinit success",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {},
		},
		{
			name:       "sysvinit fails",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("service", errors.New("stop failed"))
			},
		},
		{
			name:       "unknown init system",
			initSystem: InitUnknown,
			setupMocks: func(cmd *MockCommandRunner) {},
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

			// stopService doesn't return errors, it just prints warnings
			installer.stopService()

			// Verify appropriate commands were called for known init systems
			commands := cmd.GetCommands()
			if tt.initSystem != InitUnknown && len(commands) == 0 {
				t.Errorf("Expected commands to be called for init system %s", tt.initSystem)
			}
		})
	}
}

// Test disableService method.
func TestServiceInstaller_disableService_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		setupMocks func(*MockCommandRunner)
	}{
		{
			name:       "systemd success",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {},
		},
		{
			name:       "systemd fails",
			initSystem: InitSystemd,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("systemctl", errors.New("disable failed"))
			},
		},
		{
			name:       "openrc success",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {},
		},
		{
			name:       "openrc fails",
			initSystem: InitOpenRC,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetShouldFail("rc-update", errors.New("disable failed"))
			},
		},
		{
			name:       "sysvinit success with update-rc.d",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupPath("update-rc.d", "/usr/sbin/update-rc.d")
			},
		},
		{
			name:       "sysvinit fails with update-rc.d",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupPath("update-rc.d", "/usr/sbin/update-rc.d")
				cmd.SetShouldFail("update-rc.d", errors.New("disable failed"))
			},
		},
		{
			name:       "sysvinit without update-rc.d",
			initSystem: InitSysVInit,
			setupMocks: func(cmd *MockCommandRunner) {
				cmd.SetLookupError("update-rc.d", errors.New("not found"))
			},
		},
		{
			name:       "unknown init system",
			initSystem: InitUnknown,
			setupMocks: func(cmd *MockCommandRunner) {},
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

			// disableService doesn't return errors, it just prints warnings
			installer.disableService()

			// Verify appropriate commands or lookups were called
			if tt.initSystem == InitSysVInit {
				// For SysV, we expect at least a LookPath call
				commands := cmd.GetCommands()
				// Note: LookPath calls don't show up in GetCommands(),
				// but successful Run calls do
				if strings.Contains(tt.name, "success with update-rc.d") && len(commands) == 0 {
					t.Errorf("Expected commands to be called for successful SysV disable")
				}
			}
		})
	}
}

// Test removeServiceFiles method.
func TestServiceInstaller_removeServiceFiles_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		initSystem InitSystem
		setupMocks func(*MockFileSystem, *MockCommandRunner)
	}{
		{
			name:       "systemd success",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {},
		},
		{
			name:       "systemd file removal fails",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("Remove", "/etc/systemd/system/cloud-update.service", errors.New("removal failed"))
			},
		},
		{
			name:       "systemd daemon-reload fails",
			initSystem: InitSystemd,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				cmd.SetShouldFail("systemctl", errors.New("daemon-reload failed"))
			},
		},
		{
			name:       "openrc success",
			initSystem: InitOpenRC,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {},
		},
		{
			name:       "openrc file removal fails",
			initSystem: InitOpenRC,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("Remove", "/etc/init.d/cloud-update", errors.New("removal failed"))
			},
		},
		{
			name:       "sysvinit success",
			initSystem: InitSysVInit,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {},
		},
		{
			name:       "sysvinit file removal fails",
			initSystem: InitSysVInit,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {
				fs.SetShouldFail("Remove", "/etc/init.d/cloud-update", errors.New("removal failed"))
			},
		},
		{
			name:       "unknown init system",
			initSystem: InitUnknown,
			setupMocks: func(fs *MockFileSystem, cmd *MockCommandRunner) {},
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

			// removeServiceFiles doesn't return errors, it just prints warnings
			installer.removeServiceFiles()

			// For systemd, verify daemon-reload is called (success or fail)
			if tt.initSystem == InitSystemd {
				commands := cmd.GetCommands()
				found := false
				for _, command := range commands {
					if command.Name == "systemctl" && len(command.Args) > 0 && command.Args[0] == "daemon-reload" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected systemctl daemon-reload to be called for systemd")
				}
			}
		})
	}
}
