// Package system provides secure system-level operations and command execution.
package system

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/logger"
)

// SecureExecutor implements the Executor interface with enhanced security.
type SecureExecutor struct {
	privilegeCmd string
	timeout      time.Duration
}

// NewSecureExecutor creates a new secure system executor.
func NewSecureExecutor() Executor {
	return &SecureExecutor{
		privilegeCmd: detectPrivilegeCommand(),
		timeout:      5 * time.Minute, // Default timeout for system commands
	}
}

// runPrivilegedSecure runs commands with proper security measures.
func (e *SecureExecutor) runPrivilegedSecure(ctx context.Context, command string, args ...string) error {
	// Create a context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Log the command execution
	logger.WithField("command", command).Debug("Executing privileged command")

	// Build the command based on privilege escalation method
	var cmd *exec.Cmd

	if e.privilegeCmd == "" {
		// No privilege escalation needed/available
		cmd = exec.CommandContext(cmdCtx, command, args...)
	} else {
		switch e.privilegeCmd {
		case "doas", "sudo":
			// These tools handle arguments properly
			fullArgs := append([]string{command}, args...)
			cmd = exec.CommandContext(cmdCtx, e.privilegeCmd, fullArgs...) //nolint:gosec // Privilege cmd is validated
		default:
			// For other methods, refuse to run for security
			return fmt.Errorf("unsupported privilege escalation method: %s", e.privilegeCmd)
		}
	}

	// Execute the command
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.WithField("output", string(output)).WithField("error", err).Error("Command execution failed")

		// Check if it was a timeout
		if cmdCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out after %v", e.timeout)
		}

		return fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	logger.WithField("command", command).Info("Command executed successfully")
	return nil
}

// RunCloudInit executes cloud-init on the system.
func (e *SecureExecutor) RunCloudInit() error {
	ctx := context.Background()

	// First, clean cloud-init to ensure fresh run
	if err := e.runPrivilegedSecure(ctx, "cloud-init", "clean", "--logs"); err != nil {
		logger.WithField("error", err).Warn("Failed to clean cloud-init (non-fatal)")
	}

	// Run cloud-init
	return e.runPrivilegedSecure(ctx, "cloud-init", "init", "--local")
}

// Reboot schedules a system reboot.
func (e *SecureExecutor) Reboot() error {
	ctx := context.Background()

	// Schedule reboot in 1 minute to allow response to be sent
	logger.Info("Scheduling system reboot in 1 minute")
	return e.runPrivilegedSecure(ctx, "shutdown", "-r", "+1", "Cloud Update triggered reboot")
}

// UpdateSystem performs system updates based on the distribution.
func (e *SecureExecutor) UpdateSystem() error {
	ctx := context.Background()
	distro := e.DetectDistribution()

	logger.WithField("distribution", string(distro)).Info("Starting system update")

	switch distro {
	case DistroAlpine:
		// Update Alpine Linux
		if err := e.runPrivilegedSecure(ctx, "apk", "update"); err != nil {
			return err
		}
		return e.runPrivilegedSecure(ctx, "apk", "upgrade", "--available")

	case DistroDebian, DistroUbuntu:
		// Update Debian-based systems
		if err := e.runPrivilegedSecure(ctx, "apt-get", "update"); err != nil {
			return err
		}
		// Non-interactive upgrade
		return e.runPrivilegedSecure(ctx, "apt-get", "upgrade", "-y",
			"--with-new-pkgs", "-o", "Dpkg::Options::=--force-confdef",
			"-o", "Dpkg::Options::=--force-confold")

	case DistroRHEL, DistroCentOS, DistroFedora:
		// Update Red Hat-based systems
		return e.runPrivilegedSecure(ctx, "dnf", "upgrade", "-y", "--refresh")

	case DistroArch:
		// Update Arch Linux
		return e.runPrivilegedSecure(ctx, "pacman", "-Syu", "--noconfirm")

	case DistroSUSE:
		// Update openSUSE/SUSE
		if err := e.runPrivilegedSecure(ctx, "zypper", "refresh"); err != nil {
			return err
		}
		// Then upgrade
		return e.runPrivilegedSecure(ctx, "zypper", "update", "-y")

	default:
		return fmt.Errorf("unsupported distribution: %s", distro)
	}
}

// DetectDistribution detects the current Linux distribution.
func (e *SecureExecutor) DetectDistribution() Distribution {
	// This is a simplified version - the actual implementation is in the original file
	// We'll use the existing implementation
	executor := &DefaultExecutor{}
	return executor.DetectDistribution()
}
