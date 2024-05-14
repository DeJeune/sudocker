package cmd

import (
	"context"
	"io"

	"github.com/DeJeune/sudocker/cli/config"
	configfile "github.com/DeJeune/sudocker/cli/config/configfile"
	cliflags "github.com/DeJeune/sudocker/cli/flag"
)

type Cli interface {
	Err() io.Writer
	ConfigFile() *configfile.ConfigFile
	Apply(ops ...CLIOption) error
}

type SudockerCli struct {
	configFile *configfile.ConfigFile
	err        io.Writer
	options    *cliflags.ClientOptions
	baseCtx    context.Context
}

func (cli *SudockerCli) ConfigFile() *configfile.ConfigFile {
	if cli.configFile == nil {
		cli.configFile = config.LoadDefaultConfigFile(cli.err)
	}
	return cli.configFile
}

func (cli *SudockerCli) Err() io.Writer {
	return cli.err
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
	cli := &SudockerCli{baseCtx: context.Background()}
	if err := cli.Apply(ops...); err != nil {
		return nil, err
	}
	return cli, nil
}
