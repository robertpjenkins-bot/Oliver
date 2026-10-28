package mcreator

import (
	"strings"
)

// MCreator describes generators and mod APIs in YAML. Only a handful of
// values are needed from those files — a version string, a path, a block of
// Gradle code — so rather than take on a YAML dependency this reads the
// subset MCreator actually writes: nested mappings, plain and quoted scalars,
// and literal block scalars.

// yamlLookup returns the scalar found by following path through nested
// mappings, and whether it was present.
func yamlLookup(doc string, path ...string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(doc, "\r\n", "\n"), "\n")
	parent := -1 // indentation of the mapping key we are inside
	i := 0
	for depth, want := range path {
		child := -1 // indentation of this mapping's keys, learned from its first key
		found := false
		for ; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "---" {
				continue
			}
			indent := len(lines[i]) - len(strings.TrimLeft(lines[i], " "))
			if indent <= parent {
				return "", false // left the mapping without finding the key
			}
			if child < 0 {
				child = indent
			}
			if indent != child {
				continue // something nested deeper than the keys we compare
			}
			key, rest, ok := splitYAMLKey(trimmed)
			if !ok || key != want {
				continue
			}
			if depth == len(path)-1 {
				return yamlScalar(rest, lines[i+1:], indent), true
			}
			parent = indent
			i++
			found = true
			break
		}
		if !found {
			return "", false
		}
	}
	return "", false
}

// splitYAMLKey splits "key: value" into its parts. Quoted keys are unquoted.
func splitYAMLKey(line string) (key, rest string, ok bool) {
	if strings.HasPrefix(line, "- ") {
		return "", "", false
	}
	if line[0] == '"' || line[0] == '\'' {
		end := strings.IndexByte(line[1:], line[0])
		if end < 0 {
			return "", "", false
		}
		key = line[1 : end+1]
		rest = strings.TrimSpace(line[end+2:])
		if !strings.HasPrefix(rest, ":") {
			return "", "", false
		}
		return key, strings.TrimSpace(rest[1:]), true
	}
	// The separator is a colon followed by a space or the end of the line;
	// "a:b" inside a plain scalar is not one.
	for j := 0; j < len(line); j++ {
		if line[j] == ':' && (j+1 == len(line) || line[j+1] == ' ' || line[j+1] == '\t') {
			return strings.TrimSpace(line[:j]), strings.TrimSpace(line[j+1:]), true
		}
	}
	return "", "", false
}

// yamlScalar decodes the value written after a key. following holds the lines
// after the key, which a block scalar draws its content from.
func yamlScalar(rest string, following []string, keyIndent int) string {
	switch {
	case strings.HasPrefix(rest, "|") || strings.HasPrefix(rest, ">"):
		return yamlBlock(rest, following, keyIndent)
	case strings.HasPrefix(rest, `"`):
		return yamlDoubleQuoted(rest)
	case strings.HasPrefix(rest, "'"):
		end := strings.LastIndex(rest, "'")
		if end <= 0 {
			return rest[1:]
		}
		return strings.ReplaceAll(rest[1:end], "''", "'")
	}
	if i := strings.Index(rest, " #"); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

func yamlDoubleQuoted(s string) string {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			break
		}
		if c == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(s[i])
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// yamlBlock decodes a literal (|) or folded (>) block scalar.
func yamlBlock(header string, following []string, keyIndent int) string {
	folded := strings.HasPrefix(header, ">")
	strip := strings.Contains(header, "-")
	keep := strings.Contains(header, "+")

	var body []string
	indent := -1
	for _, line := range following {
		if strings.TrimSpace(line) == "" {
			body = append(body, "")
			continue
		}
		n := len(line) - len(strings.TrimLeft(line, " "))
		if n <= keyIndent {
			break
		}
		if indent < 0 {
			indent = n
		}
		if n < indent {
			break
		}
		body = append(body, line[indent:])
	}

	// Blank lines after the block belong to whatever follows it.
	trailing := 0
	for len(body) > 0 && body[len(body)-1] == "" {
		body = body[:len(body)-1]
		trailing++
	}

	sep := "\n"
	if folded {
		sep = " "
	}
	out := strings.Join(body, sep)
	switch {
	case strip || len(body) == 0:
	case keep:
		out += strings.Repeat("\n", trailing+1)
	default:
		out += "\n"
	}
	return out
}
