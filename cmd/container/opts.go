package container

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DeJeune/sudocker/cli/compose/loader"
	"github.com/DeJeune/sudocker/cli/opts"
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

var deviceCgroupRuleRegexp = regexp.MustCompile(`^[acb] ([0-9]+|\*):([0-9]+|\*) [rwm]{1,3}$`)

type containerOptions struct {
	hostname   string
	domainname string
	attach     opts.ListOpts
	volumes    opts.ListOpts
	stdin      bool
	tty        bool
	env        opts.ListOpts
	envFile    opts.ListOpts
	// devices            opts.ListOpts
	// deviceCgroupRules  opts.ListOpts
	// blkioWeightDevice  opts.WeightdeviceOpt
	// deviceReadBps      opts.ThrottledeviceOpt
	// deviceWriteBps     opts.ThrottledeviceOpt
	// deviceReadIOps     opts.ThrottledeviceOpt
	// deviceWriteIOps    opts.ThrottledeviceOpt
	memory            opts.MemBytes
	memoryReservation opts.MemBytes
	memorySwap        opts.MemSwapBytes
	// oomKillDisable     bool
	// oomScoreAdj        int
	pidsLimit int64
	// kernelMemory       opts.MemBytes
	// cpuCount           int64
	cpuShares uint64
	// cpuPercent         int64
	cpuPeriod          uint64
	cpuRealtimePeriod  uint64
	cpuRealtimeRuntime int64
	cpuQuota           int64
	cpus               opts.NanoCPUs
	cpusetCpus         string
	cpusetMems         string
	blkioWeight        uint16
	// ioMaxBandwidth     opts.MemBytes
	// ioMaxIOps          uint64
	swappiness int64
	publish    opts.ListOpts
	expose     opts.ListOpts
	netMode    string
	autoRemove bool
	Image      string
	Args       []string
}

func addFlags(flags *pflag.FlagSet) *containerOptions {
	copts := &containerOptions{
		attach:  opts.NewListOpts(validateAttach),
		volumes: opts.NewListOpts(nil),
		env:     opts.NewListOpts(opts.ValidateEnv),
		envFile: opts.NewListOpts(nil),
		publish: opts.NewListOpts(nil),
		expose:  opts.NewListOpts(nil),
	}
	// General purpose flags
	flags.VarP(&copts.attach, "attach", "a", "Attach to STDIN, STDOUT or STDERR")
	flags.BoolVarP(&copts.stdin, "interactive", "i", false, "Keep STDIN open even if not attached")
	flags.BoolVarP(&copts.tty, "tty", "t", false, "Allocate a pseudo-TTY")
	flags.VarP(&copts.volumes, "volume", "v", "Bind mount a volume")
	flags.VarP(&copts.env, "env", "e", "Set environment variables")
	flags.Var(&copts.envFile, "env-file", "Read in a file of environment variables")
	flags.StringVarP(&copts.hostname, "hostname", "h", "", "Container host name")
	flags.StringVar(&copts.domainname, "domainname", "", "Container NIS domain name")
	flags.BoolVar(&copts.autoRemove, "rm", false, "Automatically remove the container and its associated anonymous volumes when it exits")
	// Resource management
	flags.Uint16Var(&copts.blkioWeight, "blkio-weight", 0, "Block IO (relative weight), between 10 and 1000, or 0 to disable (default 0)")
	// flags.Var(&copts.blkioWeightDevice, "blkio-weight-device", "Block IO weight (relative device weight)")
	// flags.StringVar(&copts.containerIDFile, "cidfile", "", "Write the container ID to the file")
	flags.StringVar(&copts.cpusetCpus, "cpuset-cpus", "", "CPUs in which to allow execution (0-3, 0,1)")
	flags.StringVar(&copts.cpusetMems, "cpuset-mems", "", "MEMs in which to allow execution (0-3, 0,1)")
	// flags.Int64Var(&copts.cpuCount, "cpu-count", 0, "CPU count (Windows only)")
	// flags.SetAnnotation("cpu-count", "ostype", []string{"windows"})
	// flags.Int64Var(&copts.cpuPercent, "cpu-percent", 0, "CPU percent (Windows only)")
	// flags.SetAnnotation("cpu-percent", "ostype", []string{"windows"})
	flags.Uint64Var(&copts.cpuPeriod, "cpu-period", 0, "Limit CPU CFS (Completely Fair Scheduler) period")
	flags.Int64Var(&copts.cpuQuota, "cpu-quota", 0, "Limit CPU CFS (Completely Fair Scheduler) quota")
	flags.Uint64Var(&copts.cpuRealtimePeriod, "cpu-rt-period", 0, "Limit CPU real-time period in microseconds")
	flags.SetAnnotation("cpu-rt-period", "version", []string{"1.25"})
	flags.Int64Var(&copts.cpuRealtimeRuntime, "cpu-rt-runtime", 0, "Limit CPU real-time runtime in microseconds")
	flags.SetAnnotation("cpu-rt-runtime", "version", []string{"1.25"})
	flags.Uint64VarP(&copts.cpuShares, "cpu-shares", "c", 0, "CPU shares (relative weight)")
	flags.Var(&copts.cpus, "cpus", "Number of CPUs")
	flags.SetAnnotation("cpus", "version", []string{"1.25"})
	// flags.Var(&copts.deviceReadBps, "device-read-bps", "Limit read rate (bytes per second) from a device")
	// flags.Var(&copts.deviceReadIOps, "device-read-iops", "Limit read rate (IO per second) from a device")
	// flags.Var(&copts.deviceWriteBps, "device-write-bps", "Limit write rate (bytes per second) to a device")
	// flags.Var(&copts.deviceWriteIOps, "device-write-iops", "Limit write rate (IO per second) to a device")
	// flags.Var(&copts.ioMaxBandwidth, "io-maxbandwidth", "Maximum IO bandwidth limit for the system drive (Windows only)")
	// flags.SetAnnotation("io-maxbandwidth", "ostype", []string{"windows"})
	// flags.Uint64Var(&copts.ioMaxIOps, "io-maxiops", 0, "Maximum IOps limit for the system drive (Windows only)")
	// flags.SetAnnotation("io-maxiops", "ostype", []string{"windows"})
	// flags.Var(&copts.kernelMemory, "kernel-memory", "Kernel memory limit")
	flags.VarP(&copts.memory, "memory", "m", "Memory limit")
	flags.Var(&copts.memoryReservation, "memory-reservation", "Memory soft limit")
	flags.Var(&copts.memorySwap, "memory-swap", "Swap limit equal to memory plus swap: '-1' to enable unlimited swap")
	flags.Int64Var(&copts.swappiness, "memory-swappiness", -1, "Tune container memory swappiness (0 to 100)")
	// flags.BoolVar(&copts.oomKillDisable, "oom-kill-disable", false, "Disable OOM Killer")
	// flags.IntVar(&copts.oomScoreAdj, "oom-score-adj", 0, "Tune host's OOM preferences (-1000 to 1000)")
	flags.Int64Var(&copts.pidsLimit, "pids-limit", 0, "Tune container pids limit (set -1 for unlimited)")

	flags.VarP(&copts.publish, "publish", "p", "Publish a container's port(s) to the host")
	flags.StringVar(&copts.netMode, "net", "", "Connect a container to a network")
	flags.Var(&copts.expose, "expose", "Expose a port or a range of ports")

	return copts
}

type containerConfig struct {
	Config           *config.Config
	HostConfig       *config.HostConfig
	NetworkingConfig *config.NetworkingConfig
}

func parse(flags *pflag.FlagSet, copts *containerOptions) (*containerConfig, error) {
	var (
		attachStdin  = copts.attach.Get("stdin")
		attachStdout = copts.attach.Get("stdout")
		attachStderr = copts.attach.Get("stderr")
	)
	// -i
	if copts.stdin {
		attachStdin = true
	}
	if copts.attach.Len() == 0 {
		attachStdout = true
		attachStderr = true
	}

	// 解析-v参数
	var binds []string
	volumes := copts.volumes.GetMap()
	for bind := range copts.volumes.GetMap() {
		parsed, err := loader.ParseVolume(bind)
		if err != nil {
			return nil, err
		}

		if parsed.Source != "" {
			toBind := bind

			if parsed.Type == string(config.TypeBind) {
				if hostPart, targetPath, ok := strings.Cut(bind, ":"); ok {
					if strings.HasPrefix(hostPart, "."+string(filepath.Separator)) || hostPart == "." {
						if absHostPart, err := filepath.Abs(hostPart); err == nil {
							hostPart = absHostPart
						}
					}
					toBind = hostPart + ":" + targetPath
				}
			}

			// after creating the bind mount we want to delete it from the copts.volumes values because
			// we do not want bind mounts being committed to image configs
			binds = append(binds, toBind)
			// We should delete from the map (`volumes`) here, as deleting from copts.volumes will not work if
			// there are duplicates entries.
			delete(volumes, bind)
		}
	}

	var (
		runCmd []string
		// entrypoint strslice.StrSlice
	)

	if len(copts.Args) > 0 {
		runCmd = copts.Args
	}

	// 解析 -p
	publishOpts := copts.publish.GetAll()

	// 解析 -e 参数
	envVariables, err := opts.ReadKVEnvStrings(copts.envFile.GetAll(), copts.env.GetAll())
	if err != nil {
		return nil, err
	}

	resources := config.Resources{
		Memory:            copts.memory.Value(),
		MemoryReservation: copts.memoryReservation.Value(),
		MemorySwap:        copts.memorySwap.Value(),
		MemorySwappiness:  &copts.swappiness,
		CpuPeriod:         copts.cpuPeriod,
		CpuQuota:          copts.cpuQuota,
		CpuShares:         copts.cpuShares,
		CpuRtRuntime:      copts.cpuRealtimeRuntime,
		CpuRtPeriod:       copts.cpuRealtimePeriod,
		CpusetCpus:        copts.cpusetCpus,
		CpusetMems:        copts.cpusetMems,
		PidsLimit:         copts.pidsLimit,
		BlkioWeight:       copts.blkioWeight,
	}

	generalConfig := &config.Config{
		Hostname:     copts.hostname,
		Domainname:   copts.domainname,
		OpenStdin:    copts.stdin,
		AttachStdin:  attachStdin,
		AttachStdout: attachStdout,
		AttachStderr: attachStderr,
		Cmd:          runCmd,
		Image:        copts.Image,
		Tty:          copts.tty,
		Env:          envVariables,
	}

	hostConfig := &config.HostConfig{
		Binds:        binds,
		Resources:    &resources,
		PortBindings: publishOpts,
		AutoRemove:   copts.autoRemove,
	}

	networkingConfig := &config.NetworkingConfig{
		Endpoints: copts.netMode,
	}

	return &containerConfig{
		Config:           generalConfig,
		HostConfig:       hostConfig,
		NetworkingConfig: networkingConfig,
	}, nil
}

func validateAttach(val string) (string, error) {
	s := strings.ToLower(val)
	for _, str := range []string{"stdin", "stdout", "stderr"} {
		if s == str {
			return s, nil
		}
	}
	return val, errors.Errorf("valid streams are STDIN, STDOUT and STDERR")
}

func convertToStandardNotation(ports []string) ([]string, error) {
	optsList := []string{}
	for _, publish := range ports {
		if strings.Contains(publish, "=") {
			params := map[string]string{"protocol": "tcp"}
			for _, param := range strings.Split(publish, ",") {
				k, v, ok := strings.Cut(param, "=")
				if !ok || k == "" {
					return optsList, errors.Errorf("invalid publish opts format (should be name=value but got '%s')", param)
				}
				params[k] = v
			}
			optsList = append(optsList, fmt.Sprintf("%s:%s/%s", params["published"], params["target"], params["protocol"]))
		} else {
			optsList = append(optsList, publish)
		}
	}
	return optsList, nil
}

func parseNetworkAttachmentOpt(ep opts.NetworkAttachmentOpts) (*config.EndpointSettings, error) {
	if strings.TrimSpace(ep.Target) == "" {
		return nil, errors.New("no name set for network")
	}
	epConfig := &config.EndpointSettings{}
	// epConfig.Aliases = append(epConfig.Aliases, ep.Aliases...)
	// if len(ep.DriverOpts) > 0 {
	// 	epConfig.DriverOpts = make(map[string]string)
	// 	epConfig.DriverOpts = ep.DriverOpts
	// }
	if len(ep.Links) > 0 {
		epConfig.Links = ep.Links
	}
	if ep.IPv4Address != "" || ep.IPv6Address != "" || len(ep.LinkLocalIPs) > 0 {
		epConfig.IPAMConfig = &config.EndpointIPAMConfig{
			IPv4Address:  ep.IPv4Address,
			IPv6Address:  ep.IPv6Address,
			LinkLocalIPs: ep.LinkLocalIPs,
		}
	}
	return epConfig, nil

}
