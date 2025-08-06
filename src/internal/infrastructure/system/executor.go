// Package system provides system-level operations and command execution.
package system

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Distribution represents a Linux distribution type.
type Distribution string

// Supported Linux distributions.
const (
	DistroAlpine  Distribution = "alpine"
	DistroDebian  Distribution = "debian"
	DistroUbuntu  Distribution = "ubuntu"
	DistroRHEL    Distribution = "rhel"
	DistroCentOS  Distribution = "centos"
	DistroFedora  Distribution = "fedora"
	DistroSUSE    Distribution = "suse"
	DistroArch    Distribution = "arch"
	DistroUnknown Distribution = "unknown"
)

// Executor defines the interface for system operations.
type Executor interface {
	RunCloudInit() error
	Reboot() error
	UpdateSystem() error
	DetectDistribution() Distribution
}

// DefaultExecutor implements the Executor interface for real system operations.
type DefaultExecutor struct {
	privilegeCmd string
}

// NewSystemExecutor creates a new system executor.
func NewSystemExecutor() Executor {
	return &DefaultExecutor{
		privilegeCmd: detectPrivilegeCommand(),
	}
}

func detectPrivilegeCommand() string {
	commands := []string{"doas", "sudo", "su"}

	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err == nil {
			return cmd
		}
	}

	return ""
}

func (e *DefaultExecutor) runPrivileged(args ...string) error {
	// This function runs system commands with appropriate privileges
	// Commands are predefined and not user-controlled
	if e.privilegeCmd == "" {
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // predefined system commands only
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("command failed: %w, output: %s", err, string(output))
		}
		return nil
	}

	var cmd *exec.Cmd
	switch e.privilegeCmd {
	case "doas", "sudo":
		fullArgs := append([]string{}, args...)
		cmd = exec.Command(e.privilegeCmd, fullArgs...) //nolint:gosec // using privilege escalation tool
	case "su":
		// Use proper shell escaping to prevent injection
		escapedArgs := make([]string, len(args))
		for i, arg := range args {
			escapedArgs[i] = strconv.Quote(arg)
		}
		shellCmd := strings.Join(escapedArgs, " ")
		cmd = exec.Command("su", "-c", shellCmd) //nolint:gosec // su for privilege escalation
	default:
		cmd = exec.Command(args[0], args[1:]...) //nolint:gosec // fallback to direct execution
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}
	return nil
}

// RunCloudInit executes cloud-init on the system.
func (e *DefaultExecutor) RunCloudInit() error {
	return e.runPrivileged("cloud-init", "init")
}

// Reboot schedules a system reboot.
func (e *DefaultExecutor) Reboot() error {
	return e.runPrivileged("reboot")
}

// UpdateSystem performs a system update based on the detected distribution.
func (e *DefaultExecutor) UpdateSystem() error {
	distro := e.DetectDistribution()

	switch distro {
	case DistroAlpine:
		if err := e.runPrivileged("apk", "update"); err != nil {
			return err
		}
		return e.runPrivileged("apk", "upgrade")

	case DistroDebian, DistroUbuntu:
		if err := e.runPrivileged("apt-get", "update"); err != nil {
			return err
		}
		return e.runPrivileged("apt-get", "upgrade", "-y")

	case DistroRHEL, DistroCentOS, DistroFedora:
		if _, err := exec.LookPath("dnf"); err == nil {
			return e.runPrivileged("dnf", "update", "-y")
		}
		return e.runPrivileged("yum", "update", "-y")

	case DistroSUSE:
		if err := e.runPrivileged("zypper", "refresh"); err != nil {
			return err
		}
		return e.runPrivileged("zypper", "update", "-y")

	case DistroArch:
		return e.runPrivileged("pacman", "-Syu", "--noconfirm")

	default:
		return fmt.Errorf("unsupported distribution: %s", distro)
	}
}

// DetectDistribution detects the current Linux distribution.
func (e *DefaultExecutor) DetectDistribution() Distribution {
	if _, err := os.Stat("/etc/alpine-release"); err == nil {
		return DistroAlpine
	}

	if _, err := os.Stat("/etc/os-release"); err == nil {
		content, err := os.ReadFile("/etc/os-release")
		if err != nil {
			return DistroUnknown
		}
		osRelease := strings.ToLower(string(content))

		switch {
		case strings.Contains(osRelease, "alpine"):
			return DistroAlpine
		case strings.Contains(osRelease, "ubuntu"):
			return DistroUbuntu
		case strings.Contains(osRelease, "debian"):
			return DistroDebian
		case strings.Contains(osRelease, "rhel") || strings.Contains(osRelease, "red hat"):
			return DistroRHEL
		case strings.Contains(osRelease, "centos"):
			return DistroCentOS
		case strings.Contains(osRelease, "fedora"):
			return DistroFedora
		case strings.Contains(osRelease, "suse") || strings.Contains(osRelease, "opensuse"):
			return DistroSUSE
		case strings.Contains(osRelease, "arch"):
			return DistroArch
		}
	}

	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return DistroDebian
	}

	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		return DistroRHEL
	}

	return DistroUnknown
}
