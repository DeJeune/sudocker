package container

import (
	"github.com/DeJeune/sudocker/cmd"
	"github.com/spf13/cobra"
)

func NewContainerCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Manage containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.HelpFunc()(cmd, args)
			return nil
		},
	}
	cmd.AddCommand(
		NewCreateCommand(sudockerCli),
		NewRunCommand(sudockerCli),
		NewExecCommand(sudockerCli),
		NewKillCommand(sudockerCli),
		NewPsCommand(sudockerCli),
		NewStartCommand(sudockerCli),
		NewStopCommand(sudockerCli),
		NewRestartCommand(sudockerCli),
		NewRmCommand(sudockerCli),
		NewCommitCommand(sudockerCli),
		newListCommand(*sudockerCli),
		NewLogsCommand(sudockerCli),
		NewExecCommand(sudockerCli),
		NewStopCommand(sudockerCli),
		NewRmCommand(sudockerCli),
	)
	return cmd
}
