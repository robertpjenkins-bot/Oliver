package backend

import (
	"fmt"
	"sort"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/pe"
)

// Profile selects the trade-off to optimise for. Games want the Direct3D
// translation layers Proton bundles; ordinary applications want the lower
// startup cost and desktop integration of plain Wine.
type Profile string

const (
	ProfileAuto Profile = "auto"
	ProfileGame Profile = "game"
	ProfileApp  Profile = "app"
)

// ParseProfile validates a profile name.
func ParseProfile(s string) (Profile, error) {
	switch Profile(strings.ToLower(strings.TrimSpace(s))) {
	case "", ProfileAuto:
		return ProfileAuto, nil
	case ProfileGame:
		return ProfileGame, nil
	case ProfileApp:
		return ProfileApp, nil
	}
	return "", fmt.Errorf("unknown profile %q (want auto, game or app)", s)
}

// Reason is a single scoring decision, kept so the choice can be explained.
type Reason struct {
	Delta int    `json:"delta"`
	Text  string `json:"text"`
}

// Candidate is a backend with its score for a particular executable.
type Candidate struct {
	Backend    *Backend `json:"backend"`
	Score      int      `json:"score"`
	Reasons    []Reason `json:"reasons"`
	Blockers   []string `json:"blockers,omitempty"`
	Compatible bool     `json:"compatible"`
}

// Request is everything the ranker needs to know about what is being launched.
type Request struct {
	Info    *pe.Info
	Profile Profile
	// Prefer forces a specific backend to the top when it matches.
	Prefer string
}

// EffectiveProfile resolves ProfileAuto using what we learned from the binary.
func (r Request) EffectiveProfile() Profile {
	if r.Profile != ProfileAuto && r.Profile != "" {
		return r.Profile
	}
	if r.Info == nil {
		return ProfileApp
	}
	// An installer is an installer even when it links Direct3D.
	if r.Info.IsInstaller {
		return ProfileApp
	}
	if r.Info.LikelyGame {
		return ProfileGame
	}
	return ProfileApp
}

// Rank scores every backend for the request, best first. Incompatible
// backends are kept (so the user can see why they were rejected) but always
// sort below compatible ones.
func Rank(bs []*Backend, req Request) []*Candidate {
	profile := req.EffectiveProfile()
	cands := make([]*Candidate, 0, len(bs))
	for _, b := range bs {
		cands = append(cands, score(b, req, profile))
	}
	sort.SliceStable(cands, func(i, j int) bool {
		a, c := cands[i], cands[j]
		if a.Compatible != c.Compatible {
			return a.Compatible
		}
		if a.Score != c.Score {
			return a.Score > c.Score
		}
		if v := a.Backend.Version.Compare(c.Backend.Version); v != 0 {
			return v > 0
		}
		return a.Backend.ID < c.Backend.ID
	})
	return cands
}

// Best returns the highest-scoring compatible backend.
func Best(bs []*Backend, req Request) (*Candidate, error) {
	ranked := Rank(bs, req)
	if len(ranked) == 0 {
		return nil, fmt.Errorf("no Windows-compatibility runtime found; run `oliver doctor` for what to install")
	}
	if !ranked[0].Compatible {
		return nil, fmt.Errorf("none of the %d installed runtimes can run this executable: %s",
			len(ranked), strings.Join(ranked[0].Blockers, "; "))
	}
	return ranked[0], nil
}

func score(b *Backend, req Request, profile Profile) *Candidate {
	c := &Candidate{Backend: b, Compatible: true}
	add := func(delta int, format string, args ...any) {
		c.Score += delta
		c.Reasons = append(c.Reasons, Reason{Delta: delta, Text: fmt.Sprintf(format, args...)})
	}
	block := func(format string, args ...any) {
		c.Compatible = false
		c.Blockers = append(c.Blockers, fmt.Sprintf(format, args...))
	}

	switch b.Kind {
	case KindWine:
		add(100, "Wine")
	case KindProton:
		add(100, "Proton")
	case KindCrossOver:
		add(95, "CrossOver")
	case KindBox:
		add(60, "Wine under x86 translation")
	}

	info := req.Info
	if info != nil {
		scoreArchitecture(b, info, block, add)
		scoreWorkload(b, info, profile, add)
	}

	scoreBuildQuality(b, profile, add)

	if b.IsProton() && len(b.Launcher) == 0 {
		add(-18, "no umu-launcher installed, so Proton must run without the Steam runtime")
	}
	if b.Source == SourceSystem {
		add(4, "installed system-wide")
	}
	if b.Source == SourceFlatpak {
		add(-6, "flatpak sandbox restricts filesystem access")
	}
	if b.Kind == KindBox {
		add(-25, "x86 translation carries a large performance cost")
	}

	// An explicit preference outranks everything, but never makes an
	// incompatible runtime usable.
	if req.Prefer != "" && matches(b, req.Prefer) {
		add(1000, "matches your requested backend %q", req.Prefer)
	}

	return c
}

func scoreArchitecture(b *Backend, info *pe.Info, block func(string, ...any), add func(int, string, ...any)) {
	switch info.Kind {
	case pe.KindDOS:
		block("this is a 16-bit DOS program; Wine cannot run it, use DOSBox instead")
		return
	case pe.KindBatch:
		add(0, "batch script, run through the Windows command interpreter")
	}

	switch info.Arch {
	case pe.ArchX86:
		if !b.Has32 {
			block("%s has no 32-bit support and this is a 32-bit executable", b.Name)
			return
		}
		add(0, "32-bit executable, and this build supports 32-bit")
	case pe.ArchX64:
		if !b.Has64 {
			block("%s has no 64-bit support and this is a 64-bit executable", b.Name)
			return
		}
	case pe.ArchARM64:
		block("ARM64 Windows binaries need an ARM64EC-capable Wine build")
	case pe.ArchARM32:
		block("32-bit ARM Windows binaries are not supported by any current Wine build")
	}
}

func scoreWorkload(b *Backend, info *pe.Info, profile Profile, add func(int, string, ...any)) {
	// Installers are the clearest case: they want a plain, writable prefix and
	// they interact badly with Proton's container.
	if info.IsInstaller {
		switch b.Kind {
		case KindProton:
			add(-45, "installers are unreliable inside Proton's container")
		case KindWine, KindCrossOver:
			add(25, "plain Wine is the reliable way to run an installer")
		}
		if info.Kind == pe.KindMSI && b.Kind != KindProton {
			add(10, "MSI packages install cleanly through Wine's msiexec")
		}
	} else {
		switch profile {
		case ProfileGame:
			switch b.Kind {
			case KindProton:
				add(45, "Proton is purpose-built for games")
			case KindCrossOver:
				add(5, "CrossOver includes its own Direct3D translation")
			}
		case ProfileApp:
			switch b.Kind {
			case KindWine, KindCrossOver:
				add(35, "plain Wine starts faster and integrates with the desktop")
			case KindProton:
				add(-30, "Proton's container adds startup cost and isolates the application")
			}
		}
	}

	if len(info.GraphicsAPIs) > 0 && !info.IsInstaller {
		apis := strings.Join(info.GraphicsAPIs, ", ")
		switch {
		case b.Kind == KindProton:
			add(15, "ships DXVK and VKD3D-Proton for %s", apis)
		case b.Variant == VariantGE:
			add(10, "GE builds bundle DXVK for %s", apis)
		case b.Kind == KindWine:
			add(0, "uses %s; consider installing DXVK into the prefix", apis)
		}
	}

	if info.DotNET && !info.DotNETCore {
		if b.Kind == KindWine || b.Kind == KindCrossOver {
			add(8, ".NET Framework applications run well under Wine Mono")
		}
	}
}

func scoreBuildQuality(b *Backend, profile Profile, add func(int, string, ...any)) {
	switch b.Variant {
	case VariantStaging:
		add(6, "Staging carries patches not yet in upstream Wine")
	case VariantGE:
		if profile == ProfileGame {
			add(12, "GE builds carry game-focused patches")
		} else {
			add(2, "GE build")
		}
	case VariantExperimental:
		if profile == ProfileGame {
			add(6, "Proton Experimental gets fixes first")
		} else {
			add(-2, "Experimental builds change frequently")
		}
	case VariantTKG:
		if profile == ProfileGame {
			add(4, "TKG builds are tuned for performance")
		}
	}

	// Newer runtimes support more of the Windows API. Capped so that a very
	// new but unsuitable runtime never outranks the right kind of runtime.
	if major := b.Version.Major(); major > 0 {
		bonus := 2 * major
		if bonus > 22 {
			bonus = 22
		}
		add(bonus, "version %s", b.Version)
	} else {
		add(-2, "version could not be determined")
	}
}

func matches(b *Backend, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	return q != "" && (strings.Contains(strings.ToLower(b.ID), q) || strings.Contains(strings.ToLower(b.Name), q))
}

// Explain renders the scoring breakdown as aligned lines for `oliver why`.
func (c *Candidate) Explain() []string {
	out := make([]string, 0, len(c.Reasons)+len(c.Blockers))
	for _, r := range c.Reasons {
		sign := " "
		if r.Delta > 0 {
			sign = "+"
		}
		if r.Delta == 0 {
			out = append(out, fmt.Sprintf("      %s", r.Text))
			continue
		}
		out = append(out, fmt.Sprintf("  %s%3d  %s", sign, r.Delta, r.Text))
	}
	for _, b := range c.Blockers {
		out = append(out, fmt.Sprintf("  BLOCKED  %s", b))
	}
	return out
}
