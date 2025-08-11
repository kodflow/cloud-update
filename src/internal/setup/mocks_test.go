package setup

import (
	"errors"
	"os"
)

// MockFileSystem is a mock implementation of FileSystem interface.
type MockFileSystem struct {
	MkdirAllFunc  func(string, os.FileMode) error
	WriteFileFunc func(string, []byte, os.FileMode) error
	ReadFileFunc  func(string) ([]byte, error)
	StatFunc      func(string) (os.FileInfo, error)
	ChmodFunc     func(string, os.FileMode) error
	ChownFunc     func(string, int, int) error
	RemoveFunc    func(string) error
	RemoveAllFunc func(string) error
	// Test helpers
	files   map[string][]byte
	failOps map[string]map[string]error // operation -> path -> error
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if m.failOps != nil && m.failOps["MkdirAll"] != nil {
		if err, ok := m.failOps["MkdirAll"][path]; ok {
			return err
		}
	}
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(path, perm)
	}
	return nil
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if m.failOps != nil && m.failOps["WriteFile"] != nil {
		if err, ok := m.failOps["WriteFile"][filename]; ok {
			return err
		}
	}
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(filename, data, perm)
	}
	if m.files != nil {
		m.files[filename] = data
	}
	return nil
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if m.failOps != nil && m.failOps["ReadFile"] != nil {
		if err, ok := m.failOps["ReadFile"][filename]; ok {
			return nil, err
		}
	}
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(filename)
	}
	if m.files != nil {
		if data, ok := m.files[filename]; ok {
			return data, nil
		}
	}
	return []byte{}, nil
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(name)
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Chmod(name string, mode os.FileMode) error {
	if m.ChmodFunc != nil {
		return m.ChmodFunc(name, mode)
	}
	return nil
}

func (m *MockFileSystem) Chown(name string, uid, gid int) error {
	if m.failOps != nil && m.failOps["Chown"] != nil {
		if err, ok := m.failOps["Chown"][name]; ok {
			return err
		}
	}
	if m.ChownFunc != nil {
		return m.ChownFunc(name, uid, gid)
	}
	return nil
}

func (m *MockFileSystem) Remove(name string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(name)
	}
	return nil
}

func (m *MockFileSystem) RemoveAll(path string) error {
	if m.RemoveAllFunc != nil {
		return m.RemoveAllFunc(path)
	}
	return nil
}

// Helper methods for testing.
func (m *MockFileSystem) AddFile(path string, content []byte) {
	if m.files == nil {
		m.files = make(map[string][]byte)
	}
	m.files[path] = content
}

func (m *MockFileSystem) SetShouldFail(operation string, path string, err error) {
	if m.failOps == nil {
		m.failOps = make(map[string]map[string]error)
	}
	if m.failOps[operation] == nil {
		m.failOps[operation] = make(map[string]error)
	}
	m.failOps[operation][path] = err
}

func (m *MockFileSystem) FileExists(path string) bool {
	if m.files != nil {
		_, exists := m.files[path]
		return exists
	}
	return false
}

func (m *MockFileSystem) DirExists(path string) bool {
	// Simplified implementation for testing
	return true
}

// MockCommandRunner is a mock implementation of CommandRunner interface.
type MockCommandRunner struct {
	LookPathFunc func(string) (string, error)
	RunFunc      func(string, ...string) error
	OutputFunc   func(string, ...string) ([]byte, error)
	// Test helpers
	output     map[string][]byte
	failCmds   map[string]error
	lookupPath map[string]string
}

func (m *MockCommandRunner) LookPath(file string) (string, error) {
	if m.lookupPath != nil {
		if path, ok := m.lookupPath[file]; ok {
			if path == "" {
				return "", errors.New("command not found")
			}
			return path, nil
		}
	}
	if m.LookPathFunc != nil {
		return m.LookPathFunc(file)
	}
	return "/usr/bin/" + file, nil
}

func (m *MockCommandRunner) Run(name string, args ...string) error {
	if m.failCmds != nil {
		if err, ok := m.failCmds[name]; ok {
			return err
		}
	}
	if m.RunFunc != nil {
		return m.RunFunc(name, args...)
	}
	return nil
}

func (m *MockCommandRunner) Output(name string, args ...string) ([]byte, error) {
	if m.output != nil {
		if output, ok := m.output[name]; ok {
			return output, nil
		}
	}
	if m.OutputFunc != nil {
		return m.OutputFunc(name, args...)
	}
	return []byte{}, nil
}

// Helper methods for testing.
func (m *MockCommandRunner) SetOutput(cmd string, output []byte) {
	if m.output == nil {
		m.output = make(map[string][]byte)
	}
	m.output[cmd] = output
}

func (m *MockCommandRunner) SetShouldFail(cmd string, err error) {
	if m.failCmds == nil {
		m.failCmds = make(map[string]error)
	}
	m.failCmds[cmd] = err
}

func (m *MockCommandRunner) SetShouldFailWithArgs(cmd string, args []string, err error) {
	// Simplified: just use the command name for now
	m.SetShouldFail(cmd, err)
}

func (m *MockCommandRunner) SetLookupPath(cmd string, path string) {
	if m.lookupPath == nil {
		m.lookupPath = make(map[string]string)
	}
	m.lookupPath[cmd] = path
}

func (m *MockCommandRunner) SetLookupError(cmd string, err error) {
	// We ignore the error parameter since our implementation
	// uses empty string to trigger error
	if m.lookupPath == nil {
		m.lookupPath = make(map[string]string)
	}
	m.lookupPath[cmd] = "" // empty string will trigger error in LookPath
}

// MockOSInterface is a mock implementation of OSInterface.
type MockOSInterface struct {
	ExecutableFunc  func() (string, error)
	GetuidFunc      func() int
	GeteuidFunc     func() int
	GetgidFunc      func() int
	HostnameFunc    func() (string, error)
	GetenvFunc      func(string) string
	SetenvFunc      func(string, string) error
	UnsetenvFunc    func(string) error
	LookupEnvFunc   func(string) (string, bool)
	UserHomeDirFunc func() (string, error)
	ScanlnFunc      func(...interface{}) (int, error)
	// Test helper fields
	euid           int
	executablePath string
	executableErr  error
}

func (m *MockOSInterface) Executable() (string, error) {
	if m.ExecutableFunc != nil {
		return m.ExecutableFunc()
	}
	if m.executableErr != nil {
		return "", m.executableErr
	}
	if m.executablePath != "" {
		return m.executablePath, nil
	}
	return "/usr/local/bin/cloud-update", nil
}

// Helper method to set executable path for testing.
func (m *MockOSInterface) SetExecutable(path string, err error) {
	m.executablePath = path
	m.executableErr = err
}

func (m *MockOSInterface) Getuid() int {
	if m.GetuidFunc != nil {
		return m.GetuidFunc()
	}
	return 0
}

func (m *MockOSInterface) Geteuid() int {
	if m.GeteuidFunc != nil {
		return m.GeteuidFunc()
	}
	return m.euid
}

// Helper method to set effective UID for testing.
func (m *MockOSInterface) SetEuid(uid int) {
	m.euid = uid
}

func (m *MockOSInterface) Scanln(a ...interface{}) (n int, err error) {
	if m.ScanlnFunc != nil {
		return m.ScanlnFunc(a...)
	}
	// Default implementation for testing
	return 0, nil
}

func (m *MockOSInterface) Getgid() int {
	if m.GetgidFunc != nil {
		return m.GetgidFunc()
	}
	return 0
}

func (m *MockOSInterface) Hostname() (string, error) {
	if m.HostnameFunc != nil {
		return m.HostnameFunc()
	}
	return "test-host", nil
}

func (m *MockOSInterface) Getenv(key string) string {
	if m.GetenvFunc != nil {
		return m.GetenvFunc(key)
	}
	return ""
}

func (m *MockOSInterface) Setenv(key, value string) error {
	if m.SetenvFunc != nil {
		return m.SetenvFunc(key, value)
	}
	return nil
}

func (m *MockOSInterface) Unsetenv(key string) error {
	if m.UnsetenvFunc != nil {
		return m.UnsetenvFunc(key)
	}
	return nil
}

func (m *MockOSInterface) LookupEnv(key string) (string, bool) {
	if m.LookupEnvFunc != nil {
		return m.LookupEnvFunc(key)
	}
	return "", false
}

func (m *MockOSInterface) UserHomeDir() (string, error) {
	if m.UserHomeDirFunc != nil {
		return m.UserHomeDirFunc()
	}
	return "/home/test", nil
}

// Helper functions to create mocks with default behavior

// NewMockFileSystem creates a new MockFileSystem with default behavior.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
		MkdirAllFunc: func(path string, perm os.FileMode) error {
			return nil
		},
		// Don't set WriteFileFunc and ReadFileFunc so they use the files map
		StatFunc: func(name string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		ChmodFunc: func(name string, mode os.FileMode) error {
			return nil
		},
		ChownFunc: func(name string, uid, gid int) error {
			return nil
		},
		RemoveFunc: func(name string) error {
			return nil
		},
		RemoveAllFunc: func(path string) error {
			return nil
		},
	}
}

// NewMockCommandRunner creates a new MockCommandRunner with default behavior.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		LookPathFunc: func(file string) (string, error) {
			if file == "systemctl" || file == "service" || file == "rc-service" {
				return "/usr/bin/" + file, nil
			}
			return "", errors.New("command not found")
		},
		RunFunc: func(name string, args ...string) error {
			return nil
		},
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			return []byte{}, nil
		},
	}
}

// NewMockOSInterface creates a new MockOSInterface with default behavior.
func NewMockOSInterface() *MockOSInterface {
	return &MockOSInterface{
		// Don't set ExecutableFunc so that SetExecutable can work
		GetuidFunc: func() int {
			return 0
		},
		GetgidFunc: func() int {
			return 0
		},
		HostnameFunc: func() (string, error) {
			return "test-host", nil
		},
		GetenvFunc: func(key string) string {
			return ""
		},
		SetenvFunc: func(key, value string) error {
			return nil
		},
		UnsetenvFunc: func(key string) error {
			return nil
		},
		LookupEnvFunc: func(key string) (string, bool) {
			return "", false
		},
		UserHomeDirFunc: func() (string, error) {
			return "/home/test", nil
		},
		// Set default executable path
		executablePath: "/usr/local/bin/cloud-update",
	}
}
