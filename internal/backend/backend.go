// Package backend finds and ranks the Windows-compatibility runtimes already
// installed on the system.
package backend

import (
	"fmt"
	"sort"
	"strings"
)

// Kind is the family a runtime belongs to. Each family is launched
// differently, so this drives command construction later.
type Kind string

const (
	KindWine      Kind = "wine"
	KindProton    Kind = "proton"
	KindCrossOver Kind = "crossover"
	KindBox       Kind = "box64" // emulated x86 wine on ARM hosts
)

// Source records where a runtime came from, which is a good proxy for how
// well maintained and how fast to start it is.
type Source string

const (
	SourceSystem      Source = "system"
	SourceSteam       Source = "steam"
	SourceCompatTools Source = "compatibilitytools.d"
	SourceLutris      Source = "lutris"
	SourceBottles     Source = "bottles"
	SourceHeroic      Source = "heroic"
	SourcePlayOnLinux Source = "playonlinux"
	SourceCrossOver   Source = "crossover"
	SourceOpt         Source = "opt"
	SourceFlatpak     Source = "flatpak"
)

// Variant marks notable builds within a family.
type Variant string

const (
	VariantStable       Variant = ""
	VariantStaging      Variant = "staging"
	VariantDevel        Variant = "devel"
	VariantGE           Variant = "ge"
	VariantExperimental Variant = "experimental"
	VariantTKG          Variant = "tkg"
)

// Backend is one installed runtime we can launch an executable with.
type Backend struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Kind    Kind    `json:"kind"`
	Source  Source  `json:"source"`
	Variant Variant `json:"variant,omitempty"`

	// Exec is what we actually invoke: a wine binary, or a Proton script.
	Exec string `json:"exec"`
	// Root is the install directory, used to locate sibling files.
	Root string `json:"root,omitempty"`

	Version Version `json:"version"`

	// Has32 reports whether this build can run 32-bit Windows binaries. Modern
	// wow64-only Wine builds and distro packages missing 32-bit libraries
	// cannot, and that is the single most common cause of a failed launch.
	Has32 bool `json:"supports_32bit"`
	Has64 bool `json:"supports_64bit"`

	// Launcher is an optional wrapper command (umu-run, box64, flatpak).
	Launcher []string `json:"launcher,omitempty"`

	Notes []string `json:"notes,omitempty"`
}

// String renders a short human-readable identity.
func (b *Backend) String() string {
	v := b.Version.String()
	if v == "" {
		v = "unknown version"
	}
	return fmt.Sprintf("%s (%s)", b.Name, v)
}

// Bitness describes 32/64-bit support in one word for tables.
func (b *Backend) Bitness() string {
	switch {
	case b.Has32 && b.Has64:
		return "32+64"
	case b.Has64:
		return "64-only"
	case b.Has32:
		return "32-only"
	}
	return "unknown"
}

// IsProton reports whether launching goes through the Proton entry point.
func (b *Backend) IsProton() bool { return b.Kind == KindProton }

// Sort orders backends newest-and-best first, with a deterministic tiebreak so
// output is stable between runs.
func Sort(bs []*Backend) {
	sort.SliceStable(bs, func(i, j int) bool {
		a, b := bs[i], bs[j]
		if c := a.Version.Compare(b.Version); c != 0 {
			return c > 0
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID < b.ID
	})
}

// Find returns the backend whose ID or name matches the query. Matching is
// case-insensitive and accepts a unique prefix or substring, so users can type
// "ge" or "staging" instead of a full identifier.
func Find(bs []*Backend, query string) (*Backend, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, fmt.Errorf("no backend specified")
	}
	var exact, partial []*Backend
	for _, b := range bs {
		id, name := strings.ToLower(b.ID), strings.ToLower(b.Name)
		switch {
		case id == q || name == q:
			exact = append(exact, b)
		case strings.Contains(id, q) || strings.Contains(name, q):
			partial = append(partial, b)
		}
	}
	if len(exact) > 0 {
		return exact[0], nil
	}
	switch len(partial) {
	case 0:
		return nil, fmt.Errorf("no installed backend matches %q (try `oliver backends`)", query)
	case 1:
		return partial[0], nil
	default:
		ids := make([]string, 0, len(partial))
		for _, b := range partial {
			ids = append(ids, b.ID)
		}
		return nil, fmt.Errorf("%q matches several backends: %s", query, strings.Join(ids, ", "))
	}
}
