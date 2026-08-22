package compat

import (
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/pe"
)

func TestKnownHardwareToolIsFlaggedAsBlocking(t *testing.T) {
	// The version suffix in the real download must not defeat the match.
	for _, name := range []string{"rufus.exe", "rufus-3.8.exe", "rufus-4.5p.exe", "Rufus-3.8.EXE"} {
		got := Check(&pe.Info{Name: name, Kind: pe.KindPE, Arch: pe.ArchX86})
		if Worst(got) != Blocking {
			t.Errorf("%s: severity = %v, want Blocking", name, Worst(got))
		}
		if len(got) == 0 || !strings.Contains(got[0].Detail, "raw USB") {
			t.Errorf("%s: expected an explanation of the device-access problem, got %+v", name, got)
		}
		if got[0].Workaround == "" {
			t.Errorf("%s: a blocking caveat must suggest an alternative", name)
		}
	}
}

func TestUnrelatedProgramsAreNotFlagged(t *testing.T) {
	// The match is anchored: a name that merely contains a known program's
	// name must not trigger it.
	for _, name := range []string{"notepad.exe", "myrufus.exe", "rufusclone-tool.exe", "game.exe"} {
		if got := Check(&pe.Info{Name: name, Kind: pe.KindPE}); len(got) != 0 {
			t.Errorf("%s wrongly flagged: %+v", name, got)
		}
	}
}

func TestElevationIsReportedButNotAsBlocking(t *testing.T) {
	got := Check(&pe.Info{
		Name: "tool.exe", Kind: pe.KindPE,
		RequiresAdmin: true, ExecutionLevel: "requireAdministrator",
	})
	if len(got) != 1 {
		t.Fatalf("expected one caveat, got %+v", got)
	}
	if got[0].Severity != Degraded {
		t.Errorf("severity = %v, want Degraded", got[0].Severity)
	}
	if !strings.Contains(got[0].Detail, "ordinary Linux permissions") {
		t.Errorf("the caveat should explain what Wine actually grants: %q", got[0].Detail)
	}
}

func TestInstallersRequestingElevationAreOnlyANote(t *testing.T) {
	// Nearly every installer asks for elevation and nearly all of them work,
	// because they only write inside the prefix.
	got := Check(&pe.Info{
		Name: "setup.exe", Kind: pe.KindPE, IsInstaller: true,
		RequiresAdmin: true, ExecutionLevel: "requireAdministrator",
	})
	if Worst(got) != Note {
		t.Errorf("severity = %v, want Note for an installer", Worst(got))
	}
}

func TestKnownProgramSuppressesTheGenericElevationNotice(t *testing.T) {
	// Rufus asks for elevation too; repeating that under the real explanation
	// would just be noise.
	got := Check(&pe.Info{
		Name: "rufus-3.8.exe", Kind: pe.KindPE,
		RequiresAdmin: true, ExecutionLevel: "requireAdministrator",
	})
	if len(got) != 1 {
		t.Errorf("expected only the specific caveat, got %d: %+v", len(got), got)
	}
}

func TestPackedBinaryIsNoted(t *testing.T) {
	got := Check(&pe.Info{Name: "app.exe", Kind: pe.KindPE, Packed: true, Packer: "UPX"})
	if len(got) != 1 || got[0].Severity != Note {
		t.Fatalf("expected a single note, got %+v", got)
	}
	if !strings.Contains(got[0].Title, "UPX") {
		t.Errorf("the packer should be named: %q", got[0].Title)
	}
}

func TestCaveatsCarryAMachineReadableLevel(t *testing.T) {
	for _, c := range Check(&pe.Info{Name: "rufus.exe", Kind: pe.KindPE, Packed: true, Packer: "UPX"}) {
		if c.Level == "" {
			t.Errorf("caveat %q has no severity string", c.Title)
		}
	}
}

func TestNilInfoIsSafe(t *testing.T) {
	if got := Check(nil); got != nil {
		t.Errorf("Check(nil) = %+v, want nil", got)
	}
	if Worst(nil) != Note {
		t.Error("Worst(nil) should be the lowest severity")
	}
}

func TestNamePatternsAreAnchoredButAllowVersions(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"rufus", true},
		{"rufus-3.8", true},
		{"rufus_4.5p", true},
		{"rufusclone-tool", false},
		{"myrufus", false},
		{"cpuz", true},
		{"cpu-z", true},
		{"cpuz_x64", true},
		{"cpuzilla", false},
		{"daemon-tools-lite", true},
		{"daemontools", true},
		{"tools", false},
	}
	for _, c := range cases {
		var matched bool
		for i := range knownPrograms {
			if knownPrograms[i].match.MatchString(c.name) {
				matched = true
				break
			}
		}
		if matched != c.want {
			t.Errorf("%q matched=%v, want %v", c.name, matched, c.want)
		}
	}
}
