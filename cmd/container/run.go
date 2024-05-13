package container

import (
	"fmt"

	"github.com/DeJeune/sudocker/cli"
	"github.com/spf13/cobra"
)

type runOption struct {
	detach     bool
	sigProxy   bool
	name       string
	detachKeys string
}

func NewRunCommand() *cobra.Command {

	runCmd := &cobra.Command{
		Use:     "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
		Short:   "Create and Run a container from a image",
		Long:    `Create and Run a container from a image`,
		Aliases: []string{"sudocker container run"},
		Args:    cli.RequiresMinArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("run called")
		},
	}

	// Here you will define your flags and configuration settings.
	runCmd.Flags().BoolP("detach", "d", true, "Run the container in the background")
	runCmd.Flags().StringP("name", "n", "", "Assign a name to the container")
	runCmd.Flags().String("host", "", "Set hostname to the container")
	runCmd.Flags().StringSliceP("env", "e", []string{}, "Set environment variables")
	runCmd.Flags().String("store-opt", "overlay2", "Optional storage driver")
	return runCmd
}
