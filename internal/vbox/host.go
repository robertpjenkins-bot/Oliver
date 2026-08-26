package vbox

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// Host is what this machine has to hand a virtual machine.
//
// The numbers matter because a virtual machine takes its memory away from the
// system running it: RAM given to Windows is RAM Linux no longer has. Guessing
// this wrong is the difference between a machine that boots and a desktop that
// starts swapping the moment Windows starts.
type Host struct {
	RAMMB int `json:"host_ram_mb"`
	CPUs  int `json:"host_cpus"`
}

// ProbeHost reads this machine's memory and processor count.
func ProbeHost() Host {
	return Host{RAMMB: totalRAMMB("/proc/meminfo"), CPUs: runtime.NumCPU()}
}

// totalRAMMB reads MemTotal, which /proc/meminfo reports in kibibytes.
func totalRAMMB(meminfo string) int {
	f, err := os.Open(meminfo)
	if err != nil {
		return 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return int(kb / 1024)
	}
	return 0
}

// FreeGB reports the space available where a virtual disk would be created.
//
// The directory usually does not exist yet — it is the one the user is in the
// middle of choosing — so this walks up to the nearest parent that does, which
// is on the same filesystem and therefore has the same free space.
//
// It is a variable so tests can describe a disk this machine does not have.
var FreeGB = func(dir string) int {
	path, err := filepath.Abs(dir)
	if err != nil {
		path = dir
	}
	for {
		var st syscall.Statfs_t
		if err := syscall.Statfs(path, &st); err == nil {
			return int(uint64(st.Bavail) * uint64(st.Bsize) / (1 << 30))
		}
		parent := filepath.Dir(path)
		if parent == path {
			return 0
		}
		path = parent
	}
}

// Recommend proposes a specification this host can actually honour.
//
// The rules are deliberately conservative. Half the memory is the most a guest
// should take before the host starts to suffer, 2 GB is the least Linux needs
// left over, and beyond about 8 GB a desktop Windows guest gains nothing — so
// the suggestion stops there rather than reserving memory nothing will use.
func (h Host) Recommend(rel Release, dir string) Spec {
	spec := Spec{
		Name:    DefaultName,
		RAMMB:   rel.MinRAMMB,
		CPUs:    2,
		VRAMMB:  128,
		DiskGB:  rel.MinDiskGB,
		Dir:     dir,
		Release: rel.ID,
		EFI:     rel.NeedsEFI,
		TPM:     rel.NeedsTPM,
	}

	if h.RAMMB > 0 {
		want := roundDownGB(h.RAMMB / 2)
		if want > 8192 {
			want = 8192
		}
		if headroom := roundDownGB(h.RAMMB - 2048); want > headroom {
			want = headroom
		}
		if want > spec.RAMMB {
			spec.RAMMB = want
		}
	}

	if h.CPUs > 0 {
		cpus := h.CPUs / 2
		if cpus > 4 {
			cpus = 4
		}
		if cpus > spec.CPUs {
			spec.CPUs = cpus
		}
	}

	// The virtual disk grows as Windows fills it, so its size is a ceiling
	// rather than a reservation. Suggesting more than the minimum costs
	// nothing today and saves a painful resize later — but not more than the
	// filesystem could ever deliver.
	if want := 2 * rel.MinDiskGB; want > spec.DiskGB {
		spec.DiskGB = want
	}
	if free := FreeGB(dir); free > 0 && spec.DiskGB > free {
		spec.DiskGB = rel.MinDiskGB
	}
	return spec
}

// roundDownGB rounds a megabyte figure down to a whole gibibyte, since memory
// sizes that are not round numbers only make the summary harder to read.
func roundDownGB(mb int) int {
	if mb < 1024 {
		return 0
	}
	return (mb / 1024) * 1024
}

// FormatMB renders a memory size the way someone would say it out loud.
func FormatMB(mb int) string {
	if mb >= 1024 && mb%1024 == 0 {
		return fmt.Sprintf("%d GB", mb/1024)
	}
	if mb >= 1024 {
		return fmt.Sprintf("%.1f GB", float64(mb)/1024)
	}
	return fmt.Sprintf("%d MB", mb)
}
