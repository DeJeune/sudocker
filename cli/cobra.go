package cli

import (
	"fmt"
	"os"
	"strings"

	cliflags "github.com/DeJeune/sudocker/cli/flag"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func SetupRootCommand(rootCmd *cobra.Command) *cliflags.ClientOptions {
	rootCmd.SetVersionTemplate("Sudocker version {{.Version}}\n")
	return setupCommonRootCommand(rootCmd)
}

func setupCommonRootCommand(rootCmd *cobra.Command) *cliflags.ClientOptions {
	opts := cliflags.NewClientOptions()
	opts.InstallFlags(rootCmd.Flags())
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
	return opts
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

type TopLevelCommand struct {
	cmd       *cobra.Command
	dockerCli *cmd.SudockerCli
	opts      *cliflags.ClientOptions
	flags     *pflag.FlagSet
	args      []string
}

// NewTopLevelCommand 返回一个 new TopLevelCommand 对象
func NewTopLevelCommand(cmd *cobra.Command, dockerCli *cmd.SudockerCli, opts *cliflags.ClientOptions, flags *pflag.FlagSet) *TopLevelCommand {
	return &TopLevelCommand{
		cmd:       cmd,
		dockerCli: dockerCli,
		opts:      opts,
		flags:     flags,
		args:      os.Args[1:],
	}
}

// SetArgs 通过设置参数用来调用命令
func (tcmd *TopLevelCommand) SetArgs(args []string) {
	tcmd.args = args
	tcmd.cmd.SetArgs(args)
}

// SetFlag 设置top command的flag到flagset中
func (tcmd *TopLevelCommand) SetFlag(name, value string) {
	tcmd.cmd.Flags().Set(name, value)
}

// HandleGlobalFlags 解析全局flag
// eg. sudocker --debug true run xxx
func (tcmd *TopLevelCommand) HandleGlobalFlags() (*cobra.Command, []string, error) {
	cmd := tcmd.cmd

	flags := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)

	flags.SetInterspersed(false)

	flags.AddFlagSet(cmd.Flags())
	flags.AddFlagSet(cmd.PersistentFlags())

	if err := flags.Parse(tcmd.args); err != nil {

		if err := tcmd.Initialize(); err != nil {
			return nil, nil, err
		}
		return nil, nil, cmd.FlagErrorFunc()(cmd, err)
	}

	return cmd, flags.Args(), nil
}

// 通过解析全局option来初始化sudocker 客户端
func (tcmd *TopLevelCommand) Initialize(ops ...cmd.CLIOption) error {
	return tcmd.dockerCli.Initialize(tcmd.opts, ops...)
}

func VisitAll(root *cobra.Command, fn func(*cobra.Command)) {
	for _, cmd := range root.Commands() {
		VisitAll(cmd, fn)
	}
	fn(root)
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

{{- if not .HasParent}}
{{- if .HasAvailableFlags}}

GlobalOptions:
{{ wrappedFlagUsages . | trimRightSpace}}

{{- end}}
{{- end}}

{{- if .HasSubCommands }}

Run '{{.CommandPath}} COMMAND --help' for more information on a command.
{{- end}}
`
var helpTemplate = `
{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
