package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/exoscale/cli/pkg/account"
	"github.com/exoscale/cli/pkg/globalstate"
	"github.com/exoscale/cli/pkg/output"
	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
)

type securityGroupDeleteRuleCmd struct {
	CliCommandSettings `cli-cmd:"-"`

	_ bool `cli-cmd:"delete"`

	SecurityGroup string `cli-arg:"#" cli-usage:"SECURITY-GROUP-ID|NAME"`
	Rule          string `cli-arg:"#"`

	Force bool `cli-short:"f" cli-usage:"don't prompt for confirmation"`
}

func (c *securityGroupDeleteRuleCmd) CmdAliases() []string { return GRemoveAlias }

func (c *securityGroupDeleteRuleCmd) CmdShort() string {
	return "Delete a Security Group rule"
}

func (c *securityGroupDeleteRuleCmd) CmdLong() string {
	return fmt.Sprintf(`This command deletes a rule from a Compute instance Security Group.

Supported output template annotations: %s`,
		strings.Join(output.TemplateAnnotations(&securityGroupShowOutput{}), ", "))
}

func (c *securityGroupDeleteRuleCmd) CmdPreRun(cmd *cobra.Command, args []string) error {
	return CliCommandDefaultPreRun(c, cmd, args)
}

func (c *securityGroupDeleteRuleCmd) CmdRun(_ *cobra.Command, _ []string) error {
	zone := account.CurrentAccount.DefaultZone

	ctx := exoapi.WithEndpoint(GContext, exoapi.NewReqEndpoint(account.CurrentAccount.Environment, zone))

	securityGroup, err := globalstate.EgoscaleClient.FindSecurityGroup(ctx, zone, c.SecurityGroup)
	if err != nil {
		return err
	}

	if !c.Force {
		if !askQuestion(fmt.Sprintf(
			"Are you sure you want to delete rule %s from Security Group %q?",
			c.Rule,
			*securityGroup.Name,
		)) {
			return nil
		}
	}

	decorateAsyncOperation(fmt.Sprintf("Deleting Security Group rule %s...", c.Rule), func() {
		err = globalstate.EgoscaleClient.DeleteSecurityGroupRule(ctx, zone, securityGroup, &egoscale.SecurityGroupRule{ID: &c.Rule})
	})
	if err != nil {
		return err
	}

	return (&securityGroupShowCmd{
		CliCommandSettings: c.CliCommandSettings,
		SecurityGroup:      *securityGroup.ID,
	}).CmdRun(nil, nil)
}

func init() {
	cobra.CheckErr(RegisterCLICommand(securityGroupRuleCmd, &securityGroupDeleteRuleCmd{
		CliCommandSettings: DefaultCLICmdSettings(),
	}))
}
