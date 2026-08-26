package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/ui"
	"github.com/robertpjenkins-bot/oliver/internal/vbox"
)

// virtualbox installs a stand-in for VBoxManage in the synthetic system and
// returns the file it records its arguments in, so a whole first run can be
// checked on a machine with no VirtualBox on it.
func (s *system) virtualbox(t *testing.T, version ...string) string {
	t.Helper()
	reported := "7.0.14r161095"
	if len(version) > 0 {
		reported = version[0]
	}
	log := s.path("vbox.log")
	script := "#!/bin/sh\n" +
		"echo \"$@\" >> " + log + "\n" +
		"case \"$1\" in\n" +
		"  --version) echo '" + reported + "' ;;\n" +
		// Only shell builtins and redirections: the synthetic system's PATH
		// holds nothing else.
		"  createvm) : > " + log + ".registered ;;\n" +
		"  list) [ -f " + log + ".registered ] && echo '\"Windows\" {0}' ;;\n" +
		"  showvminfo) echo 'VMState=\"poweroff\"' ;;\n" +
		"esac\n" +
		"exit 0\n"
	p := s.write(script, "usr", "bin", "VBoxManage")
	if err := os.Chmod(p, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLIVER_VM_ROOT", s.path("home", "user", ".local", "share", "oliver", "vms"))

	loaded := vbox.KernelModuleLoaded
	vbox.KernelModuleLoaded = func() bool { return true }
	// Neither the driver nor the free space of the machine running the tests
	// should decide whether they pass.
	free := vbox.FreeGB
	vbox.FreeGB = func(string) int { return 500 }
	t.Cleanup(func() {
		vbox.KernelModuleLoaded = loaded
		vbox.FreeGB = free
	})
	return log
}

// vboxCalls returns what the stand-in VirtualBox was asked to do.
func vboxCalls(t *testing.T, log string) string {
	t.Helper()
	data, err := os.ReadFile(log)
	if err != nil {
		return ""
	}
	return string(data)
}

// answer drives a command that asks questions, returning everything the user
// would have seen — prompts included, since those go to stderr.
func (s *system) answer(answers string, args ...string) (string, error) {
	s.t.Helper()
	s.out.Reset()
	both := &bytes.Buffer{}
	orig := newUI
	newUI = func() *ui.UI {
		return &ui.UI{Out: io.MultiWriter(s.out, both), Err: both, In: strings.NewReader(answers), Interactive: true}
	}
	defer func() { newUI = orig }()
	err := run(args)
	return both.String(), err
}

// installer writes something with the volume descriptors of a real Windows
// image, since Oliver reads the image rather than trusting its name.
func (s *system) installer(name string) string {
	s.t.Helper()
	image := make([]byte, 16*2048)

	boot := make([]byte, 2048)
	boot[0], boot[6] = 0, 1
	copy(boot[1:], "CD001")
	copy(boot[7:], "EL TORITO SPECIFICATION")

	primary := make([]byte, 2048)
	primary[0], primary[6] = 1, 1
	copy(primary[1:], "CD001")
	copy(primary[40:72], []byte("CCCOMA_X64FRE_EN-US_DV9"+strings.Repeat(" ", 32)))
	copy(primary[318:446], []byte("MICROSOFT CORPORATION"+strings.Repeat(" ", 128)))

	end := make([]byte, 2048)
	end[0], end[6] = 255, 1
	copy(end[1:], "CD001")

	image = append(image, boot...)
	image = append(image, primary...)
	image = append(image, end...)
	image = append(image, make([]byte, 4<<20)...)

	p := s.path("home", "user", "Downloads", name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(p, image, 0o644); err != nil {
		s.t.Fatal(err)
	}
	return p
}

func loadRecord(t *testing.T, name string) vbox.Record {
	t.Helper()
	path := filepath.Join(os.Getenv("OLIVER_VM_ROOT"), name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no record was written: %v", err)
	}
	var r vbox.Record
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

// The whole point of the command: the first run asks how big the machine
// should be, which installer to boot and where to keep it, and nothing is
// created until those are answered.
func TestVMFirstRunAsksHowBigTheMachineShouldBe(t *testing.T) {
	s := newSystem(t)
	log := s.virtualbox(t)
	iso := s.installer("Win10_22H2_English_x64.iso")
	where := s.path("home", "user", "Machines")

	out, err := s.answer(strings.Join([]string{iso, where, "6G", "3", "80", "y"}, "\n")+"\n",
		"vm", "setup")
	if err != nil {
		t.Fatalf("vm setup: %v\n%s", err, out)
	}

	for _, want := range []string{
		"Windows installer image",
		"Memory for Windows",
		"Processors",
		"Disk size for Windows",
		"Where to keep the machine",
		"Create this machine?",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the first run should ask %q:\n%s", want, out)
		}
	}

	record := loadRecord(t, "Windows")
	if record.Spec.RAMMB != 6144 {
		t.Errorf("memory = %d MB; want the 6 GB that was asked for", record.Spec.RAMMB)
	}
	if record.Spec.CPUs != 3 {
		t.Errorf("processors = %d; want 3", record.Spec.CPUs)
	}
	if record.Spec.DiskGB != 80 {
		t.Errorf("disk = %d GB; want 80", record.Spec.DiskGB)
	}
	if record.Spec.Dir != where {
		t.Errorf("location = %q; want %q", record.Spec.Dir, where)
	}
	if record.Spec.ISO != iso {
		t.Errorf("installer = %q; want %q", record.Spec.ISO, iso)
	}

	calls := vboxCalls(t, log)
	for _, want := range []string{
		"createvm --name Windows --basefolder " + where,
		"--memory 6144 --cpus 3",
		"--size 81920",
		"--type dvddrive --medium " + iso,
	} {
		if !strings.Contains(calls, want) {
			t.Errorf("VirtualBox was never asked to %q:\n%s", want, calls)
		}
	}
}

// Having answered once, the user must never be asked again.
func TestVMSecondRunStartsWithoutAskingAnything(t *testing.T) {
	s := newSystem(t)
	log := s.virtualbox(t)
	iso := s.installer("Win10_22H2_English_x64.iso")

	if out, err := s.run("vm", "setup", "-iso", iso, "-dir", s.path("home", "user", "Machines"),
		"-ram", "4G", "-disk", "64G", "-y"); err != nil {
		t.Fatalf("vm setup: %v\n%s", err, out)
	}
	_ = os.Truncate(log, 0)

	out, err := s.answer("", "vm")
	if err != nil {
		t.Fatalf("vm: %v\n%s", err, out)
	}
	for _, unwanted := range []string{"Memory for Windows", "Disk size", "Create this machine?"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("a machine that exists should just start, but it asked %q:\n%s", unwanted, out)
		}
	}
	if calls := vboxCalls(t, log); !strings.Contains(calls, "startvm Windows --type gui") {
		t.Errorf("the machine should have been started:\n%s", calls)
	}
	if r := loadRecord(t, "Windows"); r.Starts != 1 {
		t.Errorf("the start should have been recorded: %d", r.Starts)
	}
}

func TestVMSetupRefusesToBuildASecondMachineOverTheFirst(t *testing.T) {
	s := newSystem(t)
	s.virtualbox(t)
	iso := s.installer("Win10_22H2_English_x64.iso")
	args := []string{"vm", "setup", "-iso", iso, "-dir", s.path("home", "user", "Machines"), "-y"}

	if _, err := s.run(args...); err != nil {
		t.Fatalf("first setup: %v", err)
	}
	_, err := s.run(args...)
	if err == nil || !strings.Contains(err.Error(), "already been set up") {
		t.Errorf("err = %v; want a refusal naming the existing machine", err)
	}
}

// A mistyped path is the commonest answer to the first question, and starting
// the whole setup again over it would be miserable.
func TestVMAsksAgainWhenTheInstallerPathIsWrong(t *testing.T) {
	s := newSystem(t)
	s.virtualbox(t)
	iso := s.installer("Win10_22H2_English_x64.iso")

	out, err := s.answer(strings.Join([]string{
		s.path("home", "user", "Downloads", "typo.iso"), iso, "", "", "", "", "y",
	}, "\n")+"\n", "vm", "setup")
	if err != nil {
		t.Fatalf("vm setup: %v\n%s", err, out)
	}
	if strings.Count(out, "Windows installer image") < 2 {
		t.Errorf("the question should have been repeated:\n%s", out)
	}
	if r := loadRecord(t, "Windows"); r.Spec.ISO != iso {
		t.Errorf("installer = %q; want the second answer %q", r.Spec.ISO, iso)
	}
}

func TestVMDryRunCreatesNothing(t *testing.T) {
	s := newSystem(t)
	log := s.virtualbox(t)
	iso := s.installer("Win11_23H2_English_x64.iso")

	out, err := s.run("vm", "setup", "-iso", iso, "-dir", s.path("home", "user", "Machines"), "-n")
	if err != nil {
		t.Fatalf("vm setup -n: %v\n%s", err, out)
	}
	if !strings.Contains(out, "createvm") || !strings.Contains(out, "--tpm-type 2.0") {
		t.Errorf("a dry run should print the whole plan, Windows 11's TPM included:\n%s", out)
	}
	if calls := vboxCalls(t, log); strings.Contains(calls, "createvm") {
		t.Errorf("a dry run must not create anything:\n%s", calls)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("OLIVER_VM_ROOT"), "Windows.json")); err == nil {
		t.Error("a dry run must not record a machine")
	}
}

func TestVMWithoutVirtualBoxSaysHowToInstallIt(t *testing.T) {
	s := newSystem(t)
	t.Setenv("OLIVER_VM_ROOT", s.path("vms"))
	if _, err := vbox.Find(); err == nil {
		t.Skip("VirtualBox is installed on this machine")
	}

	_, err := s.run("vm")
	if err == nil || !strings.Contains(err.Error(), "virtualbox.org") {
		t.Errorf("err = %v; want an error saying how to get VirtualBox", err)
	}
}

func TestVMRefusesAnImageNothingCanBootFrom(t *testing.T) {
	s := newSystem(t)
	s.virtualbox(t)
	notAnISO := s.write(strings.Repeat("x", 4096), "home", "user", "Downloads", "broken.iso")

	_, err := s.run("vm", "setup", "-iso", notAnISO, "-dir", s.path("home", "user", "Machines"), "-y")
	if err == nil || !strings.Contains(err.Error(), "installer image") {
		t.Errorf("err = %v; want a complaint about the image", err)
	}
}

// Windows 11 cannot install without a TPM, and VirtualBox only grew one in
// version 7 — better to say so than to let the installer refuse an hour later.
func TestVMSaysWhenVirtualBoxIsTooOldForWindows11(t *testing.T) {
	s := newSystem(t)
	s.virtualbox(t, "6.1.50")
	iso := s.installer("Win11_23H2_English_x64.iso")

	_, err := s.run("vm", "setup", "-iso", iso, "-dir", s.path("home", "user", "Machines"), "-y")
	if err == nil || !strings.Contains(err.Error(), "TPM") {
		t.Errorf("err = %v; want an explanation that this VirtualBox is too old", err)
	}
	if err != nil && !strings.Contains(err.Error(), "-release win10") {
		t.Errorf("err = %v; want it to offer the way forward", err)
	}
}

func TestVMRejectsAWindowsItDoesNotKnow(t *testing.T) {
	s := newSystem(t)
	s.virtualbox(t)
	iso := s.installer("Win10_22H2_English_x64.iso")

	_, err := s.run("vm", "setup", "-iso", iso, "-release", "vista", "-y")
	if err == nil || !strings.Contains(err.Error(), "win11") {
		t.Errorf("err = %v; want the list of releases Oliver knows", err)
	}
}

func TestVMListAndStatusReportWhatWasBuilt(t *testing.T) {
	s := newSystem(t)
	s.virtualbox(t)
	iso := s.installer("Win11_23H2_English_x64.iso")
	if _, err := s.run("vm", "setup", "-iso", iso, "-dir", s.path("home", "user", "Machines"),
		"-ram", "8G", "-disk", "128G", "-y"); err != nil {
		t.Fatal(err)
	}

	out, err := s.run("vm", "list")
	if err != nil {
		t.Fatalf("vm list: %v", err)
	}
	for _, want := range []string{"Windows", "Windows 11", "8 GB", "128 GB"} {
		if !strings.Contains(out, want) {
			t.Errorf("list should mention %q:\n%s", want, out)
		}
	}

	out, err = s.run("vm", "status", "-json")
	if err != nil {
		t.Fatalf("vm status -json: %v", err)
	}
	var status struct {
		Spec  vbox.Spec `json:"spec"`
		State string    `json:"state"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatalf("status is not machine-readable: %v\n%s", err, out)
	}
	if status.Spec.RAMMB != 8192 || status.State != "poweroff" {
		t.Errorf("status = %+v", status)
	}
}
