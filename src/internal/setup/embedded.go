package setup

import _ "embed"

// Embedded init files
var (
	//go:embed files/systemd/cloud-update.service
	SystemdService string

	//go:embed files/openrc/cloud-update
	OpenRCScript string

	//go:embed files/sysvinit/cloud-update
	SysVInitScript string
)
