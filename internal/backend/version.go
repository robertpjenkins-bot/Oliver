package backend

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// Version is a comparable runtime version. Wine, Proton and Proton-GE all use
// different formats, so we normalise them into ordered numeric fields plus the
// original string for display.
type Version struct {
	Fields []int  `json:"fields"`
	Raw    string `json:"raw"`
}

// MarshalJSON emits the display string, which is what consumers actually want.
func (v Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Raw    string `json:"raw"`
		Fields []int  `json:"fields"`
	}{v.Raw, v.Fields})
}

func (v Version) String() string { return v.Raw }

// Major returns the leading version component, or 0 if unknown.
func (v Version) Major() int {
	if len(v.Fields) == 0 {
		return 0
	}
	return v.Fields[0]
}

// IsZero reports whether no version could be determined.
func (v Version) IsZero() bool { return len(v.Fields) == 0 }

// Compare returns >0 if v is newer than o, <0 if older, 0 if equal.
func (v Version) Compare(o Version) int {
	for i := 0; i < len(v.Fields) || i < len(o.Fields); i++ {
		var a, b int
		if i < len(v.Fields) {
			a = v.Fields[i]
		}
		if i < len(o.Fields) {
			b = o.Fields[i]
		}
		if a != b {
			if a > b {
				return 1
			}
			return -1
		}
	}
	return 0
}

var numRun = regexp.MustCompile(`[0-9]+`)

// ParseVersion pulls an ordered set of numbers out of a version string.
// It copes with "wine-9.0", "wine-8.0.2 (Staging)", "GE-Proton9-20",
// "Proton 9.0", "Proton Experimental" and "proton-9.0-4c".
func ParseVersion(s string) Version {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return Version{}
	}
	// Drop a leading product name so "Proton 9.0" and "wine-9.0" both start at
	// the first real number, and trailing parenthetical build info.
	work := raw
	if i := strings.IndexByte(work, '('); i >= 0 {
		work = work[:i]
	}
	matches := numRun.FindAllString(work, -1)
	fields := make([]int, 0, len(matches))
	for _, m := range matches {
		n, err := strconv.Atoi(m)
		if err != nil {
			continue
		}
		fields = append(fields, n)
		if len(fields) == 4 {
			break
		}
	}
	return Version{Fields: fields, Raw: raw}
}

// variantPatterns maps a marker in a version or directory name to the build
// variant it identifies.
var variantPatterns = []struct {
	needle  string
	variant Variant
}{
	{"experimental", VariantExperimental},
	{"staging", VariantStaging},
	{"ge-proton", VariantGE},
	{"proton-ge", VariantGE},
	{"lutris-ge", VariantGE},
	{"wine-ge", VariantGE},
	{"tkg", VariantTKG},
	{"devel", VariantDevel},
}

// DetectVariant identifies notable builds from any descriptive string.
func DetectVariant(s string) Variant {
	l := strings.ToLower(s)
	for _, p := range variantPatterns {
		if strings.Contains(l, p.needle) {
			return p.variant
		}
	}
	// "-ge-" as a standalone token, e.g. "wine-8.26-ge-1".
	if strings.Contains(l, "-ge-") || strings.HasSuffix(l, "-ge") {
		return VariantGE
	}
	return VariantStable
}
