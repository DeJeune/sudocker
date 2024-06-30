package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"text/tabwriter"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func NewPsCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	options := psOptions{}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS]",
		Short: "List containers",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPs(cmd.Context(), sudockerCli, &options)
		},
		Annotations: map[string]string{
			"category-top": "3",
			"aliases":      "docker container ls, docker container list, docker container ps, docker ps",
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

func newListCommand(sudockerCli cmd.SudockerCli) *cobra.Command {
	cmd := *NewPsCommand(&sudockerCli)
	cmd.Aliases = []string{"ps", "list"}
	cmd.Use = "ls [OPTIONS]"
	return &cmd
}

func runPs(ctx context.Context, sudockerCli cmd.Cli, options *psOptions) error {
	// 读取存放容器信息目录下的所有文件
	files, err := os.ReadDir(utils.InfoLoc)
	if err != nil {
		return errors.Errorf("read dir %s error %v", utils.InfoLoc, err)
	}
	containers := make([]*container.Info, 0, len(files))
	for _, file := range files {
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			logrus.Errorf("get container info error %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}
	// 使用tabwriter.NewWriter在控制台打印出容器信息
	// tabwriter 是引用的text/tabwriter类库，用于在控制台打印对齐的表格
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, err = fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	if err != nil {
		return errors.Errorf("Fprint error %v", err)
	}
	for _, item := range containers {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.Created)
		if err != nil {
			logrus.Errorf("Fprint error %v", err)
		}
	}
	if err = w.Flush(); err != nil {
		return errors.Errorf("Flush error %v", err)
	}
	return nil
}

func getContainerInfo(file os.DirEntry) (*container.Info, error) {
	// 根据文件名拼接出完整路径
	configFileDir := fmt.Sprintf(utils.InfoLocFormat, file.Name())
	configFileDir = path.Join(configFileDir, utils.ConfigName)
	// 读取容器配置文件
	content, err := os.ReadFile(configFileDir)
	if err != nil {
		logrus.Errorf("read file %s error %v", configFileDir, err)
		return nil, err
	}
	info := new(container.Info)
	if err = json.Unmarshal(content, info); err != nil {
		logrus.Errorf("json unmarshal error %v", err)
		return nil, err
	}

	return info, nil
}
