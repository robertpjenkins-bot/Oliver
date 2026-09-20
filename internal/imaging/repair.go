package imaging

import (
	"encoding/binary"
	"fmt"
	"os"
)

// A drive does not usually break. What breaks is the handful of sectors that
// describe it — the partition table, the signatures a filesystem leaves at
// known offsets — and when those disagree with the drive underneath them,
// every tool that looks at the drive gets a different wrong answer. The
// commonest case by far: a 64 GB stick that has had an installation image
// written to it and now shows up as 2 GB, because the image's partition table
// is the only description of the drive left and it describes a disc.
//
// This is what Diagnose looks for and what Repair puts right. Neither recovers
// files: repairing a drive means making the drive usable again, and the way it
// does that is by overwriting what is there.

// FaultKind names something wrong with how a drive describes itself.
type FaultKind string

const (
	// Shrunken means the partition table describes far less space than the
	// drive has, which is why the drive looks smaller than it is.
	Shrunken FaultKind = "shrunken"
	// LeftoverImage means a disc image's descriptor is still present, so some
	// systems will treat the drive as a read-only disc.
	LeftoverImage FaultKind = "leftover-image"
	// Overrun means a partition claims space past the end of the drive.
	Overrun FaultKind = "overrun"
	// MissingBackupTable means a GPT has lost the copy kept at the far end,
	// which makes most partitioning tools refuse to touch the drive.
	MissingBackupTable FaultKind = "missing-backup-table"
	// NoTable means there is no partition table at all, though something is
	// written to the drive.
	NoTable FaultKind = "no-table"
	// Unreadable means the drive would not answer.
	Unreadable FaultKind = "unreadable"
)

// Fault is one thing found wrong, in terms of what it means rather than which
// byte it was found at.
type Fault struct {
	Kind    FaultKind `json:"kind"`
	Summary string    `json:"summary"`
	Detail  string    `json:"detail"`
}

// Diagnosis is what a drive says about itself, and where that is wrong.
type Diagnosis struct {
	Path string `json:"path"`
	// Capacity is what the drive really holds.
	Capacity int64 `json:"capacity_bytes"`
	// Described is how much of it the partition table accounts for.
	Described int64   `json:"described_bytes"`
	Volume    Volume  `json:"volume"`
	Faults    []Fault `json:"faults,omitempty"`
}

// Healthy reports whether nothing was found wrong.
func (d *Diagnosis) Healthy() bool { return len(d.Faults) == 0 }

// Diagnose reads a drive's first and last sectors and reports what is wrong
// with them. It writes nothing, and a drive it cannot read is itself a finding
// rather than an error.
func Diagnose(path string) (*Diagnosis, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, openError(path, err)
	}
	defer f.Close()

	capacity, err := Capacity(f)
	if err != nil {
		return nil, err
	}
	d := &Diagnosis{Path: path, Capacity: capacity, Volume: probeReader(f)}

	head := make([]byte, 512)
	if n, _ := f.ReadAt(head, 0); n < 512 {
		d.Faults = append(d.Faults, Fault{
			Kind:    Unreadable,
			Summary: "the drive would not answer",
			Detail: "Its first sector could not be read. That is either a failing drive or one " +
				"this user has no permission to read.",
		})
		return d, nil
	}

	hasMBR := head[510] == 0x55 && head[511] == 0xAA
	gpt := make([]byte, 8)
	f.ReadAt(gpt, 512)
	hasGPT := string(gpt) == "EFI PART"

	if hasGPT {
		d.checkGPT(f, capacity)
	}
	if hasMBR {
		d.checkMBR(head, capacity, hasGPT)
	}

	if !hasMBR && !hasGPT {
		if !d.Volume.Empty() {
			d.Faults = append(d.Faults, Fault{
				Kind:    NoTable,
				Summary: "no partition table, though the drive is not blank",
				Detail: fmt.Sprintf("The drive holds %s written straight to it, with nothing describing "+
					"where it begins or ends. Some systems will mount it; Windows will offer to format it.",
					d.Volume.String()),
			})
		}
		return d, nil
	}

	// An image written whole to a stick leaves its disc descriptor behind, and
	// that is what makes a drive turn up read-only or at the wrong size.
	sig := make([]byte, 5)
	if n, _ := f.ReadAt(sig, 32769); n == 5 && string(sig) == "CD001" {
		d.Faults = append(d.Faults, Fault{
			Kind:    LeftoverImage,
			Summary: "a disc image is still written across the drive",
			Detail: "The drive carries an ISO 9660 descriptor, so it was written from an " +
				"installation image. Until that is cleared, systems will keep reading the drive " +
				"as a disc — read-only, and the size of the image rather than the size of the drive.",
		})
	}
	return d, nil
}

// checkMBR reads the four partition entries and compares them with the drive.
func (d *Diagnosis) checkMBR(head []byte, capacity int64, hasGPT bool) {
	var end int64
	for i := 0; i < 4; i++ {
		e := head[446+i*16 : 446+(i+1)*16]
		if e[4] == 0 {
			continue
		}
		if e[4] == 0xEE {
			continue // the protective entry in front of a GPT
		}
		start := int64(binary.LittleEndian.Uint32(e[8:]))
		length := int64(binary.LittleEndian.Uint32(e[12:]))
		if last := (start + length) * sectorSize; last > end {
			end = last
		}
	}
	d.Described = end

	if end > capacity {
		d.Faults = append(d.Faults, Fault{
			Kind:    Overrun,
			Summary: "the partition table describes space the drive does not have",
			Detail: fmt.Sprintf("It claims %s on a %s drive. Reading the end of it will fail, and "+
				"writing there loses whatever was meant to be written.",
				HumanBytes(end), HumanBytes(capacity)),
		})
		return
	}
	if hasGPT || end == 0 {
		return
	}
	// A tenth of the drive is the threshold at which "the table is smaller
	// than the drive" stops being ordinary spare room and starts being the
	// reason somebody thinks their stick has shrunk.
	if end < capacity/10 {
		d.Faults = append(d.Faults, Fault{
			Kind: Shrunken,
			Summary: fmt.Sprintf("the drive holds %s but only %s of it is described",
				HumanBytes(capacity), HumanBytes(end)),
			Detail: fmt.Sprintf("Everything that reads this drive will show it as %s, because that "+
				"is all its partition table accounts for. The remaining %s is still there and still "+
				"works; nothing on the drive knows about it.",
				HumanBytes(end), HumanBytes(capacity-end)),
		})
	}
}

// checkGPT verifies that the copy of the table kept at the end of the drive is
// where the one at the front says it is.
func (d *Diagnosis) checkGPT(f *os.File, capacity int64) {
	header := make([]byte, 92)
	if n, _ := f.ReadAt(header, 512); n < 92 {
		return
	}
	backupLBA := int64(binary.LittleEndian.Uint64(header[32:]))
	lastLBA := capacity/sectorSize - 1

	sig := make([]byte, 8)
	if n, _ := f.ReadAt(sig, lastLBA*sectorSize); n == 8 && string(sig) == "EFI PART" {
		return // the backup is where it should be
	}
	d.Faults = append(d.Faults, Fault{
		Kind:    MissingBackupTable,
		Summary: "the backup partition table at the end of the drive is missing",
		Detail: fmt.Sprintf("A GPT keeps a second copy in the last sector, and this drive has none "+
			"there. The table at the front expects it at sector %d of %d — the usual cause is an "+
			"image being written to a larger drive. Partitioning tools will refuse to work until "+
			"it is put right.", backupLBA, lastLBA),
	})
}

// RepairResult reports what a repair restored.
type RepairResult struct {
	Before  *Diagnosis    `json:"before"`
	Format  *FormatResult `json:"format,omitempty"`
	Cleared int64         `json:"cleared_bytes"`
}

// Recovered returns how much space the drive describes now that it did not
// before.
func (r *RepairResult) Recovered() int64 {
	if r.Format == nil {
		return 0
	}
	now := r.Format.PartitionSectors * sectorSize
	if now <= r.Before.Described {
		return 0
	}
	return now - r.Before.Described
}
