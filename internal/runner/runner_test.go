package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/backend"
	"github.com/robertpjenkins-bot/oliver/internal/pe"
	"github.com/robertpjenkins-bot/oliver/internal/prefix"
)

func wineBackend() *backend.Backend {
	return &backend.Backend{
		ID: "system-wine", Name: "wine", Kind: backend.KindWine,
		Source: backend.SourceSystem, Exec: "/usr/bin/wine",
		Version: backend.ParseVersion("wine-9.0"), Has32: true, Has64: true,
	}
}

func protonBackend(withUmu bool) *backend.Backend {
	b := &backend.Backend{
		ID: "proton-ge", Name: "GE-Proton9-20", Kind: backend.KindProton,
		Source: backend.SourceCompatTools, Exec: "/steam/GE-Proton9-20/proton",
		Root: "/steam/GE-Proton9-20", Version: backend.ParseVersion("GE-Proton9-20"),
		Has32: true, Has64: true,
	}
	if withUmu {
		b.Launcher = []string{"/usr/bin/umu-run"}
	}
	return b
}

func testPrefix(t *testing.T, created bool) *prefix.Prefix {
	t.Helper()
	m := &prefix.Manager{Root: t.TempDir()}
	p := m.For("/games/demo/demo.exe", "demo")
	if created {
		if err := p.Prepare(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.WinePrefix, "system.reg"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func info(kind pe.Kind, arch pe.Arch) *pe.Info {
	return &pe.Info{Kind: kind, Arch: arch, Path: "/games/demo/demo.exe", Name: "demo.exe"}
}

func envOf(step Step) map[string]string {
	m := map[string]string{}
	for _, kv := range step.Env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			m[k] = v
		}
	}
	return m
}

func TestPlanForWineRunsFromTheProgramsOwnDirectory(t *testing.T) {
	plan, err := Build(Options{
		Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(),
		Prefix: testPrefix(t, true), Args: []string{"--flag", "a b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/usr/bin/wine", "/games/demo/demo.exe", "--flag", "a b"}
	if strings.Join(plan.Launch.Argv, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("argv = %v, want %v", plan.Launch.Argv, want)
	}
	if plan.Launch.Dir != "/games/demo" {
		t.Errorf("working directory = %q, want the program's own directory", plan.Launch.Dir)
	}
}

func TestWineArchIsSetOnlyWhenCreatingAPrefix(t *testing.T) {
	// A new prefix for a 32-bit program must be created as win32.
	plan, err := Build(Options{Info: info(pe.KindPE, pe.ArchX86), Backend: wineBackend(), Prefix: testPrefix(t, false)})
	if err != nil {
		t.Fatal(err)
	}
	if plan.WineArch != "win32" {
		t.Errorf("WineArch = %q, want win32", plan.WineArch)
	}
	if envOf(plan.Launch)["WINEARCH"] != "win32" {
		t.Error("WINEARCH must be set while the prefix is being created")
	}

	// Against an existing prefix, WINEARCH must not be sent at all: Wine
	// refuses to start when it disagrees with the prefix on disk.
	plan, err = Build(Options{Info: info(pe.KindPE, pe.ArchX86), Backend: wineBackend(), Prefix: testPrefix(t, true)})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := envOf(plan.Launch)["WINEARCH"]; ok {
		t.Error("WINEARCH must not be set for an existing prefix")
	}
}

func TestNewPrefixIsInitialisedBeforeLaunch(t *testing.T) {
	plan, err := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(), Prefix: testPrefix(t, false)})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.NewPrefix || len(plan.Setup) == 0 {
		t.Fatalf("expected a prefix-creation step, got %+v", plan.Setup)
	}
	if !strings.Contains(strings.Join(plan.Setup[0].Argv, " "), "wineboot") {
		t.Errorf("first setup step should be wineboot, got %v", plan.Setup[0].Argv)
	}

	plan, err = Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(), Prefix: testPrefix(t, true)})
	if err != nil {
		t.Fatal(err)
	}
	if plan.NewPrefix || len(plan.Setup) != 0 {
		t.Errorf("an existing prefix needs no setup, got %+v", plan.Setup)
	}
}

func TestMSIIsHandedToMsiexec(t *testing.T) {
	i := info(pe.KindMSI, pe.ArchUnknown)
	i.Path, i.Name = "/games/demo/package.msi", "package.msi"
	plan, err := Build(Options{Info: i, Backend: wineBackend(), Prefix: testPrefix(t, true)})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(plan.Launch.Argv, " ")
	if !strings.Contains(got, "msiexec /i /games/demo/package.msi") {
		t.Errorf("argv = %q, want an msiexec invocation", got)
	}
}

func TestBatchFileGoesThroughCmd(t *testing.T) {
	i := info(pe.KindBatch, pe.ArchUnknown)
	i.Path, i.Name = "/games/demo/run.bat", "run.bat"
	plan, err := Build(Options{Info: i, Backend: wineBackend(), Prefix: testPrefix(t, true)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(plan.Launch.Argv, " "), "cmd /c") {
		t.Errorf("argv = %v, want cmd /c", plan.Launch.Argv)
	}
}

func TestDOSBinaryIsRefused(t *testing.T) {
	if _, err := Build(Options{Info: info(pe.KindDOS, pe.ArchUnknown), Backend: wineBackend(), Prefix: testPrefix(t, true)}); err == nil {
		t.Fatal("expected DOS binaries to be refused")
	}
}

func TestProtonUsesUmuWhenAvailable(t *testing.T) {
	p := testPrefix(t, true)
	plan, err := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: protonBackend(true), Prefix: p})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Launch.Argv[0] != "/usr/bin/umu-run" {
		t.Errorf("argv = %v, want umu-run first", plan.Launch.Argv)
	}
	env := envOf(plan.Launch)
	if env["PROTONPATH"] != "/steam/GE-Proton9-20" {
		t.Errorf("PROTONPATH = %q", env["PROTONPATH"])
	}
	if env["STEAM_COMPAT_DATA_PATH"] != p.Dir {
		t.Errorf("STEAM_COMPAT_DATA_PATH = %q, want the prefix container %q", env["STEAM_COMPAT_DATA_PATH"], p.Dir)
	}
	if env["WINEPREFIX"] != p.WinePrefix {
		t.Errorf("WINEPREFIX = %q, want %q", env["WINEPREFIX"], p.WinePrefix)
	}
	if env["GAMEID"] == "" || env["PROTON_VERB"] != "run" {
		t.Errorf("umu needs GAMEID and PROTON_VERB, got %v", env)
	}
}

func TestProtonFallsBackToItsOwnEntryPoint(t *testing.T) {
	plan, err := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: protonBackend(false), Prefix: testPrefix(t, true)})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/steam/GE-Proton9-20/proton", "run", "/games/demo/demo.exe"}
	if strings.Join(plan.Launch.Argv, " ") != strings.Join(want, " ") {
		t.Errorf("argv = %v, want %v", plan.Launch.Argv, want)
	}
}

func TestProtonNeverGetsWineArchOrWineboot(t *testing.T) {
	// Proton builds its own prefix; driving it with Wine's variables corrupts it.
	plan, err := Build(Options{Info: info(pe.KindPE, pe.ArchX86), Backend: protonBackend(true), Prefix: testPrefix(t, false)})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := envOf(plan.Launch)["WINEARCH"]; ok {
		t.Error("WINEARCH must never be set for Proton")
	}
	for _, s := range plan.Setup {
		if strings.Contains(strings.Join(s.Argv, " "), "wineboot") {
			t.Error("Proton must not be bootstrapped with wineboot")
		}
	}
}

// TestNewProtonPrefixIsBuiltBeforeTheProgramRuns covers the second half of the
// isolation guarantee. Proton creates its prefix during launch, so without an
// explicit build step the home directories are only isolated after the program
// has already had a run with access to them.
func TestNewProtonPrefixIsBuiltBeforeTheProgramRuns(t *testing.T) {
	plan, err := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: protonBackend(true), Prefix: testPrefix(t, false)})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Setup) == 0 {
		t.Fatal("a new Proton prefix needs a build step before launch")
	}
	first := strings.Join(plan.Setup[0].Argv, " ")
	if !strings.Contains(first, "umu-run") || !strings.Contains(first, "cmd /c exit") {
		t.Errorf("setup step = %q, want a trivial command through Proton", first)
	}
	if !plan.Setup[0].Optional {
		t.Error("failing to pre-build the prefix must not abort the launch")
	}

	// An existing prefix needs no build step at all.
	plan, err = Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: protonBackend(true), Prefix: testPrefix(t, true)})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Setup) != 0 {
		t.Errorf("an existing Proton prefix needs no setup, got %+v", plan.Setup)
	}
}

func TestMenuBuilderIsAlwaysDisabledAndMonoOnlyWhenUnused(t *testing.T) {
	plan, _ := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(), Prefix: testPrefix(t, true)})
	ov := envOf(plan.Launch)["WINEDLLOVERRIDES"]
	if !strings.Contains(ov, "winemenubuilder.exe=d") {
		t.Errorf("WINEDLLOVERRIDES = %q, expected winemenubuilder to be disabled", ov)
	}
	if !strings.Contains(ov, "mscoree=d") {
		t.Errorf("WINEDLLOVERRIDES = %q, expected the Mono prompt to be suppressed", ov)
	}

	// A .NET program needs mscoree, so it must not be disabled.
	dotnet := info(pe.KindPE, pe.ArchX64)
	dotnet.DotNET = true
	plan, _ = Build(Options{Info: dotnet, Backend: wineBackend(), Prefix: testPrefix(t, true)})
	if ov := envOf(plan.Launch)["WINEDLLOVERRIDES"]; strings.Contains(ov, "mscoree=d") {
		t.Errorf("mscoree must stay enabled for .NET programs, got %q", ov)
	}
}

func TestWinVerAutoOnlyDowngrades(t *testing.T) {
	cases := map[string]string{"win7": "win7", "winxp": "winxp", "win10": "", "": ""}
	for target, want := range cases {
		i := info(pe.KindPE, pe.ArchX64)
		i.TargetWindows = target
		plan, err := Build(Options{Info: i, Backend: wineBackend(), Prefix: testPrefix(t, true), WinVer: "auto"})
		if err != nil {
			t.Fatal(err)
		}
		if plan.WinVer != want {
			t.Errorf("target %q => WinVer %q, want %q", target, plan.WinVer, want)
		}
	}

	// An explicit version is always honoured.
	plan, _ := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(), Prefix: testPrefix(t, true), WinVer: "win2k"})
	if plan.WinVer != "win2k" {
		t.Errorf("WinVer = %q, want win2k", plan.WinVer)
	}
	found := false
	for _, s := range plan.Setup {
		if strings.Contains(strings.Join(s.Argv, " "), "HKCU\\Software\\Wine") {
			found = true
			if !s.Optional {
				t.Error("a failed version tweak should not abort the launch")
			}
		}
	}
	if !found {
		t.Error("expected a registry step applying the Windows version")
	}
}

func TestVerboseControlsWineDebug(t *testing.T) {
	quiet, _ := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(), Prefix: testPrefix(t, true)})
	if envOf(quiet.Launch)["WINEDEBUG"] != "-all" {
		t.Error("expected Wine to be silenced by default")
	}
	loud, _ := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(), Prefix: testPrefix(t, true), Verbose: true})
	if envOf(loud.Launch)["WINEDEBUG"] == "-all" {
		t.Error("verbose mode must not silence Wine")
	}
}

func TestUserEnvironmentOverridesDefaults(t *testing.T) {
	plan, _ := Build(Options{
		Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(), Prefix: testPrefix(t, true),
		Env: []string{"WINEDEBUG=+relay", "DXVK_HUD=fps"},
	})
	env := envOf(plan.Launch)
	if env["WINEDEBUG"] != "+relay" {
		t.Errorf("user WINEDEBUG ignored: %q", env["WINEDEBUG"])
	}
	if env["DXVK_HUD"] != "fps" {
		t.Error("extra user variables should be passed through")
	}
}

func TestFlatpakAndTranslationLayerComposition(t *testing.T) {
	flat := wineBackend()
	flat.Source = backend.SourceFlatpak
	flat.Launcher = []string{"flatpak", "run", "--command=wine", "org.winehq.Wine"}
	plan, _ := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: flat, Prefix: testPrefix(t, true)})
	want := "flatpak run --command=wine org.winehq.Wine /games/demo/demo.exe"
	if got := strings.Join(plan.Launch.Argv, " "); got != want {
		t.Errorf("flatpak argv = %q, want %q", got, want)
	}

	box := wineBackend()
	box.Kind = backend.KindBox
	box.Launcher = []string{"/usr/bin/box64"}
	plan, _ = Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: box, Prefix: testPrefix(t, true)})
	want = "/usr/bin/box64 /usr/bin/wine /games/demo/demo.exe"
	if got := strings.Join(plan.Launch.Argv, " "); got != want {
		t.Errorf("box64 argv = %q, want %q", got, want)
	}
}

func TestEnvironmentIsDeterministic(t *testing.T) {
	p := testPrefix(t, true)
	first, _ := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: protonBackend(true), Prefix: p})
	for i := 0; i < 10; i++ {
		next, _ := Build(Options{Info: info(pe.KindPE, pe.ArchX64), Backend: protonBackend(true), Prefix: p})
		if strings.Join(first.Launch.Env, "|") != strings.Join(next.Launch.Env, "|") {
			t.Fatal("environment ordering is not deterministic")
		}
	}
}

func TestMergeEnvReplacesRatherThanDuplicates(t *testing.T) {
	got := mergeEnv([]string{"PATH=/bin", "HOME=/root"}, []string{"HOME=/prefix", "NEW=1"})
	counts := map[string]int{}
	for _, kv := range got {
		k, _, _ := strings.Cut(kv, "=")
		counts[k]++
	}
	if counts["HOME"] != 1 {
		t.Errorf("HOME appears %d times, want 1", counts["HOME"])
	}
	found := false
	for _, kv := range got {
		if kv == "HOME=/prefix" {
			found = true
		}
	}
	if !found || counts["NEW"] != 1 || counts["PATH"] != 1 {
		t.Errorf("mergeEnv = %v", got)
	}
}

func TestStepRendersAsCopyableShellCommand(t *testing.T) {
	s := Step{
		Argv: []string{"/usr/bin/wine", "/games/My Game/run it.exe", "--x"},
		Env:  []string{"WINEPREFIX=/home/u/.local/share/oliver/prefixes/my-game/pfx"},
	}
	got := s.String()
	if !strings.Contains(got, `'/games/My Game/run it.exe'`) {
		t.Errorf("paths with spaces must be quoted: %s", got)
	}
	if strings.Contains(got, `'/usr/bin/wine'`) {
		t.Errorf("simple arguments should not be quoted: %s", got)
	}
}

func TestBuildRejectsIncompleteOptions(t *testing.T) {
	for _, o := range []Options{
		{Info: info(pe.KindPE, pe.ArchX64), Prefix: testPrefix(t, true)},
		{Backend: wineBackend(), Prefix: testPrefix(t, true)},
		{Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend()},
	} {
		if _, err := Build(o); err == nil {
			t.Errorf("expected an error for incomplete options %+v", o)
		}
	}
}

// TestNonExecutableTargetsAreHandledForEveryBackend guards a real defect: the
// Proton path returned before the check for targets that are not executables,
// so an MSI was handed to Proton as if it were a program. On a machine with no
// system Wine — a Steam Deck, for instance — that is every installer.
func TestNonExecutableTargetsAreHandledForEveryBackend(t *testing.T) {
	msi := info(pe.KindMSI, pe.ArchUnknown)
	msi.Path, msi.Name = "/games/demo/Installer.msi", "Installer.msi"

	bat := info(pe.KindBatch, pe.ArchUnknown)
	bat.Path, bat.Name = "/games/demo/run.bat", "run.bat"

	cases := []struct {
		name    string
		backend *backend.Backend
		info    *pe.Info
		want    string
	}{
		{"msi via umu", protonBackend(true), msi, "/usr/bin/umu-run msiexec /i /games/demo/Installer.msi"},
		{"msi via proton", protonBackend(false), msi, "/steam/GE-Proton9-20/proton run msiexec /i /games/demo/Installer.msi"},
		{"msi via wine", wineBackend(), msi, "/usr/bin/wine msiexec /i /games/demo/Installer.msi"},
		{"batch via umu", protonBackend(true), bat, "/usr/bin/umu-run cmd /c /games/demo/run.bat"},
		{"batch via proton", protonBackend(false), bat, "/steam/GE-Proton9-20/proton run cmd /c /games/demo/run.bat"},
		{"batch via wine", wineBackend(), bat, "/usr/bin/wine cmd /c /games/demo/run.bat"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan, err := Build(Options{Info: c.info, Backend: c.backend, Prefix: testPrefix(t, true)})
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(plan.Launch.Argv, " "); got != c.want {
				t.Errorf("argv = %q\n    want %q", got, c.want)
			}
		})
	}
}

func TestProgramArgumentsFollowTheInstallerCommand(t *testing.T) {
	msi := info(pe.KindMSI, pe.ArchUnknown)
	msi.Path, msi.Name = "/games/demo/Installer.msi", "Installer.msi"
	plan, err := Build(Options{
		Info: msi, Backend: protonBackend(true), Prefix: testPrefix(t, true),
		Args: []string{"/quiet"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "/usr/bin/umu-run msiexec /i /games/demo/Installer.msi /quiet"
	if got := strings.Join(plan.Launch.Argv, " "); got != want {
		t.Errorf("argv = %q, want %q", got, want)
	}
}

func TestDOSBinaryIsRefusedForProtonToo(t *testing.T) {
	// The refusal used to live only on the Wine path.
	for _, b := range []*backend.Backend{protonBackend(true), protonBackend(false), wineBackend()} {
		_, err := Build(Options{Info: info(pe.KindDOS, pe.ArchUnknown), Backend: b, Prefix: testPrefix(t, true)})
		if err == nil {
			t.Errorf("%s: expected a DOS binary to be refused", b.ID)
		}
	}
}
