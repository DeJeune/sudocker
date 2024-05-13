package image

import "github.com/spf13/cobra"

func NewImageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage image",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.HelpFunc()(cmd, args)
			return nil
		},
	}
	cmd.AddCommand(
		NewCreateCommand(),
	)
	return cmd
}
