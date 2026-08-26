package vbox

import (
	"path/filepath"
	"strings"
	"testing"
)

// goodSpec is a machine that nothing should complain about, so each test can
// break exactly one thing.
func goodSpec(t *testing.T) (Spec, Host) {
	t.Helper()
	// A test must not depend on how full the machine running it happens to be.
	stubFreeSpace(t, 500)
	dir := t.TempDir()
	iso := writeISO(t, "Win10_22H2_English_x64.iso", isoOptions{
		label: "CCCOMA_X64FRE_EN-US_DV9", publisher: "MICROSOFT CORPORATION",
		bootable: true, pad: 4 << 20,
	})
	return Spec{
		Name: "Windows", RAMMB: 4096, CPUs: 2, VRAMMB: 128, DiskGB: 64,
		ISO: iso, Dir: dir, Release: "win10",
	}, Host{RAMMB: 16384, CPUs: 8}
}

// problems returns the problems at or above a level, as text.
func problems(list []Problem, level Level) []string {
	var out []string
	for _, p := range list {
		if p.Level >= level {
			out = append(out, p.String())
		}
	}
	return out
}

// stubFreeSpace makes the free-space probe report a fixed figure.
func stubFreeSpace(t *testing.T, gb int) {
	t.Helper()
	real := FreeGB
	FreeGB = func(string) int { return gb }
	t.Cleanup(func() { FreeGB = real })
}

func TestCheckPassesAReasonableMachine(t *testing.T) {
	spec, host := goodSpec(t)
	if got := problems(spec.Check(host), Error); len(got) > 0 {
		t.Errorf("a reasonable machine should raise nothing: %v", got)
	}
}

func TestCheckRefusesLessMemoryThanWindowsNeeds(t *testing.T) {
	spec, host := goodSpec(t)
	spec.RAMMB = 1024
	got := strings.Join(problems(spec.Check(host), Error), "; ")
	if !strings.Contains(got, "below the minimum") {
		t.Errorf("1 GB is below Windows 10's minimum: %q", got)
	}
}

// Memory handed to a guest is memory the host no longer has, and a host with
// none left stops responding before Windows finishes starting.
func TestCheckRefusesToStarveTheHost(t *testing.T) {
	spec, host := goodSpec(t)
	spec.RAMMB = host.RAMMB
	got := strings.Join(problems(spec.Check(host), Error), "; ")
	if !strings.Contains(got, "leaves the host") {
		t.Errorf("taking all the memory should be refused: %q", got)
	}
}

func TestCheckWarnsAboutMoreThanHalfTheMemory(t *testing.T) {
	spec, host := goodSpec(t)
	spec.RAMMB = 12288 // 12 of 16 GB: allowed, but worth saying
	all := spec.Check(host)
	if Fatal(all, false) {
		t.Errorf("three quarters of the memory is a warning, not a refusal: %v", problems(all, Warn))
	}
	if got := strings.Join(problems(all, Warn), "; "); !strings.Contains(got, "more than half") {
		t.Errorf("it should still be mentioned: %q", got)
	}
}

func TestCheckRefusesADiskWindowsWillNotInstallOn(t *testing.T) {
	spec, host := goodSpec(t)
	spec.DiskGB = 16
	got := strings.Join(problems(spec.Check(host), Error), "; ")
	if !strings.Contains(got, "below the minimum") {
		t.Errorf("16 GB is below Windows 10's minimum: %q", got)
	}
}

// The disk grows as Windows fills it, so a disk larger than the space left is
// worth mentioning but is not yet a problem; a disk with no room for Windows
// at all is.
func TestCheckSeparatesATightDiskFromAFullOne(t *testing.T) {
	spec, host := goodSpec(t)

	stubFreeSpace(t, 40)
	tight := spec.Check(host)
	if Fatal(tight, false) {
		t.Errorf("40 GB free for a 64 GB disk is a warning: %v", problems(tight, Warn))
	}
	if got := strings.Join(problems(tight, Warn), "; "); !strings.Contains(got, "40 GB free") {
		t.Errorf("it should still be mentioned: %q", got)
	}

	stubFreeSpace(t, 5)
	if got := strings.Join(problems(spec.Check(host), Error), "; "); !strings.Contains(got, "not enough room") {
		t.Errorf("5 GB free is not enough for any Windows: %q", got)
	}
}

func TestCheckDemandsAnInstallerImage(t *testing.T) {
	spec, host := goodSpec(t)
	spec.ISO = ""
	all := spec.Check(host)
	if !Fatal(all, true) {
		t.Error("no installer image cannot be forced past")
	}
	if got := strings.Join(problems(all, Blocker), "; "); !strings.Contains(got, "no Windows installer") {
		t.Errorf("the message should say what is missing: %q", got)
	}
}

func TestCheckReportsAnInstallerThatCannotBeBootedFrom(t *testing.T) {
	spec, host := goodSpec(t)
	spec.ISO = writeISO(t, "extract.iso", isoOptions{label: "EXTRACTED", pad: 4 << 20})
	all := spec.Check(host)
	if got := strings.Join(problems(all, Error), "; "); !strings.Contains(got, "no boot record") {
		t.Errorf("an unbootable image should be reported: %q", got)
	}
	if !Fatal(all, false) {
		t.Error("an unbootable image should stop the build by default")
	}
	if Fatal(all, true) {
		t.Error("but the user may insist")
	}
}

func TestCheckMentionsAnImageThatDisagreesWithTheChosenRelease(t *testing.T) {
	spec, host := goodSpec(t)
	spec.ISO = writeISO(t, "Win11_23H2_English_x64.iso", isoOptions{
		label: "CCCOMA_X64FRE_EN-US_DV9", publisher: "MICROSOFT CORPORATION", bootable: true, pad: 4 << 20,
	})
	got := strings.Join(problems(spec.Check(host), Warn), "; ")
	if !strings.Contains(got, "-release win11") {
		t.Errorf("preparing a Windows 10 machine for a Windows 11 image should be flagged: %q", got)
	}
}

func TestCheckRefusesAFolderThatIsAlreadyAMachine(t *testing.T) {
	spec, host := goodSpec(t)
	if err := writeDir(filepath.Join(spec.MachineDir(), "Windows.vbox")); err != nil {
		t.Fatal(err)
	}
	all := spec.Check(host)
	if !Fatal(all, true) {
		t.Error("an occupied folder cannot be forced past")
	}
	if got := strings.Join(problems(all, Blocker), "; "); !strings.Contains(got, "already exists") {
		t.Errorf("the message should say the folder is taken: %q", got)
	}
}

// A machine that fits the host is the point of suggesting values at all.
func TestRecommendFitsTheMachineItRunsOn(t *testing.T) {
	win11, _ := ReleaseByID("win11")
	stubFreeSpace(t, 500)
	dir := t.TempDir()

	for _, tc := range []struct {
		name string
		host Host
	}{
		{"a laptop", Host{RAMMB: 8192, CPUs: 4}},
		{"a desktop", Host{RAMMB: 16384, CPUs: 8}},
		{"a workstation", Host{RAMMB: 65536, CPUs: 32}},
	} {
		spec := tc.host.Recommend(win11, dir)
		if spec.RAMMB > tc.host.RAMMB/2 {
			t.Errorf("%s: suggested %s of %s, more than half", tc.name, FormatMB(spec.RAMMB), FormatMB(tc.host.RAMMB))
		}
		if spec.RAMMB < win11.MinRAMMB {
			t.Errorf("%s: suggested %s, below what Windows 11 needs", tc.name, FormatMB(spec.RAMMB))
		}
		if spec.CPUs < 2 || spec.CPUs > tc.host.CPUs/2 {
			t.Errorf("%s: suggested %d of %d processors", tc.name, spec.CPUs, tc.host.CPUs)
		}
		if spec.DiskGB < win11.MinDiskGB {
			t.Errorf("%s: suggested a %d GB disk", tc.name, spec.DiskGB)
		}
		if !spec.EFI || !spec.TPM {
			t.Errorf("%s: a Windows 11 machine needs EFI and a TPM", tc.name)
		}
	}
}

// A machine with barely enough memory of its own must still be told the truth
// rather than handed a suggestion that cannot work.
func TestRecommendOnASmallMachineStillMeetsTheMinimum(t *testing.T) {
	win11, _ := ReleaseByID("win11")
	stubFreeSpace(t, 500)
	host := Host{RAMMB: 4096, CPUs: 2}
	spec := host.Recommend(win11, t.TempDir())

	if spec.RAMMB != win11.MinRAMMB {
		t.Errorf("suggested %s; want Windows 11's minimum", FormatMB(spec.RAMMB))
	}
	if got := problems(spec.Check(host), Error); len(got) == 0 {
		t.Error("a host with only 4 GB cannot give 4 GB away, and should say so")
	}
}

func TestParseMBUnderstandsHowPeopleWriteSizes(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
	}{
		{"8G", 8192}, {"8 GB", 8192}, {"8gb", 8192}, {"8", 8192},
		{"4096M", 4096}, {"4096", 4096}, {"512M", 512},
		{"1.5G", 1536}, {"16GiB", 16384},
	} {
		got, err := ParseMB(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("ParseMB(%q) = %d, %v; want %d", tc.in, got, err, tc.want)
		}
	}
	for _, bad := range []string{"", "lots", "-4G", "0"} {
		if _, err := ParseMB(bad); err == nil {
			t.Errorf("ParseMB(%q) should fail", bad)
		}
	}
}

func TestParseGBReadsABareNumberAsGigabytes(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
	}{
		{"64", 64}, {"64G", 64}, {"128 GB", 128}, {"1T", 1024},
	} {
		got, err := ParseGB(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("ParseGB(%q) = %d, %v; want %d", tc.in, got, err, tc.want)
		}
	}
	if _, err := ParseGB("512M"); err == nil {
		t.Error("a disk measured in megabytes is a mistake, not a request")
	}
}

func TestFormatMBSaysWhatPeopleSay(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want string
	}{{8192, "8 GB"}, {4096, "4 GB"}, {1536, "1.5 GB"}, {512, "512 MB"}} {
		if got := FormatMB(tc.in); got != tc.want {
			t.Errorf("FormatMB(%d) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
