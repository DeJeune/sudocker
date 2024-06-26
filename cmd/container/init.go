package container

import (
	"context"
	"fmt"

	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewInitCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "init in a container, can't be used outside",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("init called")
			return Init(cmd.Context(), sudockerCli)
		},
	}
	return cmd
}

func Init(ctx context.Context, sudockerCli cmd.Cli) error {
	logrus.Infof("init come on")
	err := container.RunContainerInitProcess()
	return err
}
