package setup

import (
	"fmt"
	"os"
	"os/exec"
)

// FileSystem interface for file operations (for testing).
type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadFile(filename string) ([]byte, error)
	Remove(name string) error
	RemoveAll(path string) error
	Stat(name string) (os.FileInfo, error)
	Chmod(name string, mode os.FileMode) error
	Chown(name string, uid, gid int) error
}

// CommandRunner interface for executing system commands (for testing).
type CommandRunner interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) ([]byte, error)
	LookPath(file string) (string, error)
}

// OSInterface provides OS-level operations.
type OSInterface interface {
	Executable() (string, error)
	Geteuid() int
	Scanln(a ...interface{}) (n int, err error)
}

// RealFileSystem implements FileSystem using actual OS operations.
type RealFileSystem struct{}

func (r RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

func (r RealFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(filename, data, perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func (r RealFileSystem) ReadFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

func (r RealFileSystem) Remove(name string) error {
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}
	return nil
}

func (r RealFileSystem) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove directory: %w", err)
	}
	return nil
}

func (r RealFileSystem) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	return info, nil
}

func (r RealFileSystem) Chmod(name string, mode os.FileMode) error {
	if err := os.Chmod(name, mode); err != nil {
		return fmt.Errorf("failed to change file permissions: %w", err)
	}
	return nil
}

func (r RealFileSystem) Chown(name string, uid, gid int) error {
	if err := os.Chown(name, uid, gid); err != nil {
		return fmt.Errorf("failed to change file ownership: %w", err)
	}
	return nil
}

// RealCommandRunner implements CommandRunner using actual exec commands.
type RealCommandRunner struct{}

func (r RealCommandRunner) Run(name string, args ...string) error {
	if err := exec.Command(name, args...).Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}
	return nil
}

func (r RealCommandRunner) Output(name string, args ...string) ([]byte, error) {
	output, err := exec.Command(name, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get command output: %w", err)
	}
	return output, nil
}

func (r RealCommandRunner) LookPath(file string) (string, error) {
	path, err := exec.LookPath(file)
	if err != nil {
		return "", fmt.Errorf("failed to find command in PATH: %w", err)
	}
	return path, nil
}

// RealOSInterface implements OSInterface using actual OS operations.
type RealOSInterface struct{}

func (r RealOSInterface) Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return exe, nil
}

func (r RealOSInterface) Geteuid() int {
	return os.Geteuid()
}

func (r RealOSInterface) Scanln(a ...interface{}) (n int, err error) {
	n, err = fmt.Scanln(a...)
	if err != nil {
		return n, fmt.Errorf("failed to scan input: %w", err)
	}
	return n, nil
}
