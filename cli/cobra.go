package cli

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func SetupRootCommand(rootCmd *cobra.Command) {
	cobra.AddTemplateFunc("hasSubCommands", hasSubCommands)
	cobra.AddTemplateFunc("hasManagementSubCommands", hasManagementSubCommands)
	cobra.AddTemplateFunc("operationSubCommands", operationSubCommands)
	cobra.AddTemplateFunc("managementSubCommands", managementSubCommands)
	cobra.AddTemplateFunc("wrappedFlagUsages", wrappedFlagUsages)

	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetFlagErrorFunc(FlagErrorFunc)
	rootCmd.SetHelpCommand(helpCommand)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().MarkShorthandDeprecated("help", "please use --help")
	rootCmd.PersistentFlags().Lookup("help").Hidden = true
}

func FlagErrorFunc(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}

	usage := ""
	if cmd.HasSubCommands() {
		usage = "\n\n" + cmd.UsageString()
	}
	return StatusError{
		Status:     fmt.Sprintf("%s\nSee '%s --help'.%s", err, cmd.CommandPath(), usage),
		StatusCode: 125,
	}
}

var helpCommand = &cobra.Command{
	Use:   "help [command]",
	Short: "help about the command",
	RunE: func(c *cobra.Command, args []string) error {
		cmd, args, e := c.Root().Find(args)
		if cmd == nil || e != nil || len(args) > 0 {
			return errors.Errorf("unknown help topic %v", strings.Join(args, " "))
		}
		helpFunc := cmd.HelpFunc()
		helpFunc(cmd, args)
		return nil
	},
}

func hasSubCommands(cmd *cobra.Command) bool {
	return len(operationSubCommands(cmd)) > 0
}

func hasManagementSubCommands(cmd *cobra.Command) bool {
	return len(managementSubCommands(cmd)) > 0
}

func operationSubCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() && !sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func managementSubCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() && sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func RequiresMinArgs(min int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) >= min {
			return nil
		}
		return errors.Errorf(
			"\"%s\" requires at least %d argument(s).\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			min,
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

func wrappedFlagUsages(cmd *cobra.Command) string {
	width := 80
	return cmd.Flags().FlagUsagesWrapped(width - 1)
}

var usageTemplate = `Usage:

{{- if not .HasSubCommands}}	{{.UseLine}}{{end}}
{{- if .HasSubCommands}}	{{ .CommandPath}} {{- if .HasAvailableFlags}} [OPTIONS]{{end}} COMMAND{{end}}

{{if ne .Long ""}}{{ .Long | trim }}{{ else }}{{ .Short | trim }}{{end}}

{{- if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}

{{- end}}
{{- if .HasExample}}

Examples:
{{ .Example }}

{{- end}}
{{- if .HasParent}}
{{- if .HasAvailableFlags}}

Options:
{{ wrappedFlagUsages . | trimRightSpace}}

{{- end}}
{{- end}}
{{- if hasManagementSubCommands . }}

Management Commands:

{{- range managementSubCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}

{{- end}}
{{- if hasSubCommands .}}

Commands:

{{- range operationSubCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}
{{- end}}

{{- if .HasSubCommands }}

Run '{{.CommandPath}} COMMAND --help' for more information on a command.
{{- end}}
`
var helpTemplate = `
{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
