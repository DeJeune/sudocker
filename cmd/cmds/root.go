/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmds

import (
	"context"
	"fmt"

	"github.com/DeJeune/sudocker/cli"
	cliflags "github.com/DeJeune/sudocker/cli/flag"
	"github.com/DeJeune/sudocker/cli/version"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/spf13/cobra"
)

func NewSudockerCommand(sudockerCli *cmd.SudockerCli) *cli.TopLevelCommand {
	var opts *cliflags.ClientOptions

	cmd := &cobra.Command{
		Use:     "sudocker [OPTIONS] COMMAND [ARG...]",
		Short:   `A docker-like container runtime`,
		Version: fmt.Sprintf("%s, build %s", version.Version, version.GitCommit),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				return nil
			}
			return fmt.Errorf("sudocker: '%s' is not a sudocker command.\nSee 'sudocker --help'", args[0])
		},
		TraverseChildren: true,
		SilenceUsage:     true,
		SilenceErrors:    true,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
	}
	opts = cli.SetupRootCommand(cmd)
	cmd.Flags().BoolP("version", "v", false, "Pring version and quit")

	setFlagErrorFunc(sudockerCli, cmd)
	AddCommands(cmd, sudockerCli)
	setValidateArgs(sudockerCli, cmd)
	return cli.NewTopLevelCommand(cmd, sudockerCli, opts, cmd.Flags())
}

func RunSudocker(ctx context.Context, sudockerCli *cmd.SudockerCli) error {
	tcmd := NewSudockerCommand(sudockerCli)
	cmd, args, err := tcmd.HandleGlobalFlags()
	if err != nil {
		return err
	}
	if err := tcmd.Initialize(); err != nil {
		return err
	}
	cmd.SetArgs(args)
	err = cmd.ExecuteContext(ctx)
	return err
}

func setFlagErrorFunc(sudockerCli *cmd.SudockerCli, cmd *cobra.Command) {
	flagErrorFunc := cmd.FlagErrorFunc()
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return flagErrorFunc(cmd, err)
	})
}

func setValidateArgs(sudockerCli *cmd.SudockerCli, cmd *cobra.Command) {
	cli.VisitAll(cmd, func(c *cobra.Command) {
		if !hasTags(cmd) {
			return
		}
		if c.Args == nil {
			return
		}
	})
}

func hasTags(cmd *cobra.Command) bool {
	for curr := cmd; curr != nil; curr = curr.Parent() {
		if len(curr.Annotations) > 0 {
			return true
		}
	}
	return false
}
