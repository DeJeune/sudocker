package main

import (
	"context"
	"fmt"
	"os"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/cmd/cmds"
)

func main() {
	ctx := context.Background()
	sudockerCli, err := cmd.NewSudockerCLi(cmd.WithBaseContext(ctx))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := cmds.RunSudocker(ctx, sudockerCli); err != nil {
		if sterr, ok := err.(cli.StatusError); ok {
			if sterr.Status != "" {
				fmt.Fprintln(os.Stderr, sterr.Status)
			}

			if sterr.StatusCode == 0 {
				os.Exit(1)
			}
			os.Exit(sterr.StatusCode)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
