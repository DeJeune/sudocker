package system

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

type ParentDeathSignal int

func (p ParentDeathSignal) Restore() error {
	if p == 0 {
		return nil
	}
	current, err := GetParentDeathSignal()
	if err != nil {
		return err
	}
	if p == current {
		return nil
	}
	return p.Set()
}
func (p ParentDeathSignal) Set() error {
	return SetParentDeathSignal(uintptr(p))
}

func SetParentDeathSignal(sig uintptr) error {
	if err := unix.Prctl(unix.PR_SET_PDEATHSIG, sig, 0, 0, 0); err != nil {
		return err
	}
	return nil
}

func Setctty() error {
	if err := unix.IoctlSetInt(0, unix.TIOCSCTTY, 0); err != nil {
		return err
	}
	return nil
}

func Exec(cmd string, args []string, env []string) error {
	for {
		err := unix.Exec(cmd, args, env)
		if err != unix.EINTR {
			return &os.PathError{Op: "exec", Path: cmd, Err: err}
		}
	}
}

func GetParentDeathSignal() (ParentDeathSignal, error) {
	var sig int
	if err := unix.Prctl(unix.PR_GET_PDEATHSIG, uintptr(unsafe.Pointer(&sig)), 0, 0, 0); err != nil {
		return -1, err
	}
	return ParentDeathSignal(sig), nil
}

func SetKeepCaps() error {
	if err := unix.Prctl(unix.PR_SET_KEEPCAPS, 1, 0, 0, 0); err != nil {
		return err
	}

	return nil
}

func ClearKeepCaps() error {
	if err := unix.Prctl(unix.PR_SET_KEEPCAPS, 0, 0, 0, 0); err != nil {
		return err
	}

	return nil
}
