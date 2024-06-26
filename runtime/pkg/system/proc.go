package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type State rune

const ( // Only values for Linux 3.14 and later are listed here
	Dead        State = 'X'
	DiskSleep   State = 'D'
	Running     State = 'R'
	Sleeping    State = 'S'
	Stopped     State = 'T'
	TracingStop State = 't'
	Zombie      State = 'Z'
	Parked      State = 'P'
	Idle        State = 'I'
)

func (s State) String() string {
	switch s {
	case Dead:
		return "dead"
	case DiskSleep:
		return "disk sleep"
	case Running:
		return "running"
	case Sleeping:
		return "sleeping"
	case Stopped:
		return "stopped"
	case TracingStop:
		return "tracing stop"
	case Zombie:
		return "zombie"
	case Parked:
		return "parked"
	case Idle:
		return "idle" // kernel thread
	default:
		return fmt.Sprintf("unknown (%c)", s)
	}
}

type Stat_t struct {
	// Name is the command run by the process.
	Name string

	// State is the state of the process.
	State State

	// StartTime is the number of clock ticks after system boot (since
	// Linux 2.6).
	StartTime uint64
}

// Stat返回具体进程的一个Stat_t实例
func Stat(pid int) (stat Stat_t, err error) {
	bytes, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return stat, err
	}
	return parseStat(string(bytes))
}

func parseStat(data string) (stat Stat_t, err error) {
	const minAfterName = 20*2 + 1 // the min field is '0 '.

	first := strings.IndexByte(data, '(')
	if first < 0 || first+minAfterName >= len(data) {
		return stat, fmt.Errorf("invalid stat data (no comm or too short): %q", data)
	}

	last := strings.LastIndexByte(data, ')')
	if last <= first || last+minAfterName >= len(data) {
		return stat, fmt.Errorf("invalid stat data (no comm or too short): %q", data)
	}

	stat.Name = data[first+1 : last]

	// 2. Remove fields 1 and 2 and a space after. State is right after.
	data = data[last+2:]
	stat.State = State(data[0])

	// 3. StartTime is field 22, data is at field 3 now, so we need to skip 19 spaces.
	skipSpaces := 22 - 3
	for first = 0; skipSpaces > 0 && first < len(data); first++ {
		if data[first] == ' ' {
			skipSpaces--
		}
	}
	// Now first points to StartTime; look for space right after.
	i := strings.IndexByte(data[first:], ' ')
	if i < 0 {
		return stat, fmt.Errorf("invalid stat data (too short): %q", data)
	}
	stat.StartTime, err = strconv.ParseUint(data[first:first+i], 10, 64)
	if err != nil {
		return stat, fmt.Errorf("invalid stat data (bad start time): %w", err)
	}

	return stat, nil
}
