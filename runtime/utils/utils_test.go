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

func TestGetSth(t *testing.T) {
	containerId := "12345678"
	imageName := "test"
	cases := []struct {
		name          string
		arg, expected string
	}{
		{"imagePath", imageName, "/var/lib/sudocker/image/test.tar"},
		{"lowerPath", containerId, "/var/lib/sudocker/overlay2/12345678/lower"},
		{"upperPath", containerId, "/var/lib/sudocker/overlay2/12345678/upper"},
		{"workerPath", containerId, "/var/lib/sudocker/overlay2/12345678/worker"},
		{"mergedPath", containerId, "/var/lib/sudocker/overlay2/12345678/merged"},
	}
	t.Run(cases[0].name, func(t *testing.T) {
		if imagePath := GetImage(imageName); imagePath != cases[0].expected {
			t.Errorf("expected %s but %s got", cases[0].expected, imagePath)
		}
	})

	t.Run(cases[1].name, func(t *testing.T) {
		if ans := GetLower(containerId); ans != cases[1].expected {
			t.Errorf("expected %s but %s got", cases[0].expected, ans)
		}
	})
}
