package utils

import "testing"

func TestCleanPath(t *testing.T) {
	path := CleanPath("")
	if path != "" {
		t.Errorf("expected to receive empty string and received %s", path)
	}

	path = CleanPath("rootfs")
	if path != "rootfs" {
		t.Errorf("expected to receive 'rootfs' and received %s", path)
	}
	path = CleanPath("/../../../var")
	if path != "/var" {
		t.Errorf("expected to receive '/var' and received %s", path)
	}
}
