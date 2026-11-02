package scripts

import _ "embed"

//go:embed bootstrap-vps.sh
var Bootstrap string

//go:embed teardown-vps.sh
var Teardown string
