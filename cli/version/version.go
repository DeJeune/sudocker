package version

// build时的变量
// 通过ldflags覆写
var (
	Version   string
	GitCommit string
	BuildTime string
	GoVersion string
)
