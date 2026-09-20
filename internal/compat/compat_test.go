package compat

import (
	"errors"
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
		if len(got) == 0 || !strings.Contains(got[0].Detail, "setupapi") {
			t.Errorf("%s: expected an explanation of the device-access problem, got %+v", name, got)
		}
		if got[0].Workaround == "" {
			t.Errorf("%s: a blocking caveat must suggest an alternative", name)
		}
		// These programs do launch, so the caveat has to describe what the
		// user will actually observe rather than implying nothing happens.
		if got[0].Symptom == "" {
			t.Errorf("%s: a blocking caveat must describe the symptom", name)
		}
	}
}

func TestRenamedProgramIsStillRecognisedByItsVersionResource(t *testing.T) {
	// The file on disk can be called anything: a browser's "(1)" suffix, a
	// download manager's hash prefix, or a deliberate rename. What the
	// executable says about itself is what matters.
	for _, name := range []string{
		"totally-innocent.exe",
		"117e7b59-rufus3.10.exe",
		"rufus-3.10 (1).exe",
		"downloaded_file.exe",
	} {
		got := Check(&pe.Info{
			Name: name, Kind: pe.KindPE, Arch: pe.ArchX86,
			VersionInfo: pe.VersionInfo{ProductName: "Rufus", OriginalFilename: "rufus-3.10.exe"},
		})
		if Worst(got) != Blocking {
			t.Errorf("%s: severity = %v, want Blocking", name, Worst(got))
		}
	}
}

func TestVersionSuffixesInFileNamesStillMatch(t *testing.T) {
	// With no version resource the file name is all there is to go on.
	for _, name := range []string{
		"rufus.exe", "rufus-3.10.exe", "rufus3.10.exe",
		"rufus-3.10 (1).exe", "rufus_4.5p.exe", "rufus-4.9_x64.exe",
	} {
		if Worst(Check(&pe.Info{Name: name, Kind: pe.KindPE})) != Blocking {
			t.Errorf("%s was not recognised", name)
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

	// A different program whose version resource says so must not be caught
	// just because someone named the file rufus.exe.
	got := Check(&pe.Info{
		Name: "rufusclone.exe", Kind: pe.KindPE,
		VersionInfo: pe.VersionInfo{ProductName: "Notepad++"},
	})
	if len(got) != 0 {
		t.Errorf("unrelated program flagged: %+v", got)
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

func TestBlockingWordingDoesNotClaimTheProgramFailsToStart(t *testing.T) {
	// Most of these programs open a window and respond to some controls. A
	// label of "will not work" gets contradicted by the first person who runs
	// one, and then the real warning stops being believed.
	if got := Blocking.String(); strings.Contains(got, "will not work") {
		t.Errorf("Blocking.String() = %q, which overstates what happens", got)
	}

	c := Check(&pe.Info{Name: "rufus-3.10.exe", Kind: pe.KindPE})
	if len(c) == 0 {
		t.Fatal("expected a caveat")
	}
	if !strings.Contains(c[0].Symptom, "opens") {
		t.Errorf("the symptom should say the program opens: %q", c[0].Symptom)
	}
	// The commonly suggested dosdevices workaround does not help here, and
	// saying so saves the user trying it.
	if !strings.Contains(c[0].Workaround, "dosdevices") {
		t.Errorf("the caveat should head off the dosdevices suggestion: %q", c[0].Workaround)
	}
}

// withInstalled makes lookPath report exactly the given commands as present.
func withInstalled(t *testing.T, present ...string) {
	t.Helper()
	set := map[string]bool{}
	for _, p := range present {
		set[p] = true
	}
	original := lookPath
	lookPath = func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = original })
}

func TestInstalledAlternativesAreMarkedAndListedFirst(t *testing.T) {
	withInstalled(t, "gnome-disks", "dd")

	got := Check(&pe.Info{Name: "rufus-3.10.exe", Kind: pe.KindPE})
	if len(got) == 0 {
		t.Fatal("expected a caveat")
	}
	alts := got[0].Alternatives
	if len(alts) == 0 {
		t.Fatal("expected native alternatives to be offered")
	}

	// Everything installed must sort ahead of everything that is not, so the
	// advice ends on something the user can run now.
	var seenMissing bool
	installed := map[string]bool{}
	for _, a := range alts {
		if a.Installed {
			if seenMissing {
				t.Errorf("%s is installed but listed after a missing tool", a.Name)
			}
			installed[a.Command] = true
		} else {
			seenMissing = true
		}
	}
	if !installed["gnome-disks"] || !installed["dd"] {
		t.Errorf("expected gnome-disks and dd to be marked installed, got %+v", alts)
	}
	for _, a := range alts {
		if a.Command == "isoimagewriter" && a.Installed {
			t.Error("isoimagewriter is not installed and must not be marked as such")
		}
	}
}

func TestAlternativesWithoutACommandAreNeverMarkedInstalled(t *testing.T) {
	// Ventoy is run from an extracted download, so there is nothing on PATH to
	// find. It must be offered without ever claiming to be present.
	withInstalled(t) // nothing installed
	got := Check(&pe.Info{Name: "rufus.exe", Kind: pe.KindPE})
	var found bool
	for _, a := range got[0].Alternatives {
		if a.Name == "Ventoy" {
			found = true
			if a.Installed {
				t.Error("Ventoy has no command to detect and must not be reported as installed")
			}
		}
	}
	if !found {
		t.Error("Ventoy should be offered for a USB writer")
	}
}

func TestAlternativesAreOfferedForEachKnownProgram(t *testing.T) {
	withInstalled(t)
	for _, name := range []string{"rufus.exe", "cpuz.exe", "hwmonitor.exe", "crystaldiskinfo.exe", "daemontools.exe", "aida64.exe"} {
		got := Check(&pe.Info{Name: name, Kind: pe.KindPE})
		if len(got) == 0 {
			t.Errorf("%s: no caveat", name)
			continue
		}
		if len(got[0].Alternatives) == 0 {
			t.Errorf("%s: no alternatives offered", name)
		}
	}
}

func TestResolveDoesNotMutateTheSharedTable(t *testing.T) {
	// The caveat tables are package-level, so resolving must copy rather than
	// write detection results back into them.
	withInstalled(t, "dd")
	_ = Check(&pe.Info{Name: "rufus.exe", Kind: pe.KindPE})

	withInstalled(t) // now nothing is installed
	for _, a := range Check(&pe.Info{Name: "rufus.exe", Kind: pe.KindPE})[0].Alternatives {
		if a.Command == "oliver" {
			// Oliver is running, so it is always available; it is not a
			// detection result that could have leaked.
			continue
		}
		if a.Installed {
			t.Errorf("%s reported installed from a previous run's state", a.Name)
		}
	}
}

func TestOliverOffersItselfForTheProgramsItReplaces(t *testing.T) {
	// Nothing at all is on this machine's PATH.
	withInstalled(t)

	for _, name := range []string{"rufus.exe", "win32diskimager.exe", "balenaetcher.exe"} {
		caveats := Check(&pe.Info{Name: name, Kind: pe.KindPE})
		if len(caveats) == 0 {
			t.Errorf("%s: no caveat", name)
			continue
		}
		alts := caveats[0].Alternatives
		if len(alts) == 0 || alts[0].Command != "oliver" {
			t.Errorf("%s: the advice does not lead with the one tool that is certainly present: %+v", name, alts)
			continue
		}
		if !alts[0].Installed {
			t.Errorf("%s: oliver reported as not installed while it is running", name)
		}
	}
}
