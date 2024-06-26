package fs2

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
)

// numToStr转换一个int64类型变量为字符串类型变量
// 然后写到一个cgroupv2文件，文件名以.min, .max, .low, .high结尾
func numToStr(value int64) (ret string) {
	switch {
	case value == 0:
		ret = ""
	default:
		ret = strconv.FormatInt(value, 10)
	}

	return ret
}

// 记录解析错误
type ParseError struct {
	Path string
	File string
	Err  error
}

func (e *ParseError) Error() string {
	return "unable to parse " + path.Join(e.Path, e.File) + ": " + e.Err.Error()
}

func (e *ParseError) UnWrap() error {
	return e.Err
}

// 转换字符串为64位无符号整数
func ParseUint(s string, base, bitSize int) (uint64, error) {
	value, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if errors.Is(intErr, strconv.ErrRange) && intValue < 0 {
			return 0, nil
		}

		return value, err
	}

	return value, nil
}

// 解析cgroup里的"key value"
// 例如,"io_service_bytes_1234" 将要返回"io_service_bytes", 1234
func ParseKeyValue(t string) (string, uint64, error) {
	parts := strings.SplitN(t, " ", 3)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("line %q is not in key value format", t)
	}

	value, err := ParseUint(parts[1], 10, 64)
	if err != nil {
		return "", 0, err
	}

	return parts[0], value, nil
}
