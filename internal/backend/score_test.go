package backend

import (
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/pe"
)

func testBackends() []*Backend {
	return []*Backend{
		{ID: "system-wine", Name: "wine", Kind: KindWine, Source: SourceSystem,
			Version: ParseVersion("wine-9.0"), Has32: true, Has64: true},
		{ID: "opt-wine-staging", Name: "wine-staging", Kind: KindWine, Source: SourceOpt,
			Variant: VariantStaging, Version: ParseVersion("wine-9.2"), Has32: true, Has64: true},
		{ID: "proton-ge", Name: "GE-Proton9-20", Kind: KindProton, Source: SourceCompatTools,
			Variant: VariantGE, Version: ParseVersion("GE-Proton9-20"), Has32: true, Has64: true,
			Launcher: []string{"/usr/bin/umu-run"}},
		{ID: "proton-valve", Name: "Proton 9.0", Kind: KindProton, Source: SourceSteam,
			Version: ParseVersion("Proton 9.0"), Has32: true, Has64: true,
			Launcher: []string{"/usr/bin/umu-run"}},
	}
}

func rankTop(t *testing.T, info *pe.Info, profile Profile) *Candidate {
	t.Helper()
	ranked := Rank(testBackends(), Request{Info: info, Profile: profile})
	if len(ranked) == 0 {
		t.Fatal("no candidates ranked")
	}
	return ranked[0]
}

func TestGamePrefersProton(t *testing.T) {
	info := &pe.Info{
		Kind: pe.KindPE, Arch: pe.ArchX64, GUI: true, LikelyGame: true,
		GraphicsAPIs: []string{"Direct3D 11"},
		GameHints:    []string{"Steamworks SDK alongside the executable"},
	}
	top := rankTop(t, info, ProfileAuto)
	if !top.Backend.IsProton() {
		t.Errorf("a Direct3D game should pick Proton, got %s", top.Backend.ID)
	}
	if top.Backend.ID != "proton-ge" {
		t.Errorf("GE should outrank stock Proton for games, got %s", top.Backend.ID)
	}
}

func TestPlainApplicationPrefersWine(t *testing.T) {
	info := &pe.Info{Kind: pe.KindPE, Arch: pe.ArchX64, GUI: true}
	top := rankTop(t, info, ProfileAuto)
	if top.Backend.Kind != KindWine {
		t.Errorf("an ordinary application should pick Wine, got %s (%s)", top.Backend.ID, top.Backend.Kind)
	}
}

func TestInstallerPrefersWineEvenWhenItLooksLikeAGame(t *testing.T) {
	info := &pe.Info{
		Kind: pe.KindPE, Arch: pe.ArchX86, GUI: true,
		IsInstaller: true, Installer: pe.InstallerInno,
		GraphicsAPIs: []string{"Direct3D 9"},
	}
	top := rankTop(t, info, ProfileAuto)
	if top.Backend.Kind != KindWine {
		t.Errorf("installers should run under Wine, got %s (%s)", top.Backend.ID, top.Backend.Kind)
	}

	// Even when the caller insists it is a game, the installer rule holds.
	top = rankTop(t, info, ProfileGame)
	if top.Backend.Kind != KindWine {
		t.Errorf("installer rule should survive an explicit game profile, got %s", top.Backend.ID)
	}
}

func TestThirtyTwoBitExecutableBlocksSixtyFourOnlyRuntimes(t *testing.T) {
	bs := []*Backend{
		{ID: "wine64only", Name: "wine", Kind: KindWine, Version: ParseVersion("wine-10.0"), Has32: false, Has64: true},
		{ID: "wine-old", Name: "wine-old", Kind: KindWine, Version: ParseVersion("wine-7.0"), Has32: true, Has64: true},
	}
	info := &pe.Info{Kind: pe.KindPE, Arch: pe.ArchX86, GUI: true}

	ranked := Rank(bs, Request{Info: info, Profile: ProfileAuto})
	if ranked[0].Backend.ID != "wine-old" {
		t.Errorf("expected the older 32-bit-capable build to win, got %s", ranked[0].Backend.ID)
	}
	// The newer runtime must still be listed, marked incompatible with a reason.
	last := ranked[len(ranked)-1]
	if last.Compatible || len(last.Blockers) == 0 {
		t.Errorf("64-bit-only runtime should be blocked with an explanation, got %+v", last)
	}

	best, err := Best(bs, Request{Info: info})
	if err != nil || best.Backend.ID != "wine-old" {
		t.Errorf("Best() = %v, %v", best, err)
	}
}

func TestBestFailsWhenNothingCanRunIt(t *testing.T) {
	bs := []*Backend{{ID: "wine64only", Name: "wine", Kind: KindWine, Version: ParseVersion("wine-10.0"), Has64: true}}
	info := &pe.Info{Kind: pe.KindPE, Arch: pe.ArchX86}
	if _, err := Best(bs, Request{Info: info}); err == nil {
		t.Fatal("expected an error when no runtime is compatible")
	}

	if _, err := Best(nil, Request{Info: info}); err == nil {
		t.Fatal("expected an error when no runtime is installed at all")
	}
}

func TestDOSBinaryIsRejectedWithAdvice(t *testing.T) {
	info := &pe.Info{Kind: pe.KindDOS, Arch: pe.ArchUnknown}
	ranked := Rank(testBackends(), Request{Info: info})
	for _, c := range ranked {
		if c.Compatible {
			t.Fatalf("%s should not claim to run a DOS binary", c.Backend.ID)
		}
	}
	if _, err := Best(testBackends(), Request{Info: info}); err == nil {
		t.Fatal("expected DOS binaries to be rejected")
	}
}

func TestPreferOverridesRankingButNotCompatibility(t *testing.T) {
	info := &pe.Info{Kind: pe.KindPE, Arch: pe.ArchX64, GUI: true} // an app: Wine would normally win
	ranked := Rank(testBackends(), Request{Info: info, Prefer: "proton-ge"})
	if ranked[0].Backend.ID != "proton-ge" {
		t.Errorf("explicit preference ignored, got %s", ranked[0].Backend.ID)
	}

	bs := []*Backend{{ID: "wine64only", Name: "wine", Kind: KindWine, Version: ParseVersion("wine-10.0"), Has64: true}}
	ranked = Rank(bs, Request{Info: &pe.Info{Kind: pe.KindPE, Arch: pe.ArchX86}, Prefer: "wine64only"})
	if ranked[0].Compatible {
		t.Error("a preference must not make an incompatible runtime compatible")
	}
}

func TestProtonWithoutUmuIsPenalised(t *testing.T) {
	withUmu := &Backend{ID: "p1", Name: "Proton 9.0", Kind: KindProton, Version: ParseVersion("Proton 9.0"),
		Has32: true, Has64: true, Launcher: []string{"/usr/bin/umu-run"}}
	withoutUmu := &Backend{ID: "p2", Name: "Proton 9.0", Kind: KindProton, Version: ParseVersion("Proton 9.0"),
		Has32: true, Has64: true}

	info := &pe.Info{Kind: pe.KindPE, Arch: pe.ArchX64, LikelyGame: true}
	req := Request{Info: info, Profile: ProfileGame}
	a := Rank([]*Backend{withUmu}, req)[0]
	b := Rank([]*Backend{withoutUmu}, req)[0]
	if a.Score <= b.Score {
		t.Errorf("umu-backed Proton should score higher: %d vs %d", a.Score, b.Score)
	}
}

func TestRankingIsDeterministic(t *testing.T) {
	info := &pe.Info{Kind: pe.KindPE, Arch: pe.ArchX64, LikelyGame: true}
	first := Rank(testBackends(), Request{Info: info})
	for i := 0; i < 20; i++ {
		next := Rank(testBackends(), Request{Info: info})
		for j := range first {
			if first[j].Backend.ID != next[j].Backend.ID || first[j].Score != next[j].Score {
				t.Fatalf("ranking is not stable at position %d: %s vs %s", j, first[j].Backend.ID, next[j].Backend.ID)
			}
		}
	}
}

func TestEveryCandidateCarriesAnExplanation(t *testing.T) {
	info := &pe.Info{Kind: pe.KindPE, Arch: pe.ArchX64, LikelyGame: true, GraphicsAPIs: []string{"Direct3D 12"}}
	for _, c := range Rank(testBackends(), Request{Info: info}) {
		if len(c.Explain()) == 0 {
			t.Errorf("%s has no explanation", c.Backend.ID)
		}
	}
}

func TestParseProfile(t *testing.T) {
	for _, in := range []string{"", "auto", "GAME", "app"} {
		if _, err := ParseProfile(in); err != nil {
			t.Errorf("ParseProfile(%q) failed: %v", in, err)
		}
	}
	if _, err := ParseProfile("nonsense"); err == nil {
		t.Error("expected an error for an unknown profile")
	}
}
