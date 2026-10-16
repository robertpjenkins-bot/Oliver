package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The real Batocera boot script, from a checkout of batocera.linux if one is
// around (set BATOCERA_INIT), else a stand-in with the same anchors.
func batoceraInit(t *testing.T) string {
	if p := os.Getenv("BATOCERA_INIT"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	return "#!/bin/ash\ndo_root() {\n    # update the squashfs\n    true\n    # mount the future root in read write\n" +
		"    mount -t overlay overlay -o rw,lowerdir=/overlay_root/base$EXTRALAYERS,upperdir=/u /new_root\n}\n"
}

func makeInitrd(t *testing.T, init string, uboot bool) []byte {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "init"), []byte(init), 0o755); err != nil {
		t.Fatal(err)
	}
	os.Mkdir(filepath.Join(dir, "bin"), 0o755)
	cmd := exec.Command("sh", "-c", "find . -mindepth 1 | cpio -o -H newc --quiet | lz4 -l -q")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Skip("needs cpio and lz4: ", err)
	}
	if !uboot {
		return out
	}
	lz := filepath.Join(dir, "x.lz4")
	os.WriteFile(lz, out, 0o644)
	ui := filepath.Join(dir, "uInitrd")
	if err := exec.Command("mkimage", "-A", "arm64", "-O", "linux", "-T", "ramdisk", "-C", "none", "-n", "uInitrd", "-d", lz, ui).Run(); err != nil {
		t.Skip("needs mkimage: ", err)
	}
	b, _ := os.ReadFile(ui)
	return b
}

func unpack(t *testing.T, initrd []byte, uboot bool) map[string]string {
	dir := t.TempDir()
	f := filepath.Join(dir, "in")
	if uboot {
		// mkimage -l checks both CRCs.
		os.WriteFile(f, initrd, 0o644)
		if out, err := exec.Command("mkimage", "-l", f).CombinedOutput(); err != nil || !strings.Contains(strings.ToLower(string(out)), "ramdisk") || strings.Contains(strings.ToLower(string(out)), "bad") {
			t.Fatalf("mkimage -l: %v\n%s", err, out)
		}
		initrd = initrd[64:]
	}
	os.WriteFile(f, initrd, 0o644)
	root := filepath.Join(dir, "root")
	os.Mkdir(root, 0o755)
	cmd := exec.Command("sh", "-c", "lz4 -dc "+f+" | cpio -id --quiet")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("unpacking: %v\n%s", err, out)
	}
	files := map[string]string{}
	for _, n := range []string{"init", "etros-hook.sh"} {
		b, err := os.ReadFile(filepath.Join(root, n))
		if err != nil {
			t.Fatalf("%s missing", n)
		}
		files[n] = string(b)
	}
	return files
}

func TestPatchInitrd(t *testing.T) {
	init := batoceraInit(t)
	for _, uboot := range []bool{false, true} {
		out, err := PatchInitrd(makeInitrd(t, init, uboot))
		if err != nil {
			t.Fatal(err)
		}
		files := unpack(t, out, uboot)
		if !strings.Contains(files["init"], ". /etros-hook.sh && etros_start") ||
			!strings.Contains(files["init"], "lowerdir=${ETROS_LAYER}/overlay_root/base") {
			t.Errorf("init not patched:\n%s", files["init"])
		}
		if !bytes.Equal([]byte(files["etros-hook.sh"]), etrosHook) {
			t.Error("hook differs")
		}
	}
}

func TestPatchInitRefusesUnknownScript(t *testing.T) {
	if _, err := patchInit("#!/bin/sh\necho hello\n"); err == nil {
		t.Error("patched a script without the anchors")
	}
}
