package cmds

import (
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/cmd/container"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, sudockerCli *cmd.SudockerCli) {
	cmd.AddCommand(
		container.NewContainerCommand(sudockerCli),
		container.NewInitCommand(sudockerCli),
		container.NewRunCommand(sudockerCli),
		container.NewPsCommand(sudockerCli),
		container.NewLogsCommand(sudockerCli),
		container.NewCreateCommand(sudockerCli),
	)
}
