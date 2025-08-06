// Package system provides system-level operations with timeout support.
package system

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// ExecutorWithTimeout wraps an executor with timeout capabilities.
type ExecutorWithTimeout struct {
	*DefaultExecutor
	defaultTimeout time.Duration
}

// NewExecutorWithTimeout creates an executor with timeout support.
func NewExecutorWithTimeout(timeout time.Duration) *ExecutorWithTimeout {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &ExecutorWithTimeout{
		DefaultExecutor: &DefaultExecutor{
			privilegeCmd: detectPrivilegeCommand(),
		},
		defaultTimeout: timeout,
	}
}

// RunCommandWithTimeout executes a command with a specific timeout.
func (e *ExecutorWithTimeout) RunCommandWithTimeout(
	ctx context.Context, timeout time.Duration, command string, args ...string,
) error {
	if timeout <= 0 {
		timeout = e.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("command timed out after %v", timeout)
	}

	if err != nil {
		return fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	return nil
}

// getTimeoutForDistro returns appropriate timeout for distribution.
func getTimeoutForDistro(distro Distribution) time.Duration {
	switch distro {
	case DistroAlpine:
		return 3 * time.Minute // Alpine is usually faster
	case DistroDebian, DistroUbuntu, DistroRHEL, DistroCentOS, DistroFedora:
		return 10 * time.Minute // apt/yum/dnf can be slow
	default:
		return 5 * time.Minute
	}
}

// runUpdate runs the update command for the distribution.
func (e *ExecutorWithTimeout) runUpdate(ctx context.Context, distro Distribution, timeout time.Duration) error {
	updateCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var updateCmd *exec.Cmd
	switch distro {
	case DistroAlpine:
		updateCmd = exec.CommandContext(updateCtx, "apk", "update")
	case DistroDebian, DistroUbuntu:
		updateCmd = exec.CommandContext(updateCtx, "apt-get", "update")
	case DistroRHEL, DistroCentOS:
		updateCmd = exec.CommandContext(updateCtx, "yum", "check-update")
	case DistroFedora:
		updateCmd = exec.CommandContext(updateCtx, "dnf", "check-update")
	case DistroArch:
		updateCmd = exec.CommandContext(updateCtx, "pacman", "-Sy")
	default:
		return fmt.Errorf("unsupported distribution: %s", distro)
	}

	if output, err := updateCmd.CombinedOutput(); err != nil {
		if updateCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("update check timed out after %v", timeout)
		}
		// For yum/dnf, exit code 100 means updates available
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 100 {
			// Exit code 100 means updates are available - this is expected
			return nil
		}
		return fmt.Errorf("update failed: %w, output: %s", err, string(output))
	}
	return nil
}

// runUpgrade runs the upgrade command for the distribution.
func (e *ExecutorWithTimeout) runUpgrade(ctx context.Context, distro Distribution, timeout time.Duration) error {
	upgradeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var upgradeCmd *exec.Cmd
	switch distro {
	case DistroAlpine:
		upgradeCmd = exec.CommandContext(upgradeCtx, "apk", "upgrade")
	case DistroDebian, DistroUbuntu:
		upgradeCmd = exec.CommandContext(upgradeCtx, "apt-get", "upgrade", "-y")
	case DistroRHEL, DistroCentOS:
		upgradeCmd = exec.CommandContext(upgradeCtx, "yum", "update", "-y")
	case DistroFedora:
		upgradeCmd = exec.CommandContext(upgradeCtx, "dnf", "upgrade", "-y")
	case DistroArch:
		upgradeCmd = exec.CommandContext(upgradeCtx, "pacman", "-Su", "--noconfirm")
	default:
		return fmt.Errorf("unsupported distribution: %s", distro)
	}

	if output, err := upgradeCmd.CombinedOutput(); err != nil {
		if upgradeCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("upgrade timed out after %v", timeout)
		}
		return fmt.Errorf("upgrade failed: %w, output: %s", err, string(output))
	}
	return nil
}

// UpdateSystemWithTimeout updates system with timeout.
func (e *ExecutorWithTimeout) UpdateSystemWithTimeout(ctx context.Context) error {
	distro := e.DetectDistribution()
	timeout := getTimeoutForDistro(distro)

	// Update package lists
	if err := e.runUpdate(ctx, distro, timeout); err != nil {
		return err
	}

	// Upgrade packages with double timeout
	return e.runUpgrade(ctx, distro, timeout*2)
}

// RebootWithDelay schedules a reboot with a delay.
func (e *ExecutorWithTimeout) RebootWithDelay(delay time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use shutdown command with delay
	seconds := int(delay.Seconds())
	minutes := fmt.Sprintf("+%d", seconds/60)
	cmd := exec.CommandContext(ctx, "shutdown", "-r", minutes) //nolint:gosec // shutdown is safe

	if output, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("reboot command timed out")
		}
		return fmt.Errorf("reboot scheduling failed: %w, output: %s", err, string(output))
	}

	return nil
}
