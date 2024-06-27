package container

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cli/opts"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type commitOptions struct {
	container string
	reference string

	pause   bool
	comment string
	author  string
	changes opts.ListOpts
}

var options commitOptions

func NewCommitCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]",
		Short: "Create a new image from a container's changes",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.container = args[0]
			if len(args) > 1 {
				options.reference = args[1]
			}
			return runCommit(cmd.Context(), sudockerCli, &options)
		},
		Annotations: map[string]string{
			"aliases": "docker container commit, docker commit",
		},
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	flags.BoolVarP(&options.pause, "pause", "p", true, "Pause container during commit")
	flags.StringVarP(&options.comment, "message", "m", "", "Commit message")
	flags.StringVarP(&options.author, "author", "a", "", `Author (e.g., "John Hannibal Smith <hannibal@a-team.com>")`)

	options.changes = opts.NewListOpts(nil)
	flags.VarP(&options.changes, "change", "c", "Apply Dockerfile instruction to the created image")

	return cmd
}

func runCommit(ctx context.Context, sudockerCli cmd.Cli, options *commitOptions) error {
	mntPath := "/root/merged"
	imageTar := "/root/" + options.reference + ".tar"
	fmt.Println("commitContainer imageTar:", imageTar)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntPath, ".").CombinedOutput(); err != nil {
		logrus.Errorf("tar folder %s error %v", mntPath, err)
	}
	return nil
}
