// Package version provides version information for the Cloud Update service.
package version

import (
	"fmt"
	"runtime"
)

// Build information, injected at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// GetFullVersion returns the full version string.
func GetFullVersion() string {
	return fmt.Sprintf("cloud-update %s (%s) built on %s with %s for %s/%s",
		Version, Commit, Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// GetShortVersion returns just the version number.
func GetShortVersion() string {
	return Version
}
