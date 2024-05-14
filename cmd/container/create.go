package container

import (
	"fmt"

	"github.com/DeJeune/sudocker/cmd"
	"github.com/spf13/cobra"
)

func NewCreateCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new container",
		Long:  `Create a new container`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("create called")
		},
	}

	return cmd
}
