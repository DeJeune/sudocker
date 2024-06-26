package cmd

import (
	"context"

	"github.com/DeJeune/sudocker/cli/streams"
	"github.com/DeJeune/sudocker/cli/term"
)

// CLIOption 是传递给Sudocker Cli的函数式参数
type CLIOption func(cli *SudockerCli) error

func WithStandardStreams() CLIOption {
	return func(cli *SudockerCli) error {
		// Set terminal emulation based on platform as required.
		stdin, stdout, stderr := term.StdStreams()
		cli.in = streams.NewIn(stdin)
		cli.out = streams.NewOut(stdout)
		cli.err = streams.NewOut(stderr)
		return nil
	}
}

// WithBaseContext 设置cli基本的上下文环境
func WithBaseContext(ctx context.Context) CLIOption {
	return func(cli *SudockerCli) error {
		cli.baseCtx = ctx
		return nil
	}
}
