package container

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/DeJeune/sudocker/cli/opts"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"tags.cncf.io/container-device-interface/pkg/cdi"
)

var deviceCgroupRuleRegexp = regexp.MustCompile(`^[acb] ([0-9]+|\*):([0-9]+|\*) [rwm]{1,3}$`)

type containerOptions struct {
	// 加载卷
	volumes opts.ListOpts
	// 卷，用于持久化数据
	tmpfs opts.ListOpts

	// blkioWeightDevice opts.WeightdeviceOpt
	// deviceReadBps     opts.ThrottledeviceOpt
	// deviceWriteBps    opts.ThrottledeviceOpt
	// links             opts.ListOpts
	// 容器网络别名
	// aliases         opts.ListOpts
	// linkLocalIPs    opts.ListOpts
	// deviceReadIOps  opts.ThrottledeviceOpt
	// deviceWriteIOps opts.ThrottledeviceOpt
	// 环境变量
	env opts.ListOpts
	// 设置容器元数据
	// labels opts.ListOpts
	// 允许您指定容器可以访问哪些设备，以及访问设备的权限
	deviceCgroupRules opts.ListOpts
	devices           opts.ListOpts
	// 连接到host指定端口
	publish opts.ListOpts
	// sysctls *opts.MapOpts
	expose  opts.ListOpts
	// 共享卷
	volumesFrom opts.ListOpts
	// 环境文件
	envFile            opts.ListOpts
	// capAdd             opts.ListOpts
	// capDrop            opts.ListOpts
	// groupAdd           opts.ListOpts
	// storageOpt         opts.ListOpts
	// labelsFile         opts.ListOpts
	// privileged         bool
	// pidMode            string
	// utsMode            string
	// usernsMode         string
	// cgroupnsMode       string
	// stdin              bool
	// publishAll         bool
	// oomKillDisable     bool
	// oomScoreAdj        int
	// entrypoint         string
	// containerIDFile    string
	// hostname           string
	// domainname         string
	// memory             opts.MemBytes
	// memoryReservation  opts.MemBytes
	// memorySwap         opts.MemSwapBytes
	// kernelMemory       opts.MemBytes
	// user               string
	// workingDir         string
	// cpuCount           int64
	// cpuShares          int64
	// cpuPercent         int64
	// cpuPeriod          int64
	// cpuRealtimePeriod  int64
	// cpuRealtimeRuntime int64
	// cpuQuota           int64
	// cpus               opts.NanoCPUs
	// cpusetCpus         string
	// cpusetMems         string
	// blkioWeight        uint16
	// ioMaxBandwidth     opts.MemBytes
	// ioMaxIOps          uint64
	// swappiness         int64
	netMode            opts.NetworkOpt
	macAddress         string
	ipv4Address        string
	ipv6Address        string
	// ipcMode            string
	// pidsLimit          int64
	// restartPolicy      string
	// readonlyRootfs     bool
	// cgroupParent       string
	volumeDriver       string
	// stopSignal         string
	// stopTimeout        int
	// isolation          string
	// shmSize            opts.MemBytes
	// healthRetries      int
	// runtime            string
	// autoRemove         bool
	// init               bool
	// annotations        *opts.MapOpts

	Image string
	Args  []string
}

func parseFlags(flags *pflag.FlagSet) *containerOptions {
	copts := &containerOptions{
		aliases:           opts.NewListOpts(nil),
		blkioWeightDevice: opts.NewWeightdeviceOpt(opts.ValidateWeightDevice),
		capAdd:            opts.NewListOpts(nil),
		capDrop:           opts.NewListOpts(nil),
		deviceCgroupRules: opts.NewListOpts(validateDeviceCgroupRule),
		deviceReadBps:     opts.NewThrottledeviceOpt(opts.ValidateThrottleBpsDevice),
		deviceReadIOps:    opts.NewThrottledeviceOpt(opts.ValidateThrottleIOpsDevice),
		deviceWriteBps:    opts.NewThrottledeviceOpt(opts.ValidateThrottleBpsDevice),
		deviceWriteIOps:   opts.NewThrottledeviceOpt(opts.ValidateThrottleIOpsDevice),
		devices:           opts.NewListOpts(nil), // devices can only be validated after we know the server OS
		env:               opts.NewListOpts(opts.ValidateEnv),
		envFile:           opts.NewListOpts(nil),
		expose:            opts.NewListOpts(nil),
		groupAdd:          opts.NewListOpts(nil),
		labels:            opts.NewListOpts(opts.ValidateLabel),
		labelsFile:        opts.NewListOpts(nil),
		linkLocalIPs:      opts.NewListOpts(nil),
		links:             opts.NewListOpts(opts.ValidateLink),
		storageOpt:        opts.NewListOpts(nil),
		sysctls:           opts.NewMapOpts(nil, opts.ValidateSysctl),
		tmpfs:             opts.NewListOpts(nil),
		volumes:           opts.NewListOpts(nil),
		volumesFrom:       opts.NewListOpts(nil),
		annotations:       opts.NewMapOpts(nil, nil),
	}

	// flags.Var(&copts.deviceCgroupRules, "device-cgroup-rule", "Add a rule to the cgroup allowed devices list")
	// flags.Var(&copts.devices, "device", "Add a host device to the container")

	flags.VarP(&copts.env, "env", "e", "Set environment variables")
	flags.Var(&copts.envFile, "env-file", "Read in a file of environment variables")
	flags.StringVar(&copts.entrypoint, "entrypoint", "", "Overwrite the default ENTRYPOINT of the image")
	// flags.Var(&copts.groupAdd, "group-add", "Add additional groups to join")
	flags.StringVarP(&copts.hostname, "hostname", "h", "", "Container host name")
	// flags.StringVar(&copts.domainname, "domainname", "", "Container NIS domain name")
	flags.BoolVarP(&copts.stdin, "interactive", "i", false, "Keep STDIN open even if not attached")
	flags.VarP(&copts.publish,"publish","p","Publish the container's port to the host")
	flags.SetAnnotation("stop-timeout", "version", []string{"1.25"})
	flags.Var(copts.sysctls, "sysctl", "Sysctl options")

	flags.StringVarP(&copts.user, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	// flags.StringVarP(&copts.workingDir, "workdir", "w", "", "Working directory inside the container")
	flags.BoolVar(&copts.autoRemove, "rm", false, "Automatically remove the container when it exits")

	// Security
// 	flags.Var(&copts.capAdd, "cap-add", "Add Linux capabilities")
// 	flags.Var(&copts.capDrop, "cap-drop", "Drop Linux capabilities")
// 	flags.BoolVar(&copts.privileged, "privileged", false, "Give extended privileges to this container")
// 	flags.StringVar(&copts.usernsMode, "userns", "", "User namespace to use")
// 	flags.StringVar(&copts.cgroupnsMode, "cgroupns", "", `Cgroup namespace to use (host|private)
// 'host':    Run the container in the Docker host's cgroup namespace
// 'private': Run the container in its own private cgroup namespace
// '':        Use the cgroup namespace as configured by the
//            default-cgroupns-mode option on the daemon (default)`)
// 	flags.SetAnnotation("cgroupns", "version", []string{"1.41"})

	flags.Var(&copts.expose, "expose", "Expose a port or a range of ports")
	flags.StringVar(&copts.ipv4Address, "ip", "", "IPv4 address (e.g., 172.30.100.104)")
	flags.StringVar(&copts.ipv6Address, "ip6", "", "IPv6 address (e.g., 2001:db8::33)")
	flags.Var(&copts.links, "link", "Add link to another container")
	flags.Var(&copts.linkLocalIPs, "link-local-ip", "Container IPv4/IPv6 link-local addresses")
	flags.StringVar(&copts.macAddress, "mac-address", "", "Container MAC address (e.g., 92:d0:c6:0a:29:33)")
	// We allow for both "--net" and "--network", although the latter is the recommended way.
	flags.Var(&copts.netMode, "net", "Connect a container to a network")
	flags.Var(&copts.netMode, "network", "Connect a container to a network")
	flags.MarkHidden("net")
	// We allow for both "--net-alias" and "--network-alias", although the latter is the recommended way.
	flags.Var(&copts.aliases, "net-alias", "Add network-scoped alias for the container")
	flags.Var(&copts.aliases, "network-alias", "Add network-scoped alias for the container")
	flags.MarkHidden("net-alias")

	// Logging and storage
	// flags.StringVar(&copts.loggingDriver, "log-driver", "", "Logging driver for the container")
	flags.StringVar(&copts.volumeDriver, "volume-driver", "", "Optional volume driver for the container")
	flags.Var(&copts.storageOpt, "storage-opt", "Storage driver options for the container")
	flags.Var(&copts.tmpfs, "tmpfs", "Mount a tmpfs directory")
	flags.Var(&copts.volumesFrom, "volumes-from", "Mount volumes from the specified container(s)")
	flags.VarP(&copts.volumes, "volume", "v", "Bind mount a volume")
	// flags.Var(&copts.mounts, "mount", "Attach a filesystem mount to the container")


	// Resource management
	// flags.Uint16Var(&copts.blkioWeight, "blkio-weight", 0, "Block IO (relative weight), between 10 and 1000, or 0 to disable (default 0)")
	// flags.Var(&copts.blkioWeightDevice, "blkio-weight-device", "Block IO weight (relative device weight)")
	// flags.StringVar(&copts.containerIDFile, "cidfile", "", "Write the container ID to the file")
	// flags.StringVar(&copts.cpusetCpus, "cpuset-cpus", "", "CPUs in which to allow execution (0-3, 0,1)")
	// flags.StringVar(&copts.cpusetMems, "cpuset-mems", "", "MEMs in which to allow execution (0-3, 0,1)")
	// flags.Int64Var(&copts.cpuCount, "cpu-count", 0, "CPU count (Windows only)")
	// flags.SetAnnotation("cpu-count", "ostype", []string{"windows"})
	// flags.Int64Var(&copts.cpuPercent, "cpu-percent", 0, "CPU percent (Windows only)")
	// flags.SetAnnotation("cpu-percent", "ostype", []string{"windows"})
	// flags.Int64Var(&copts.cpuPeriod, "cpu-period", 0, "Limit CPU CFS (Completely Fair Scheduler) period")
	// flags.Int64Var(&copts.cpuQuota, "cpu-quota", 0, "Limit CPU CFS (Completely Fair Scheduler) quota")
	// flags.Int64Var(&copts.cpuRealtimePeriod, "cpu-rt-period", 0, "Limit CPU real-time period in microseconds")
	// flags.SetAnnotation("cpu-rt-period", "version", []string{"1.25"})
	// flags.Int64Var(&copts.cpuRealtimeRuntime, "cpu-rt-runtime", 0, "Limit CPU real-time runtime in microseconds")
	// flags.SetAnnotation("cpu-rt-runtime", "version", []string{"1.25"})
	// flags.Int64VarP(&copts.cpuShares, "cpu-shares", "c", 0, "CPU shares (relative weight)")
	// flags.Var(&copts.cpus, "cpus", "Number of CPUs")
	// flags.SetAnnotation("cpus", "version", []string{"1.25"})
	// flags.Var(&copts.deviceReadBps, "device-read-bps", "Limit read rate (bytes per second) from a device")
	// flags.Var(&copts.deviceReadIOps, "device-read-iops", "Limit read rate (IO per second) from a device")
	// flags.Var(&copts.deviceWriteBps, "device-write-bps", "Limit write rate (bytes per second) to a device")
	// flags.Var(&copts.deviceWriteIOps, "device-write-iops", "Limit write rate (IO per second) to a device")
	// flags.Var(&copts.ioMaxBandwidth, "io-maxbandwidth", "Maximum IO bandwidth limit for the system drive (Windows only)")
	// flags.SetAnnotation("io-maxbandwidth", "ostype", []string{"windows"})
	// flags.Uint64Var(&copts.ioMaxIOps, "io-maxiops", 0, "Maximum IOps limit for the system drive (Windows only)")
	// flags.SetAnnotation("io-maxiops", "ostype", []string{"windows"})
	// flags.Var(&copts.kernelMemory, "kernel-memory", "Kernel memory limit")
	// flags.VarP(&copts.memory, "memory", "m", "Memory limit")
	// flags.Var(&copts.memoryReservation, "memory-reservation", "Memory soft limit")
	// flags.Var(&copts.memorySwap, "memory-swap", "Swap limit equal to memory plus swap: '-1' to enable unlimited swap")
	// flags.Int64Var(&copts.swappiness, "memory-swappiness", -1, "Tune container memory swappiness (0 to 100)")
	// flags.BoolVar(&copts.oomKillDisable, "oom-kill-disable", false, "Disable OOM Killer")
	// flags.IntVar(&copts.oomScoreAdj, "oom-score-adj", 0, "Tune host's OOM preferences (-1000 to 1000)")
	// flags.Int64Var(&copts.pidsLimit, "pids-limit", 0, "Tune container pids limit (set -1 for unlimited)")

	// // Low-level execution (cgroups, namespaces, ...)
	// flags.StringVar(&copts.cgroupParent, "cgroup-parent", "", "Optional parent cgroup for the container")
	// flags.StringVar(&copts.ipcMode, "ipc", "", "IPC mode to use")
	// flags.StringVar(&copts.isolation, "isolation", "", "Container isolation technology")
	// flags.StringVar(&copts.pidMode, "pid", "", "PID namespace to use")
	// flags.Var(&copts.shmSize, "shm-size", "Size of /dev/shm")
	// flags.StringVar(&copts.utsMode, "uts", "", "UTS namespace to use")
	// flags.StringVar(&copts.runtime, "runtime", "", "Runtime to use for this container")

	// flags.BoolVar(&copts.init, "init", false, "Run an init inside the container that forwards signals and reaps processes")
	// flags.SetAnnotation("init", "version", []string{"1.25"})

	// flags.Var(copts.annotations, "annotation", "Add an annotation to the container (passed through to the OCI runtime)")
	// flags.SetAnnotation("annotation", "version", []string{"1.43"})

	return copts
}

type containerConfig struct {
	Config           *container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *networktypes.NetworkingConfig
}

func parse(flags *pflag.FlagSet, copts *containerOptions, serverOS string) (*containerConfig, error) {
	attachStdin := true
	attachStdout := true
	attachStderr := true
	// Validate the input mac address
	if copts.macAddress != "" {
		if _, err := opts.ValidateMACAddress(copts.macAddress); err != nil {
			return nil, errors.Errorf("%s is not a valid mac address", copts.macAddress)
		}
	}

	var err error

	swappiness := copts.swappiness
	if swappiness != -1 && (swappiness < 0 || swappiness > 100) {
		return nil, errors.Errorf("invalid value: %d. Valid memory swappiness range is 0-100", swappiness)
	}

	// mounts := copts.mounts.Value()
	// if len(mounts) > 0 && copts.volumeDriver != "" {
	// 	logrus.Warn("`--volume-driver` is ignored for volumes specified via `--mount`. Use `--mount type=volume,volume-driver=...` instead.")
	// }
	// var binds []string
	// volumes := copts.volumes.GetMap()
	// // add any bind targets to the list of container volumes
	// for bind := range copts.volumes.GetMap() {
	// 	parsed, err := loader.ParseVolume(bind)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if parsed.Source != "" {
	// 		toBind := bind

	// 		if parsed.Type == string(mounttypes.TypeBind) {
	// 			if hostPart, targetPath, ok := strings.Cut(bind, ":"); ok {
	// 				if strings.HasPrefix(hostPart, "."+string(filepath.Separator)) || hostPart == "." {
	// 					if absHostPart, err := filepath.Abs(hostPart); err == nil {
	// 						hostPart = absHostPart
	// 					}
	// 				}
	// 				toBind = hostPart + ":" + targetPath
	// 			}
	// 		}

	// 		// after creating the bind mount we want to delete it from the copts.volumes values because
	// 		// we do not want bind mounts being committed to image configs
	// 		binds = append(binds, toBind)
	// 		// We should delete from the map (`volumes`) here, as deleting from copts.volumes will not work if
	// 		// there are duplicates entries.
	// 		delete(volumes, bind)
	// 	}
	// }

	// Can't evaluate options passed into --tmpfs until we actually mount
	tmpfs := make(map[string]string)
	for _, t := range copts.tmpfs.GetAll() {
		k, v, _ := strings.Cut(t, ":")
		tmpfs[k] = v
	}

	var (
		runCmd     strslice.StrSlice
		entrypoint strslice.StrSlice
	)

	if len(copts.Args) > 0 {
		runCmd = copts.Args
	}

	if copts.entrypoint != "" {
		entrypoint = strslice.StrSlice{copts.entrypoint}
	} else if flags.Changed("entrypoint") {
		// if `--entrypoint=` is parsed then Entrypoint is reset
		entrypoint = []string{""}
	}

	publishOpts := copts.publish.GetAll()
	var (
		ports         map[nat.Port]struct{}
		portBindings  map[nat.Port][]nat.PortBinding
		convertedOpts []string
	)

	convertedOpts, err = convertToStandardNotation(publishOpts)
	if err != nil {
		return nil, err
	}

	ports, portBindings, err = nat.ParsePortSpecs(convertedOpts)
	if err != nil {
		return nil, err
	}

	// Merge in exposed ports to the map of published ports
	for _, e := range copts.expose.GetAll() {
		if strings.Contains(e, ":") {
			return nil, errors.Errorf("invalid port format for --expose: %s", e)
		}
		// support two formats for expose, original format <portnum>/[<proto>]
		// or <startport-endport>/[<proto>]
		proto, port := nat.SplitProtoPort(e)
		// parse the start and end port and create a sequence of ports to expose
		// if expose a port, the start and end port are the same
		start, end, err := nat.ParsePortRange(port)
		if err != nil {
			return nil, errors.Errorf("invalid range format for --expose: %s, error: %s", e, err)
		}
		for i := start; i <= end; i++ {
			p, err := nat.NewPort(proto, strconv.FormatUint(i, 10))
			if err != nil {
				return nil, err
			}
			if _, exists := ports[p]; !exists {
				ports[p] = struct{}{}
			}
		}
	}

	// validate and parse device mappings. Note we do late validation of the
	// device path (as opposed to during flag parsing), as at the time we are
	// parsing flags, we haven't yet sent a _ping to the daemon to determine
	// what operating system it is.
	deviceMappings := []container.DeviceMapping{}
	var cdiDeviceNames []string
	for _, device := range copts.devices.GetAll() {
		var (
			validated     string
			deviceMapping container.DeviceMapping
			err           error
		)
		if cdi.IsQualifiedName(device) {
			cdiDeviceNames = append(cdiDeviceNames, device)
			continue
		}
		validated, err = validateDevice(device, serverOS)
		if err != nil {
			return nil, err
		}
		deviceMapping, err = parseDevice(validated, serverOS)
		if err != nil {
			return nil, err
		}
		deviceMappings = append(deviceMappings, deviceMapping)
	}

	// collect all the environment variables for the container
	envVariables, err := opts.ReadKVEnvStrings(copts.envFile.GetAll(), copts.env.GetAll())
	if err != nil {
		return nil, err
	}

	// collect all the labels for the container
	labels, err := opts.ReadKVStrings(copts.labelsFile.GetAll(), copts.labels.GetAll())
	if err != nil {
		return nil, err
	}

	pidMode := container.PidMode(copts.pidMode)
	if !pidMode.Valid() {
		return nil, errors.Errorf("--pid: invalid PID mode")
	}

	// utsMode := container.UTSMode(copts.utsMode)
	// if !utsMode.Valid() {
	// 	return nil, errors.Errorf("--uts: invalid UTS mode")
	// }

	// usernsMode := container.UsernsMode(copts.usernsMode)
	// if !usernsMode.Valid() {
	// 	return nil, errors.Errorf("--userns: invalid USER mode")
	// }

	// cgroupnsMode := container.CgroupnsMode(copts.cgroupnsMode)
	// if !cgroupnsMode.Valid() {
	// 	return nil, errors.Errorf("--cgroupns: invalid CGROUP mode")
	}

	// restartPolicy, err := opts.ParseRestartPolicy(copts.restartPolicy)result := make(map[string]string, len(values))
	// if err != nil {
	// 	return nil, err
	// }

	// securityOpts, maskedPaths, readonlyPaths := parseSystemPaths(securityOpts)

	// deviceRequests := copts.gpus.Value()
	// if len(cdiDeviceNames) > 0 {
	// 	cdiDeviceRequest := container.DeviceRequest{
	// 		Driver:    "cdi",
	// 		DeviceIDs: cdiDeviceNames,
	// 	}
	// 	deviceRequests = append(deviceRequests, cdiDeviceRequest)
	// }

	// resources := container.Resources{
		
	// }

	config := &container.Config{
		Hostname:     copts.hostname,
		Domainname:   copts.domainname,
		ExposedPorts: ports,
		User:         copts.user,
		OpenStdin:    copts.stdin,
		AttachStdin:  attachStdin,
		AttachStdout: attachStdout,
		AttachStderr: attachStderr,
		Env:          envVariables,
		Cmd:          runCmd,
		Image:        copts.Image,
		Volumes:      volumes,
		MacAddress:   copts.macAddress,
		Entrypoint:   entrypoint,
	}
	// if flags.Changed("stop-timeout") {
	// 	config.StopTimeout = &copts.stopTimeout
	// }

	hostConfig := &container.HostConfig{
		Binds:           binds,
		ContainerIDFile: copts.containerIDFile,
		AutoRemove:      copts.autoRemove,
		Privileged:      copts.privileged,
		PortBindings:    portBindings,
		Links:           copts.links.GetAll(),
		PublishAllPorts: copts.publishAll,
		// Make sure the dns fields are never nil.
		// New containers don't ever have those fields nil,
		// but pre created containers can still have those nil values.
		// See https://github.com/docker/docker/pull/17779
		// for a more detailed explanation on why we don't want that.
		DNS:            copts.dns.GetAllOrEmpty(),
		DNSSearch:      copts.dnsSearch.GetAllOrEmpty(),
		DNSOptions:     copts.dnsOptions.GetAllOrEmpty(),
		ExtraHosts:     copts.extraHosts.GetAll(),
		VolumesFrom:    copts.volumesFrom.GetAll(),
		IpcMode:        container.IpcMode(copts.ipcMode),
		NetworkMode:    container.NetworkMode(copts.netMode.NetworkMode()),
		// PidMode:        pidMode,
		// UTSMode:        utsMode,
		// UsernsMode:     usernsMode,
		// CgroupnsMode:   cgroupnsMode,
		// CapAdd:         strslice.StrSlice(copts.capAdd.GetAll()),
		// CapDrop:        strslice.StrSlice(copts.capDrop.GetAll()),
		// GroupAdd:       copts.groupAdd.GetAll(),
		// RestartPolicy:  restartPolicy,
		// SecurityOpt:    securityOpts,
		StorageOpt:     storageOpts,
		ReadonlyRootfs: copts.readonlyRootfs,
		LogConfig:      container.LogConfig{Type: copts.loggingDriver, Config: loggingOpts},
		VolumeDriver:   copts.volumeDriver,
		Isolation:      container.Isolation(copts.isolation),
		// ShmSize:        copts.shmSize.Value(),
		// Resources:      resources,
		Tmpfs:          tmpfs,
		Sysctls:        copts.sysctls.GetAll(),
		Runtime:        copts.runtime,
		Mounts:         mounts,
		MaskedPaths:    maskedPaths,
		ReadonlyPaths:  readonlyPaths,
		Annotations:    copts.annotations.GetAll(),
	}

	// if copts.autoRemove && !hostConfig.RestartPolicy.IsNone() {
	// 	return nil, errors.Errorf("Conflicting options: --restart and --rm")
	// }

	// only set this value if the user provided the flag, else it should default to nil
	if flags.Changed("init") {
		hostConfig.Init = &copts.init
	}

	// When allocating stdin in attached mode, close stdin at client disconnect
	if config.OpenStdin && config.AttachStdin {
		config.StdinOnce = true
	}

	networkingConfig := &networktypes.NetworkingConfig{
		EndpointsConfig: make(map[string]*networktypes.EndpointSettings),
	}

	networkingConfig.EndpointsConfig, err = parseNetworkOpts(copts)
	if err != nil {
		return nil, err
	}

	// Put the endpoint-specific MacAddress of the "main" network attachment into the container Config for backward
	// compatibility with older daemons.
	if nw, ok := networkingConfig.EndpointsConfig[hostConfig.NetworkMode.NetworkName()]; ok {
		config.MacAddress = nw.MacAddress //nolint:staticcheck // ignore SA1019: field is deprecated, but still used on API < v1.44.
	}

	return &containerConfig{
		Config:           config,
		HostConfig:       hostConfig,
		NetworkingConfig: networkingConfig,
	}, nil
}

func parseNetworkOpts(copts *containerOptions) (map[string]*networktypes.EndpointSettings, error) {
	var (
		endpoints                         = make(map[string]*networktypes.EndpointSettings, len(copts.netMode.Value()))
		hasUserDefined, hasNonUserDefined bool
	)

	if len(copts.netMode.Value()) == 0 {
		n := opts.NetworkAttachmentOpts{
			Target: "default",
		}
		if err := applyContainerOptions(&n, copts); err != nil {
			return nil, err
		}
		ep, err := parseNetworkAttachmentOpt(n)
		if err != nil {
			return nil, err
		}
		endpoints["default"] = ep
	}

	for i, n := range copts.netMode.Value() {
		n := n
		if container.NetworkMode(n.Target).IsUserDefined() {
			hasUserDefined = true
		} else {
			hasNonUserDefined = true
		}
		if i == 0 {
			// The first network corresponds with what was previously the "only"
			// network, and what would be used when using the non-advanced syntax
			// `--network-alias`, `--link`, `--ip`, `--ip6`, and `--link-local-ip`
			// are set on this network, to preserve backward compatibility with
			// the non-advanced notation
			if err := applyContainerOptions(&n, copts); err != nil {
				return nil, err
			}
		}
		ep, err := parseNetworkAttachmentOpt(n)
		if err != nil {
			return nil, err
		}
		if _, ok := endpoints[n.Target]; ok {
			return nil, errdefs.InvalidParameter(errors.Errorf("network %q is specified multiple times", n.Target))
		}

		// For backward compatibility: if no custom options are provided for the network,
		// and only a single network is specified, omit the endpoint-configuration
		// on the client (the daemon will still create it when creating the container)
		if i == 0 && len(copts.netMode.Value()) == 1 {
			if ep == nil || reflect.DeepEqual(*ep, networktypes.EndpointSettings{}) {
				continue
			}
		}
		endpoints[n.Target] = ep
	}
	if hasUserDefined && hasNonUserDefined {
		return nil, errdefs.InvalidParameter(errors.New("conflicting options: cannot attach both user-defined and non-user-defined network-modes"))
	}
	return endpoints, nil
}

func applyContainerOptions(n *opts.NetworkAttachmentOpts, copts *containerOptions) error { //nolint:gocyclo
	// TODO should we error if _any_ advanced option is used? (i.e. forbid to combine advanced notation with the "old" flags (`--network-alias`, `--link`, `--ip`, `--ip6`)?
	if len(n.Aliases) > 0 && copts.aliases.Len() > 0 {
		return errdefs.InvalidParameter(errors.New("conflicting options: cannot specify both --network-alias and per-network alias"))
	}
	if len(n.Links) > 0 && copts.links.Len() > 0 {
		return errdefs.InvalidParameter(errors.New("conflicting options: cannot specify both --link and per-network links"))
	}
	if n.IPv4Address != "" && copts.ipv4Address != "" {
		return errdefs.InvalidParameter(errors.New("conflicting options: cannot specify both --ip and per-network IPv4 address"))
	}
	if n.IPv6Address != "" && copts.ipv6Address != "" {
		return errdefs.InvalidParameter(errors.New("conflicting options: cannot specify both --ip6 and per-network IPv6 address"))
	}
	if n.MacAddress != "" && copts.macAddress != "" {
		return errdefs.InvalidParameter(errors.New("conflicting options: cannot specify both --mac-address and per-network MAC address"))
	}
	if len(n.LinkLocalIPs) > 0 && copts.linkLocalIPs.Len() > 0 {
		return errdefs.InvalidParameter(errors.New("conflicting options: cannot specify both --link-local-ip and per-network link-local IP addresses"))
	}
	if copts.aliases.Len() > 0 {
		n.Aliases = make([]string, copts.aliases.Len())
		copy(n.Aliases, copts.aliases.GetAll())
	}
	if n.Target != "default" && copts.links.Len() > 0 {
		n.Links = make([]string, copts.links.Len())
		copy(n.Links, copts.links.GetAll())
	}
	if copts.ipv4Address != "" {
		n.IPv4Address = copts.ipv4Address
	}
	if copts.ipv6Address != "" {
		n.IPv6Address = copts.ipv6Address
	}
	if copts.macAddress != "" {
		n.MacAddress = copts.macAddress
	}
	if copts.linkLocalIPs.Len() > 0 {
		n.LinkLocalIPs = make([]string, copts.linkLocalIPs.Len())
		copy(n.LinkLocalIPs, copts.linkLocalIPs.GetAll())
	}
	return nil
}

func parseNetworkAttachmentOpt(ep opts.NetworkAttachmentOpts) (*networktypes.EndpointSettings, error) {
	if strings.TrimSpace(ep.Target) == "" {
		return nil, errors.New("no name set for network")
	}
	if !container.NetworkMode(ep.Target).IsUserDefined() {
		if len(ep.Aliases) > 0 {
			return nil, errors.New("network-scoped aliases are only supported for user-defined networks")
		}
		if len(ep.Links) > 0 {
			return nil, errors.New("links are only supported for user-defined networks")
		}
	}

	epConfig := &networktypes.EndpointSettings{}
	epConfig.Aliases = append(epConfig.Aliases, ep.Aliases...)
	if len(ep.DriverOpts) > 0 {
		epConfig.DriverOpts = make(map[string]string)
		epConfig.DriverOpts = ep.DriverOpts
	}
	if len(ep.Links) > 0 {
		epConfig.Links = ep.Links
	}
	if ep.IPv4Address != "" || ep.IPv6Address != "" || len(ep.LinkLocalIPs) > 0 {
		epConfig.IPAMConfig = &networktypes.EndpointIPAMConfig{
			IPv4Address:  ep.IPv4Address,
			IPv6Address:  ep.IPv6Address,
			LinkLocalIPs: ep.LinkLocalIPs,
		}
	}
	if ep.MacAddress != "" {
		if _, err := opts.ValidateMACAddress(ep.MacAddress); err != nil {
			return nil, errors.Errorf("%s is not a valid mac address", ep.MacAddress)
		}
		epConfig.MacAddress = ep.MacAddress
	}
	return epConfig, nil
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

func parseLoggingOpts(loggingDriver string, loggingOpts []string) (map[string]string, error) {
	loggingOptsMap := opts.ConvertKVStringsToMap(loggingOpts)
	if loggingDriver == "none" && len(loggingOpts) > 0 {
		return map[string]string{}, errors.Errorf("invalid logging opts for driver %s", loggingDriver)
	}
	return loggingOptsMap, nil
}

// takes a local seccomp daemon, reads the file contents for sending to the daemon
func parseSecurityOpts(securityOpts []string) ([]string, error) {
	for key, opt := range securityOpts {
		k, v, ok := strings.Cut(opt, "=")
		if !ok && k != "no-new-privileges" {
			k, v, ok = strings.Cut(opt, ":")
		}
		if (!ok || v == "") && k != "no-new-privileges" {
			// "no-new-privileges" is the only option that does not require a value.
			return securityOpts, errors.Errorf("Invalid --security-opt: %q", opt)
		}
		if k == "seccomp" {
			switch v {
			case seccompProfileDefault, seccompProfileUnconfined:
				// known special names for built-in profiles, nothing to do.
			default:
				// value may be a filename, in which case we send the profile's
				// content if it's valid JSON.
				f, err := os.ReadFile(v)
				if err != nil {
					return securityOpts, errors.Errorf("opening seccomp profile (%s) failed: %v", v, err)
				}
				b := bytes.NewBuffer(nil)
				if err := json.Compact(b, f); err != nil {
					return securityOpts, errors.Errorf("compacting json for seccomp profile (%s) failed: %v", v, err)
				}
				securityOpts[key] = fmt.Sprintf("seccomp=%s", b.Bytes())
			}
		}
	}

	return securityOpts, nil
}

// parseSystemPaths checks if `systempaths=unconfined` security option is set,
// and returns the `MaskedPaths` and `ReadonlyPaths` accordingly. An updated
// list of security options is returned with this option removed, because the
// `unconfined` option is handled client-side, and should not be sent to the
// daemon.
func parseSystemPaths(securityOpts []string) (filtered, maskedPaths, readonlyPaths []string) {
	filtered = securityOpts[:0]
	for _, opt := range securityOpts {
		if opt == "systempaths=unconfined" {
			maskedPaths = []string{}
			readonlyPaths = []string{}
		} else {
			filtered = append(filtered, opt)
		}
	}

	return filtered, maskedPaths, readonlyPaths
}

// parses storage options per container into a map
func parseStorageOpts(storageOpts []string) (map[string]string, error) {
	m := make(map[string]string)
	for _, option := range storageOpts {
		k, v, ok := strings.Cut(option, "=")
		if !ok {
			return nil, errors.Errorf("invalid storage option")
		}
		m[k] = v
	}
	return m, nil
}

// parseDevice parses a device mapping string to a container.DeviceMapping struct
func parseDevice(device, serverOS string) (container.DeviceMapping, error) {
	switch serverOS {
	case "linux":
		return parseLinuxDevice(device)
	case "windows":
		return parseWindowsDevice(device)
	}
	return container.DeviceMapping{}, errors.Errorf("unknown server OS: %s", serverOS)
}

// parseLinuxDevice parses a device mapping string to a container.DeviceMapping struct
// knowing that the target is a Linux daemon
func parseLinuxDevice(device string) (container.DeviceMapping, error) {
	var src, dst string
	permissions := "rwm"
	// We expect 3 parts at maximum; limit to 4 parts to detect invalid options.
	arr := strings.SplitN(device, ":", 4)
	switch len(arr) {
	case 3:
		permissions = arr[2]
		fallthrough
	case 2:
		if validDeviceMode(arr[1]) {
			permissions = arr[1]
		} else {
			dst = arr[1]
		}
		fallthrough
	case 1:
		src = arr[0]
	default:
		return container.DeviceMapping{}, errors.Errorf("invalid device specification: %s", device)
	}

	if dst == "" {
		dst = src
	}

	deviceMapping := container.DeviceMapping{
		PathOnHost:        src,
		PathInContainer:   dst,
		CgroupPermissions: permissions,
	}
	return deviceMapping, nil
}

// parseWindowsDevice parses a device mapping string to a container.DeviceMapping struct
// knowing that the target is a Windows daemon
func parseWindowsDevice(device string) (container.DeviceMapping, error) {
	return container.DeviceMapping{PathOnHost: device}, nil
}

// validateDeviceCgroupRule validates a device cgroup rule string format
// It will make sure 'val' is in the form:
//
//	'type major:minor mode'
func validateDeviceCgroupRule(val string) (string, error) {
	if deviceCgroupRuleRegexp.MatchString(val) {
		return val, nil
	}

	return val, errors.Errorf("invalid device cgroup format '%s'", val)
}

// validDeviceMode checks if the mode for device is valid or not.
// Valid mode is a composition of r (read), w (write), and m (mknod).
func validDeviceMode(mode string) bool {
	legalDeviceMode := map[rune]bool{
		'r': true,
		'w': true,
		'm': true,
	}
	if mode == "" {
		return false
	}
	for _, c := range mode {
		if !legalDeviceMode[c] {
			return false
		}
		legalDeviceMode[c] = false
	}
	return true
}

// validateDevice validates a path for devices
func validateDevice(val string, serverOS string) (string, error) {
	switch serverOS {
	case "linux":
		return validateLinuxPath(val, validDeviceMode)
	case "windows":
		// Windows does validation entirely server-side
		return val, nil
	}
	return "", errors.Errorf("unknown server OS: %s", serverOS)
}

// validateLinuxPath is the implementation of validateDevice knowing that the
// target server operating system is a Linux daemon.
// It will make sure 'val' is in the form:
//
//	[host-dir:]container-path[:mode]
//
// It also validates the device mode.
func validateLinuxPath(val string, validator func(string) bool) (string, error) {
	var containerPath string
	var mode string

	if strings.Count(val, ":") > 2 {
		return val, errors.Errorf("bad format for path: %s", val)
	}

	split := strings.SplitN(val, ":", 3)
	if split[0] == "" {
		return val, errors.Errorf("bad format for path: %s", val)
	}
	switch len(split) {
	case 1:
		containerPath = split[0]
		val = path.Clean(containerPath)
	case 2:
		if isValid := validator(split[1]); isValid {
			containerPath = split[0]
			mode = split[1]
			val = fmt.Sprintf("%s:%s", path.Clean(containerPath), mode)
		} else {
			containerPath = split[1]
			val = fmt.Sprintf("%s:%s", split[0], path.Clean(containerPath))
		}
	case 3:
		containerPath = split[1]
		mode = split[2]
		if isValid := validator(split[2]); !isValid {
			return val, errors.Errorf("bad mode specified: %s", mode)
		}
		val = fmt.Sprintf("%s:%s:%s", split[0], containerPath, mode)
	}

	if !path.IsAbs(containerPath) {
		return val, errors.Errorf("%s is not an absolute path", containerPath)
	}
	return val, nil
}

// validateAttach validates that the specified string is a valid attach option.
func validateAttach(val string) (string, error) {
	s := strings.ToLower(val)
	for _, str := range []string{"stdin", "stdout", "stderr"} {
		if s == str {
			return s, nil
		}
	}
	return val, errors.Errorf("valid streams are STDIN, STDOUT and STDERR")
}