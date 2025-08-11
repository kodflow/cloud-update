package setup

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

// Test Setup() method enableService failure to reach 100% coverage.
func TestServiceInstaller_Setup_EnableServiceFailure(t *testing.T) {
	fs := NewMockFileSystem()
	cmd := NewMockCommandRunner()
	osIface := NewMockOSInterface()

	// Set up successful path until enableService
	osIface.SetEuid(0) // root
	osIface.SetExecutable("/test/cloud-update", nil)
	fs.WriteFile("/test/cloud-update", []byte("binary content"), 0755)
	cmd.SetOutput("openssl", []byte("deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678"))

	// Make enableService fail specifically (systemctl enable, not daemon-reload)
	cmd.SetShouldFailWithArgs("systemctl", []string{"enable", "cloud-update"}, fmt.Errorf("systemctl enable failed"))

	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
		fs:         fs,
		cmd:        cmd,
		os:         osIface,
	}

	err := installer.Setup()
	if err == nil {
		t.Error("Expected Setup() to fail when enableService fails")
		return
	}

	if !strings.Contains(err.Error(), "failed to enable service") {
		t.Errorf("Expected error to contain 'failed to enable service', got: %v", err)
	}
}
