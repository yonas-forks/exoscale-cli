package deploy_target

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	exocmd "github.com/exoscale/cli/cmd"
	"github.com/exoscale/cli/pkg/globalstate"
	"github.com/exoscale/cli/pkg/output"
	v3 "github.com/exoscale/egoscale/v3"
)

type deployTargetShowOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Zone        string `json:"zone"`
}

func (o *deployTargetShowOutput) ToJSON()  { output.JSON(o) }
func (o *deployTargetShowOutput) ToText()  { output.Text(o) }
func (o *deployTargetShowOutput) ToTable() { output.Table(o) }

type deployTargetShowCmd struct {
	exocmd.CliCommandSettings `cli-cmd:"-"`

	_ bool `cli-cmd:"show"`

	DeployTarget string `cli-arg:"#" cli-usage:"NAME|ID"`

	Zone string `cli-short:"z" cli-usage:"Deploy Target zone"`
}

func (c *deployTargetShowCmd) CmdAliases() []string { return exocmd.GShowAlias }

func (c *deployTargetShowCmd) CmdShort() string { return "Show a Deploy Target details" }

func (c *deployTargetShowCmd) CmdLong() string {
	return fmt.Sprintf(`This command shows a Deploy Target details.

Supported output template annotations: %s`,
		strings.Join(output.TemplateAnnotations(&deployTargetShowOutput{}), ", "))
}

func (c *deployTargetShowCmd) CmdPreRun(cmd *cobra.Command, args []string) error {
	exocmd.CmdSetZoneFlagFromDefault(cmd)
	return exocmd.CliCommandDefaultPreRun(c, cmd, args)
}

func (c *deployTargetShowCmd) CmdRun(_ *cobra.Command, _ []string) error {
	ctx := exocmd.GContext

	client, err := exocmd.SwitchClientZoneV3(ctx, globalstate.EgoscaleV3Client, v3.ZoneName(c.Zone))
	if err != nil {
		return err
	}

	deployTargetListResp, err := client.ListDeployTargets(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving Deploy Target: %w", err)
	}

	dt, err := deployTargetListResp.FindDeployTarget(c.DeployTarget)
	if err != nil {
		return fmt.Errorf("error retrieving Deploy Target: %w", err)
	}

	return c.OutputFunc(&deployTargetShowOutput{
		ID:          dt.ID.String(),
		Name:        dt.Name,
		Description: dt.Description,
		Type:        string(dt.Type),
		Zone:        c.Zone,
	}, nil)
}

func init() {
	cobra.CheckErr(exocmd.RegisterCLICommand(deployTargetCmd, &deployTargetShowCmd{
		CliCommandSettings: exocmd.DefaultCLICmdSettings(),
	}))
}
