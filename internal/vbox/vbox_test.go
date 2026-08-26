package vbox

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testManager returns a manager whose VirtualBox is a recording of what was
// asked of it, so the whole build can be checked without VirtualBox existing.
func testManager(t *testing.T, respond func(argv []string) (string, error)) (*Manager, *[]string) {
	t.Helper()
	var log []string
	m := &Manager{Bin: "VBoxManage", Major: 7, Minor: 0, Root: t.TempDir()}
	m.Run = func(argv []string) (string, error) {
		log = append(log, strings.Join(argv, " "))
		if respond == nil {
			return "", nil
		}
		return respond(argv)
	}
	return m, &log
}

func specFor(t *testing.T, release string) Spec {
	t.Helper()
	dir := t.TempDir()
	return Spec{
		Name: "Windows", RAMMB: 8192, CPUs: 4, VRAMMB: 128, DiskGB: 64,
		ISO: filepath.Join(dir, "win.iso"), Dir: dir, Release: release,
		EFI: release == "win11", TPM: release == "win11",
	}
}

func planText(m *Manager, spec Spec) string {
	var b strings.Builder
	for _, s := range m.Plan(spec) {
		b.WriteString(s.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func TestPlanGivesWindows11TheFirmwareItRefusesToInstallWithout(t *testing.T) {
	m, _ := testManager(t, nil)
	text := planText(m, specFor(t, "win11"))

	for _, want := range []string{
		"--ostype Windows11_64",
		"--firmware efi",
		"--tpm-type 2.0",
		"modifynvram Windows enrollmssignatures",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("plan for Windows 11 is missing %q:\n%s", want, text)
		}
	}
}

func TestPlanLeavesOlderWindowsOnBIOS(t *testing.T) {
	m, _ := testManager(t, nil)
	text := planText(m, specFor(t, "win10"))

	for _, unwanted := range []string{"--firmware efi", "--tpm-type"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("plan for Windows 10 should not contain %q:\n%s", unwanted, text)
		}
	}
	if !strings.Contains(text, "--ostype Windows10_64") {
		t.Errorf("plan should tell VirtualBox which Windows it is:\n%s", text)
	}
}

// The installer must run while the disk is empty and never again afterwards,
// which is what booting from the disk first achieves.
func TestPlanBootsFromTheDiskBeforeTheInstaller(t *testing.T) {
	m, _ := testManager(t, nil)
	text := planText(m, specFor(t, "win10"))
	if !strings.Contains(text, "--boot1 disk --boot2 dvd") {
		t.Errorf("the disk must come before the installer in the boot order:\n%s", text)
	}
}

func TestPlanSizesTheDiskInMegabytesAndAttachesTheInstaller(t *testing.T) {
	spec := specFor(t, "win10")
	m, _ := testManager(t, nil)
	text := planText(m, spec)

	if !strings.Contains(text, "--size 65536") {
		t.Errorf("a 64 GB disk should be requested as 65536 MB:\n%s", text)
	}
	if !strings.Contains(text, "--type dvddrive --medium "+spec.ISO) {
		t.Errorf("the installer image should be in the DVD drive:\n%s", text)
	}
	if !strings.Contains(text, "--type hdd --medium "+spec.DiskPath()) {
		t.Errorf("the disk should be attached:\n%s", text)
	}
}

func TestPlanUsesTheAudioFlagThisVirtualBoxUnderstands(t *testing.T) {
	spec := specFor(t, "win10")

	modern, _ := testManager(t, nil)
	if !strings.Contains(planText(modern, spec), "--audio-driver default") {
		t.Error("VirtualBox 7 takes --audio-driver")
	}

	old, _ := testManager(t, nil)
	old.Major, old.Minor = 6, 1
	if !strings.Contains(planText(old, spec), "--audio pulse") {
		t.Error("VirtualBox 6 only understands --audio")
	}
}

// A half-built machine holds a name that cannot be reused and leaves a disk
// image behind, so a failure must undo what it started.
func TestCreateUndoesAHalfBuiltMachine(t *testing.T) {
	spec := specFor(t, "win10")
	m, log := testManager(t, func(argv []string) (string, error) {
		if argv[0] == "createmedium" {
			return "", errors.New("VBOX_E_FILE_ERROR")
		}
		return "", nil
	})

	if _, err := m.Create(spec, nil); err == nil {
		t.Fatal("a failed disk creation should fail the build")
	}
	if !strings.Contains(strings.Join(*log, "\n"), "unregistervm Windows --delete") {
		t.Errorf("the registered machine should have been removed again:\n%s", strings.Join(*log, "\n"))
	}
	if _, ok, _ := m.Load(spec.Name); ok {
		t.Error("a machine that was not built should not be recorded")
	}
}

func TestCreateTreatsComfortSettingsAsOptional(t *testing.T) {
	spec := specFor(t, "win10")
	m, _ := testManager(t, func(argv []string) (string, error) {
		for _, a := range argv {
			if strings.HasPrefix(a, "--audio") {
				return "", errors.New("unknown option: --audio-driver")
			}
		}
		return "", nil
	})

	warnings, err := m.Create(spec, nil)
	if err != nil {
		t.Fatalf("audio should not fail the build: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "sound") {
		t.Errorf("the skipped setting should be reported: %v", warnings)
	}
	if _, ok, _ := m.Load(spec.Name); !ok {
		t.Error("the machine should have been recorded")
	}
}

// The point of the record is that the second run does not ask the first run's
// questions again.
func TestARecordedMachineRemembersHowItWasSetUp(t *testing.T) {
	spec := specFor(t, "win11")
	m, _ := testManager(t, nil)

	if _, ok, err := m.Load(spec.Name); err != nil || ok {
		t.Fatalf("nothing should be known before setup: %v %v", ok, err)
	}
	if _, err := m.Create(spec, nil); err != nil {
		t.Fatal(err)
	}

	got, ok, err := m.Load(spec.Name)
	if err != nil || !ok {
		t.Fatalf("the machine should be recorded: %v %v", ok, err)
	}
	if got.Spec.RAMMB != spec.RAMMB || got.Spec.DiskGB != spec.DiskGB || got.Spec.Dir != spec.Dir {
		t.Errorf("the record lost the choices: %+v", got.Spec)
	}
	if got.Starts != 0 {
		t.Errorf("a new machine has not been started: %d", got.Starts)
	}

	if err := m.MarkStarted(spec.Name); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := m.Load(spec.Name); got.Starts != 1 || got.LastStarted.IsZero() {
		t.Errorf("a start should be recorded: %+v", got)
	}

	list, err := m.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list should hold the one machine: %v %v", len(list), err)
	}
}

func TestRegisteredReadsVirtualBoxOwnList(t *testing.T) {
	m, _ := testManager(t, func(argv []string) (string, error) {
		return "\"Other\" {aaa}\n\"Windows\" {bbb}\n", nil
	})
	if ok, err := m.Registered("Windows"); err != nil || !ok {
		t.Errorf("Windows should be registered: %v %v", ok, err)
	}
	if ok, _ := m.Registered("Missing"); ok {
		t.Error("a machine VirtualBox does not list is not registered")
	}
}

func TestStateReadsTheMachineReadableReport(t *testing.T) {
	m, _ := testManager(t, func(argv []string) (string, error) {
		return "name=\"Windows\"\nVMState=\"running\"\nVMStateChangeTime=\"...\"\n", nil
	})
	state, err := m.State("Windows")
	if err != nil || state != "running" {
		t.Errorf("state = %q, %v; want running", state, err)
	}
}

func TestFindReportsAMissingVirtualBoxAsSuch(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	// The fixed search paths are absolute, so this only holds where VirtualBox
	// is genuinely absent; skip rather than fail on a machine that has it.
	if _, err := Find(); err != nil && !errors.Is(err, ErrNotInstalled) {
		t.Errorf("a missing VirtualBox should be reported as ErrNotInstalled, got %v", err)
	} else if err == nil {
		t.Skip("VirtualBox is installed on this machine")
	}
}

func TestParseVersionIgnoresTheBuildSuffix(t *testing.T) {
	for _, tc := range []struct {
		in           string
		major, minor int
	}{
		{"7.0.14r161095\n", 7, 0},
		{"7.1.4_Debianr165100", 7, 1},
		{"6.1.50", 6, 1},
		{"not a version", 0, 0},
	} {
		major, minor := parseVersion(tc.in)
		if major != tc.major || minor != tc.minor {
			t.Errorf("parseVersion(%q) = %d.%d; want %d.%d", tc.in, major, minor, tc.major, tc.minor)
		}
	}
}

func TestAnUnreadableVersionIsTreatedAsNewEnough(t *testing.T) {
	m := &Manager{}
	if !m.AtLeast(7, 0) {
		t.Error("refusing to work over an unparsable version string helps nobody")
	}
}

func TestDefaultRootFollowsXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("OLIVER_VM_ROOT", "")
	if got, want := DefaultRoot(), filepath.Join(dir, "oliver", "vms"); got != want {
		t.Errorf("DefaultRoot() = %q; want %q", got, want)
	}
	t.Setenv("OLIVER_VM_ROOT", filepath.Join(dir, "elsewhere"))
	if got, want := DefaultRoot(), filepath.Join(dir, "elsewhere"); got != want {
		t.Errorf("DefaultRoot() = %q; want %q", got, want)
	}
}

func TestRemoveDeletesTheFilesUnlessAskedNotTo(t *testing.T) {
	m, log := testManager(t, nil)
	if err := m.Save(Record{Spec: Spec{Name: "Windows"}}); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("Windows", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains((*log)[0], "--delete") {
		t.Errorf("removing a machine should delete its disk: %q", (*log)[0])
	}
	if _, ok, _ := m.Load("Windows"); ok {
		t.Error("the record should be gone too")
	}

	m2, log2 := testManager(t, nil)
	_ = m2.Save(Record{Spec: Spec{Name: "Windows"}})
	if err := m2.Remove("Windows", true); err != nil {
		t.Fatal(err)
	}
	if strings.Contains((*log2)[0], "--delete") {
		t.Errorf("-keep-files should leave the disk alone: %q", (*log2)[0])
	}
}

func TestStepQuotesPathsWithSpaces(t *testing.T) {
	spec := specFor(t, "win10")
	spec.Dir = filepath.Join(os.TempDir(), "VirtualBox VMs")
	m, _ := testManager(t, nil)
	text := planText(m, spec)
	if !strings.Contains(text, "'"+spec.Dir+"'") {
		t.Errorf("a folder with a space must survive being pasted into a shell:\n%s", text)
	}
}
