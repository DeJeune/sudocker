package devices

import "os"

type Device struct {
	Type     rune        `json:"type"`
	Path     string      `json:"path"`
	Major    int64       `json:"major"`
	Minor    int64       `json:"minor"`
	FileMode os.FileMode `json:"file_mode"`
	Uid      uint32      `json:"uid"`
	Gid      uint32      `json:"gid"`
}
