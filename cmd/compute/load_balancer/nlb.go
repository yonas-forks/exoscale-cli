package load_balancer

import (
	exocmd "github.com/exoscale/cli/cmd"
	"github.com/spf13/cobra"
)

var nlbCmd = &cobra.Command{
	Use:     "load-balancer",
	Short:   "Network Load Balancers management",
	Aliases: []string{"nlb"},
}

func init() {
	exocmd.ComputeCmd.AddCommand(nlbCmd)
}
