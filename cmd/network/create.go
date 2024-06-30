package network

import (
	"context"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cli/opts"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/pkg/network"
	"github.com/spf13/cobra"
)

type createOptions struct {
	name       string
	scope      string
	driver     string
	driverOpts opts.MapOpts
	labels     opts.ListOpts
	internal   bool
	ipv6       *bool
	attachable bool
	ingress    bool
	configOnly bool
	configFrom string

	ipamDriver  string
	ipamSubnet  string
	ipamIPRange string
	ipamGateway string
	ipamAux     opts.MapOpts
	ipamOpt     opts.MapOpts
}

func NewCreateCommand(sudockerCli cmd.Cli) *cobra.Command {
	options := &createOptions{
		driverOpts: *opts.NewMapOpts(nil, nil),
		labels:     opts.NewListOpts(opts.ValidateLabel),
		ipamAux:    *opts.NewMapOpts(nil, nil),
		ipamOpt:    *opts.NewMapOpts(nil, nil),
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] NETWORK",
		Short: "Create a network",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.name = args[0]
			return runCreate(cmd.Context(), sudockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&options.driver, "driver", "d", "bridge", "Driver to manage the Network")
	flags.StringVar(&options.ipamSubnet, "subnet", "", "Subnet in CIDR format that represents a network segment")
	return cmd
}

func runCreate(ctx context.Context, sudockerCli cmd.Cli, options *createOptions) error {
	return network.CreateNetwork(options.driver, options.ipamSubnet, options.name)
}
