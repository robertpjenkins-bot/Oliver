// Package design is the saved form of an OS designed in OS Maker: what the
// desktop looks like, which apps it ships, and the system settings.
package design

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

// The designer's canvas size. Items are positioned on this canvas and scaled
// to whatever screen the finished OS boots on.
const (
	CanvasW = 1280
	CanvasH = 720
)

// FileExt is the extension of saved designs.
const FileExt = ".osmaker"

// Design is a whole OS.
type Design struct {
	Version int      `json:"version"`
	System  System   `json:"system"`
	Desktop Desktop  `json:"desktop"`
	Taskbar Taskbar  `json:"taskbar"`
	Apps    []string `json:"apps"`
	Items   []Item   `json:"items"`
}

// System holds the settings that are not about looks.
type System struct {
	OSName       string `json:"osName"`
	Hostname     string `json:"hostname"`
	Username     string `json:"username"`
	FullName     string `json:"fullName"`
	PasswordHash string `json:"passwordHash"`
	Autologin    bool   `json:"autologin"`
	Timezone     string `json:"timezone"`
	Keyboard     string `json:"keyboard"`
}

// Desktop is the background and colour theme.
type Desktop struct {
	Theme      string     `json:"theme"` // "dark" or "light"
	Accent     string     `json:"accent"`
	Background Background `json:"background"`
}

// Background is a solid colour, a gradient, or a picture.
type Background struct {
	Kind   string `json:"kind"` // "color", "gradient" or "image"
	Color1 string `json:"color1"`
	Color2 string `json:"color2"`
	Angle  int    `json:"angle"`
	// Image is a data: URL so a design is a single self-contained file.
	Image string `json:"image,omitempty"`
}

// Taskbar is the panel along the top or bottom edge.
type Taskbar struct {
	Position  string   `json:"position"` // "top" or "bottom"
	Size      int      `json:"size"`
	Color     string   `json:"color"`
	Opacity   float64  `json:"opacity"`
	MenuLabel string   `json:"menuLabel"`
	ShowClock bool     `json:"showClock"`
	Pinned    []string `json:"pinned"`
}

// Item is one thing placed on the desktop.
type Item struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	X         int     `json:"x"`
	Y         int     `json:"y"`
	W         int     `json:"w"`
	H         int     `json:"h"`
	App       string  `json:"app,omitempty"`
	Text      string  `json:"text,omitempty"`
	Color     string  `json:"color"`
	TextColor string  `json:"textColor"`
	Opacity   float64 `json:"opacity"`
	Radius    int     `json:"radius"`
	FontSize  int     `json:"fontSize"`
	Format    string  `json:"format,omitempty"` // clock: "24h" or "12h"
}

// ItemTypes are the kinds of thing that can be placed on the desktop.
var ItemTypes = []string{"app", "search", "clock", "calendar", "sysmon", "note", "text"}

// Default returns the design a new project starts from.
func Default() *Design {
	return &Design{
		Version: 1,
		System: System{
			OSName:    "My OS",
			Hostname:  "my-os",
			Username:  "user",
			FullName:  "User",
			Autologin: true,
			Timezone:  "UTC",
			Keyboard:  "us",
		},
		Desktop: Desktop{
			Theme:  "dark",
			Accent: "#4f8cff",
			Background: Background{
				Kind:   "gradient",
				Color1: "#1e3c72",
				Color2: "#7b2ff7",
				Angle:  135,
			},
		},
		Taskbar: Taskbar{
			Position:  "bottom",
			Size:      40,
			Color:     "#111827",
			Opacity:   0.9,
			MenuLabel: "Start",
			ShowClock: true,
			Pinned:    []string{"files", "terminal", "firefox"},
		},
		Apps: []string{"firefox"},
		Items: []Item{
			{ID: "search1", Type: "search", X: 390, Y: 120, W: 500, H: 48, Color: "#ffffff", TextColor: "#111827", Opacity: 0.95, Radius: 24, FontSize: 18},
			{ID: "clock1", Type: "clock", X: 1010, Y: 30, W: 240, H: 110, Color: "#000000", TextColor: "#ffffff", Opacity: 0.35, Radius: 16, FontSize: 40, Format: "24h"},
			{ID: "app1", Type: "app", X: 30, Y: 30, W: 90, H: 90, App: "files", TextColor: "#ffffff", Color: "#000000", Radius: 12, FontSize: 13},
			{ID: "app2", Type: "app", X: 30, Y: 140, W: 90, H: 90, App: "firefox", TextColor: "#ffffff", Color: "#000000", Radius: 12, FontSize: 13},
			{ID: "app3", Type: "app", X: 30, Y: 250, W: 90, H: 90, App: "terminal", TextColor: "#ffffff", Color: "#000000", Radius: 12, FontSize: 13},
		},
	}
}

// Load reads a saved design.
func Load(path string) (*Design, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse decodes and validates a design.
func Parse(b []byte) (*Design, error) {
	var d Design
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("not an OS Maker design: %w", err)
	}
	d.Normalize()
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return &d, nil
}

// Save writes the design to path.
func (d *Design) Save(path string) error {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// Normalize fills in defaults and makes sure every app that appears on the
// desktop or the taskbar is also installed.
func (d *Design) Normalize() {
	if d.Version == 0 {
		d.Version = 1
	}
	def := Default()
	if d.System.Username == "" {
		d.System.Username = def.System.Username
	}
	if d.System.Hostname == "" {
		d.System.Hostname = def.System.Hostname
	}
	if d.System.OSName == "" {
		d.System.OSName = def.System.OSName
	}
	if d.System.Timezone == "" {
		d.System.Timezone = "UTC"
	}
	if d.System.Keyboard == "" {
		d.System.Keyboard = "us"
	}
	if d.Desktop.Theme == "" {
		d.Desktop.Theme = "dark"
	}
	if d.Desktop.Accent == "" {
		d.Desktop.Accent = def.Desktop.Accent
	}
	if d.Desktop.Background.Kind == "" {
		d.Desktop.Background = def.Desktop.Background
	}
	if d.Desktop.Background.Color2 == "" {
		d.Desktop.Background.Color2 = d.Desktop.Background.Color1
	}
	if d.Taskbar.Position == "" {
		d.Taskbar.Position = "bottom"
	}
	if d.Taskbar.Size == 0 {
		d.Taskbar.Size = 40
	}
	if d.Taskbar.Color == "" {
		d.Taskbar.Color = def.Taskbar.Color
	}
	for i := range d.Items {
		it := &d.Items[i]
		if it.Color == "" {
			it.Color = "#000000"
		}
		if it.TextColor == "" {
			it.TextColor = "#ffffff"
		}
		if it.FontSize == 0 {
			it.FontSize = 14
		}
		if it.Type == "app" && it.App != "" {
			d.addApp(it.App)
		}
	}
	for _, p := range d.Taskbar.Pinned {
		d.addApp(p)
	}
	// Base apps are always present, so they need no entry.
	d.Apps = slices.DeleteFunc(d.Apps, func(id string) bool {
		a, ok := AppByID(id)
		return ok && a.Base
	})
}

func (d *Design) addApp(id string) {
	a, ok := AppByID(id)
	if !ok || a.Base || slices.Contains(d.Apps, id) {
		return
	}
	d.Apps = append(d.Apps, id)
}

var (
	reUser     = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	reHost     = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	reOSName   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._'!-]{0,31}$`)
	reTimezone = regexp.MustCompile(`^[A-Za-z0-9_+-]+(/[A-Za-z0-9_+-]+)*$`)
	reColor    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	reID       = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)
	reHash     = regexp.MustCompile(`^\$6\$[./A-Za-z0-9]{1,16}\$[./A-Za-z0-9]{86}$`)
	reFullName = regexp.MustCompile(`^[^:\n\r,]{0,64}$`)
)

// reserved names already used by system accounts on Arch.
var reservedUsers = []string{"root", "bin", "daemon", "mail", "ftp", "http", "nobody", "dbus", "polkitd", "lightdm", "systemd-network", "systemd-resolve", "systemd-timesync", "systemd-coredump", "uuidd", "git", "rtkit"}

// Validate reports the first thing wrong with the design, in words the
// person who made it can act on.
func (d *Design) Validate() error {
	var errs []error
	bad := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	s := d.System
	if !reOSName.MatchString(s.OSName) {
		bad("OS name %q: use 1-32 letters, numbers, spaces and . _ ' ! -", s.OSName)
	}
	if !reHost.MatchString(s.Hostname) {
		bad("computer name %q: use lowercase letters, numbers and dashes", s.Hostname)
	}
	if !reUser.MatchString(s.Username) || slices.Contains(reservedUsers, s.Username) {
		bad("username %q: use lowercase letters and numbers, starting with a letter (and not a system name like root)", s.Username)
	}
	if !reFullName.MatchString(s.FullName) {
		bad("full name: no colons, commas or line breaks")
	}
	if s.PasswordHash != "" && !reHash.MatchString(s.PasswordHash) {
		bad("password is damaged; set it again")
	}
	if s.PasswordHash == "" && !s.Autologin {
		bad("set a password, or turn on automatic login, or nobody can log in")
	}
	if !reTimezone.MatchString(s.Timezone) || strings.Contains(s.Timezone, "..") {
		bad("time zone %q is not valid", s.Timezone)
	}
	if _, ok := KeyboardByID(s.Keyboard); !ok {
		bad("keyboard layout %q is not one OS Maker knows", s.Keyboard)
	}

	if d.Desktop.Theme != "dark" && d.Desktop.Theme != "light" {
		bad("theme must be dark or light")
	}
	if !reColor.MatchString(d.Desktop.Accent) {
		bad("accent colour %q is not a colour", d.Desktop.Accent)
	}
	bg := d.Desktop.Background
	switch bg.Kind {
	case "color", "gradient":
		if !reColor.MatchString(bg.Color1) || !reColor.MatchString(bg.Color2) {
			bad("background colours must be colours like #1e3c72")
		}
	case "image":
		if _, _, err := DecodeImage(bg.Image); err != nil {
			bad("wallpaper picture: %v", err)
		}
	default:
		bad("background must be a colour, gradient or picture")
	}

	t := d.Taskbar
	if t.Position != "top" && t.Position != "bottom" {
		bad("taskbar must be at the top or bottom")
	}
	if t.Size < 24 || t.Size > 72 {
		bad("taskbar height must be between 24 and 72")
	}
	if !reColor.MatchString(t.Color) {
		bad("taskbar colour %q is not a colour", t.Color)
	}
	if t.Opacity < 0 || t.Opacity > 1 {
		bad("taskbar see-through amount must be between 0 and 1")
	}
	if len([]rune(t.MenuLabel)) > 16 || strings.ContainsAny(t.MenuLabel, "\n\r\t") {
		bad("start button text: up to 16 characters on one line")
	}
	if len(t.Pinned) > 12 {
		bad("you can pin up to 12 apps to the taskbar")
	}
	for _, p := range t.Pinned {
		if _, ok := AppByID(p); !ok {
			bad("pinned app %q is not in the catalog", p)
		}
	}

	for _, id := range d.Apps {
		if _, ok := AppByID(id); !ok {
			bad("app %q is not in the catalog", id)
		}
	}

	if len(d.Items) > 200 {
		bad("too many things on the desktop (200 is the limit)")
	}
	seen := map[string]bool{}
	for _, it := range d.Items {
		name := it.Type
		if !reID.MatchString(it.ID) || seen[it.ID] {
			bad("%s has a bad or repeated id %q", name, it.ID)
		}
		seen[it.ID] = true
		if !slices.Contains(ItemTypes, it.Type) {
			bad("unknown desktop item %q", it.Type)
			continue
		}
		if it.W < 16 || it.H < 16 || it.X < 0 || it.Y < 0 || it.X+it.W > CanvasW || it.Y+it.H > CanvasH {
			bad("%s %q does not fit on the screen", name, it.ID)
		}
		if !reColor.MatchString(it.Color) || !reColor.MatchString(it.TextColor) {
			bad("%s %q has a colour that is not a colour", name, it.ID)
		}
		if it.Opacity < 0 || it.Opacity > 1 {
			bad("%s %q: see-through amount must be between 0 and 1", name, it.ID)
		}
		if it.Radius < 0 || it.Radius > 200 || it.FontSize < 6 || it.FontSize > 120 {
			bad("%s %q: corner or text size out of range", name, it.ID)
		}
		if len(it.Text) > 2000 {
			bad("%s %q: text is too long", name, it.ID)
		}
		if it.Type == "app" {
			if _, ok := AppByID(it.App); !ok {
				bad("app icon %q points at unknown app %q", it.ID, it.App)
			}
		}
		if it.Type == "clock" && it.Format != "" && it.Format != "24h" && it.Format != "12h" {
			bad("clock %q: format must be 24h or 12h", it.ID)
		}
	}
	return errors.Join(errs...)
}

// Packages returns the Arch packages the selected apps need, in catalog order.
func (d *Design) Packages() []string {
	var out []string
	for _, a := range Catalog {
		if a.Base || slices.Contains(d.Apps, a.ID) {
			for _, p := range a.Packages {
				if !slices.Contains(out, p) {
					out = append(out, p)
				}
			}
		}
	}
	return out
}

// NeedsMultilib reports whether any selected app needs 32-bit packages.
func (d *Design) NeedsMultilib() bool {
	for _, id := range d.Apps {
		if a, ok := AppByID(id); ok && a.Multilib {
			return true
		}
	}
	return false
}

// maxImage bounds the wallpaper so a design file stays a sensible size.
const maxImage = 15 << 20

// DecodeImage unpacks a wallpaper data: URL into bytes and a file extension.
func DecodeImage(dataURL string) ([]byte, string, error) {
	rest, ok := strings.CutPrefix(dataURL, "data:")
	if !ok {
		return nil, "", errors.New("no picture chosen")
	}
	mime, payload, ok := strings.Cut(rest, ";base64,")
	if !ok {
		return nil, "", errors.New("picture is not base64 encoded")
	}
	ext := map[string]string{"image/png": "png", "image/jpeg": "jpg", "image/webp": "webp"}[mime]
	if ext == "" {
		return nil, "", fmt.Errorf("pictures must be PNG, JPEG or WebP, not %s", mime)
	}
	if base64.StdEncoding.DecodedLen(len(payload)) > maxImage {
		return nil, "", errors.New("picture is bigger than 15 MB")
	}
	b, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", fmt.Errorf("picture is damaged: %w", err)
	}
	return b, ext, nil
}
