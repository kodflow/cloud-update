// Package setup provides service installation and management
package setup

import _ "embed"

// Embedded init files for system services.
// These files are embedded at build time from the subdirectories.
var (
	//go:embed systemd/cloud-update.service
	SystemdService string

	//go:embed openrc/cloud-update
	OpenRCScript string

	//go:embed sysvinit/cloud-update
	SysVInitScript string
)
