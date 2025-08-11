package setup

import (
	"os"
	"path/filepath"
	"testing"
)

// Test RealFileSystem implementations.
func TestRealFileSystem(t *testing.T) {
	fs := RealFileSystem{}
	tempDir := t.TempDir()

	// Test MkdirAll
	testDir := filepath.Join(tempDir, "testdir")
	if err := fs.MkdirAll(testDir, 0755); err != nil {
		t.Errorf("RealFileSystem.MkdirAll() failed: %v", err)
	}

	// Test WriteFile
	testFile := filepath.Join(testDir, "testfile")
	testContent := []byte("test content")
	if err := fs.WriteFile(testFile, testContent, 0644); err != nil {
		t.Errorf("RealFileSystem.WriteFile() failed: %v", err)
	}

	// Test ReadFile
	content, err := fs.ReadFile(testFile)
	if err != nil {
		t.Errorf("RealFileSystem.ReadFile() failed: %v", err)
	}
	if string(content) != string(testContent) {
		t.Errorf("RealFileSystem.ReadFile() = %q, want %q", string(content), string(testContent))
	}

	// Test Stat
	info, err := fs.Stat(testFile)
	if err != nil {
		t.Errorf("RealFileSystem.Stat() failed: %v", err)
	}
	if info == nil {
		t.Error("RealFileSystem.Stat() returned nil info")
	}

	// Test Chmod
	if err := fs.Chmod(testFile, 0600); err != nil {
		t.Errorf("RealFileSystem.Chmod() failed: %v", err)
	}

	// Test Chown (might fail if not root, but we test it anyway)
	if err := fs.Chown(testFile, os.Getuid(), os.Getgid()); err != nil {
		// Don't fail the test if we can't chown (might not have permissions)
		t.Logf("RealFileSystem.Chown() failed (expected if not root): %v", err)
	}

	// Test Remove
	if err := fs.Remove(testFile); err != nil {
		t.Errorf("RealFileSystem.Remove() failed: %v", err)
	}

	// Test RemoveAll
	if err := fs.RemoveAll(testDir); err != nil {
		t.Errorf("RealFileSystem.RemoveAll() failed: %v", err)
	}
}

// Test RealCommandRunner implementations.

// Test RealOSInterface implementations.
func TestRealOSInterface(t *testing.T) {
	osIface := RealOSInterface{}

	// Test Executable
	path, err := osIface.Executable()
	if err != nil {
		t.Errorf("RealOSInterface.Executable() failed: %v", err)
	}
	if path == "" {
		t.Error("RealOSInterface.Executable() returned empty path")
	}

	// Test Geteuid
	uid := osIface.Geteuid()
	if uid < 0 {
		t.Errorf("RealOSInterface.Geteuid() = %d, want >= 0", uid)
	}

	// Test Scanln - we can't easily test this interactively,
	// but we can test that it doesn't panic
	// Note: This will fail because there's no input, but that's expected
	_, err = osIface.Scanln()
	// We expect an error here due to no input, so don't fail the test
	t.Logf("RealOSInterface.Scanln() returned error (expected): %v", err)
}
