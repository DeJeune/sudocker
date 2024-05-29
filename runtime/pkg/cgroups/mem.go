package cgroups

import "github.com/DeJeune/sudocker/runtime/config"

func isMemorySet(r *config.Resources) bool {
	return r.MemoryReservation != 0 || r.Memory != 0 || r.MemorySwap != 0
}

func setMemory(dirPath string, r *config.Resources) error {
	if !isMemorySet(r) {
		return nil
	}

	if err := CheckMemoryUsage(dirPath, r); err != nil {
		return err
	}

	swap, err := ConvertMemorySwapToCgroupV2Value(r.MemorySwap, r.Memory)
	if err != nil {
		return err
	}

	swapStr := numToStr(swap)
	if swapStr == "" && swap == 0 && r.MemorySwap > 0 {
		swapStr = "0"
	}

	if swapStr != "" {
		if err := WriteFile(dirPath, "memory.swap.max", swapStr); err != nil {
			return err
		}
	}

	if val := numToStr(r.Memory); val != "" {
		if err := WriteFile(dirPath, "memory.max", val); err != nil {
			return err
		}
	}

	if val := numToStr(r.MemoryReservation); val != "" {
		if err := WriteFile(dirPath, "memory.low", val); err != nil {
			return err
		}
	}
	return nil
}
