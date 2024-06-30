package container

import (
	"fmt"
	"testing"

	"github.com/DeJeune/sudocker/runtime/utils"
)

func TestLogFileLocation(t *testing.T) {
	containerId := "12345678"
	logFileLocation := fmt.Sprintf(utils.InfoLocFormat, containerId) + utils.GetLogfile(containerId)
	expectedLocation := "/var/lib/sudocker/containers/12345678/12345678-json.log"
	if logFileLocation != expectedLocation {
		t.Errorf("exepcted %s bug error %s got", expectedLocation, logFileLocation)
	}
}
