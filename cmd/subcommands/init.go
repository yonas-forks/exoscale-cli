/*
The purpose of this file is to initialize all of the commands, that are spread across different folders,
and to keep the imports in one place, keeping main.go clean.
*/

package subcommands

import (
	_ "github.com/exoscale/cli/cmd/compute/load_balancer"
)
