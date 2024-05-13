package container

import (
	"github.com/DeJeune/sudocker/cli"
	"github.com/spf13/cobra"
)

type psOptions struct {
	quiet   bool
	size    bool
	all     bool
	noTrunc bool
	nLatest bool
	last    int
	format  string
}

func NewPsCommand() *cobra.Command {
	options := psOptions{}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS]",
		Short: "List containers",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display numeric IDs")
	flags.BoolVarP(&options.size, "size", "s", false, "Display total file sizes")
	flags.BoolVarP(&options.all, "all", "a", false, "Show all containers (default shows just running)")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.BoolVarP(&options.nLatest, "latest", "l", false, "Show the latest created container (includes all states)")
	flags.IntVarP(&options.last, "last", "n", -1, "Show n last created containers (includes all states)")
	flags.StringVarP(&options.format, "format", "", "", "Pretty-print containers using a Go template")
	// flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}
