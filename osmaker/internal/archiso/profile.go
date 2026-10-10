// Package archiso turns a design into a folder that builds a bootable Arch
// Linux ISO: the apps to install, the files to lay over the live system, and
// a build.sh that hands it all to archiso's mkarchiso.
package archiso

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/robertpjenkins-bot/oliver/osmaker/internal/design"
)

//go:embed assets/build.sh
var buildScript string

//go:embed assets/osmaker-desktop.py
var desktopShell []byte

//go:embed assets/osmaker-install
var installer []byte

//go:embed assets/osmaker-install-backend
var installerBackend []byte

// BasePackages are installed in every OS on top of archiso's releng set:
// the Xfce desktop, a login manager, sound, networking, fonts and icons, and
// what the OS Maker desktop shell needs to run.
var BasePackages = []string{
	"xorg-server", "xorg-xrandr", "mesa",
	"xfce4", "xfce4-whiskermenu-plugin", "xfce4-pulseaudio-plugin",
	"lightdm", "lightdm-gtk-greeter",
	"networkmanager", "network-manager-applet",
	"pipewire", "pipewire-pulse", "wireplumber", "pavucontrol",
	"python", "python-gobject", "python-cairo", "gtk3",
	"papirus-icon-theme", "gnome-themes-extra",
	"noto-fonts", "noto-fonts-emoji", "ttf-dejavu",
	"gvfs", "xdg-utils", "xdg-user-dirs", "sudo",
	// For installing from the live system onto a disk.
	"grub", "efibootmgr", "dosfstools", "e2fsprogs", "rsync", "arch-install-scripts",
}

// File is one file of the generated build folder.
type File struct {
	Data []byte
	Mode os.FileMode
	// Link, when set, makes this a symbolic link to Link instead of a file.
	Link string
}

// Files returns every file of the build folder for d, keyed by path relative
// to the folder. d must already be valid.
func Files(d *design.Design) (map[string]File, error) {
	f := map[string]File{}
	put := func(path, data string, mode os.FileMode) { f[path] = File{Data: []byte(data), Mode: mode} }
	link := func(path, target string) { f[path] = File{Link: target} }
	root := func(path, data string) { put("airootfs/"+path, data, 0o644) }

	s := d.System
	user := s.Username
	home := "home/" + user

	// build.sh and what it reads.
	script := strings.NewReplacer(
		"@OS_NAME@", shellQuote(s.OSName),
		"@ISO_NAME@", shellQuote(s.Hostname),
		"@MULTILIB@", map[bool]string{true: "1", false: "0"}[d.NeedsMultilib()],
	).Replace(buildScript)
	put("build.sh", script, 0o755)

	pkgs := append(slices.Clone(BasePackages), d.Packages()...)
	put("packages.txt", "# Installed on top of archiso's releng package list.\n"+strings.Join(pkgs, "\n")+"\n", 0o644)

	put("permissions.sh", fmt.Sprintf(`
# Added by OS Maker 2.0
file_permissions+=(
  ["/etc/shadow"]="0:0:400"
  ["/etc/gshadow"]="0:0:400"
  ["/etc/sudoers.d/10-osmaker"]="0:0:440"
  ["/etc/sudoers.d/20-osmaker-live"]="0:0:440"
  ["/usr/local/bin/osmaker-desktop"]="0:0:755"
  ["/usr/local/bin/osmaker-install"]="0:0:755"
  ["/usr/local/bin/osmaker-install-backend"]="0:0:755"
  ["/home/%s"]="1000:1000:750"
)
`, user), 0o644)

	saved, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	put("design"+design.FileExt, string(saved)+"\n", 0o644)
	put("README.txt", readme(d), 0o644)

	// Accounts. archiso uses these files as they are, so they list root too.
	full := s.FullName
	root("etc/hostname", s.Hostname+"\n")
	root("etc/passwd", fmt.Sprintf("root:x:0:0:root:/root:/usr/bin/zsh\n%s:x:1000:1000:%s:/home/%s:/bin/bash\n", user, full, user))
	hash := s.PasswordHash
	if hash == "" {
		// No password: the only way in is automatic login.
		hash = "!*"
	}
	root("etc/shadow", fmt.Sprintf("root:!*:14871::::::\n%s:%s:14871::::::\n", user, hash))
	root("etc/group", fmt.Sprintf("root:x:0:root\nadm:x:999:%[1]s\nwheel:x:998:%[1]s\nautologin:x:1001:%[1]s\n%[1]s:x:1000:\n", user))
	root("etc/gshadow", fmt.Sprintf("root:!*::root\nadm:!*::%[1]s\nwheel:!*::%[1]s\nautologin:!*::%[1]s\n%[1]s:!*::\n", user))
	root("etc/sudoers.d/10-osmaker", "%wheel ALL=(ALL:ALL) ALL\n")
	// On the live USB the user may run the installer without a password,
	// since a design need not have one. The installer deletes this rule from
	// the system it installs.
	root("etc/sudoers.d/20-osmaker-live", "%wheel ALL=(root) NOPASSWD: /usr/local/bin/osmaker-install-backend\n")

	// Time, keyboard.
	link("airootfs/etc/localtime", "/usr/share/zoneinfo/"+s.Timezone)
	kb, _ := design.KeyboardByID(s.Keyboard)
	root("etc/vconsole.conf", "KEYMAP="+kb.Console+"\n")
	root("etc/X11/xorg.conf.d/00-keyboard.conf", fmt.Sprintf(`Section "InputClass"
    Identifier "system-keyboard"
    MatchIsKeyboard "on"
    Option "XkbLayout" "%s"
EndSection
`, kb.ID))

	// Boot into the desktop.
	link("airootfs/etc/systemd/system/default.target", "/usr/lib/systemd/system/graphical.target")
	link("airootfs/etc/systemd/system/display-manager.service", "/usr/lib/systemd/system/lightdm.service")
	link("airootfs/etc/systemd/system/multi-user.target.wants/NetworkManager.service", "/usr/lib/systemd/system/NetworkManager.service")
	link("airootfs/etc/systemd/system/multi-user.target.wants/osmaker-home.service", "/etc/systemd/system/osmaker-home.service")
	root("etc/systemd/system/osmaker-home.service", fmt.Sprintf(`[Unit]
Description=Give %[1]s their home folder
Before=display-manager.service

[Service]
Type=oneshot
ExecStart=/usr/bin/chown -R %[1]s:%[1]s /home/%[1]s

[Install]
WantedBy=multi-user.target
`, user))

	for _, svc := range d.Services() {
		link("airootfs/etc/systemd/system/multi-user.target.wants/"+svc, "/usr/lib/systemd/system/"+svc)
	}

	if slices.Contains(d.Apps, "store") {
		// The App Store finds apps on Flathub; add it once there is internet.
		link("airootfs/etc/systemd/system/multi-user.target.wants/osmaker-flathub.service", "/etc/systemd/system/osmaker-flathub.service")
		root("etc/systemd/system/osmaker-flathub.service", `[Unit]
Description=Add Flathub to the App Store
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/bin/flatpak remote-add --system --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo
Restart=on-failure
RestartSec=30

[Install]
WantedBy=multi-user.target
`)
	}

	theme, icons := "Adwaita", "Papirus-Light"
	if d.Desktop.Theme == "dark" {
		theme, icons = "Adwaita-dark", "Papirus-Dark"
	}
	lightdm := "[LightDM]\n\n[Seat:*]\ngreeter-session=lightdm-gtk-greeter\nuser-session=xfce\n"
	if s.Autologin {
		lightdm += fmt.Sprintf("autologin-user=%s\nautologin-session=xfce\n", user)
	}
	root("etc/lightdm/lightdm.conf", lightdm)

	// Wallpaper and the desktop layout the shell draws.
	wallpaper := ""
	bg := d.Desktop.Background
	if bg.Kind == "image" {
		img, ext, err := design.DecodeImage(bg.Image)
		if err != nil {
			return nil, err
		}
		wallpaper = "/etc/osmaker/wallpaper." + ext
		f["airootfs"+wallpaper] = File{Data: img, Mode: 0o644}
	}
	greeterBG := wallpaper
	if greeterBG == "" {
		greeterBG = bg.Color1
	}
	root("etc/lightdm/lightdm-gtk-greeter.conf", fmt.Sprintf(
		"[greeter]\ntheme-name=%s\nicon-theme-name=%s\nbackground=%s\nfont-name=Noto Sans 11\n", theme, icons, greeterBG))

	layout, err := shellLayout(d, wallpaper)
	if err != nil {
		return nil, err
	}
	root("etc/osmaker/layout.json", string(layout))
	f["airootfs/usr/local/bin/osmaker-desktop"] = File{Data: desktopShell, Mode: 0o755}
	f["airootfs/usr/local/bin/osmaker-install"] = File{Data: installer, Mode: 0o755}
	f["airootfs/usr/local/bin/osmaker-install-backend"] = File{Data: installerBackend, Mode: 0o755}
	root("etc/osmaker/os-name", s.OSName+"\n")
	root("usr/share/applications/osmaker-install.desktop", fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Install %s
Comment=Put %[1]s on this computer's disk
Exec=/usr/local/bin/osmaker-install
Icon=system-software-install
Terminal=false
Categories=System;
`, s.OSName))

	// The user's own settings: Xfce panel, theme and session, and the shell.
	xfconf := home + "/.config/xfce4/xfconf/xfce-perchannel-xml/"
	root(xfconf+"xfce4-panel.xml", panelXML(d))
	root(xfconf+"xsettings.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<channel name="xsettings" version="1.0">
  <property name="Net" type="empty">
    <property name="ThemeName" type="string" value="%s"/>
    <property name="IconThemeName" type="string" value="%s"/>
  </property>
  <property name="Gtk" type="empty">
    <property name="FontName" type="string" value="Noto Sans 10"/>
  </property>
</channel>
`, theme, icons))
	root(xfconf+"xfwm4.xml", `<?xml version="1.0" encoding="UTF-8"?>
<channel name="xfwm4" version="1.0">
  <property name="general" type="empty">
    <property name="theme" type="string" value="Default"/>
    <property name="workspace_count" type="int" value="1"/>
  </property>
</channel>
`)
	root(xfconf+"xfce4-session.xml", sessionXML)
	root(home+"/.config/autostart/osmaker-desktop.desktop", `[Desktop Entry]
Type=Application
Name=OS Maker desktop
Exec=/usr/local/bin/osmaker-desktop
X-XFCE-Autostart-Override=true
`)
	for i, id := range d.Taskbar.Pinned {
		a, _ := design.AppByID(id)
		root(fmt.Sprintf("%s/.config/xfce4/panel/launcher-%d/%s.desktop", home, launcherID(i), a.ID), launcherDesktop(a))
	}
	return f, nil
}

// Write creates the build folder for d at dir.
func Write(d *design.Design, dir string) error {
	files, err := Files(d)
	if err != nil {
		return err
	}
	// Start clean so files from an earlier design cannot leak into this one.
	for _, sub := range []string{"airootfs", "work"} {
		if err := os.RemoveAll(filepath.Join(dir, sub)); err != nil {
			return fmt.Errorf("clearing old build files: %w (if an earlier build ran as root, delete %s by hand)", err, filepath.Join(dir, sub))
		}
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		file := files[p]
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if file.Link != "" {
			os.Remove(full)
			if err := os.Symlink(file.Link, full); err != nil {
				return err
			}
			continue
		}
		if err := os.WriteFile(full, file.Data, file.Mode); err != nil {
			return err
		}
		if err := os.Chmod(full, file.Mode); err != nil {
			return err
		}
	}
	return nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

type shellApp struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Desktop     string `json:"desktop"`
	Exec        string `json:"exec"`
	Icon        string `json:"icon"`
}

type shellItem struct {
	design.Item
	AppInfo *shellApp `json:"appInfo,omitempty"`
}

// shellLayout is the file the desktop shell reads at login.
func shellLayout(d *design.Design, wallpaper string) ([]byte, error) {
	bg := d.Desktop.Background
	items := make([]shellItem, 0, len(d.Items))
	for _, it := range d.Items {
		si := shellItem{Item: it}
		if it.Type == "app" {
			a, _ := design.AppByID(it.App)
			si.AppInfo = &shellApp{a.Name, a.Description, a.Desktop, a.Exec, a.Icon}
		}
		items = append(items, si)
	}
	return json.MarshalIndent(map[string]any{
		"canvas": map[string]int{"w": design.CanvasW, "h": design.CanvasH},
		"theme":  d.Desktop.Theme,
		"accent": d.Desktop.Accent,
		"osName": d.System.OSName,
		"background": map[string]any{
			"kind":   bg.Kind,
			"color1": bg.Color1,
			"color2": bg.Color2,
			"angle":  bg.Angle,
			"image":  wallpaper,
		},
		"items": items,
	}, "", "  ")
}

// Panel plugin ids: 1 is the start menu, launchers follow, then the rest.
func launcherID(i int) int { return 2 + i }

func panelXML(d *design.Design) string {
	t := d.Taskbar
	var ids []int
	var plugins strings.Builder
	add := func(kind, props string) {
		id := len(ids) + 1
		ids = append(ids, id)
		fmt.Fprintf(&plugins, "    <property name=\"plugin-%d\" type=\"string\" value=\"%s\">\n%s    </property>\n", id, kind, props)
	}

	label := xmlEscape(t.MenuLabel)
	add("whiskermenu", fmt.Sprintf(`      <property name="button-title" type="string" value="%s"/>
      <property name="show-button-title" type="bool" value="%t"/>
      <property name="show-button-icon" type="bool" value="true"/>
`, label, label != ""))
	for _, id := range t.Pinned {
		add("launcher", fmt.Sprintf(`      <property name="items" type="array">
        <value type="string" value="%s.desktop"/>
      </property>
`, id))
	}
	add("separator", `      <property name="style" type="uint" value="0"/>
`)
	add("tasklist", `      <property name="grouping" type="uint" value="1"/>
`)
	add("separator", `      <property name="expand" type="bool" value="true"/>
      <property name="style" type="uint" value="0"/>
`)
	add("systray", "")
	add("pulseaudio", "")
	if t.ShowClock {
		add("clock", `      <property name="digital-layout" type="uint" value="3"/>
`)
	}
	add("actions", `      <property name="appearance" type="uint" value="0"/>
`)

	var idList strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&idList, "        <value type=\"int\" value=\"%d\"/>\n", id)
	}
	r, g, b := hexRGB(t.Color)
	pos := "p=8;x=0;y=0"
	if t.Position == "top" {
		pos = "p=6;x=0;y=0"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<channel name="xfce4-panel" version="1.0">
  <property name="configver" type="int" value="2"/>
  <property name="panels" type="array">
    <value type="int" value="1"/>
    <property name="dark-mode" type="bool" value="%t"/>
    <property name="panel-1" type="empty">
      <property name="position" type="string" value="%s"/>
      <property name="length" type="uint" value="100"/>
      <property name="position-locked" type="bool" value="true"/>
      <property name="size" type="uint" value="%d"/>
      <property name="icon-size" type="uint" value="0"/>
      <property name="background-style" type="uint" value="1"/>
      <property name="background-rgba" type="array">
        <value type="double" value="%.4f"/>
        <value type="double" value="%.4f"/>
        <value type="double" value="%.4f"/>
        <value type="double" value="%.4f"/>
      </property>
      <property name="plugin-ids" type="array">
%s      </property>
    </property>
  </property>
  <property name="plugins" type="empty">
%s  </property>
</channel>
`, d.Desktop.Theme == "dark", pos, t.Size, r, g, b, t.Opacity, idList.String(), plugins.String())
}

func launcherDesktop(a design.App) string {
	exec := a.Exec
	if a.Desktop == "" {
		// A command-line program: run it in a terminal that stays open.
		exec = "xfce4-terminal --hold -e " + a.Exec
	}
	return fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nComment=%s\nExec=%s\nIcon=%s\nTerminal=false\n",
		a.Name, a.Description, exec, a.Icon)
}

// sessionXML is Xfce's standard session minus xfdesktop: the OS Maker shell
// draws the desktop instead.
const sessionXML = `<?xml version="1.0" encoding="UTF-8"?>
<channel name="xfce4-session" version="1.0">
  <property name="general" type="empty">
    <property name="FailsafeSessionName" type="string" value="Failsafe"/>
    <property name="SaveOnExit" type="bool" value="false"/>
  </property>
  <property name="sessions" type="empty">
    <property name="Failsafe" type="empty">
      <property name="IsFailsafe" type="bool" value="true"/>
      <property name="Count" type="int" value="4"/>
      <property name="Client0_Command" type="array">
        <value type="string" value="xfwm4"/>
      </property>
      <property name="Client0_Priority" type="int" value="15"/>
      <property name="Client0_PerScreen" type="bool" value="false"/>
      <property name="Client1_Command" type="array">
        <value type="string" value="xfsettingsd"/>
      </property>
      <property name="Client1_Priority" type="int" value="20"/>
      <property name="Client1_PerScreen" type="bool" value="false"/>
      <property name="Client2_Command" type="array">
        <value type="string" value="xfce4-panel"/>
      </property>
      <property name="Client2_Priority" type="int" value="25"/>
      <property name="Client2_PerScreen" type="bool" value="false"/>
      <property name="Client3_Command" type="array">
        <value type="string" value="Thunar"/>
        <value type="string" value="--daemon"/>
      </property>
      <property name="Client3_Priority" type="int" value="30"/>
      <property name="Client3_PerScreen" type="bool" value="false"/>
    </property>
  </property>
</channel>
`

func hexRGB(c string) (r, g, b float64) {
	var ri, gi, bi int
	fmt.Sscanf(c, "#%02x%02x%02x", &ri, &gi, &bi)
	return float64(ri) / 255, float64(gi) / 255, float64(bi) / 255
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func readme(d *design.Design) string {
	return fmt.Sprintf(`%s - made with OS Maker 2.0

This folder builds a bootable ISO of %[1]s, based on Arch Linux.

  On Arch Linux:          sudo bash build.sh
  Anywhere with Docker:   docker run --rm --privileged -e OSMAKER_CONTAINER=1 \
                            -e WORK=/tmp/work -v "$PWD":/osmaker \
                            archlinux:latest bash /osmaker/build.sh

The finished ISO appears in the out/ folder. Write it to a USB stick (for
example with Fedora Media Writer, balenaEtcher or dd) or boot it in a virtual
machine such as VirtualBox, GNOME Boxes or QEMU.

Login: %s (password: %s)

What is in here:
  build.sh        the build itself
  packages.txt    the apps you picked, as Arch package names
  airootfs/       files placed into the OS (your desktop lives in
                  airootfs/etc/osmaker/layout.json)
  design.osmaker  your design; open it in OS Maker to change it
`, d.System.OSName, d.System.Username, map[bool]string{true: "the one you set", false: "none, it logs in by itself"}[d.System.PasswordHash != ""])
}
