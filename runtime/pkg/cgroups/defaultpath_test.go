package cgroups

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseCgroupFromReader(t *testing.T) {
	cases := map[string]string{
		"0::/user.slice/user-1001.slice/session-1.scope\n":                                  "/user.slice/user-1001.slice/session-1.scope",
		"2:cpuset:/foo\n1:name=systemd:/\n":                                                 "",
		"2:cpuset:/foo\n1:name=systemd:/\n0::/user.slice/user-1001.slice/session-1.scope\n": "/user.slice/user-1001.slice/session-1.scope",
	}
	for s, expected := range cases {
		g, err := parseCgroupFromReader(strings.NewReader(s))
		if expected != "" {
			fmt.Println(expected)
			if g != expected {
				t.Errorf("expected %q, got %q", expected, g)
			}
			if err != nil {
				t.Error(err)
			}
		} else {
			if err == nil {
				t.Error("error is expected")
			}
		}
	}
}
