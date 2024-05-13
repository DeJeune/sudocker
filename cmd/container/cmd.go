package container

import "github.com/spf13/cobra"

func NewContainerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Manage containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.HelpFunc()(cmd, args)
			return nil
		},
	}
	cmd.AddCommand(
		NewRunCommand(),
		NewExecCommand(),
		NewKillCommand(),
		NewPsCommand(),
		NewStartCommand(),
		NewStopCommand(),
		NewRestartCommand(),
		NewRmCommand(),
	)
	return cmd
}
