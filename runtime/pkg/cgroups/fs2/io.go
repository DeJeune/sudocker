package fs2

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
)

func isIoSet(r *config.Resources) bool {
	return r.BlkioWeight != 0 ||
		len(r.BlkioWeightDevice) > 0 ||
		len(r.BlkioThrottleReadBpsDevice) > 0 ||
		len(r.BlkioThrottleWriteBpsDevice) > 0 ||
		len(r.BlkioThrottleReadIOPSDevice) > 0 ||
		len(r.BlkioThrottleWriteIOPSDevice) > 0
}

func bfqDeviceWeightSupported(bfq *os.File) bool {
	if bfq == nil {
		return false
	}

	_, _ = bfq.Seek(0, 0)
	buf := make([]byte, 32)
	_, _ = bfq.Read(buf)
	_, err := strconv.ParseInt(string(bytes.TrimSpace(buf)), 10, 64)

	return err != nil
}

func setIo(dirPath string, r *config.Resources) error {
	if !isIoSet(r) {
		return nil
	}
	var bfq *os.File
	if r.BlkioWeight != 0 || len(r.BlkioWeightDevice) > 0 {
		var err error
		bfq, err = cgroups.OpenFile(dirPath, "io.bfq.weight", os.O_RDWR)
		if err == nil {
			defer bfq.Close()
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	if r.BlkioWeight != 0 {
		if bfq != nil {
			if _, err := bfq.WriteString(strconv.FormatUint(uint64(r.BlkioWeight), 10)); err != nil {
				return err
			}
		} else {
			v := cgroups.ConvertBlkIOToIOWeightValue(r.BlkioWeight)
			if err := cgroups.WriteFile(dirPath, "io.weight", strconv.FormatUint(v, 10)); err != nil {
				return err
			}
		}
	}

	if bfqDeviceWeightSupported(bfq) {
		for _, wd := range r.BlkioWeightDevice {
			if _, err := bfq.WriteString(wd.WeightString() + "\n"); err != nil {
				return fmt.Errorf("setting device weight %q: %w", wd.WeightString(), err)
			}
		}
	}

	for _, td := range r.BlkioThrottleReadBpsDevice {
		if err := cgroups.WriteFile(dirPath, "io.max", td.StringName("rbps")); err != nil {
			return err
		}
	}

	for _, td := range r.BlkioThrottleWriteBpsDevice {
		if err := cgroups.WriteFile(dirPath, "io.max", td.StringName("wbps")); err != nil {
			return err
		}
	}

	for _, td := range r.BlkioThrottleReadIOPSDevice {
		if err := cgroups.WriteFile(dirPath, "io.max", td.StringName("riops")); err != nil {
			return err
		}
	}

	for _, td := range r.BlkioThrottleWriteIOPSDevice {
		if err := cgroups.WriteFile(dirPath, "io.max", td.StringName("wiops")); err != nil {
			return err
		}
	}

	return nil

}
