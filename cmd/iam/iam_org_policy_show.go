package iam

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	exocmd "github.com/exoscale/cli/cmd"
	"github.com/exoscale/cli/pkg/account"
	"github.com/exoscale/cli/pkg/globalstate"
	"github.com/exoscale/cli/pkg/output"
	v3 "github.com/exoscale/egoscale/v3"
)

type iamOrgPolicyShowCmd struct {
	exocmd.CliCommandSettings `cli-cmd:"-"`

	_ bool `cli-cmd:"show"`
}

func (c *iamOrgPolicyShowCmd) CmdAliases() []string { return exocmd.GShowAlias }

func (c *iamOrgPolicyShowCmd) CmdShort() string {
	return "Show Org policy details"
}

func (c *iamOrgPolicyShowCmd) CmdLong() string {
	return fmt.Sprintf(`This command shows IAM Org Policy details.

Supported output template annotations: %s`,
		strings.Join(output.TemplateAnnotations(&iamPolicyServiceOutput{}), ", "))
}

func (c *iamOrgPolicyShowCmd) CmdPreRun(cmd *cobra.Command, args []string) error {
	return exocmd.CliCommandDefaultPreRun(c, cmd, args)
}

func (c *iamOrgPolicyShowCmd) CmdRun(_ *cobra.Command, _ []string) error {
	ctx := exocmd.GContext
	client, err := exocmd.SwitchClientZoneV3(ctx, globalstate.EgoscaleV3Client, v3.ZoneName(account.CurrentAccount.DefaultZone))
	if err != nil {
		return err
	}

	policy, err := client.GetIAMOrganizationPolicy(ctx)
	if err != nil {
		return err
	}

	out := iamPolicyOutput{
		DefaultServiceStrategy: string(policy.DefaultServiceStrategy),
		Services:               map[string]iamPolicyServiceOutput{},
	}

	for name, service := range policy.Services {
		rules := []iamPolicyServiceRuleOutput{}
		for _, rule := range service.Rules {
			rules = append(rules, iamPolicyServiceRuleOutput{
				Action:     string(rule.Action),
				Expression: rule.Expression,
			})
		}

		out.Services[name] = iamPolicyServiceOutput{
			Type:  string(service.Type),
			Rules: rules,
		}
	}

	return c.OutputFunc(&out, nil)
}

func init() {
	cobra.CheckErr(exocmd.RegisterCLICommand(iamOrgPolicyCmd, &iamOrgPolicyShowCmd{
		CliCommandSettings: exocmd.DefaultCLICmdSettings(),
	}))
}
