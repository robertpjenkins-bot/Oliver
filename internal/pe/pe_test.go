package pe

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/petest"
)

// buildPE writes a fixture image into dir and returns its path.
func buildPE(t *testing.T, dir, name string, o petest.Options) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := petest.WriteFile(path, o); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestInspectArchitectureAndSubsystem(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name    string
		opts    petest.Options
		arch    Arch
		gui     bool
		console bool
		is64    bool
	}{
		{"app32.exe", petest.Options{Machine: petest.MachineI386, Subsystem: petest.SubsystemConsole}, ArchX86, false, true, false},
		{"app64.exe", petest.Options{Machine: petest.MachineAMD64, Subsystem: petest.SubsystemConsole, PE32Plus: true}, ArchX64, false, true, true},
		{"gui64.exe", petest.Options{Machine: petest.MachineAMD64, Subsystem: petest.SubsystemGUI, PE32Plus: true}, ArchX64, true, false, true},
		{"arm64.exe", petest.Options{Machine: petest.MachineARM64, Subsystem: petest.SubsystemGUI, PE32Plus: true}, ArchARM64, true, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := buildPE(t, dir, c.name, c.opts)
			got, err := Inspect(path)
			if err != nil {
				t.Fatalf("Inspect: %v", err)
			}
			if got.Kind != KindPE {
				t.Errorf("Kind = %q, want pe", got.Kind)
			}
			if got.Arch != c.arch {
				t.Errorf("Arch = %q, want %q", got.Arch, c.arch)
			}
			if got.GUI != c.gui || got.Console != c.console {
				t.Errorf("GUI/Console = %v/%v, want %v/%v", got.GUI, got.Console, c.gui, c.console)
			}
			if got.Is64 != c.is64 {
				t.Errorf("Is64 = %v, want %v", got.Is64, c.is64)
			}
		})
	}
}

func TestInspectDetectsManagedAssembly(t *testing.T) {
	dir := t.TempDir()
	managed := buildPE(t, dir, "managed.exe", petest.Options{Machine: petest.MachineI386, Subsystem: petest.SubsystemGUI, CLRHeader: true})
	native := buildPE(t, dir, "native.exe", petest.Options{Machine: petest.MachineI386, Subsystem: petest.SubsystemGUI})

	if got, err := Inspect(managed); err != nil || !got.DotNET {
		t.Errorf("expected a .NET assembly to be detected (err=%v)", err)
	}
	if got, err := Inspect(native); err != nil || got.DotNET {
		t.Errorf("native binary wrongly reported as .NET (err=%v)", err)
	}
}

func TestInspectDetectsDotNetCoreViaRuntimeConfig(t *testing.T) {
	dir := t.TempDir()
	path := buildPE(t, dir, "tool.exe", petest.Options{Machine: petest.MachineAMD64, Subsystem: petest.SubsystemConsole, PE32Plus: true})
	if err := os.WriteFile(filepath.Join(dir, "tool.runtimeconfig.json"), []byte(`{"runtimeOptions":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.DotNETCore || !got.DotNET {
		t.Errorf("expected .NET Core detection, got DotNET=%v Core=%v", got.DotNET, got.DotNETCore)
	}
}

func TestInspectTargetWindowsVersion(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		maj, min uint16
		want     string
	}{
		{5, 1, "winxp"},
		{6, 0, "vista"},
		{6, 1, "win7"},
		{6, 2, "win8"},
		{6, 3, "win81"},
		{10, 0, "win10"},
	}
	for _, c := range cases {
		p := buildPE(t, dir, "v.exe", petest.Options{Machine: petest.MachineAMD64, Subsystem: petest.SubsystemGUI, PE32Plus: true, SubsystemMajor: c.maj, SubsystemMinor: c.min})
		got, err := Inspect(p)
		if err != nil {
			t.Fatal(err)
		}
		if got.TargetWindows != c.want {
			t.Errorf("subsystem %d.%d => %q, want %q", c.maj, c.min, got.TargetWindows, c.want)
		}
	}
}

func TestInspectRecognisesNonPEInputs(t *testing.T) {
	dir := t.TempDir()

	msi := filepath.Join(dir, "package.msi")
	if err := os.WriteFile(msi, append(oleMagic, make([]byte, 512)...), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Inspect(msi)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != KindMSI || !got.IsInstaller || got.Installer != InstallerMSI {
		t.Errorf("MSI detection failed: %+v", got)
	}

	// An MZ header with no PE payload is a real-mode DOS binary.
	dos := filepath.Join(dir, "old.exe")
	body := make([]byte, 256)
	copy(body, "MZ")
	if err := os.WriteFile(dos, body, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Inspect(dos)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != KindDOS {
		t.Errorf("Kind = %q, want dos", got.Kind)
	}
	if len(got.Warnings) == 0 {
		t.Error("expected a warning pointing at DOSBox")
	}

	notExe := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(notExe, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(notExe); err == nil {
		t.Error("expected an error for a non-executable file")
	}
}

func TestInstallerDetectionFromSignatureAndName(t *testing.T) {
	dir := t.TempDir()

	inno := buildPE(t, dir, "totally-normal-app.exe", petest.Options{
		Machine: petest.MachineI386, Subsystem: petest.SubsystemGUI,
		Trailing: []byte("....This installation was built with Inno Setup 6.2...."),
	})
	got, err := Inspect(inno)
	if err != nil {
		t.Fatal(err)
	}
	if got.Installer != InstallerInno || !got.IsInstaller {
		t.Errorf("Installer = %q, want Inno Setup", got.Installer)
	}

	nsis := buildPE(t, dir, "app2.exe", petest.Options{
		Machine: petest.MachineI386, Subsystem: petest.SubsystemGUI,
		Trailing: []byte("Nullsoft Install System v3.08"),
	})
	if got, _ := Inspect(nsis); got.Installer != InstallerNSIS {
		t.Errorf("Installer = %q, want NSIS", got.Installer)
	}

	// No signature, but the name is unambiguous.
	named := buildPE(t, dir, "setup.exe", petest.Options{Machine: petest.MachineI386, Subsystem: petest.SubsystemGUI})
	if got, _ := Inspect(named); !got.IsInstaller || got.Installer != InstallerGeneric {
		t.Errorf("setup.exe not recognised as an installer")
	}

	plain := buildPE(t, dir, "notepad.exe", petest.Options{Machine: petest.MachineI386, Subsystem: petest.SubsystemGUI})
	if got, _ := Inspect(plain); got.IsInstaller {
		t.Errorf("notepad.exe wrongly flagged as an installer")
	}
}

func TestGameDetectionFromSidecarFiles(t *testing.T) {
	dir := t.TempDir()
	path := buildPE(t, dir, "Game.exe", petest.Options{Machine: petest.MachineAMD64, Subsystem: petest.SubsystemGUI, PE32Plus: true})

	got, _ := Inspect(path)
	if got.LikelyGame {
		t.Error("a bare executable should not be classified as a game")
	}

	// Storefront SDK and engine data alongside the binary are strong signals.
	if err := os.WriteFile(filepath.Join(dir, "steam_api64.dll"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "Game_Data"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, _ = Inspect(path)
	if !got.LikelyGame {
		t.Errorf("expected a game classification, hints=%v", got.GameHints)
	}
	if len(got.GameHints) < 2 {
		t.Errorf("expected both the Steamworks and Unity hints, got %v", got.GameHints)
	}

	// An installer that ships next to game files is still an installer.
	setup := buildPE(t, dir, "setup.exe", petest.Options{Machine: petest.MachineI386, Subsystem: petest.SubsystemGUI})
	if got, _ := Inspect(setup); got.LikelyGame {
		t.Error("an installer must not be classified as a game")
	}
}

func TestGraphicsAPIMapping(t *testing.T) {
	got := graphicsFromImports([]string{"kernel32.dll", "d3d11.dll", "dxgi.dll", "xinput1_4.dll", "xaudio2_9.dll", "user32.dll"})
	want := map[string]bool{"Direct3D 11": true, "DXGI": true, "XInput": true, "XAudio2": true}
	if len(got) != len(want) {
		t.Fatalf("graphicsFromImports = %v, want %v entries", got, len(want))
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected API %q", g)
		}
	}
}

// TestInspectRealBinary validates the parser against an executable produced by
// a real linker rather than by our own test helper.
func TestInspectRealBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cross-compilation in short mode")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain unavailable")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\n\nfunc main() { println(\"hi\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		goarch string
		arch   Arch
	}{{"386", ArchX86}, {"amd64", ArchX64}} {
		out := filepath.Join(dir, "fixture-"+c.goarch+".exe")
		cmd := exec.Command(goBin, "build", "-o", out, ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH="+c.goarch, "GOFLAGS=")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("cross-compile for %s unavailable: %v: %s", c.goarch, err, b)
		}
		got, err := Inspect(out)
		if err != nil {
			t.Fatalf("Inspect(%s): %v", out, err)
		}
		if got.Arch != c.arch {
			t.Errorf("%s: Arch = %q, want %q", c.goarch, got.Arch, c.arch)
		}
		if got.Kind != KindPE {
			t.Errorf("%s: Kind = %q, want pe", c.goarch, got.Kind)
		}
		if !got.Console {
			t.Errorf("%s: expected a console subsystem", c.goarch)
		}
		if got.IsInstaller {
			t.Errorf("%s: wrongly flagged as an installer", c.goarch)
		}
	}
}
