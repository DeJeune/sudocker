package utils

// 容器相关的目录
const (
	ImagePath       = "/var/lib/sudocker/images/"
	RootPath        = "/var/lib/sudocker/overlay2/"
	lowerDirFormat  = RootPath + "%s/lower"
	upperDirFormat  = RootPath + "%s/upper"
	workDirFormat   = RootPath + "%s/work"
	mergedDirFormat = RootPath + "%s/merged"
	overlayFSFormat = "lowerdir=%s,upperdir=%s,workdir=%s"
)

const (
	InfoLoc       = "/var/lib/sudocker/containers/"
	InfoLocFormat = InfoLoc + "%s/"
	ConfigName    = "config.json"
	IDLength      = 10
	LogFile       = "%s-json.log"
)
