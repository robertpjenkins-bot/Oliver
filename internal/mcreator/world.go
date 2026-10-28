package mcreator

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// A jar is a mod, not a save, so Minecraft does nothing with a world that is
// merely packed inside one. Bundling a world therefore means two things: the
// world's files go into the jar as resources, and a small class goes in with
// them that copies those files into the player's saves folder when the mod
// first loads. The class touches no loader API at compile time — it finds
// the game directory reflectively — so the same source compiles against
// every Forge, NeoForge and Fabric version MCreator supports.

// worldResourceRoot is where a bundled world lives inside the jar. The hyphen
// keeps the path from ever being mistaken for a Java package, which matters on
// Forge and NeoForge: resources in a module's packages are encapsulated.
func worldResourceRoot(modid string) string {
	return "ziptransform-world/" + modid + "/"
}

// constructorMarker opens the user code block at the top of the generated
// main class: the constructor on Forge and NeoForge, onInitialize on Fabric.
const constructorMarker = "// Start of user code block mod constructor"

const installerClass = "ZipTransformWorld"

// InstalledWorldName is the folder under saves the bundled world is copied
// to. The mod ID keeps it from colliding with a player's own "New World". The
// world list itself shows the name stored inside the world.
func InstalledWorldName(p *Project, w World) string {
	return safeFileName(w.Name) + " (" + p.ModID + ")"
}

// bundleWorld copies a world out of the ZIP into the build's resources and
// hooks the installer into the mod's main class.
func bundleWorld(p *Project, w World, work string) error {
	mainClass, pkg, err := findMainClass(filepath.Join(work, "src", "main", "java"))
	if err != nil {
		return err
	}

	res := filepath.Join(work, "src", "main", "resources", filepath.FromSlash(worldResourceRoot(p.ModID)))
	var index []string
	for _, f := range p.zr.File {
		rel, ok := strings.CutPrefix(f.Name, w.prefix)
		if !ok || f.FileInfo().IsDir() || rel == "session.lock" {
			continue // the lock belongs to the copy that was open in the game
		}
		if !filepath.IsLocal(rel) || strings.Contains(rel, "\n") {
			return fmt.Errorf("the world contains an unsafe file name: %q", rel)
		}
		if err := extractFile(f, filepath.Join(res, "world", filepath.FromSlash(rel))); err != nil {
			return err
		}
		index = append(index, rel)
	}
	if len(index) == 0 {
		return fmt.Errorf("the world %q has no files", w.Name)
	}
	if err := os.WriteFile(filepath.Join(res, "index.txt"), []byte(strings.Join(index, "\n")+"\n"), 0o644); err != nil {
		return err
	}

	src := installerSource(pkg, p.ModID, InstalledWorldName(p, w))
	if err := os.WriteFile(filepath.Join(filepath.Dir(mainClass), installerClass+".java"), []byte(src), 0o644); err != nil {
		return err
	}
	return hookInstaller(mainClass)
}

var packageLine = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`)

// findMainClass locates the generated main mod class and its package.
func findMainClass(javaRoot string) (file, pkg string, err error) {
	var candidates []string
	err = filepath.WalkDir(javaRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".java") {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), constructorMarker) {
			candidates = append(candidates, p)
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}

	// Only the main class has a "mod constructor" block, but prefer the one
	// that declares itself a mod should a custom element copy the comment.
	for _, c := range candidates {
		data, _ := os.ReadFile(c)
		s := string(data)
		if strings.Contains(s, "@Mod(") || strings.Contains(s, "implements ModInitializer") {
			file = c
			break
		}
	}
	if file == "" && len(candidates) > 0 {
		file = candidates[0]
	}
	if file == "" {
		return "", "", fmt.Errorf("could not find the mod's main class (looked for %q), so the world cannot be installed from it", constructorMarker)
	}

	data, _ := os.ReadFile(file)
	m := packageLine.FindStringSubmatch(string(data))
	if m == nil {
		return "", "", fmt.Errorf("%s has no package declaration", file)
	}
	return file, m[1], nil
}

// hookInstaller adds the installer call at the start of the constructor.
func hookInstaller(mainClass string) error {
	data, err := os.ReadFile(mainClass)
	if err != nil {
		return err
	}
	s := string(data)
	i := strings.Index(s, constructorMarker)
	lineStart := strings.LastIndexByte(s[:i], '\n') + 1
	indent := s[lineStart:i]
	if strings.TrimSpace(indent) != "" {
		indent = "\t\t"
	}
	end := i + len(constructorMarker)
	if nl := strings.IndexByte(s[end:], '\n'); nl >= 0 {
		end += nl + 1
	} else {
		s += "\n"
		end = len(s)
	}
	call := indent + installerClass + ".install(); // added by Zip Transform\n"
	return os.WriteFile(mainClass, []byte(s[:end]+call+s[end:]), 0o644)
}

// extractFile writes one ZIP entry to disk.
func extractFile(f *zip.File, dest string) error {
	if f.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("reading %s from the ZIP: %w", f.Name, err)
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return fmt.Errorf("reading %s from the ZIP: %w", f.Name, err)
	}
	return out.Close()
}

// javaString quotes s as a Java string literal, escaping everything outside
// printable ASCII so the source file's encoding cannot matter.
func javaString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x80 && unicode.IsPrint(r):
			b.WriteRune(r)
		case r > 0xFFFF:
			for _, u := range utf16Pair(r) {
				fmt.Fprintf(&b, `\u%04x`, u)
			}
		default:
			fmt.Fprintf(&b, `\u%04x`, r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func utf16Pair(r rune) [2]rune {
	r -= 0x10000
	return [2]rune{0xD800 + (r>>10)&0x3FF, 0xDC00 + r&0x3FF}
}

// installerSource is the class that installs the bundled world.
func installerSource(pkg, modid, folder string) string {
	return fmt.Sprintf(installerTemplate, pkg, javaString("/"+worldResourceRoot(modid)), javaString(folder))
}

// The installer must compile on Java 17, the oldest Java a supported Forge
// uses, and must never take the game down with it: every failure is logged
// and swallowed.
const installerTemplate = `package %s;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.stream.Stream;

/**
 * Added by Zip Transform. Copies the world bundled in this mod's jar into the
 * player's saves folder the first time the game starts with the mod installed.
 * A world already in that folder is never replaced, so a player's progress in
 * it is safe across restarts and mod updates.
 */
public final class ZipTransformWorld {

	private static final String RESOURCES = %s;
	private static final String FOLDER = %s;

	private ZipTransformWorld() {
	}

	public static void install() {
		try {
			if (!isClient())
				return; // a dedicated server has no world list to add to
			Path saves = gameDirectory().resolve("saves");
			Path target = saves.resolve(FOLDER);
			if (Files.exists(target))
				return;
			List<String> files = index();
			if (files.isEmpty())
				return;

			// Copy beside the target and rename at the end, so a game closed
			// halfway through never leaves a broken world in the list.
			Path staging = saves.resolve(FOLDER + ".installing");
			deleteTree(staging);
			for (String name : files) {
				Path out = staging.resolve(name).normalize();
				if (!out.startsWith(staging))
					continue;
				try (InputStream in = ZipTransformWorld.class.getResourceAsStream(RESOURCES + "world/" + name)) {
					if (in == null)
						throw new IOException("missing from the jar: " + name);
					Files.createDirectories(out.getParent());
					Files.copy(in, out, StandardCopyOption.REPLACE_EXISTING);
				}
			}
			Files.move(staging, target);
			System.out.println("[Zip Transform] added the world \"" + FOLDER + "\" to " + saves);
		} catch (Throwable t) {
			System.err.println("[Zip Transform] could not add the bundled world: " + t);
		}
	}

	private static List<String> index() throws IOException {
		List<String> files = new ArrayList<>();
		try (InputStream in = ZipTransformWorld.class.getResourceAsStream(RESOURCES + "index.txt")) {
			if (in == null)
				return files;
			BufferedReader reader = new BufferedReader(new InputStreamReader(in, StandardCharsets.UTF_8));
			for (String line = reader.readLine(); line != null; line = reader.readLine()) {
				if (!line.isEmpty())
					files.add(line);
			}
		}
		return files;
	}

	private static void deleteTree(Path dir) throws IOException {
		if (!Files.exists(dir))
			return;
		List<Path> paths = new ArrayList<>();
		try (Stream<Path> walk = Files.walk(dir)) {
			walk.forEach(paths::add);
		}
		Collections.reverse(paths);
		for (Path p : paths)
			Files.deleteIfExists(p);
	}

	// The game directory and the side are looked up by name, so this class
	// compiles whichever loader the mod is built for.

	private static Path gameDirectory() {
		Object dir = constant("net.neoforged.fml.loading.FMLPaths", "GAMEDIR", "get");
		if (dir == null)
			dir = constant("net.minecraftforge.fml.loading.FMLPaths", "GAMEDIR", "get");
		if (dir == null)
			dir = fabric("getGameDir");
		if (dir instanceof Path)
			return (Path) dir;
		return Paths.get("").toAbsolutePath(); // the launcher starts the game in its directory
	}

	private static boolean isClient() {
		Object side = field("net.neoforged.fml.loading.FMLEnvironment", "dist");
		if (side == null)
			side = method("net.neoforged.fml.loading.FMLEnvironment", "getDist");
		if (side == null)
			side = field("net.minecraftforge.fml.loading.FMLEnvironment", "dist");
		if (side == null)
			side = fabric("getEnvironmentType");
		return side == null || "CLIENT".equals(String.valueOf(side));
	}

	private static Object constant(String className, String name, String getter) {
		try {
			Class<?> c = Class.forName(className);
			return c.getMethod(getter).invoke(c.getField(name).get(null));
		} catch (Throwable t) {
			return null;
		}
	}

	private static Object field(String className, String name) {
		try {
			return Class.forName(className).getField(name).get(null);
		} catch (Throwable t) {
			return null;
		}
	}

	private static Object method(String className, String name) {
		try {
			return Class.forName(className).getMethod(name).invoke(null);
		} catch (Throwable t) {
			return null;
		}
	}

	private static Object fabric(String name) {
		try {
			Class<?> loader = Class.forName("net.fabricmc.loader.api.FabricLoader");
			return loader.getMethod(name).invoke(loader.getMethod("getInstance").invoke(null));
		} catch (Throwable t) {
			return null;
		}
	}
}
`
