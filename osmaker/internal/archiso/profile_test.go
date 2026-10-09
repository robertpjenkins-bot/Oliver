package archiso

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/osmaker/internal/design"
)

func testDesign(t *testing.T) *design.Design {
	t.Helper()
	d := design.Default()
	d.System.OSName = "Rob's OS"
	d.System.PasswordHash = design.HashPassword("pw")
	d.Taskbar.Pinned = append(d.Taskbar.Pinned, "steam", "python")
	d.Taskbar.MenuLabel = "Go & play"
	d.Normalize()
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestFilesAreWellFormed(t *testing.T) {
	files, err := Files(testDesign(t))
	if err != nil {
		t.Fatal(err)
	}
	for path, f := range files {
		switch {
		case strings.HasSuffix(path, ".xml"):
			dec := xml.NewDecoder(strings.NewReader(string(f.Data)))
			for {
				if _, err := dec.Token(); err == io.EOF {
					break
				} else if err != nil {
					t.Errorf("%s: bad XML: %v", path, err)
					break
				}
			}
		case strings.HasSuffix(path, ".json"):
			var v any
			if err := json.Unmarshal(f.Data, &v); err != nil {
				t.Errorf("%s: bad JSON: %v", path, err)
			}
		}
	}

	panel := string(files["airootfs/home/user/.config/xfce4/xfconf/xfce-perchannel-xml/xfce4-panel.xml"].Data)
	for _, want := range []string{`value="Go &amp; play"`, `value="steam.desktop"`, `value="p=8;x=0;y=0"`, `value="clock"`} {
		if !strings.Contains(panel, want) {
			t.Errorf("panel config missing %s", want)
		}
	}
	if l := files["airootfs/home/user/.config/xfce4/panel/launcher-6/python.desktop"]; !strings.Contains(string(l.Data), "xfce4-terminal --hold -e python3") {
		t.Errorf("command-line launcher should open a terminal:\n%s", l.Data)
	}
	pkgs := string(files["packages.txt"].Data)
	for _, want := range []string{"\nsteam\n", "\nlib32-mesa\n", "\nfirefox\n", "\nxfce4\n", "\nthunar\n"} {
		if !strings.Contains(pkgs, want) {
			t.Errorf("packages missing %q", want)
		}
	}
	script := string(files["build.sh"].Data)
	if !strings.Contains(script, `OS_NAME='Rob'\''s OS'`) || !strings.Contains(script, "MULTILIB=1") {
		t.Errorf("build.sh settings wrong:\n%s", script[:600])
	}
	if strings.Contains(script, "@") && strings.Contains(script, "@OS_NAME@") {
		t.Error("placeholder left in build.sh")
	}
	shadow := string(files["airootfs/etc/shadow"].Data)
	if !strings.Contains(shadow, "user:$6$") {
		t.Errorf("password hash missing from shadow:\n%s", shadow)
	}
	if files["airootfs/etc/localtime"].Link != "/usr/share/zoneinfo/UTC" {
		t.Error("timezone link wrong")
	}
}

func TestWallpaperImageIsWritten(t *testing.T) {
	d := testDesign(t)
	d.Desktop.Background = design.Background{Kind: "image", Image: "data:image/png;base64,iVBORw0KGgo="}
	files, err := Files(d)
	if err != nil {
		t.Fatal(err)
	}
	if string(files["airootfs/etc/osmaker/wallpaper.png"].Data) != "\x89PNG\r\n\x1a\n" {
		t.Error("wallpaper not decoded")
	}
	if !strings.Contains(string(files["airootfs/etc/osmaker/layout.json"].Data), `"/etc/osmaker/wallpaper.png"`) {
		t.Error("layout does not point at the wallpaper")
	}
}

func TestWriteProducesRunnableFolder(t *testing.T) {
	dir := t.TempDir()
	if err := Write(testDesign(t), dir); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("bash", "-n", filepath.Join(dir, "build.sh")).CombinedOutput(); err != nil {
		t.Fatalf("build.sh does not parse: %v\n%s", err, out)
	}
	if out, err := exec.Command("bash", "-n", filepath.Join(dir, "permissions.sh")).CombinedOutput(); err != nil {
		t.Fatalf("permissions.sh does not parse: %v\n%s", err, out)
	}
	if target, err := os.Readlink(filepath.Join(dir, "airootfs/etc/systemd/system/default.target")); err != nil || !strings.HasSuffix(target, "graphical.target") {
		t.Fatalf("default.target link: %q %v", target, err)
	}
	fi, err := os.Stat(filepath.Join(dir, "airootfs/usr/local/bin/osmaker-desktop"))
	if err != nil || fi.Mode().Perm() != 0o755 {
		t.Fatalf("desktop shell not executable: %v %v", fi, err)
	}
	if py, err := exec.LookPath("python3"); err == nil {
		if out, err := exec.Command(py, "-m", "py_compile", filepath.Join(dir, "airootfs/usr/local/bin/osmaker-desktop")).CombinedOutput(); err != nil {
			t.Fatalf("desktop shell does not compile: %s", out)
		}
	}
	// Writing again over an existing folder must work.
	if err := Write(testDesign(t), dir); err != nil {
		t.Fatal(err)
	}
}
