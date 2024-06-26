package container

import (
	"context"
	"fmt"
	"os"
	gosignal "os/signal"
	"runtime"
	"time"

	"github.com/DeJeune/sudocker/cmd"

	"github.com/moby/sys/signal"
)

func initTtySize(ctx context.Context, cli cmd.Cli, id string, isExec bool, resizeTtyFunc func(ctx context.Context, cli cmd.Cli, id string, isExec bool) error) {
	rttyFunc := resizeTtyFunc
	if rttyFunc == nil {
		rttyFunc = resizeTty
	}
	if err := rttyFunc(ctx, cli, id, isExec); err != nil {
		go func() {
			var err error
			for retry := 0; retry < 10; retry++ {
				time.Sleep(time.Duration(retry+1) * 10 * time.Millisecond)
				if err = rttyFunc(ctx, cli, id, isExec); err == nil {
					break
				}
			}
			if err != nil {
				fmt.Fprintln(cli.Err(), "failed to resize tty, using default size")
			}
		}()
	}
}

func resizeTty(ctx context.Context, cli cmd.Cli, id string, isExec bool) error {
	height, width := cli.Out().GetTtySize()
	return resizeTtyTo(ctx, id, height, width, isExec)
}

// resizeTtyTo resizes tty to specific height and width
func resizeTtyTo(ctx context.Context, id string, height, width uint, isExec bool) error {
	if height == 0 && width == 0 {
		return nil
	}

	// options := container.ResizeOptions{
	// 	Height: height,
	// 	Width:  width,
	// }

	// var err error
	// if isExec {
	// 	err = apiClient.ContainerExecResize(ctx, id, options)
	// } else {
	// 	err = apiClient.ContainerResize(ctx, id, options)
	// }

	// if err != nil {
	// 	logrus.Debugf("Error resize: %s\r", err)
	// }
	return nil
}

// MonitorTtySize updates the container tty size when the terminal tty changes size
func MonitorTtySize(ctx context.Context, cli cmd.Cli, id string, isExec bool) error {
	initTtySize(ctx, cli, id, isExec, resizeTty)
	if runtime.GOOS == "windows" {
		go func() {
			prevH, prevW := cli.Out().GetTtySize()
			for {
				time.Sleep(time.Millisecond * 250)
				h, w := cli.Out().GetTtySize()

				if prevW != w || prevH != h {
					resizeTty(ctx, cli, id, isExec)
				}
				prevH = h
				prevW = w
			}
		}()
	} else {
		sigchan := make(chan os.Signal, 1)
		gosignal.Notify(sigchan, signal.SIGWINCH)
		go func() {
			for range sigchan {
				resizeTty(ctx, cli, id, isExec)
			}
		}()
	}
	return nil
}
