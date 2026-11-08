package scripts

import (
	_ "embed"
	"strings"
)

//go:embed bootstrap-vps.sh
var Bootstrap string

//go:embed teardown-vps.sh
var Teardown string

func init() {
	Bootstrap = Unix(Bootstrap)
	Teardown = Unix(Teardown)
}

// Unix makes a shell script safe for bash on Linux.
// A Windows build embeds the files with CRLF, and bash then reads
// "set -euo pipefail\r" as an invalid option.
func Unix(script string) string {
	script = strings.ReplaceAll(script, "\r\n", "\n")
	script = strings.ReplaceAll(script, "\r", "\n")
	return script
}
