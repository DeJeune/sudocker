package fscommon

import (
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
)

type ParseError struct {
	Path string
	File string
	Err  error
}

func (e *ParseError) Error() string {
	return "unable to parse " + path.Join(e.Path, e.File) + ": " + e.Err.Error()
}

func (e *ParseError) Unwrap() error { return e.Err }

func GetCgroupParamUint(path, file string) (uint64, error) {
	contents, err := GetCgroupParamString(path, file)
	if err != nil {
		return 0, err
	}
	contents = strings.TrimSpace(contents)
	if contents == "max" {
		return math.MaxUint64, nil
	}

	res, err := strconv.ParseUint(contents, 10, 64)
	if err != nil {
		return res, &ParseError{Path: path, File: file, Err: err}
	}
	return res, nil
}

func GetCgroupParamString(path, file string) (string, error) {
	contents, err := cgroups.ReadFile(path, file)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(contents), nil
}
