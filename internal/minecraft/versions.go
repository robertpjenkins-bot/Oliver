package minecraft

import (
	"strconv"
	"strings"
)

// Mods state which Minecraft versions they run on in their loader's own
// syntax: Forge and NeoForge use Maven ranges, "[1.21.1,1.21.2)"; Fabric and
// Quilt use predicates, "~1.21.1" or ">=1.20 <1.22". Both are handled here.
// Each check also reports whether it understood the range at all, so that a
// snapshot or an unusual range reads as "cannot tell" rather than "no".

// compareVersions orders release versions numerically: 1.9 < 1.21 < 26.1.
// The second result is false when either side is not a plain release.
func compareVersions(a, b string) (int, bool) {
	as, ok1 := versionParts(a)
	bs, ok2 := versionParts(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x = as[i]
		}
		if i < len(bs) {
			y = bs[i]
		}
		if x != y {
			if x < y {
				return -1, true
			}
			return 1, true
		}
	}
	return 0, true
}

func versionParts(v string) ([]int, bool) {
	v = strings.TrimSpace(v)
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i] // pre-release and build suffixes
	}
	if v == "" {
		return nil, false
	}
	var parts []int
	for _, p := range strings.Split(v, ".") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		parts = append(parts, n)
	}
	return parts, true
}

// Accepts reports whether a range allows a Minecraft version, and whether the
// range could be understood. The syntax is chosen by the loader.
func Accepts(loader, versions, minecraft string) (ok, understood bool) {
	versions = strings.TrimSpace(versions)
	if versions == "" || versions == "*" {
		return true, true
	}
	if loader == Fabric || loader == Quilt {
		return predicateAllows(versions, minecraft)
	}
	return mavenAllows(versions, minecraft)
}

// mavenAllows checks a Maven version range such as "[1.21.1]",
// "[1.20,1.21)" or "[1.18,1.19),[1.20,)". A bare version, as in Maven,
// constrains nothing.
func mavenAllows(spec, v string) (bool, bool) {
	spec = strings.ReplaceAll(spec, " ", "")
	if !strings.ContainsAny(spec, "[(") {
		_, ok := versionParts(spec)
		return true, ok
	}
	for spec != "" {
		end := strings.IndexAny(spec, "])")
		if end < 0 || (spec[0] != '[' && spec[0] != '(') {
			return false, false
		}
		ok, understood := mavenInterval(spec[:end+1], v)
		if !understood {
			return false, false
		}
		if ok {
			return true, true
		}
		spec = strings.TrimPrefix(spec[end+1:], ",")
	}
	return false, true
}

func mavenInterval(r, v string) (bool, bool) {
	lowInclusive, highInclusive := r[0] == '[', r[len(r)-1] == ']'
	body := r[1 : len(r)-1]
	low, high, isRange := strings.Cut(body, ",")
	if !isRange { // [1.21.1] is exactly that version
		c, ok := compareVersions(v, body)
		return ok && c == 0, ok
	}
	if low != "" {
		c, ok := compareVersions(v, low)
		if !ok {
			return false, false
		}
		if c < 0 || (c == 0 && !lowInclusive) {
			return false, true
		}
	}
	if high != "" {
		c, ok := compareVersions(v, high)
		if !ok {
			return false, false
		}
		if c > 0 || (c == 0 && !highInclusive) {
			return false, true
		}
	}
	return true, true
}

// predicateAllows checks a Fabric version predicate. Alternatives are
// separated by "||"; within one, space-separated conditions must all hold.
func predicateAllows(spec, v string) (bool, bool) {
	for _, alt := range strings.Split(spec, "||") {
		all, understood := true, true
		for _, cond := range strings.Fields(alt) {
			ok, u := predicateTerm(cond, v)
			if !u {
				understood = false
			}
			if !ok {
				all = false
			}
		}
		if !understood {
			return false, false
		}
		if all {
			return true, true
		}
	}
	return false, true
}

func predicateTerm(cond, v string) (bool, bool) {
	if cond == "*" {
		return true, true
	}
	op := ""
	for _, o := range []string{">=", "<=", ">", "<", "=", "~", "^"} {
		if strings.HasPrefix(cond, o) {
			op, cond = o, cond[len(o):]
			break
		}
	}

	// "1.21.x" matches every 1.21 release.
	if strings.HasSuffix(cond, ".x") || strings.HasSuffix(cond, ".*") {
		prefix, ok := versionParts(cond[:len(cond)-2])
		parts, ok2 := versionParts(v)
		if !ok || !ok2 || op != "" && op != "=" {
			return false, false
		}
		for i, p := range prefix {
			if i >= len(parts) || parts[i] != p {
				return false, true
			}
		}
		return true, true
	}

	c, ok := compareVersions(v, cond)
	if !ok {
		return false, false
	}
	switch op {
	case "", "=":
		return c == 0, true
	case ">=":
		return c >= 0, true
	case "<=":
		return c <= 0, true
	case ">":
		return c > 0, true
	case "<":
		return c < 0, true
	case "~": // same major and minor, at least this
		return c >= 0 && sameLeading(v, cond, 2), true
	case "^": // same major, at least this
		return c >= 0 && sameLeading(v, cond, 1), true
	}
	return false, false
}

func sameLeading(a, b string, n int) bool {
	as, _ := versionParts(a)
	bs, _ := versionParts(b)
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(as) {
			x = as[i]
		}
		if i < len(bs) {
			y = bs[i]
		}
		if x != y {
			return false
		}
	}
	return true
}

// DescribeRange turns a range into words for people: "[26.1.2]" becomes
// "26.1.2", "~1.21.1" becomes "1.21.1 or a later 1.21".
func DescribeRange(r string) string {
	r = strings.TrimSpace(r)
	switch {
	case r == "" || r == "*":
		return "any version"
	case strings.HasPrefix(r, "[") && strings.HasSuffix(r, "]") && !strings.Contains(r, ","):
		return r[1 : len(r)-1]
	case strings.HasPrefix(r, "~"):
		v := r[1:]
		if parts, ok := versionParts(v); ok && len(parts) >= 2 {
			return v + " or a later " + strconv.Itoa(parts[0]) + "." + strconv.Itoa(parts[1])
		}
	case strings.HasPrefix(r, "[") && strings.HasSuffix(r, ",)"):
		return r[1:len(r)-2] + " or later"
	case strings.HasPrefix(r, ">="):
		if !strings.ContainsAny(r[2:], " |") {
			return r[2:] + " or later"
		}
	}
	return r
}
