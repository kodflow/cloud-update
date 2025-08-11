package setup

import (
	"fmt"
	"strings"
	"testing"
)

func TestEmbeddedSystemdService(t *testing.T) {
	if SystemdService == "" {
		t.Error("SystemdService should not be empty")
	}

	// Check that it contains expected systemd service elements
	expectedElements := []string{
		"[Unit]",
		"[Service]",
		"[Install]",
		"Description=",
		"ExecStart=",
	}

	for _, element := range expectedElements {
		if !strings.Contains(SystemdService, element) {
			t.Errorf("SystemdService should contain %q", element)
		}
	}

	// Should be a valid systemd service file structure
	lines := strings.Split(SystemdService, "\n")
	if len(lines) < 5 {
		t.Errorf("SystemdService should have at least 5 lines, got %d", len(lines))
	}

	// Test specific systemd service expectations
	if !strings.Contains(SystemdService, "cloud-update") {
		t.Error("SystemdService should reference cloud-update")
	}

	t.Logf("SystemdService content preview: %q...", truncateString(SystemdService, 100))
}

func TestEmbeddedOpenRCScript(t *testing.T) {
	if OpenRCScript == "" {
		t.Error("OpenRCScript should not be empty")
	}

	// Check that it contains expected OpenRC script elements
	expectedElements := []string{
		"#!/",
		"stop()",
		"depend()",
	}

	for _, element := range expectedElements {
		if !strings.Contains(OpenRCScript, element) {
			t.Errorf("OpenRCScript should contain %q", element)
		}
	}

	// Should start with shebang
	if !strings.HasPrefix(OpenRCScript, "#!") {
		t.Error("OpenRCScript should start with shebang")
	}

	// Should be executable script
	lines := strings.Split(OpenRCScript, "\n")
	if len(lines) < 5 {
		t.Errorf("OpenRCScript should have at least 5 lines, got %d", len(lines))
	}

	// Test specific OpenRC script expectations
	if !strings.Contains(OpenRCScript, "cloud-update") {
		t.Error("OpenRCScript should reference cloud-update")
	}

	t.Logf("OpenRCScript content preview: %q...", truncateString(OpenRCScript, 100))
}

func TestEmbeddedSysVInitScript(t *testing.T) {
	if SysVInitScript == "" {
		t.Error("SysVInitScript should not be empty")
	}

	// Check that it contains expected SysV init script elements
	expectedElements := []string{
		"#!/",
		"start)",
		"stop)",
		"restart|force-reload)",
		"status)",
	}

	for _, element := range expectedElements {
		if !strings.Contains(SysVInitScript, element) {
			t.Errorf("SysVInitScript should contain %q", element)
		}
	}

	// Should start with shebang
	if !strings.HasPrefix(SysVInitScript, "#!") {
		t.Error("SysVInitScript should start with shebang")
	}

	// Should be executable script
	lines := strings.Split(SysVInitScript, "\n")
	if len(lines) < 10 {
		t.Errorf("SysVInitScript should have at least 10 lines, got %d", len(lines))
	}

	// Test specific SysV init script expectations
	if !strings.Contains(SysVInitScript, "cloud-update") {
		t.Error("SysVInitScript should reference cloud-update")
	}

	// Should have standard SysV init structure
	if !strings.Contains(SysVInitScript, "case") {
		t.Error("SysVInitScript should contain case statement")
	}

	t.Logf("SysVInitScript content preview: %q...", truncateString(SysVInitScript, 100))
}

func TestEmbeddedScripts_Consistency(t *testing.T) {
	// All scripts should reference the same binary name
	binaryName := "cloud-update"

	scripts := map[string]string{
		"SystemdService": SystemdService,
		"OpenRCScript":   OpenRCScript,
		"SysVInitScript": SysVInitScript,
	}

	for name, script := range scripts {
		if !strings.Contains(script, binaryName) {
			t.Errorf("%s should contain binary name %q", name, binaryName)
		}
	}

	// All should be non-empty
	for name, script := range scripts {
		if len(script) == 0 {
			t.Errorf("%s should not be empty", name)
		}
	}

	// All should have reasonable length (not truncated)
	for name, script := range scripts {
		if len(script) < 50 {
			t.Errorf("%s seems too short (%d chars), might be truncated", name, len(script))
		}
	}
}

func TestEmbeddedScripts_NoSecrets(t *testing.T) {
	// Ensure scripts don't contain hardcoded secrets or sensitive data
	sensitivePatterns := []string{
		"password",
		"secret",
		"token",
		"key=",
		"pass=",
		"pwd=",
	}

	scripts := map[string]string{
		"SystemdService": SystemdService,
		"OpenRCScript":   OpenRCScript,
		"SysVInitScript": SysVInitScript,
	}

	for scriptName, script := range scripts {
		lowerScript := strings.ToLower(script)
		for _, pattern := range sensitivePatterns {
			if strings.Contains(lowerScript, pattern) {
				// Ignore environment variable references like $SECRET or $CLOUD_UPDATE_SECRET
				if !strings.Contains(lowerScript, "$"+pattern) &&
					!strings.Contains(lowerScript, "cloud_update_"+pattern) &&
					!strings.Contains(lowerScript, "export cloud_update_"+pattern) {
					t.Errorf("%s should not contain potentially sensitive pattern %q", scriptName, pattern)
				}
			}
		}
	}
}

func TestEmbeddedScripts_Paths(t *testing.T) {
	// Test that scripts reference appropriate paths
	expectedPaths := []string{
		"/usr/local/bin",
		"/etc/cloud-update",
	}

	scripts := map[string]string{
		"SystemdService": SystemdService,
		"OpenRCScript":   OpenRCScript,
		"SysVInitScript": SysVInitScript,
	}

	for scriptName, script := range scripts {
		foundPath := false
		for _, expectedPath := range expectedPaths {
			if strings.Contains(script, expectedPath) {
				foundPath = true
				break
			}
		}
		if !foundPath {
			t.Errorf("%s should contain at least one of the expected paths %v", scriptName, expectedPaths)
		}
	}
}

func TestEmbeddedSystemdService_Specific(t *testing.T) {
	tests := []struct {
		name          string
		expectedText  string
		shouldContain bool
		description   string
	}{
		{
			name:          "service type",
			expectedText:  "Type=",
			shouldContain: true,
			description:   "Systemd service should specify service type",
		},
		{
			name:          "wanted by target",
			expectedText:  "WantedBy=",
			shouldContain: true,
			description:   "Systemd service should specify WantedBy target",
		},
		{
			name:          "restart policy",
			expectedText:  "Restart=",
			shouldContain: true,
			description:   "Systemd service should have restart policy",
		},
		{
			name:          "user specification",
			expectedText:  "User=",
			shouldContain: true,
			description:   "Systemd service should specify user",
		},
		{
			name:          "exec start path",
			expectedText:  "/usr/local/bin/cloud-update",
			shouldContain: true,
			description:   "Systemd service should reference correct binary path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contains := strings.Contains(SystemdService, tt.expectedText)
			if contains != tt.shouldContain {
				if tt.shouldContain {
					t.Errorf("SystemdService should contain %q - %s", tt.expectedText, tt.description)
				} else {
					t.Errorf("SystemdService should not contain %q - %s", tt.expectedText, tt.description)
				}
			}
		})
	}
}

func TestEmbeddedOpenRCScript_Specific(t *testing.T) {
	tests := []struct {
		name          string
		expectedText  string
		shouldContain bool
		description   string
	}{
		{
			name:          "command variable",
			expectedText:  "command=",
			shouldContain: true,
			description:   "OpenRC script should define command variable",
		},
		{
			name:          "pidfile variable",
			expectedText:  "pidfile=",
			shouldContain: true,
			description:   "OpenRC script should define pidfile variable",
		},
		{
			name:          "depends function",
			expectedText:  "depend() {",
			shouldContain: true,
			description:   "OpenRC script should have depend function",
		},
		{
			name:          "need net dependency",
			expectedText:  "need net",
			shouldContain: true,
			description:   "OpenRC script should depend on network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contains := strings.Contains(OpenRCScript, tt.expectedText)
			if contains != tt.shouldContain {
				if tt.shouldContain {
					t.Errorf("OpenRCScript should contain %q - %s", tt.expectedText, tt.description)
				} else {
					t.Errorf("OpenRCScript should not contain %q - %s", tt.expectedText, tt.description)
				}
			}
		})
	}
}

func TestEmbeddedSysVInitScript_Specific(t *testing.T) {
	tests := []struct {
		name          string
		expectedText  string
		shouldContain bool
		description   string
	}{
		{
			name:          "pidfile variable",
			expectedText:  "PIDFILE=",
			shouldContain: true,
			description:   "SysV init script should define pidfile variable",
		},
		{
			name:          "daemon variable",
			expectedText:  "DAEMON=",
			shouldContain: true,
			description:   "SysV init script should define daemon variable",
		},
		{
			name:          "case esac structure",
			expectedText:  "esac",
			shouldContain: true,
			description:   "SysV init script should have case statement",
		},
		{
			name:          "return codes",
			expectedText:  "exit $",
			shouldContain: true,
			description:   "SysV init script should exit with proper return codes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contains := strings.Contains(SysVInitScript, tt.expectedText)
			if contains != tt.shouldContain {
				if tt.shouldContain {
					t.Errorf("SysVInitScript should contain %q - %s", tt.expectedText, tt.description)
				} else {
					t.Errorf("SysVInitScript should not contain %q - %s", tt.expectedText, tt.description)
				}
			}
		})
	}
}

func TestEmbeddedScripts_ValidSyntax(t *testing.T) {
	// Test that scripts have basic valid syntax
	tests := []struct {
		name   string
		script string
		checks []func(string) error
	}{
		{
			name:   "SystemdService",
			script: SystemdService,
			checks: []func(string) error{
				validateSystemdSyntax,
			},
		},
		{
			name:   "OpenRCScript",
			script: OpenRCScript,
			checks: []func(string) error{
				validateShellSyntax,
			},
		},
		{
			name:   "SysVInitScript",
			script: SysVInitScript,
			checks: []func(string) error{
				validateShellSyntax,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, check := range tt.checks {
				if err := check(tt.script); err != nil {
					t.Errorf("%s syntax validation failed: %v", tt.name, err)
				}
			}
		})
	}
}

// Helper functions for validation.

func validateSystemdSyntax(content string) error {
	// Basic systemd unit file validation
	requiredSections := []string{"[Unit]", "[Service]", "[Install]"}

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			return fmt.Errorf("missing required section: %s", section)
		}
	}

	// Check for basic structure
	lines := strings.Split(content, "\n")
	inSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inSection = true
			continue
		}

		if inSection && !strings.Contains(line, "=") {
			// Non-empty line in section should contain =
			if line != "" {
				return fmt.Errorf("invalid line in systemd unit: %s", line)
			}
		}
	}

	return nil
}

func validateShellSyntax(content string) error {
	// Basic shell script validation
	if !strings.HasPrefix(content, "#!") {
		return fmt.Errorf("shell script should start with shebang")
	}

	// Check for balanced parentheses in functions
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Look for function definitions
		if strings.Contains(line, "() {") || strings.Contains(line, ")") && strings.Contains(line, "{") {
			// This is a basic check - more sophisticated validation would require parsing
			if strings.Count(line, "(") != strings.Count(line, ")") {
				return fmt.Errorf("unbalanced parentheses on line %d: %s", i+1, line)
			}
		}
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Test that embedded files are actually embedded (not empty due to build issues).
func TestEmbeddedFiles_NotEmpty(t *testing.T) {
	scripts := map[string]string{
		"SystemdService": SystemdService,
		"OpenRCScript":   OpenRCScript,
		"SysVInitScript": SysVInitScript,
	}

	for name, content := range scripts {
		if content == "" {
			t.Errorf("Embedded %s is empty - check embed paths and build process", name)
		} else {
			t.Logf("%s size: %d bytes", name, len(content))
		}
	}
}

func TestEmbeddedFiles_ValidContent(t *testing.T) {
	// Test that embedded files contain expected markers
	if !strings.Contains(SystemdService, "systemd") && !strings.Contains(SystemdService, "Service") {
		t.Error("SystemdService doesn't appear to be a systemd service file")
	}

	if !strings.Contains(OpenRCScript, "openrc") && !strings.Contains(OpenRCScript, "#!/") {
		t.Error("OpenRCScript doesn't appear to be an OpenRC script")
	}

	if !strings.Contains(SysVInitScript, "init") && !strings.Contains(SysVInitScript, "#!/") {
		t.Error("SysVInitScript doesn't appear to be a SysV init script")
	}
}

// Benchmark tests.
func BenchmarkEmbeddedAccess(b *testing.B) {
	b.Run("SystemdService", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = SystemdService
		}
	})

	b.Run("OpenRCScript", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = OpenRCScript
		}
	})

	b.Run("SysVInitScript", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = SysVInitScript
		}
	})
}

func BenchmarkEmbeddedContentLength(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = len(SystemdService)
		_ = len(OpenRCScript)
		_ = len(SysVInitScript)
	}
}

// Test edge cases.
func TestEmbeddedFiles_LineEndings(t *testing.T) {
	scripts := map[string]string{
		"SystemdService": SystemdService,
		"OpenRCScript":   OpenRCScript,
		"SysVInitScript": SysVInitScript,
	}

	for name, content := range scripts {
		// Check for consistent line endings
		if strings.Contains(content, "\r\n") {
			t.Logf("%s contains Windows line endings", name)
		}

		if strings.Contains(content, "\n") {
			t.Logf("%s contains Unix line endings", name)
		}

		// Ensure it has some line breaks
		if !strings.Contains(content, "\n") {
			t.Errorf("%s should contain line breaks", name)
		}
	}
}

func TestEmbeddedFiles_Encoding(t *testing.T) {
	scripts := map[string]string{
		"SystemdService": SystemdService,
		"OpenRCScript":   OpenRCScript,
		"SysVInitScript": SysVInitScript,
	}

	for name, content := range scripts {
		// Check that content is valid UTF-8
		if !isValidUTF8(content) {
			t.Errorf("%s contains invalid UTF-8", name)
		}

		// Check for null bytes (shouldn't be in text files)
		if strings.Contains(content, "\x00") {
			t.Errorf("%s contains null bytes", name)
		}
	}
}

func isValidUTF8(s string) bool {
	return len(s) == len([]rune(s))
}
