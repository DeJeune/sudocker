package devices

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

const (
	Wildcard = -1
)

type Device struct {
	Rule
	Path     string      `json:"path"`
	FileMode os.FileMode `json:"file_mode"`
	Uid      uint32      `json:"uid"`
	Gid      uint32      `json:"gid"`
}

type Permissions string

type Type rune

const (
	WildcardDevice Type = 'a'
	BlockDevice    Type = 'b'
	CharDevice     Type = 'c' // or 'u'
	FifoDevice     Type = 'p'
)

func (t Type) IsValid() bool {
	switch t {
	case WildcardDevice, BlockDevice, CharDevice, FifoDevice:
		return true
	default:
		return false
	}
}

func (t Type) CanMknod() bool {
	switch t {
	case BlockDevice, CharDevice, FifoDevice:
		return true
	default:
		return false
	}
}

func (t Type) CanCgroup() bool {
	switch t {
	case WildcardDevice, BlockDevice, CharDevice:
		return true
	default:
		return false
	}
}

type Rule struct {
	// Type of device ('c' for char, 'b' for block). If set to 'a', this rule
	// acts as a wildcard and all fields other than Allow are ignored.
	Type Type `json:"type"`

	// Major is the device's major number.
	Major int64 `json:"major"`

	// Minor is the device's minor number.
	Minor int64 `json:"minor"`

	// Permissions is the set of permissions that this rule applies to (in the
	// cgroupv1 format -- any combination of "rwm").
	Permissions Permissions `json:"permissions"`

	// Allow specifies whether this rule is allowed.
	Allow bool `json:"allow"`
}

func mkDev(d *Rule) (uint64, error) {
	if d.Major == Wildcard || d.Minor == Wildcard {
		return 0, errors.New("cannot mkdev() device with wildcards")
	}
	return unix.Mkdev(uint32(d.Major), uint32(d.Minor)), nil
}

func (d *Rule) Mkdev() (uint64, error) {
	return mkDev(d)
}
