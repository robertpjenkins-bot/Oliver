package gamekit

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

// ExportHTML renders a game as one self-contained page.
//
// Everything is inlined: the engine, the project, every sprite and every
// sound. The result has no external requests at all, so it plays from a file://
// URL, from a USB stick, from a web host, or from inside an AppImage, and it
// keeps playing when whatever produced it is long gone.
func (g *Game) ExportHTML() ([]byte, error) {
	data, err := g.JSON()
	if err != nil {
		return nil, err
	}
	title := g.Name
	if g.Settings.Title != "" {
		title = g.Settings.Title
	}
	page := playerHTML
	page = strings.ReplaceAll(page, "__TITLE__", html.EscapeString(title))
	page = strings.ReplaceAll(page, "__NAME__", html.EscapeString(g.Name))
	page = strings.ReplaceAll(page, "__TAGLINE__", html.EscapeString(g.Tagline))
	// Order matters: the engine and the project are inserted last so that a
	// game whose text happens to contain a placeholder cannot rewrite the page.
	page = strings.Replace(page, "__ENGINE__", engineJS, 1)
	page = strings.Replace(page, "__GAME__", string(data), 1)
	return []byte(page), nil
}

// WriteHTML exports the game to a file, returning the path written.
func (g *Game) WriteHTML(path string) (string, error) {
	if path == "" {
		path = g.Slug() + ".html"
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, g.Slug()+".html")
	}
	data, err := g.ExportHTML()
	if err != nil {
		return "", err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Summary describes a game in the few lines a person needs to decide whether
// it is the one they meant to make.
func (g *Game) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n", g.Name, g.Tagline)
	fmt.Fprintf(&b, "%s, %d levels, %d objects, %d sprites, %d rules\n",
		g.Genre, len(g.Levels), len(g.Objects), len(g.Sprites), len(g.Rules))
	for _, l := range g.Levels {
		enemies, items := 0, 0
		for _, s := range l.Spawns {
			switch o := g.Object(s.Object); {
			case o == nil:
			case o.Kind == KindEnemy:
				enemies++
			case o.Collect || o.Kind == KindItem:
				items++
			}
		}
		fmt.Fprintf(&b, "  %-10s %-22s %3dx%-3d  %2d enemies  %2d to collect\n",
			l.ID, l.Name, l.Width, l.Height, enemies, items)
	}
	return b.String()
}

// IconSVG draws the game's own hero as its icon.
//
// A packaged game should not wear the toolkit's logo: the sprite the player
// spends the whole game looking at is the most recognisable thing about it,
// and it is already in the project.
func (g *Game) IconSVG() []byte {
	var s *Sprite
	if p := g.PlayerObject(); p != nil {
		s = g.Sprite(p.Sprite)
	}
	if s == nil && len(g.Sprites) > 0 {
		s = &g.Sprites[0]
	}
	background := "#101018"
	if len(g.Palette) > 15 {
		background = g.Palette[15]
	}
	var b strings.Builder
	w, h := 16, 16
	if s != nil {
		w, h = s.Width, s.Height
	}
	side := w
	if h > side {
		side = h
	}
	pad := side / 8
	box := side + pad*2
	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`+"\n", box, box)
	fmt.Fprintf(&b, `  <rect width="%d" height="%d" rx="%d" fill="%s"/>`+"\n", box, box, pad, background)
	if s != nil && len(s.Frames) > 0 {
		offX := pad + (side-w)/2
		offY := pad + (side-h)/2
		for y, row := range s.Frames[0] {
			for x, ch := range row {
				if ch == Transparent || ch == ' ' {
					continue
				}
				idx := int(hexValue(byte(ch)))
				if idx >= len(g.Palette) {
					continue
				}
				fmt.Fprintf(&b, `  <rect x="%d" y="%d" width="1" height="1" fill="%s"/>`+"\n",
					offX+x, offY+y, g.Palette[idx])
			}
		}
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}
