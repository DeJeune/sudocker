package cmd

import (
	"context"

	"github.com/DeJeune/sudocker/cli/config"
	configfile "github.com/DeJeune/sudocker/cli/config/configfile"
	cliflags "github.com/DeJeune/sudocker/cli/flag"
	"github.com/DeJeune/sudocker/cli/streams"
)

type Streams interface {
	In() *streams.In
	Out() *streams.Out
	Err() *streams.Out
}

type Cli interface {
	Streams
	SetIn(in *streams.In)
	ConfigFile() *configfile.ConfigFile
	Apply(ops ...CLIOption) error
}

type SudockerCli struct {
	configFile *configfile.ConfigFile
	in         *streams.In
	out        *streams.Out
	err        *streams.Out
	options    *cliflags.ClientOptions
	baseCtx    context.Context
}

func (cli *SudockerCli) ConfigFile() *configfile.ConfigFile {
	if cli.configFile == nil {
		cli.configFile = config.LoadDefaultConfigFile(cli.err)
	}
	return cli.configFile
}

// Out returns the writer used for stdout
func (cli *SudockerCli) Out() *streams.Out {
	return cli.out
}

func (cli *SudockerCli) Err() *streams.Out {
	return cli.err
}

func (cli *SudockerCli) SetIn(in *streams.In) {
	cli.in = in
}

func (cli *SudockerCli) In() *streams.In {
	return cli.in
}

func (cli *SudockerCli) Initialize(opts *cliflags.ClientOptions, ops ...CLIOption) error {
	for _, o := range ops {
		if err := o(cli); err != nil {
			return err
		}
	}
	cliflags.SetLogLevel(opts.LogLevel)

	if opts.ConfigDir != "" {
		config.SetDir(opts.ConfigDir)
	}

	cli.options = opts
	cli.configFile = config.LoadDefaultConfigFile(cli.err)
	return nil
}

func (cli *SudockerCli) Apply(ops ...CLIOption) error {
	for _, op := range ops {
		if err := op(cli); err != nil {
			return err
		}
	}
	return nil
}

func NewSudockerCLi(ops ...CLIOption) (*SudockerCli, error) {
	defaultOps := []CLIOption{
		WithStandardStreams(),
	}
	ops = append(defaultOps, ops...)
	cli := &SudockerCli{baseCtx: context.Background()}
	if err := cli.Apply(ops...); err != nil {
		return nil, err
	}
	return cli, nil
}
