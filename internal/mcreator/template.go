package mcreator

import (
	"fmt"
	"path"
	"strings"
)

// MCreator passes a generator's Gradle files through FreeMarker before writing
// them into a workspace, so build.gradle can name the exact Forge, NeoForge or
// Fabric version. The templates only ever interpolate a few plain values;
// those are filled in here, and anything more elaborate is reported rather
// than written out half-rendered.

// isTemplated reports whether MCreator renders a workspace base file, which it
// decides by extension.
func isTemplated(name string) bool {
	switch path.Ext(name) {
	case ".gradle", ".properties", ".txt":
		return true
	}
	return false
}

// templateValues are the expressions a workspace base file may interpolate.
func templateValues(p *Project, g *Generator) map[string]string {
	return map[string]string{
		"generator.getGeneratorBuildFileVersion()": g.BuildFileVersion,
		"generator.getGeneratorMinecraftVersion()": g.MinecraftVersion(),
		"modid":                            p.ModID,
		"settings.getModID()":              p.ModID,
		"JavaModName":                      p.JavaModName(),
		"settings.getJavaModName()":        p.JavaModName(),
		"package":                          p.Package,
		"settings.getModElementsPackage()": p.Package,
		"settings.getModName()":            p.ModName,
		"settings.getVersion()":            p.Version,
	}
}

// renderTemplate fills in ${...} interpolations. It fails on directives and on
// expressions it does not know, since a build file with a hole in it fails
// later with an error that points nowhere near the cause.
func renderTemplate(src string, values map[string]string) (string, error) {
	for _, directive := range []string{"<#", "<@"} {
		if i := strings.Index(src, directive); i >= 0 {
			return "", fmt.Errorf("uses a template directive Zip Transform does not understand: %q", excerpt(src[i:]))
		}
	}

	var b strings.Builder
	for {
		start := strings.Index(src, "${")
		if start < 0 {
			b.WriteString(src)
			return b.String(), nil
		}
		b.WriteString(src[:start])
		src = src[start+2:]

		// A string literal, raw or not, is how a template writes a literal
		// "${" for Groovy to see: ${r"${project.version}"}.
		if lit, rest, ok := templateLiteral(src); ok {
			b.WriteString(lit)
			src = rest
			continue
		}

		end := strings.IndexByte(src, '}')
		if end < 0 {
			return "", fmt.Errorf("has an unterminated ${ in it")
		}
		expr := strings.TrimSpace(src[:end])
		value, ok := values[expr]
		if !ok {
			return "", fmt.Errorf("uses a template value Zip Transform does not know: ${%s}", expr)
		}
		b.WriteString(value)
		src = src[end+1:]
	}
}

// templateLiteral decodes r"..." or "..." followed by the closing brace.
func templateLiteral(s string) (lit, rest string, ok bool) {
	raw := strings.HasPrefix(s, `r"`)
	if raw {
		s = s[1:]
	}
	if !strings.HasPrefix(s, `"`) {
		return "", "", false
	}
	end := strings.Index(s[1:], `"}`)
	if end < 0 {
		return "", "", false
	}
	lit = s[1 : end+1]
	if !raw {
		lit = strings.NewReplacer(`\"`, `"`, `\\`, `\`).Replace(lit)
	}
	return lit, s[end+3:], true
}

func excerpt(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 60 {
		s = s[:60] + "…"
	}
	return s
}
