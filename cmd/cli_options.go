package cmd

import "context"

// CLIOption 是传递给Sudocker Cli的函数式参数
type CLIOption func(cli *SudockerCli) error

// WithBaseContext 设置cli基本的上下文环境
func WithBaseContext(ctx context.Context) CLIOption {
	return func(cli *SudockerCli) error {
		cli.baseCtx = ctx
		return nil
	}
}
