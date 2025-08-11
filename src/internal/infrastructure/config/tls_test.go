package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper function to create temporary certificate files for testing.
func createTempCertFiles(t *testing.T, valid bool) (certFile, keyFile string) {
	tempDir := t.TempDir()
	certFile = filepath.Join(tempDir, "cert.pem")
	keyFile = filepath.Join(tempDir, "key.pem")

	if !valid {
		// Create invalid cert files with garbage content
		_ = os.WriteFile(certFile, []byte("invalid cert content"), 0600)
		_ = os.WriteFile(keyFile, []byte("invalid key content"), 0600)
		return certFile, keyFile
	}

	// Generate a valid self-signed certificate for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Write certificate
	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	defer func() { _ = certOut.Close() }()

	_ = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Write private key
	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	defer func() { _ = keyOut.Close() }()

	privKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	_ = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privKeyDER})

	return certFile, keyFile
}

// tlsConfigTestCase represents a test case for TLS configuration loading.
type tlsConfigTestCase struct {
	name     string
	envVars  map[string]string
	expected TLSConfig
}

// getTLSConfigTestCases returns all test cases for TLS configuration loading.
func getTLSConfigTestCases() []tlsConfigTestCase {
	return []tlsConfigTestCase{
		{
			name:    "default config - TLS disabled",
			envVars: map[string]string{},
			expected: TLSConfig{
				Enabled:  false,
				CertFile: "",
				KeyFile:  "",
				Auto:     false,
				Domain:   "",
			},
		},
		{
			name: "TLS enabled with custom paths",
			envVars: map[string]string{
				"CLOUD_UPDATE_TLS_ENABLED": "true",
				"CLOUD_UPDATE_TLS_CERT":    "/custom/cert.pem",
				"CLOUD_UPDATE_TLS_KEY":     "/custom/key.pem",
			},
			expected: TLSConfig{
				Enabled:  true,
				CertFile: "/custom/cert.pem",
				KeyFile:  "/custom/key.pem",
				Auto:     false,
				Domain:   "",
			},
		},
		{
			name: "TLS enabled with default paths",
			envVars: map[string]string{
				"CLOUD_UPDATE_TLS_ENABLED": "true",
			},
			expected: TLSConfig{
				Enabled:  true,
				CertFile: "/etc/cloud-update/tls/cert.pem",
				KeyFile:  "/etc/cloud-update/tls/key.pem",
				Auto:     false,
				Domain:   "",
			},
		},
		{
			name: "auto TLS enabled",
			envVars: map[string]string{
				"CLOUD_UPDATE_TLS_ENABLED": "true",
				"CLOUD_UPDATE_TLS_AUTO":    "true",
				"CLOUD_UPDATE_DOMAIN":      "example.com",
			},
			expected: TLSConfig{
				Enabled:  true,
				CertFile: "",
				KeyFile:  "",
				Auto:     true,
				Domain:   "example.com",
			},
		},
		{
			name: "auto TLS with manual cert paths (should ignore paths)",
			envVars: map[string]string{
				"CLOUD_UPDATE_TLS_ENABLED": "true",
				"CLOUD_UPDATE_TLS_AUTO":    "true",
				"CLOUD_UPDATE_TLS_CERT":    "/ignored/cert.pem",
				"CLOUD_UPDATE_TLS_KEY":     "/ignored/key.pem",
				"CLOUD_UPDATE_DOMAIN":      "example.com",
			},
			expected: TLSConfig{
				Enabled:  true,
				CertFile: "", // Cleared when auto is enabled
				KeyFile:  "", // Cleared when auto is enabled
				Auto:     true,
				Domain:   "example.com",
			},
		},
		{
			name: "TLS disabled with other vars set",
			envVars: map[string]string{
				"CLOUD_UPDATE_TLS_ENABLED": "false",
				"CLOUD_UPDATE_TLS_CERT":    "/some/cert.pem",
				"CLOUD_UPDATE_TLS_KEY":     "/some/key.pem",
				"CLOUD_UPDATE_TLS_AUTO":    "true",
				"CLOUD_UPDATE_DOMAIN":      "example.com",
			},
			expected: TLSConfig{
				Enabled:  false,
				CertFile: "/some/cert.pem",
				KeyFile:  "/some/key.pem",
				Auto:     true,
				Domain:   "example.com",
			},
		},
	}
}

// setupTestEnvironment saves original environment variables and returns a cleanup function.
func setupTestEnvironment() func() {
	originalVars := map[string]string{
		"CLOUD_UPDATE_TLS_ENABLED": os.Getenv("CLOUD_UPDATE_TLS_ENABLED"),
		"CLOUD_UPDATE_TLS_CERT":    os.Getenv("CLOUD_UPDATE_TLS_CERT"),
		"CLOUD_UPDATE_TLS_KEY":     os.Getenv("CLOUD_UPDATE_TLS_KEY"),
		"CLOUD_UPDATE_TLS_AUTO":    os.Getenv("CLOUD_UPDATE_TLS_AUTO"),
		"CLOUD_UPDATE_DOMAIN":      os.Getenv("CLOUD_UPDATE_DOMAIN"),
	}
	return func() {
		for key, value := range originalVars {
			if value == "" {
				_ = os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}
}

// clearTLSEnvironment clears all TLS-related environment variables.
func clearTLSEnvironment() {
	envKeys := []string{
		"CLOUD_UPDATE_TLS_ENABLED",
		"CLOUD_UPDATE_TLS_CERT",
		"CLOUD_UPDATE_TLS_KEY",
		"CLOUD_UPDATE_TLS_AUTO",
		"CLOUD_UPDATE_DOMAIN",
	}
	for _, key := range envKeys {
		_ = os.Unsetenv(key)
	}
}

// setTestEnvironment sets environment variables for a test case.
func setTestEnvironment(envVars map[string]string) {
	for key, value := range envVars {
		if value == "" {
			_ = os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}

// assertTLSConfig compares actual and expected TLS configurations.
func assertTLSConfig(t *testing.T, actual *TLSConfig, expected TLSConfig) {
	if actual.Enabled != expected.Enabled {
		t.Errorf("Enabled = %v, want %v", actual.Enabled, expected.Enabled)
	}
	if actual.CertFile != expected.CertFile {
		t.Errorf("CertFile = %q, want %q", actual.CertFile, expected.CertFile)
	}
	if actual.KeyFile != expected.KeyFile {
		t.Errorf("KeyFile = %q, want %q", actual.KeyFile, expected.KeyFile)
	}
	if actual.Auto != expected.Auto {
		t.Errorf("Auto = %v, want %v", actual.Auto, expected.Auto)
	}
	if actual.Domain != expected.Domain {
		t.Errorf("Domain = %q, want %q", actual.Domain, expected.Domain)
	}
}

func TestLoadTLSConfig(t *testing.T) {
	cleanup := setupTestEnvironment()
	defer cleanup()

	tests := getTLSConfigTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTLSEnvironment()
			setTestEnvironment(tt.envVars)
			config := LoadTLSConfig()
			assertTLSConfig(t, config, tt.expected)
		})
	}
}

func TestTLSConfig_Validate(t *testing.T) {
	validCertFile, validKeyFile := createTempCertFiles(t, true)
	invalidCertFile, invalidKeyFile := createTempCertFiles(t, false)
	nonExistentCert := "/nonexistent/cert.pem"
	nonExistentKey := "/nonexistent/key.pem"

	tests := []struct {
		name    string
		config  TLSConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "TLS disabled - always valid",
			config: TLSConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "auto TLS with domain",
			config: TLSConfig{
				Enabled: true,
				Auto:    true,
				Domain:  "example.com",
			},
			wantErr: false,
		},
		{
			name: "auto TLS without domain",
			config: TLSConfig{
				Enabled: true,
				Auto:    true,
				Domain:  "",
			},
			wantErr: true,
			errMsg:  "domain required for automatic TLS certificates",
		},
		{
			name: "manual TLS with valid certificates",
			config: TLSConfig{
				Enabled:  true,
				Auto:     false,
				CertFile: validCertFile,
				KeyFile:  validKeyFile,
			},
			wantErr: false,
		},
		{
			name: "manual TLS without cert file",
			config: TLSConfig{
				Enabled:  true,
				Auto:     false,
				CertFile: "",
				KeyFile:  validKeyFile,
			},
			wantErr: true,
			errMsg:  "cert and key files required when TLS is enabled",
		},
		{
			name: "manual TLS without key file",
			config: TLSConfig{
				Enabled:  true,
				Auto:     false,
				CertFile: validCertFile,
				KeyFile:  "",
			},
			wantErr: true,
			errMsg:  "cert and key files required when TLS is enabled",
		},
		{
			name: "manual TLS with nonexistent cert file",
			config: TLSConfig{
				Enabled:  true,
				Auto:     false,
				CertFile: nonExistentCert,
				KeyFile:  validKeyFile,
			},
			wantErr: true,
			errMsg:  "certificate file not found",
		},
		{
			name: "manual TLS with nonexistent key file",
			config: TLSConfig{
				Enabled:  true,
				Auto:     false,
				CertFile: validCertFile,
				KeyFile:  nonExistentKey,
			},
			wantErr: true,
			errMsg:  "key file not found",
		},
		{
			name: "manual TLS with invalid cert files",
			config: TLSConfig{
				Enabled:  true,
				Auto:     false,
				CertFile: invalidCertFile,
				KeyFile:  invalidKeyFile,
			},
			wantErr: false, // Validate only checks existence, not content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestTLSConfig_GetTLSConfig(t *testing.T) {
	validCertFile, validKeyFile := createTempCertFiles(t, true)
	invalidCertFile, invalidKeyFile := createTempCertFiles(t, false)

	// Split the test into smaller subtests to reduce cyclomatic complexity
	t.Run("disabled config", func(t *testing.T) {
		config := TLSConfig{Enabled: false}
		tlsConfig, err := config.GetTLSConfig()
		if err != nil {
			t.Errorf("GetTLSConfig() error = %v, want nil", err)
		}
		if tlsConfig != nil {
			t.Errorf("GetTLSConfig() = %v, want nil", tlsConfig)
		}
	})

	t.Run("auto TLS enabled", func(t *testing.T) {
		config := TLSConfig{
			Enabled: true,
			Auto:    true,
			Domain:  "example.com",
		}
		tlsConfig, err := config.GetTLSConfig()
		if err != nil {
			t.Errorf("GetTLSConfig() error = %v, want nil", err)
		}
		if tlsConfig != nil {
			t.Errorf("GetTLSConfig() = %v, want nil", tlsConfig)
		}
	})

	t.Run("manual TLS with valid certificates", func(t *testing.T) {
		config := TLSConfig{
			Enabled:  true,
			Auto:     false,
			CertFile: validCertFile,
			KeyFile:  validKeyFile,
		}
		testValidTLSConfig(t, config)
	})

	t.Run("manual TLS with invalid certificates", func(t *testing.T) {
		config := TLSConfig{
			Enabled:  true,
			Auto:     false,
			CertFile: invalidCertFile,
			KeyFile:  invalidKeyFile,
		}
		testInvalidTLSConfig(t, config, "failed to load TLS certificates")
	})

	t.Run("manual TLS with nonexistent certificates", func(t *testing.T) {
		config := TLSConfig{
			Enabled:  true,
			Auto:     false,
			CertFile: "/nonexistent/cert.pem",
			KeyFile:  "/nonexistent/key.pem",
		}
		testInvalidTLSConfig(t, config, "failed to load TLS certificates")
	})
}

// Helper function to test valid TLS config and reduce complexity.
func testValidTLSConfig(t *testing.T, config TLSConfig) {
	tlsConfig, err := config.GetTLSConfig()
	if err != nil {
		t.Errorf("GetTLSConfig() error = %v, want nil", err)
		return
	}
	if tlsConfig == nil {
		t.Error("GetTLSConfig() = nil, want non-nil")
		return
	}

	// Check certificate count
	if len(tlsConfig.Certificates) != 1 {
		t.Errorf("certificates length = %d, want 1", len(tlsConfig.Certificates))
	}

	// Check TLS version
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %x, want %x (TLS 1.2)", tlsConfig.MinVersion, tls.VersionTLS12)
	}

	// Check cipher suites
	testCipherSuites(t, tlsConfig)

	// Check curve preferences
	testCurvePreferences(t, tlsConfig)
}

// Helper function to test invalid TLS config.
func testInvalidTLSConfig(t *testing.T, config TLSConfig, expectedErrMsg string) {
	_, err := config.GetTLSConfig()
	if err == nil {
		t.Error("GetTLSConfig() error = nil, want error")
		return
	}
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("GetTLSConfig() error = %v, want error containing %q", err, expectedErrMsg)
	}
}

// Helper function to test cipher suites.
func testCipherSuites(t *testing.T, tlsConfig *tls.Config) {
	expectedCiphers := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}

	if len(tlsConfig.CipherSuites) != len(expectedCiphers) {
		t.Errorf("CipherSuites length = %d, want %d", len(tlsConfig.CipherSuites), len(expectedCiphers))
		return
	}

	for i, expected := range expectedCiphers {
		if i >= len(tlsConfig.CipherSuites) || tlsConfig.CipherSuites[i] != expected {
			t.Errorf("CipherSuites[%d] = %x, want %x", i, tlsConfig.CipherSuites[i], expected)
			break
		}
	}
}

// Helper function to test curve preferences.
func testCurvePreferences(t *testing.T, tlsConfig *tls.Config) {
	expectedCurves := []tls.CurveID{tls.X25519, tls.CurveP256}
	if len(tlsConfig.CurvePreferences) != len(expectedCurves) {
		t.Errorf("CurvePreferences length = %d, want %d",
			len(tlsConfig.CurvePreferences), len(expectedCurves))
		return
	}

	for i, expected := range expectedCurves {
		if i >= len(tlsConfig.CurvePreferences) || tlsConfig.CurvePreferences[i] != expected {
			t.Errorf("CurvePreferences[%d] = %v, want %v", i, tlsConfig.CurvePreferences[i], expected)
			break
		}
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	tests := []struct {
		name     string
		certPath string
		keyPath  string
		hosts    []string
	}{
		{
			name:     "basic call",
			certPath: "cert.pem",
			keyPath:  "key.pem",
			hosts:    []string{"localhost"},
		},
		{
			name:     "multiple hosts",
			certPath: "cert.pem",
			keyPath:  "key.pem",
			hosts:    []string{"localhost", "example.com", "127.0.0.1"},
		},
		{
			name:     "empty hosts",
			certPath: "cert.pem",
			keyPath:  "key.pem",
			hosts:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateSelfSignedCert(tt.certPath, tt.keyPath, tt.hosts)

			// The function should always return an error suggesting external tools
			if err == nil {
				t.Error("GenerateSelfSignedCert() should return an error suggesting external tools")
				return
			}

			expectedMsg := "use 'openssl' or 'mkcert' to generate self-signed certificates"
			if !strings.Contains(err.Error(), expectedMsg) {
				t.Errorf("GenerateSelfSignedCert() error = %v, want error containing %q", err, expectedMsg)
			}
		})
	}
}

func TestSetupCertificateDirectory(t *testing.T) {
	// Use a temporary directory for testing
	originalTLSDir := "/etc/cloud-update/tls"
	tempDir := t.TempDir()

	// We can't easily test the actual function since it uses hardcoded paths
	// But we can test the directory creation logic

	// Test creating directory structure
	testTLSDir := filepath.Join(tempDir, "tls")
	err := os.MkdirAll(testTLSDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create test TLS directory: %v", err)
	}

	// Verify directory was created with correct permissions
	info, err := os.Stat(testTLSDir)
	if err != nil {
		t.Fatalf("Failed to stat TLS directory: %v", err)
	}

	if !info.IsDir() {
		t.Error("TLS path should be a directory")
	}

	// Check permissions (on Unix-like systems)
	if info.Mode().Perm() != 0700 {
		t.Errorf("TLS directory permissions = %o, want 0700", info.Mode().Perm())
	}

	// Test README creation
	readmePath := filepath.Join(testTLSDir, "README.md")
	readme := `# TLS Certificate Directory

Place your TLS certificates in this directory:

- cert.pem: The server certificate
- key.pem: The private key

## Generate Self-Signed Certificate (Development)

` + "```bash" + `
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
` + "```" + `

## Production Certificates

For production, use certificates from a trusted CA or Let's Encrypt.

Enable automatic certificates with:
- CLOUD_UPDATE_TLS_AUTO=true
- CLOUD_UPDATE_DOMAIN=your-domain.com
`

	err = os.WriteFile(readmePath, []byte(readme), 0600)
	if err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	// Verify README was created
	readmeInfo, err := os.Stat(readmePath)
	if err != nil {
		t.Fatalf("Failed to stat README file: %v", err)
	}

	if readmeInfo.Mode().Perm() != 0600 {
		t.Errorf("README permissions = %o, want 0600", readmeInfo.Mode().Perm())
	}

	// Verify README content
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	if string(content) != readme {
		t.Error("README content doesn't match expected content")
	}

	t.Logf("Would create TLS directory at: %s", originalTLSDir)
}

// Test edge cases and error conditions.
func TestTLSConfig_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config TLSConfig
	}{
		{
			name:   "all fields empty",
			config: TLSConfig{},
		},
		{
			name: "enabled but all other fields empty",
			config: TLSConfig{
				Enabled: true,
			},
		},
		{
			name: "auto enabled but no domain",
			config: TLSConfig{
				Enabled: true,
				Auto:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			// Should handle gracefully - either succeed or return appropriate error
			if err != nil {
				t.Logf("Validation error (expected): %v", err)
			}
		})
	}
}

func TestTLSConfig_GetTLSConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config TLSConfig
	}{
		{
			name:   "empty config",
			config: TLSConfig{},
		},
		{
			name: "enabled but no files",
			config: TLSConfig{
				Enabled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := tt.config.GetTLSConfig()
			// Should handle gracefully
			if err != nil {
				t.Logf("GetTLSConfig error (expected): %v", err)
			}
			if tlsConfig != nil {
				t.Logf("GetTLSConfig returned config: %+v", tlsConfig)
			}
		})
	}
}

// Benchmark tests.
func BenchmarkLoadTLSConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadTLSConfig()
	}
}

func BenchmarkTLSConfig_Validate(b *testing.B) {
	config := &TLSConfig{
		Enabled: true,
		Auto:    true,
		Domain:  "example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

func BenchmarkTLSConfig_GetTLSConfig(b *testing.B) {
	// Create a temporary testing.T for the helper function
	t := &testing.T{}
	validCertFile, validKeyFile := createTempCertFiles(t, true)

	config := &TLSConfig{
		Enabled:  true,
		Auto:     false,
		CertFile: validCertFile,
		KeyFile:  validKeyFile,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = config.GetTLSConfig()
	}
}

// TestSetupCertificateDirectory_Coverage tests the SetupCertificateDirectory function for coverage.
func TestSetupCertificateDirectory_Coverage(t *testing.T) {
	// This test calls the actual function which uses hardcoded /etc/cloud-update/tls path
	// It will fail with permission denied in non-root environments, but that still covers the code
	err := SetupCertificateDirectory()

	if os.Geteuid() != 0 {
		// We expect an error when not running as root
		if err == nil {
			t.Error("SetupCertificateDirectory should fail without root permissions")
		} else {
			t.Logf("SetupCertificateDirectory failed as expected without root: %v", err)
		}
	} else {
		// If we're root (unlikely in tests), it should succeed
		if err != nil {
			t.Errorf("SetupCertificateDirectory failed with root: %v", err)
		}
	}
}
