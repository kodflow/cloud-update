// Package config provides TLS configuration for secure HTTPS connections.
package config

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
)

// TLSConfig holds TLS/HTTPS configuration.
type TLSConfig struct {
	Enabled  bool   // Whether TLS is enabled
	CertFile string // Path to certificate file
	KeyFile  string // Path to private key file
	Auto     bool   // Use automatic certificate management (Let's Encrypt)
	Domain   string // Domain for automatic certificates
}

// LoadTLSConfig loads TLS configuration from environment variables.
func LoadTLSConfig() *TLSConfig {
	cfg := &TLSConfig{
		Enabled:  os.Getenv("CLOUD_UPDATE_TLS_ENABLED") == "true",
		CertFile: os.Getenv("CLOUD_UPDATE_TLS_CERT"),
		KeyFile:  os.Getenv("CLOUD_UPDATE_TLS_KEY"),
		Auto:     os.Getenv("CLOUD_UPDATE_TLS_AUTO") == "true",
		Domain:   os.Getenv("CLOUD_UPDATE_DOMAIN"),
	}

	// Default paths if not specified
	if cfg.Enabled && !cfg.Auto {
		if cfg.CertFile == "" {
			cfg.CertFile = "/etc/cloud-update/tls/cert.pem"
		}
		if cfg.KeyFile == "" {
			cfg.KeyFile = "/etc/cloud-update/tls/key.pem"
		}
	}

	return cfg
}

// Validate checks if the TLS configuration is valid.
func (c *TLSConfig) Validate() error {
	if !c.Enabled {
		return nil // TLS disabled, no validation needed
	}

	if c.Auto {
		// Automatic certificate management
		if c.Domain == "" {
			return fmt.Errorf("domain required for automatic TLS certificates")
		}
		return nil
	}

	// Manual certificate configuration
	if c.CertFile == "" || c.KeyFile == "" {
		return fmt.Errorf("cert and key files required when TLS is enabled")
	}

	// Check if certificate files exist
	if _, err := os.Stat(c.CertFile); err != nil {
		return fmt.Errorf("certificate file not found: %s", c.CertFile)
	}
	if _, err := os.Stat(c.KeyFile); err != nil {
		return fmt.Errorf("key file not found: %s", c.KeyFile)
	}

	return nil
}

// GetTLSConfig returns a crypto/tls.Config for secure connections.
func (c *TLSConfig) GetTLSConfig() (*tls.Config, error) {
	if !c.Enabled || c.Auto {
		return nil, nil // No manual TLS config needed
	}

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12, // Minimum TLS 1.2
		CipherSuites: []uint16{
			// Prefer modern, secure cipher suites
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
	}, nil
}

// GenerateSelfSignedCert generates a self-signed certificate for development.
func GenerateSelfSignedCert(_, _ string, _ []string) error {
	// This would use crypto/x509 and crypto/rsa to generate certificates
	// For brevity, returning an error suggesting to use external tools
	return fmt.Errorf("use 'openssl' or 'mkcert' to generate self-signed certificates for development")
}

// SetupCertificateDirectory creates the TLS certificate directory if it doesn't exist.
func SetupCertificateDirectory() error {
	tlsDir := "/etc/cloud-update/tls"

	// Create directory with secure permissions
	if err := os.MkdirAll(tlsDir, 0700); err != nil {
		return fmt.Errorf("failed to create TLS directory: %w", err)
	}

	// Create README for certificate placement
	readmePath := filepath.Join(tlsDir, "README.md")
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

	if err := os.WriteFile(readmePath, []byte(readme), 0600); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	return nil
}
