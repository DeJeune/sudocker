package cmds

import (
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/cmd/container"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, sudockerCli *cmd.SudockerCli) {
	cmd.AddCommand(
		container.NewContainerCommand(sudockerCli),
	)
}
