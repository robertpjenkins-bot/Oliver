package minecraft

import (
	"strings"
)

// Forge and NeoForge describe a mod in TOML. The handful of values needed
// here — IDs, names, versions, version ranges — are strings in tables, so
// this reads that much of TOML and skips the rest: multi-line strings are
// stepped over, arrays and inline tables kept as raw text.

// tomlTable is one table: the root, a [table] or one [[array]] entry.
type tomlTable struct {
	name   string
	values map[string]string
}

func parseTOML(src string) []tomlTable {
	tables := []tomlTable{{values: map[string]string{}}}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			header := stripTOMLComment(line)
			header = strings.Trim(header, "[] \t")
			tables = append(tables, tomlTable{name: unquoteKey(header), values: map[string]string{}})
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue // a continuation line of a multi-line array
		}
		key = unquoteKey(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		var v string
		switch {
		case strings.HasPrefix(value, `"""`) || strings.HasPrefix(value, `'''`):
			delim := value[:3]
			rest := value[3:]
			var b strings.Builder
			for {
				if end := strings.Index(rest, delim); end >= 0 {
					b.WriteString(rest[:end])
					break
				}
				b.WriteString(rest + "\n")
				i++
				if i >= len(lines) {
					break
				}
				rest = lines[i]
			}
			v = strings.TrimPrefix(b.String(), "\n")
		case strings.HasPrefix(value, `"`):
			v = tomlBasicString(value)
		case strings.HasPrefix(value, "'"):
			end := strings.IndexByte(value[1:], '\'')
			if end < 0 {
				end = len(value) - 1
			}
			v = value[1 : end+1]
		default:
			v = strings.TrimSpace(stripTOMLComment(value))
		}
		tables[len(tables)-1].values[key] = v
	}
	return tables
}

func tomlBasicString(s string) string {
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

func stripTOMLComment(s string) string {
	inString := byte(0)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case inString != 0:
			if c == inString {
				inString = 0
			}
		case c == '"' || c == '\'':
			inString = c
		case c == '#':
			return s[:i]
		}
	}
	return s
}

func unquoteKey(k string) string {
	if len(k) >= 2 && (k[0] == '"' || k[0] == '\'') && k[len(k)-1] == k[0] {
		return k[1 : len(k)-1]
	}
	return k
}
