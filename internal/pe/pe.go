// Package pe inspects Windows executables so a runner can make an informed
// choice about how to launch them, rather than guessing from the file name.
package pe

import (
	"bytes"
	"debug/pe"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Arch is the CPU architecture a PE image targets.
type Arch string

const (
	ArchX86     Arch = "x86"
	ArchX64     Arch = "x86_64"
	ArchARM64   Arch = "arm64"
	ArchARM32   Arch = "arm"
	ArchIA64    Arch = "ia64"
	ArchUnknown Arch = "unknown"
)

// Kind describes what sort of file we were handed. Not everything with a
// Windows icon is a PE image.
type Kind string

const (
	KindPE      Kind = "pe"    // ordinary Windows executable
	KindMSI     Kind = "msi"   // Windows Installer package (OLE compound file)
	KindDOS     Kind = "dos"   // MZ header but no PE payload: 16-bit DOS era
	KindBatch   Kind = "batch" // .bat / .cmd
	KindUnknown Kind = "unknown"
)

// Installer identifies the packaging technology behind a setup executable.
type Installer string

const (
	InstallerNone          Installer = ""
	InstallerInno          Installer = "Inno Setup"
	InstallerNSIS          Installer = "NSIS"
	InstallerInstallShield Installer = "InstallShield"
	InstallerMSI           Installer = "MSI"
	InstallerWiX           Installer = "WiX"
	InstallerSFX           Installer = "self-extracting archive"
	InstallerGeneric       Installer = "generic"
)

// Info is everything we could learn about a target without running it.
type Info struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Kind Kind   `json:"kind"`
	Arch Arch   `json:"arch"`
	Size int64  `json:"size_bytes"`

	// Subsystem tells us whether Windows would give this a window or a console.
	Subsystem     uint16 `json:"subsystem"`
	GUI           bool   `json:"gui"`
	Console       bool   `json:"console"`
	TargetWindows string `json:"target_windows"` // "winxp", "win7", "win10", ...

	DotNET     bool `json:"dotnet"`      // managed assembly (CLR header present)
	DotNETCore bool `json:"dotnet_core"` // .NET Core/5+ apphost rather than Framework
	Is64       bool `json:"is_64bit"`

	Installer   Installer `json:"installer,omitempty"`
	IsInstaller bool      `json:"is_installer"`

	Imports      []string `json:"imports,omitempty"`
	GraphicsAPIs []string `json:"graphics_apis,omitempty"`

	// Game-ness signals gathered from the executable and its directory.
	LikelyGame bool     `json:"likely_game"`
	GameHints  []string `json:"game_hints,omitempty"`

	Warnings []string `json:"warnings,omitempty"`
}

// JSON renders the info as indented JSON.
func (i *Info) JSON() string {
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// oleMagic is the header of an OLE2 compound document, which is what an
// .msi actually is.
var oleMagic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// Inspect reads path and reports what it is and what it will need to run.
// It never executes anything.
func Inspect(path string) (*Info, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not an executable", path)
	}

	info := &Info{
		Path: abs,
		Name: filepath.Base(abs),
		Size: st.Size(),
		Kind: KindUnknown,
		Arch: ArchUnknown,
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	head = head[:n]

	switch {
	case bytes.HasPrefix(head, oleMagic):
		// OLE compound file. An .msi is the only one of these we care about.
		info.Kind = KindMSI
		info.Installer = InstallerMSI
		info.IsInstaller = true
		info.Arch = ArchUnknown
		return info, nil
	case strings.EqualFold(filepath.Ext(abs), ".bat"), strings.EqualFold(filepath.Ext(abs), ".cmd"):
		info.Kind = KindBatch
		info.Console = true
		return info, nil
	case !bytes.HasPrefix(head, []byte("MZ")):
		return nil, fmt.Errorf("%s is not a Windows executable (no MZ header)", info.Name)
	}

	pf, err := pe.Open(abs)
	if err != nil {
		// MZ but unparseable as PE: almost always a real-mode DOS binary.
		info.Kind = KindDOS
		info.Warnings = append(info.Warnings,
			"MZ header without a PE image: this looks like a 16-bit DOS program, which needs DOSBox rather than Wine")
		return info, nil
	}
	defer pf.Close()

	info.Kind = KindPE
	info.Arch = archFromMachine(pf.Machine)
	info.Is64 = info.Arch == ArchX64 || info.Arch == ArchARM64 || info.Arch == ArchIA64

	var (
		subsystem uint16
		dirs      []pe.DataDirectory
		majorOS   uint16
	)
	switch oh := pf.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		subsystem = oh.Subsystem
		majorOS = oh.MajorSubsystemVersion
		dirs = oh.DataDirectory[:]
	case *pe.OptionalHeader64:
		subsystem = oh.Subsystem
		majorOS = oh.MajorSubsystemVersion
		dirs = oh.DataDirectory[:]
	default:
		info.Warnings = append(info.Warnings, "executable has no optional header; treating it as a console program")
	}

	info.Subsystem = subsystem
	info.GUI = subsystem == 2     // IMAGE_SUBSYSTEM_WINDOWS_GUI
	info.Console = subsystem == 3 // IMAGE_SUBSYSTEM_WINDOWS_CUI
	info.TargetWindows = windowsVersion(majorOS, minorSubsystem(pf.OptionalHeader))

	// DataDirectory index 14 is IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR: its
	// presence is the definitive marker of a managed .NET assembly.
	if len(dirs) > 14 && dirs[14].VirtualAddress != 0 && dirs[14].Size != 0 {
		info.DotNET = true
	}

	if libs, err := pf.ImportedLibraries(); err == nil {
		info.Imports = normalizeImports(libs)
	}
	// A .NET Core app is a native apphost that pulls in hostfxr, and ships a
	// runtimeconfig.json beside it. .NET Framework apps do neither.
	for _, imp := range info.Imports {
		if strings.HasPrefix(imp, "hostfxr") || strings.HasPrefix(imp, "hostpolicy") {
			info.DotNETCore = true
		}
	}
	base := strings.TrimSuffix(info.Name, filepath.Ext(info.Name))
	if _, err := os.Stat(filepath.Join(filepath.Dir(abs), base+".runtimeconfig.json")); err == nil {
		info.DotNETCore = true
		info.DotNET = true
	}

	info.GraphicsAPIs = graphicsFromImports(info.Imports)
	info.Installer, info.IsInstaller = detectInstaller(abs, info.Name, f)
	info.LikelyGame, info.GameHints = detectGame(abs, info)

	return info, nil
}

func minorSubsystem(oh any) uint16 {
	switch h := oh.(type) {
	case *pe.OptionalHeader32:
		return h.MinorSubsystemVersion
	case *pe.OptionalHeader64:
		return h.MinorSubsystemVersion
	}
	return 0
}

func archFromMachine(m uint16) Arch {
	switch m {
	case pe.IMAGE_FILE_MACHINE_I386:
		return ArchX86
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return ArchX64
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return ArchARM64
	case pe.IMAGE_FILE_MACHINE_ARM, pe.IMAGE_FILE_MACHINE_ARMNT, pe.IMAGE_FILE_MACHINE_THUMB:
		return ArchARM32
	case pe.IMAGE_FILE_MACHINE_IA64:
		return ArchIA64
	}
	return ArchUnknown
}

// windowsVersion maps a PE subsystem version to the Windows release a prefix
// should report, so apps that gate on the OS version behave.
func windowsVersion(major, minor uint16) string {
	switch {
	case major >= 10:
		return "win10"
	case major == 6 && minor >= 3:
		return "win81"
	case major == 6 && minor == 2:
		return "win8"
	case major == 6 && minor == 1:
		return "win7"
	case major == 6 && minor == 0:
		return "vista"
	case major == 5 && minor >= 1:
		return "winxp"
	case major == 5:
		return "win2k"
	}
	return ""
}

func normalizeImports(libs []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(libs))
	for _, l := range libs {
		l = strings.ToLower(strings.TrimSpace(l))
		if l == "" || seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	return out
}

// graphicsAPI maps a DLL import to the user-facing API name it implies.
var graphicsAPI = map[string]string{
	"d3d8.dll":      "Direct3D 8",
	"d3d9.dll":      "Direct3D 9",
	"d3d10.dll":     "Direct3D 10",
	"d3d11.dll":     "Direct3D 11",
	"d3d12.dll":     "Direct3D 12",
	"dxgi.dll":      "DXGI",
	"ddraw.dll":     "DirectDraw",
	"opengl32.dll":  "OpenGL",
	"vulkan-1.dll":  "Vulkan",
	"dinput.dll":    "DirectInput",
	"dinput8.dll":   "DirectInput 8",
	"dsound.dll":    "DirectSound",
	"xaudio2_9.dll": "XAudio2",
	"xaudio2_8.dll": "XAudio2",
}

func graphicsFromImports(imports []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, imp := range imports {
		name, ok := graphicsAPI[imp]
		if !ok && strings.HasPrefix(imp, "xinput") {
			name, ok = "XInput", true
		}
		if !ok && strings.HasPrefix(imp, "xaudio2") {
			name, ok = "XAudio2", true
		}
		if ok && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// installerMarkers are byte signatures the major installer toolkits leave in
// their stub. Matching these is far more reliable than a file-name guess.
var installerMarkers = []struct {
	needle []byte
	kind   Installer
}{
	{[]byte("Inno Setup"), InstallerInno},
	{[]byte("Nullsoft Install System"), InstallerNSIS},
	{[]byte("NullsoftInst"), InstallerNSIS},
	{[]byte("InstallShield"), InstallerInstallShield},
	{[]byte("Windows Installer XML"), InstallerWiX},
	{[]byte("WixBundle"), InstallerWiX},
	{[]byte("7-Zip SFX"), InstallerSFX},
	{[]byte("WinRAR SFX"), InstallerSFX},
}

var installerName = regexp.MustCompile(`(?i)^(setup|install(er)?|unins[0-9]*|uninstall(er)?)([-_. ].*)?$|[-_.](setup|install(er)?)$`)

// detectInstaller scans the head and tail of the file for toolkit signatures,
// falling back to the file name. Installers behave differently enough from
// applications that it is worth getting right.
func detectInstaller(path, name string, f *os.File) (Installer, bool) {
	if strings.EqualFold(filepath.Ext(name), ".msi") {
		return InstallerMSI, true
	}

	const window = 2 << 20 // 2 MiB is enough to cover the stub and its resources
	scan := func(off int64, size int) Installer {
		buf := make([]byte, size)
		n, err := f.ReadAt(buf, off)
		if n <= 0 && err != nil && !errors.Is(err, io.EOF) {
			return InstallerNone
		}
		buf = buf[:n]
		for _, m := range installerMarkers {
			if bytes.Contains(buf, m.needle) {
				return m.kind
			}
		}
		return InstallerNone
	}

	if k := scan(0, window); k != InstallerNone {
		return k, true
	}
	// Some SFX archives stash their banner near the end of the payload.
	if st, err := f.Stat(); err == nil && st.Size() > window {
		if k := scan(st.Size()-window, window); k != InstallerNone {
			return k, true
		}
	}

	stem := strings.TrimSuffix(name, filepath.Ext(name))
	if installerName.MatchString(stem) {
		return InstallerGeneric, true
	}
	return InstallerNone, false
}

// gameSidecars are files that, when they sit next to an executable, are strong
// evidence it is a game: they ship with engines and storefront SDKs.
var gameSidecars = map[string]string{
	"steam_api.dll":             "Steamworks SDK",
	"steam_api64.dll":           "Steamworks SDK",
	"unityplayer.dll":           "Unity engine",
	"galaxy.dll":                "GOG Galaxy SDK",
	"galaxy64.dll":              "GOG Galaxy SDK",
	"eossdk-win64-shipping.dll": "Epic Online Services",
	"d3dcompiler_47.dll":        "Direct3D shader compiler",
	"openvr_api.dll":            "OpenVR",
	"fmod.dll":                  "FMOD audio engine",
	"fmodstudio.dll":            "FMOD audio engine",
	"bink2w64.dll":              "Bink video",
}

// detectGame combines import-table evidence with a scan of the executable's
// own directory. Engines commonly load Direct3D lazily, so imports alone miss
// a lot of real games.
func detectGame(path string, info *Info) (bool, []string) {
	var hints []string

	for _, api := range info.GraphicsAPIs {
		switch api {
		case "Direct3D 8", "Direct3D 9", "Direct3D 10", "Direct3D 11", "Direct3D 12", "DXGI", "DirectDraw", "Vulkan":
			hints = append(hints, "imports "+api)
		case "XInput", "DirectInput", "DirectInput 8":
			hints = append(hints, "imports "+api+" (gamepad)")
		}
	}

	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err == nil {
		seen := map[string]bool{}
		for _, e := range entries {
			lower := strings.ToLower(e.Name())
			if desc, ok := gameSidecars[lower]; ok && !seen[desc] {
				seen[desc] = true
				hints = append(hints, desc+" alongside the executable")
			}
			// Unity ships a "<Name>_Data" directory next to the player.
			if e.IsDir() && strings.HasSuffix(lower, "_data") && !seen["Unity data directory"] {
				stem := strings.ToLower(strings.TrimSuffix(info.Name, filepath.Ext(info.Name)))
				if lower == stem+"_data" {
					seen["Unity data directory"] = true
					hints = append(hints, "Unity data directory")
				}
			}
		}
	}

	// An installer that happens to link DirectX is still an installer.
	if info.IsInstaller {
		return false, hints
	}
	return len(hints) > 0, hints
}
