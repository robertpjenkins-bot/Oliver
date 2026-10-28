package minecraft

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// A mod is on when its file ends in .jar and off when it ends in
// .jar.disabled: no loader loads the second, and Prism Launcher and others
// show it as a switched-off mod. Turning a mod on or off is a rename, so
// nothing is ever lost.
const disabledSuffix = ".disabled"

// InstalledMod is a mod file in a mods folder.
type InstalledMod struct {
	File    string `json:"file"`
	Enabled bool   `json:"enabled"`
	// Mod is nil when the jar has no metadata this package can read.
	Mod *Mod `json:"mod,omitempty"`
}

// Name is what to call the mod: its own name, or its file name.
func (im InstalledMod) Name() string {
	if im.Mod != nil {
		return im.Mod.Name
	}
	return strings.TrimSuffix(strings.TrimSuffix(im.File, disabledSuffix), ".jar")
}

// isModFile reports whether a file name is a mod, on or off.
func isModFile(name string) (enabled, ok bool) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(name, "."):
		return false, false
	case strings.HasSuffix(lower, ".jar"):
		return true, true
	case strings.HasSuffix(lower, ".jar"+disabledSuffix):
		return false, true
	}
	return false, false
}

// metadataCache remembers what each jar says, keyed by path, size and time,
// so listing a large mods folder again is cheap.
var metadataCache sync.Map

type cacheKey struct {
	path string
	size int64
	mod  time.Time
}

func cachedMod(path string, info os.FileInfo) *Mod {
	key := cacheKey{path, info.Size(), info.ModTime()}
	if m, ok := metadataCache.Load(key); ok {
		return m.(*Mod)
	}
	m, err := ReadMod(path)
	if err != nil {
		m = nil
	}
	metadataCache.Store(key, m)
	return m
}

// ListMods lists the mods in a folder, on and off, by name.
func ListMods(modsDir string) []InstalledMod {
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return nil
	}
	var out []InstalledMod
	for _, e := range entries {
		enabled, ok := isModFile(e.Name())
		if !ok || e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, InstalledMod{
			File:    e.Name(),
			Enabled: enabled,
			Mod:     cachedMod(filepath.Join(modsDir, e.Name()), info),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name()) < strings.ToLower(out[j].Name())
	})
	return out
}

// InstallResult is what Install did.
type InstallResult struct {
	File string `json:"file"` // the mod's file in the mods folder
	// TurnedOff are other copies of the same mod that were switched off, so
	// that only the new one loads.
	TurnedOff []string `json:"turnedOff,omitempty"`
}

// Install copies a mod jar into a mods folder, switched on. Another copy of
// the same mod already there — an older build, typically — is switched off
// rather than deleted.
func Install(src string, m *Mod, modsDir string) (*InstallResult, error) {
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		return nil, err
	}
	name := filepath.Base(src)
	name = strings.TrimSuffix(name, disabledSuffix)
	if !strings.HasSuffix(strings.ToLower(name), ".jar") {
		name += ".jar"
	}
	res := &InstallResult{File: name}

	for _, im := range ListMods(modsDir) {
		if im.File == name || im.File == name+disabledSuffix || !im.Enabled || im.Mod == nil || im.Mod.ID != m.ID {
			continue
		}
		if _, err := SetEnabled(modsDir, im.File, false); err != nil {
			return nil, fmt.Errorf("switching off the older %s: %w", im.File, err)
		}
		res.TurnedOff = append(res.TurnedOff, im.File)
	}

	// A switched-off copy with the same name is replaced by the new file.
	_ = os.Remove(filepath.Join(modsDir, name+disabledSuffix))
	if err := copyFile(src, filepath.Join(modsDir, name)); err != nil {
		return nil, err
	}
	return res, nil
}

// SetEnabled switches a mod on or off and returns its new file name.
func SetEnabled(modsDir, file string, on bool) (string, error) {
	if file != filepath.Base(file) || file == "" {
		return "", errors.New("not a file in the mods folder")
	}
	enabled, ok := isModFile(file)
	if !ok {
		return "", fmt.Errorf("%s is not a mod", file)
	}
	if enabled == on {
		return file, nil
	}
	target := file + disabledSuffix
	if on {
		target = file[:len(file)-len(disabledSuffix)]
	}
	if _, err := os.Stat(filepath.Join(modsDir, target)); err == nil {
		return "", fmt.Errorf("%s is already there", target)
	}
	if err := os.Rename(filepath.Join(modsDir, file), filepath.Join(modsDir, target)); err != nil {
		return "", err
	}
	return target, nil
}

// copyFile copies through a temporary name a loader would never pick up, so
// a game starting mid-copy cannot load half a jar.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".ziptransform-*.part")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

// Fit is whether a mod can run in an instance.
type Fit struct {
	OK bool `json:"ok"`
	// Unsure means the instance's versions could not be read, so it might
	// or might not work.
	Unsure bool   `json:"unsure"`
	Reason string `json:"reason,omitempty"`
	// Missing lists required mods the instance does not have switched on.
	Missing []string `json:"missing,omitempty"`
}

// Check works out whether a mod fits an instance, and if not, why.
func Check(m *Mod, in Instance, installed []InstalledMod) Fit {
	if !in.Known {
		return Fit{OK: true, Unsure: true, Reason: "Zip Transform cannot tell which Minecraft this is. Make sure it is " + m.Needs() + "."}
	}
	if in.Loader == "" {
		return Fit{Reason: "This is plain Minecraft, with no mod loader. The mod needs " + m.Needs() + "."}
	}

	// Quilt also loads Fabric mods.
	loader := ""
	for _, l := range m.Loaders {
		if l == in.Loader || (l == Fabric && in.Loader == Quilt && loader == "") {
			loader = l
		}
	}
	if loader == "" {
		return Fit{Reason: "This is " + LoaderName(in.Loader) + ". The mod needs " + m.Needs() + "."}
	}

	fit := Fit{OK: true}
	if in.Minecraft == "" {
		fit.Unsure = true
		fit.Reason = "Zip Transform cannot tell which Minecraft version this is."
	} else if versions := m.Minecraft[loader]; versions != "" {
		ok, understood := Accepts(loader, versions, in.Minecraft)
		switch {
		case !understood:
			fit.Unsure = true
			fit.Reason = "The mod asks for Minecraft " + versions + ", which Zip Transform cannot compare with " + in.Minecraft + "."
		case !ok:
			return Fit{Reason: "This is Minecraft " + in.Minecraft + ". The mod needs Minecraft " + DescribeRange(versions) + "."}
		}
	}

	have := map[string]bool{}
	for _, im := range installed {
		if im.Enabled && im.Mod != nil {
			have[im.Mod.ID] = true
		}
	}
	for _, d := range m.Requires {
		if !have[d.ID] {
			fit.Missing = append(fit.Missing, d.ID)
		}
	}
	return fit
}
