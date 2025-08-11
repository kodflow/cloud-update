package setup

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MockFileSystem implements FileSystem interface for testing.
type MockFileSystem struct {
	mu           sync.Mutex
	files        map[string][]byte
	dirs         map[string]os.FileMode
	shouldFail   map[string]error
	chownCalls   []ChownCall
	statFailures map[string]error
}

type ChownCall struct {
	Name string
	UID  int
	GID  int
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:        make(map[string][]byte),
		dirs:         make(map[string]os.FileMode),
		shouldFail:   make(map[string]error),
		chownCalls:   make([]ChownCall, 0),
		statFailures: make(map[string]error),
	}
}

func (m *MockFileSystem) AddFile(path string, content []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = content
}

func (m *MockFileSystem) SetShouldFail(operation, path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := operation + ":" + path
	m.shouldFail[key] = err
}

func (m *MockFileSystem) SetStatError(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statFailures[path] = err
}

func (m *MockFileSystem) GetChownCalls() []ChownCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]ChownCall, len(m.chownCalls))
	copy(calls, m.chownCalls)
	return calls
}

func (m *MockFileSystem) FileExists(path string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.files[path]
	return exists
}

func (m *MockFileSystem) DirExists(path string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.dirs[path]
	return exists
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.shouldFail["MkdirAll:"+path]; exists {
		return err
	}

	m.dirs[path] = perm
	return nil
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.shouldFail["WriteFile:"+filename]; exists {
		return err
	}

	m.files[filename] = make([]byte, len(data))
	copy(m.files[filename], data)
	return nil
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.shouldFail["ReadFile:"+filename]; exists {
		return nil, err
	}

	if data, exists := m.files[filename]; exists {
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil
	}

	return nil, &os.PathError{Op: "open", Path: filename, Err: os.ErrNotExist}
}

func (m *MockFileSystem) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.shouldFail["Remove:"+name]; exists {
		return err
	}

	delete(m.files, name)
	return nil
}

func (m *MockFileSystem) RemoveAll(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.shouldFail["RemoveAll:"+path]; exists {
		return err
	}

	// Remove all files and directories under the path
	for file := range m.files {
		if strings.HasPrefix(file, path) {
			delete(m.files, file)
		}
	}
	for dir := range m.dirs {
		if strings.HasPrefix(dir, path) {
			delete(m.dirs, dir)
		}
	}

	return nil
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.statFailures[name]; exists {
		return nil, err
	}

	if err, exists := m.shouldFail["Stat:"+name]; exists {
		return nil, err
	}

	if _, exists := m.files[name]; exists {
		return &mockFileInfo{name: filepath.Base(name), isDir: false, mode: 0644}, nil
	}

	if mode, exists := m.dirs[name]; exists {
		return &mockFileInfo{name: filepath.Base(name), isDir: true, mode: mode}, nil
	}

	// Check if any file exists under this directory path (for directory detection)
	for filePath := range m.files {
		if strings.HasPrefix(filePath, name+"/") || strings.HasPrefix(filePath, name) {
			return &mockFileInfo{name: filepath.Base(name), isDir: true, mode: 0755}, nil
		}
	}

	return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
}

func (m *MockFileSystem) Chmod(name string, mode os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.shouldFail["Chmod:"+name]; exists {
		return err
	}

	return nil
}

func (m *MockFileSystem) Chown(name string, uid, gid int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.shouldFail["Chown:"+name]; exists {
		return err
	}

	m.chownCalls = append(m.chownCalls, ChownCall{
		Name: name,
		UID:  uid,
		GID:  gid,
	})

	return nil
}

type mockFileInfo struct {
	name  string
	isDir bool
	mode  os.FileMode
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// MockCommandRunner implements CommandRunner interface for testing.
type MockCommandRunner struct {
	mu           sync.Mutex
	commands     []Command
	shouldFail   map[string]error
	outputs      map[string][]byte
	lookupPaths  map[string]string
	lookupErrors map[string]error
}

type Command struct {
	Name string
	Args []string
}

func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		commands:     make([]Command, 0),
		shouldFail:   make(map[string]error),
		outputs:      make(map[string][]byte),
		lookupPaths:  make(map[string]string),
		lookupErrors: make(map[string]error),
	}
}

func (m *MockCommandRunner) SetShouldFail(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail[name] = err
}

func (m *MockCommandRunner) SetShouldFailWithArgs(name string, args []string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := name + ":" + strings.Join(args, ":")
	m.shouldFail[key] = err
}

func (m *MockCommandRunner) SetOutput(name string, output []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs[name] = output
}

func (m *MockCommandRunner) SetLookupPath(file, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lookupPaths[file] = path
}

func (m *MockCommandRunner) SetLookupError(file string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lookupErrors[file] = err
}

func (m *MockCommandRunner) GetCommands() []Command {
	m.mu.Lock()
	defer m.mu.Unlock()
	commands := make([]Command, len(m.commands))
	copy(commands, m.commands)
	return commands
}

func (m *MockCommandRunner) Run(name string, args ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.commands = append(m.commands, Command{Name: name, Args: args})

	// Check for specific command+args combination first
	key := name + ":" + strings.Join(args, ":")
	if err, exists := m.shouldFail[key]; exists {
		return err
	}

	// Fallback to command name only
	if err, exists := m.shouldFail[name]; exists {
		return err
	}

	return nil
}

func (m *MockCommandRunner) Output(name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.commands = append(m.commands, Command{Name: name, Args: args})

	if err, exists := m.shouldFail[name]; exists {
		return nil, err
	}

	if output, exists := m.outputs[name]; exists {
		result := make([]byte, len(output))
		copy(result, output)
		return result, nil
	}

	return []byte("mock output"), nil
}

func (m *MockCommandRunner) LookPath(file string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.lookupErrors[file]; exists {
		return "", err
	}

	if path, exists := m.lookupPaths[file]; exists {
		return path, nil
	}

	return "/usr/bin/" + file, nil
}

// MockOSInterface implements OSInterface for testing.
type MockOSInterface struct {
	mu            sync.Mutex
	executable    string
	executableErr error
	euid          int
	scanInput     string
	scanErr       error
	scanCallCount int
}

func NewMockOSInterface() *MockOSInterface {
	return &MockOSInterface{
		executable: "/test/path/cloud-update",
		euid:       0, // Default to root for tests
	}
}

func (m *MockOSInterface) SetExecutable(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executable = path
	m.executableErr = err
}

func (m *MockOSInterface) SetEuid(euid int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.euid = euid
}

func (m *MockOSInterface) SetScanInput(input string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanInput = input
	m.scanErr = err
}

func (m *MockOSInterface) GetScanCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scanCallCount
}

func (m *MockOSInterface) Executable() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executable, m.executableErr
}

func (m *MockOSInterface) Geteuid() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.euid
}

func (m *MockOSInterface) Scanln(a ...interface{}) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.scanCallCount++

	if m.scanErr != nil {
		return 0, m.scanErr
	}

	if len(a) > 0 {
		if str, ok := a[0].(*string); ok {
			*str = m.scanInput
			return 1, nil
		}
	}

	return 0, nil
}
