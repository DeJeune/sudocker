package utils

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// WriteJSON writes the provided struct v to w using standard json marshaling
// without a trailing newline. This is used instead of json.Encoder because
// there might be a problem in json decoder in some cases, see:
// https://github.com/docker/docker/issues/14203#issuecomment-174177790
func WriteJSON(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func CleanPath(path string) string {
	if path == "" {
		return ""
	}

	path = filepath.Clean(path)

	if !filepath.IsAbs(path) {
		path = filepath.Clean(string(os.PathSeparator) + path)
		path, _ = filepath.Rel(string(os.PathSeparator), path)
	}
	return filepath.Clean(path)
}

func NewSockPair(name string) (parent, child *os.File, err error) {
	// 创建一个套接字对，AF_LOCAL本地套接字，unix.SOCK_STREAM|unix.SOCK_CLOEXEC 指定
	//套接字类型为流式套接字，并设置 CLOEXEC 标志（在执行新程序时关闭套接字）
	fds, err := unix.Socketpair(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), name+"-p"), os.NewFile(uintptr(fds[0]), name+"-c"), nil
}
