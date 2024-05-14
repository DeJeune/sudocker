package flag

import (
	"fmt"
	"os"

	"github.com/DeJeune/sudocker/cli/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

type ClientOptions struct {
	Debug     bool
	LogLevel  string
	ConfigDir string
}

func NewClientOptions() *ClientOptions {
	return &ClientOptions{}
}

// InstallFlags 添加全局flag
func (o *ClientOptions) InstallFlags(flags *pflag.FlagSet) {
	configDir := config.Dir()
	flags.StringVar(&o.ConfigDir, "config", configDir, "Location of client config files")
	flags.BoolVarP(&o.Debug, "debug", "D", false, "Enable debug mode")
	flags.StringVarP(&o.LogLevel, "log-level", "l", "info", `Set the logging level ("debug", "info", "warn", "error", "fatal")`)
}

// SetLogLevel 设置日志等级
func SetLogLevel(logLevel string) {
	if logLevel != "" {
		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse logging level: %s\n", logLevel)
			os.Exit(1)
		}
		logrus.SetLevel(lvl)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}
