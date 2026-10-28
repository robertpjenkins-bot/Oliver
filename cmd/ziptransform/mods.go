package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/minecraft"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

// The second half of Zip Transform: a mod jar, built here or dropped onto
// the window, goes into the mods folder of one of the Minecraft instances on
// this computer, switched on.

// isModPath reports whether a file is a mod jar, on or off.
func isModPath(p string) bool {
	lower := strings.ToLower(p)
	return strings.HasSuffix(lower, ".jar") || strings.HasSuffix(lower, ".jar.disabled")
}

// modFile is a mod jar the window knows about.
type modFile struct {
	id   string
	path string
	mod  *minecraft.Mod
}

// modInfo is what the window shows about a mod jar.
type modInfo struct {
	ID       string `json:"id"`
	File     string `json:"file"`
	Name     string `json:"name"`
	ModID    string `json:"modid"`
	Version  string `json:"version"`
	Needs    string `json:"needs"`
	Requires string `json:"requires,omitempty"`
}

func describeMod(mf *modFile) modInfo {
	var requires []string
	for _, d := range mf.mod.Requires {
		requires = append(requires, minecraft.ModName(d.ID))
	}
	return modInfo{
		ID:       mf.id,
		File:     filepath.Base(mf.path),
		Name:     mf.mod.Name,
		ModID:    mf.mod.ID,
		Version:  mf.mod.Version,
		Needs:    mf.mod.Needs(),
		Requires: strings.Join(requires, ", "),
	}
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *server) addMod(path string) (*modFile, error) {
	m, err := minecraft.ReadMod(path)
	if err != nil {
		return nil, err
	}
	mf := &modFile{id: randomID(), path: path, mod: m}
	s.mu.Lock()
	s.modFiles[mf.id] = mf
	s.mu.Unlock()
	return mf, nil
}

func (s *server) modFile(id string) *modFile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.modFiles[id]
}

// openModPath loads a jar given on the command line.
func (s *server) openModPath(path string) (*modFile, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	mf, err := s.addMod(abs)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.preloadedMod = mf.id
	s.mu.Unlock()
	return mf, nil
}

func (s *server) handleModOpen(w http.ResponseWriter, r *http.Request) {
	dest, ok := s.receiveUpload(w, r, "mod.jar")
	if !ok {
		return
	}
	mf, err := s.addMod(dest)
	if err != nil {
		os.RemoveAll(filepath.Dir(dest))
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, describeMod(mf))
}

// handleModFromBuild hands the jar a build just made to the mods side.
func (s *server) handleModFromBuild(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	jar := ""
	if b := s.buildOf(req.ID); b != nil {
		b.mu.Lock()
		jar = b.jar
		b.mu.Unlock()
	}
	if jar == "" {
		writeError(w, http.StatusNotFound, errors.New("that build has no jar"))
		return
	}
	mf, err := s.addMod(jar)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, describeMod(mf))
}

// instanceView is an instance as the window lists it.
type instanceView struct {
	minecraft.Instance
	Label string          `json:"label"`
	Mods  []installedView `json:"mods"`
	// Fit is set when the window asks about a particular mod.
	Fit          *minecraft.Fit `json:"fit,omitempty"`
	MissingNames []string       `json:"missingNames,omitempty"`
}

type installedView struct {
	File    string `json:"file"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Enabled bool   `json:"enabled"`
}

// instanceViews lists the instances, and for a mod, how well it fits each,
// best first.
func instanceViews(m *minecraft.Mod) []instanceView {
	views := []instanceView{}
	for _, in := range minecraft.FindInstances(loadConfig().ModsFolders...) {
		installed := minecraft.ListMods(in.ModsDir)
		v := instanceView{Instance: in, Label: in.Label(), Mods: []installedView{}}
		for _, im := range installed {
			iv := installedView{File: im.File, Name: im.Name(), Enabled: im.Enabled}
			if im.Mod != nil {
				iv.Version = im.Mod.Version
			}
			v.Mods = append(v.Mods, iv)
		}
		if m != nil {
			fit := minecraft.Check(m, in, installed)
			v.Fit = &fit
			for _, id := range fit.Missing {
				v.MissingNames = append(v.MissingNames, minecraft.ModName(id))
			}
		}
		views = append(views, v)
	}
	if m != nil {
		sort.SliceStable(views, func(i, j int) bool { return fitRank(views[i].Fit) < fitRank(views[j].Fit) })
	}
	return views
}

func fitRank(f *minecraft.Fit) int {
	switch {
	case f.OK && !f.Unsure:
		return 0
	case f.OK:
		return 1
	}
	return 2
}

func (s *server) handleInstances(w http.ResponseWriter, r *http.Request) {
	var m *minecraft.Mod
	if mf := s.modFile(r.URL.Query().Get("mod")); mf != nil {
		m = mf.mod
	}
	writeJSON(w, map[string]any{"instances": instanceViews(m)})
}

func findInstance(id string) (minecraft.Instance, bool) {
	for _, in := range minecraft.FindInstances(loadConfig().ModsFolders...) {
		if in.ID == id {
			return in, true
		}
	}
	return minecraft.Instance{}, false
}

var errNoInstance = errors.New("that Minecraft instance is no longer there")

func (s *server) handleModInstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mod      string `json:"mod"`
		Instance string `json:"instance"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	mf := s.modFile(req.Mod)
	if mf == nil {
		writeError(w, http.StatusNotFound, errors.New("that mod is no longer open; drop it again"))
		return
	}
	in, ok := findInstance(req.Instance)
	if !ok {
		writeError(w, http.StatusNotFound, errNoInstance)
		return
	}
	res, err := minecraft.Install(mf.path, mf.mod, in.ModsDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"file": res.File, "turnedOff": res.TurnedOff, "instance": in.Name, "launcher": in.Launcher})
}

func (s *server) handleModToggle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Instance string `json:"instance"`
		File     string `json:"file"`
		On       bool   `json:"on"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	in, ok := findInstance(req.Instance)
	if !ok {
		writeError(w, http.StatusNotFound, errNoInstance)
		return
	}
	file, err := minecraft.SetEnabled(in.ModsDir, req.File, req.On)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, map[string]string{"file": file})
}

// handleModFolder adds a mods folder Zip Transform did not find by itself.
func (s *server) handleModFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir string `json:"dir"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := rememberModsFolder(strings.TrimSpace(req.Dir)); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, map[string]any{"instances": instanceViews(nil)})
}

// handleModShow opens an instance's mods folder in the file manager.
func (s *server) handleModShow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Instance string `json:"instance"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	in, ok := findInstance(req.Instance)
	if !ok {
		writeError(w, http.StatusNotFound, errNoInstance)
		return
	}
	if err := os.MkdirAll(in.ModsDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !openFolder(in.ModsDir) {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("could not open a file manager; the folder is %s", in.ModsDir))
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func openFolder(dir string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if cmd.Start() != nil {
		return false
	}
	go cmd.Wait()
	return true
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

const instancesSynopsis = `ziptransform instances

Lists the Minecraft instances on this computer that mods can go into: those
of Prism Launcher, the official launcher, ATLauncher, Modrinth App and
CurseForge, and any mods folder added with -folder.`

func cmdInstances(u *ui.UI, args []string) error {
	fs := newFlagSet(u, "instances", instancesSynopsis)
	folder := fs.String("folder", "", "also use this mods `folder` from now on")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *folder != "" {
		if err := rememberModsFolder(*folder); err != nil {
			return err
		}
	}
	views := instanceViews(nil)
	if len(views) == 0 {
		u.Printf("No Minecraft was found. Install Prism Launcher or the Minecraft Launcher and make an instance,")
		u.Printf("or name a mods folder: ziptransform instances -folder ~/path/to/mods")
		return nil
	}
	t := u.NewTable("NAME", "LAUNCHER", "RUNS", "MODS")
	for _, v := range views {
		on := 0
		for _, m := range v.Mods {
			if m.Enabled {
				on++
			}
		}
		count := strconv.Itoa(on) + " on"
		if off := len(v.Mods) - on; off > 0 {
			count += ", " + strconv.Itoa(off) + " off"
		}
		t.Row(v.Name, v.Launcher, v.Label, count)
	}
	t.Render()
	return nil
}

func rememberModsFolder(dir string) error {
	dir = expandHome(dir)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return fmt.Errorf("%s is not a folder", dir)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	cfg := loadConfig()
	for _, d := range cfg.ModsFolders {
		if d == abs {
			return nil
		}
	}
	cfg.ModsFolders = append(cfg.ModsFolders, abs)
	return cfg.save()
}

const installSynopsis = `ziptransform install [-instance NAME] <mod.jar>

Puts a mod into the mods folder of a Minecraft instance, switched on. Without
-instance, the one instance that can run it is used, or you are asked which.
An older copy of the same mod is switched off rather than deleted.`

func cmdInstall(u *ui.UI, args []string) error {
	fs := newFlagSet(u, "install", installSynopsis)
	name := fs.String("instance", "", "the instance to add the mod to, by `name`")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return flag.ErrHelp
	}
	m, err := minecraft.ReadMod(fs.Arg(0))
	if err != nil {
		return err
	}
	return installMod(u, fs.Arg(0), m, *name)
}

// installMod adds a mod to the instance named, or the one that fits.
func installMod(u *ui.UI, path string, m *minecraft.Mod, name string) error {
	views := instanceViews(m)
	var chosen *instanceView
	switch {
	case name != "":
		for i, v := range views {
			if strings.EqualFold(v.Name, name) || v.ID == name {
				if chosen != nil {
					return fmt.Errorf("more than one instance is called %q", name)
				}
				chosen = &views[i]
			}
		}
		if chosen == nil {
			return fmt.Errorf("no instance is called %q (see `ziptransform instances`)", name)
		}
	default:
		var fitting []int
		for i, v := range views {
			if v.Fit.OK && !v.Fit.Unsure {
				fitting = append(fitting, i)
			}
		}
		switch {
		case len(fitting) == 1:
			chosen = &views[fitting[0]]
		case len(fitting) == 0:
			return fmt.Errorf("no Minecraft instance on this computer can run %s, which needs %s. Make an instance with that in your launcher, or name one with -instance", m.Name, m.Needs())
		case !u.Interactive:
			return fmt.Errorf("%d instances can run %s; choose one with -instance", len(fitting), m.Name)
		default:
			fmt.Fprintf(u.Err, "Which Minecraft should %s go into?\n", m.Name)
			for n, i := range fitting {
				fmt.Fprintf(u.Err, "  %d) %s (%s, %s)\n", n+1, views[i].Name, views[i].Launcher, views[i].Label)
			}
			for chosen == nil {
				fmt.Fprint(u.Err, "Choose a number: ")
				var answer string
				if _, err := fmt.Fscanln(os.Stdin, &answer); err != nil {
					return errors.New("no instance chosen")
				}
				if n, err := strconv.Atoi(strings.TrimSpace(answer)); err == nil && n >= 1 && n <= len(fitting) {
					chosen = &views[fitting[n-1]]
				}
			}
		}
	}

	if !chosen.Fit.OK {
		return fmt.Errorf("%s cannot run in %q: %s", m.Name, chosen.Name, chosen.Fit.Reason)
	}
	if chosen.Fit.Unsure {
		u.Warn("%s", chosen.Fit.Reason)
	}
	res, err := minecraft.Install(path, m, chosen.ModsDir)
	if err != nil {
		return err
	}
	u.Printf("%s %s added to %q (%s), switched on", u.Green("✓"), m.Name, chosen.Name, chosen.Launcher)
	for _, f := range res.TurnedOff {
		u.Info("  switched off the older %s", f)
	}
	if len(chosen.MissingNames) > 0 {
		u.Warn("it also needs %s, which this instance does not have", strings.Join(chosen.MissingNames, ", "))
	}
	u.Info("  If Minecraft is open, close it and start %q again.", chosen.Name)
	return nil
}
