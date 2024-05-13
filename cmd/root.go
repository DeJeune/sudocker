/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd/container"
	"github.com/DeJeune/sudocker/cmd/image"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "sudocker [OPTIONS] COMMAND [ARG...]",
	Short: `A docker-like container runtime:
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
	TraverseChildren: true,
	// SilenceUsage:     true,
	// SilenceErrors:    true,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cli.SetupRootCommand(rootCmd)
	rootCmd.AddCommand(container.NewContainerCommand())
	rootCmd.AddCommand(image.NewImageCommand())
}
