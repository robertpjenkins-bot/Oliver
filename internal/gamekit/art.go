package gamekit

import "math/rand"

// Palette roles. Art is written against these slots rather than against
// colours, so a theme can repaint an entire game — every sprite, every tile —
// by swapping sixteen strings.
const (
	cOutline = 0x0
	cLight   = 0x1
	cShade   = 0x2
	cMid     = 0x3
	cHot     = 0x4
	cWarm    = 0x5
	cBright  = 0x6
	cGreen   = 0x7
	cDeep    = 0x8
	cSky     = 0x9
	cBlue    = 0xa
	cPurple  = 0xb
	cPink    = 0xc
	cBrown   = 0xd
	cGold    = 0xe
	cBack    = 0xf
)

// art is a small grid of palette indices, the shape every sprite is drawn on.
type art struct {
	w, h int
	px   []byte
}

func newArt(w, h int) *art {
	a := &art{w: w, h: h, px: make([]byte, w*h)}
	for i := range a.px {
		a.px[i] = byte(Transparent)
	}
	return a
}

func hexDigit(i int) byte {
	const digits = "0123456789abcdef"
	if i < 0 || i > 15 {
		return byte(Transparent)
	}
	return digits[i]
}

func (a *art) set(x, y, color int) {
	if x < 0 || y < 0 || x >= a.w || y >= a.h {
		return
	}
	a.px[y*a.w+x] = hexDigit(color)
}

func (a *art) get(x, y int) byte {
	if x < 0 || y < 0 || x >= a.w || y >= a.h {
		return byte(Transparent)
	}
	return a.px[y*a.w+x]
}

func (a *art) empty(x, y int) bool { return a.get(x, y) == byte(Transparent) }

func (a *art) rect(x, y, w, h, color int) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			a.set(xx, yy, color)
		}
	}
}

// mirror copies the left half onto the right. Living things are symmetrical,
// and symmetry is most of what makes a handful of random pixels read as a
// creature rather than as noise.
func (a *art) mirror() {
	for y := 0; y < a.h; y++ {
		for x := 0; x < a.w/2; x++ {
			a.px[y*a.w+(a.w-1-x)] = a.px[y*a.w+x]
		}
	}
}

// outline draws a dark edge around everything solid, which is what keeps a
// sprite legible against a background of any colour.
func (a *art) outline() {
	edges := make([][2]int, 0, a.w*a.h/4)
	for y := 0; y < a.h; y++ {
		for x := 0; x < a.w; x++ {
			if !a.empty(x, y) {
				continue
			}
			if !a.empty(x-1, y) || !a.empty(x+1, y) || !a.empty(x, y-1) || !a.empty(x, y+1) {
				edges = append(edges, [2]int{x, y})
			}
		}
	}
	for _, e := range edges {
		a.set(e[0], e[1], cOutline)
	}
}

// shade darkens the bottom row of each column, a one-pass fake of a light
// source that costs nothing and makes flat shapes look solid.
func (a *art) shade(color int) {
	for x := 0; x < a.w; x++ {
		for y := a.h - 1; y >= 0; y-- {
			if a.empty(x, y) || a.get(x, y) == hexDigit(cOutline) {
				continue
			}
			a.set(x, y, color)
			break
		}
	}
}

func (a *art) rows() []string {
	out := make([]string, a.h)
	for y := 0; y < a.h; y++ {
		out[y] = string(a.px[y*a.w : (y+1)*a.w])
	}
	return out
}

func (a *art) clone() *art {
	b := newArt(a.w, a.h)
	copy(b.px, a.px)
	return b
}

// shift moves the whole image, used to make a walk cycle out of one drawing.
func (a *art) shift(dx, dy int) *art {
	b := newArt(a.w, a.h)
	for y := 0; y < a.h; y++ {
		for x := 0; x < a.w; x++ {
			if !a.empty(x, y) {
				b.set(x+dx, y+dy, int(hexValue(a.get(x, y))))
			}
		}
	}
	return b
}

func hexValue(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	}
	return 0
}

func sprite(id string, fps float64, frames ...*art) Sprite {
	s := Sprite{ID: id, FPS: fps}
	if len(frames) > 0 {
		s.Width, s.Height = frames[0].w, frames[0].h
	}
	for _, f := range frames {
		s.Frames = append(s.Frames, f.rows())
	}
	return s
}

// ---------------------------------------------------------------- characters

// heroArt draws a humanoid in the proportions retro heroes are drawn in:
// a head about a third of the height, a body as wide as the head, and legs
// that carry the walk. The measurements are fractions of the sprite size, so
// the same drawing works at 12, 16 or 24 pixels.
func heroArt(size int, body, trim, skin int, r *rand.Rand) (*art, *art) {
	a := newArt(size, size)
	unit := float64(size) / 16
	px := func(v float64) int { return int(v*unit + 0.5) }
	mid := size / 2

	headW, headH := px(7), px(5)
	headX, headY := mid-headW/2, px(1)
	bodyW, bodyH := px(8), px(5)
	bodyX, bodyY := mid-bodyW/2, headY+headH+px(1)
	legY := bodyY + bodyH
	legH := size - legY - px(1)

	// hair or helmet, then the face under it
	a.rect(headX, headY, headW, headH, skin)
	a.rect(headX, headY, headW, px(2), trim)
	a.set(headX+px(2), headY+px(3), cOutline)
	a.set(headX+headW-1-px(2), headY+px(3), cOutline)

	// neck, body and belt
	a.rect(mid-px(1), headY+headH, px(2), px(1), skin)
	a.rect(bodyX, bodyY, bodyW, bodyH, body)
	a.rect(bodyX, bodyY+bodyH-px(1), bodyW, px(1), trim)

	// arms down the sides, a shade darker than the body reads as behind it
	a.rect(bodyX-px(2), bodyY+px(1), px(2), bodyH-px(1), skin)
	a.rect(bodyX+bodyW, bodyY+px(1), px(2), bodyH-px(1), skin)

	// two legs with a gap between them
	legW := px(3)
	a.rect(mid-legW-px(1), legY, legW, legH, trim)
	a.rect(mid+px(1), legY, legW, legH, trim)
	a.rect(mid-legW-px(1), size-px(1)-1, legW+px(1), px(1)+1, cOutline)
	a.rect(mid+px(1)-px(1), size-px(1)-1, legW+px(1), px(1)+1, cOutline)
	a.outline()

	// The second frame swings the legs and drops one arm: enough to read as
	// walking, and cheap enough to draw for every character in the game.
	b := a.clone()
	for y := legY; y < size; y++ {
		for x := 0; x < size; x++ {
			b.set(x, y, -1)
		}
	}
	b.rect(mid-legW-px(2), legY+px(1), legW+px(1), legH-px(1), trim)
	b.rect(mid+px(1), legY, legW, legH, trim)
	b.rect(mid-legW-px(2), size-px(1)-1, legW+px(1), px(1)+1, cOutline)
	b.outline()
	_ = r
	return a, b
}

// creatureArt draws a symmetrical blob with eyes: the universal retro enemy.
func creatureArt(size int, body, eye int, r *rand.Rand, legs bool) (*art, *art) {
	a := newArt(size, size)
	half := size / 2
	// A random silhouette, mirrored: bumpy on top, wide at the base.
	for y := 2; y < size-2; y++ {
		width := half - 1
		if y < size/3 {
			width = half - 2 + r.Intn(2)
		}
		if y > size*2/3 {
			width = half - 1
		}
		for x := half - width; x < half; x++ {
			a.set(x, y, body)
		}
	}
	a.mirror()
	// eyes
	eyeY := size/2 - 1
	a.set(half-3, eyeY, cLight)
	a.set(half+2, eyeY, cLight)
	a.set(half-3, eyeY, eye)
	a.set(half+2, eyeY, eye)
	// mouth
	for x := half - 2; x <= half+1; x++ {
		a.set(x, size-5, cOutline)
	}
	if legs {
		a.set(half-3, size-2, body)
		a.set(half+2, size-2, body)
	}
	a.shade(cOutline)
	a.outline()

	b := a.shift(0, -1)
	if legs {
		b.set(half-4, size-2, body)
		b.set(half+3, size-2, body)
		b.outline()
	}
	return a, b
}

// shipArt draws a craft pointing up, for shooters.
func shipArt(size, body, trim, flame int) (*art, *art) {
	a := newArt(size, size)
	half := size / 2
	for y := 2; y < size-3; y++ {
		width := (y - 1) / 2
		for x := half - width; x < half; x++ {
			a.set(x, y, body)
		}
	}
	a.mirror()
	a.rect(half-1, 1, 2, 3, trim)
	a.rect(1, size-5, 2, 3, trim)
	a.rect(size-3, size-5, 2, 3, trim)
	a.outline()

	b := a.clone()
	b.rect(half-1, size-3, 2, 2, flame)
	b.outline()
	a.rect(half-1, size-3, 2, 1, flame)
	a.outline()
	return a, b
}

// ------------------------------------------------------------------- objects

// coinArt draws a spinning pickup: a disc that narrows to an edge and back.
// Four frames is the whole animation, and it is the most satisfying twenty
// pixels in games.
func coinArt(size, face, edge int) []*art {
	fractions := []float64{1.0, 0.62, 0.18, 0.62}
	frames := make([]*art, 0, len(fractions))
	inset := size / 8
	ry := float64(size)/2 - float64(inset)
	for _, f := range fractions {
		a := newArt(size, size)
		rx := ry * f
		cx, cy := float64(size)/2-0.5, float64(size)/2-0.5
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx, dy := (float64(x)-cx)/maxF(rx, 0.6), (float64(y)-cy)/ry
				d := dx*dx + dy*dy
				if d > 1 {
					continue
				}
				// A brighter core inside a darker rim gives the disc a face.
				if d < 0.35 {
					a.set(x, y, edge)
				} else {
					a.set(x, y, face)
				}
			}
		}
		a.outline()
		frames = append(frames, a)
	}
	return frames
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func gemArt(size, face, shine int) []*art {
	a := newArt(size, size)
	half := size / 2
	for y := 2; y < size-2; y++ {
		width := y - 1
		if y > half {
			width = size - y - 1
		}
		if width < 1 {
			width = 1
		}
		for x := half - width; x < half; x++ {
			a.set(x, y, face)
		}
	}
	a.mirror()
	a.set(half-2, half-1, shine)
	a.set(half-1, half-2, shine)
	a.outline()
	b := a.clone()
	b.set(half+1, half, shine)
	b.outline()
	return []*art{a, b}
}

func bulletArt(size, core, glow int) *art {
	a := newArt(size, size)
	c := size / 2
	a.rect(c-1, c-2, 2, 4, core)
	a.rect(c-2, c-1, 4, 2, core)
	a.set(c-1, c-1, glow)
	a.outline()
	return a
}

func ballArt(size, body, shine int) *art {
	a := newArt(size, size)
	c := size / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := float64(x-c)+0.5, float64(y-c)+0.5
			if dx*dx+dy*dy <= float64(c*c) {
				a.set(x, y, body)
			}
		}
	}
	a.set(c-1, c-1, shine)
	a.outline()
	return a
}

func paddleArt(w, h, body, trim int) *art {
	a := newArt(w, h)
	a.rect(0, 1, w, h-1, body)
	a.rect(0, 1, w, 1, trim)
	a.outline()
	return a
}

func brickArt(w, h, body, trim int) *art {
	a := newArt(w, h)
	a.rect(0, 0, w, h, body)
	a.rect(0, 0, w, 1, trim)
	a.rect(0, h-1, w, 1, cOutline)
	a.rect(w-1, 0, 1, h, cOutline)
	return a
}

// --------------------------------------------------------------------- tiles

// groundArt draws a solid block with a lit top edge and a scatter of darker
// pixels, so a wall of them does not look like graph paper.
func groundArt(size, top, body, speck int, r *rand.Rand) *art {
	a := newArt(size, size)
	a.rect(0, 0, size, size, body)
	a.rect(0, 0, size, 2, top)
	a.rect(0, 2, size, 1, cOutline)
	for i := 0; i < size; i++ {
		a.set(r.Intn(size), 3+r.Intn(size-3), speck)
	}
	a.rect(0, size-1, size, 1, cOutline)
	return a
}

func stoneArt(size, body, mortar, speck int, r *rand.Rand) *art {
	a := newArt(size, size)
	a.rect(0, 0, size, size, body)
	rowH := size / 4
	for y := 0; y < size; y += rowH {
		a.rect(0, y, size, 1, mortar)
		offset := 0
		if (y/rowH)%2 == 1 {
			offset = size / 2
		}
		for x := offset; x < size+offset; x += size / 2 {
			a.rect(x%size, y, 1, rowH, mortar)
		}
	}
	for i := 0; i < size/2; i++ {
		a.set(r.Intn(size), r.Intn(size), speck)
	}
	return a
}

func spikeArt(size, body, tip int) *art {
	a := newArt(size, size)
	spikes := size / 4
	for s := 0; s < spikes; s++ {
		baseX := s * 4
		for y := 0; y < size; y++ {
			width := (y * 2) / size
			for x := baseX + 1 - width; x <= baseX+2+width; x++ {
				if x >= baseX && x < baseX+4 {
					a.set(x, y, body)
				}
			}
		}
		a.set(baseX+1, 0, tip)
		a.set(baseX+2, 0, tip)
	}
	a.rect(0, size-2, size, 2, cOutline)
	return a
}

func doorArt(size, frame, panel, handle int) *art {
	a := newArt(size, size)
	a.rect(1, 1, size-2, size-1, frame)
	a.rect(2, 3, size-4, size-4, panel)
	a.set(size-4, size/2, handle)
	a.outline()
	return a
}

func platformArt(size, top, body int) *art {
	a := newArt(size, size)
	a.rect(0, 0, size, 3, body)
	a.rect(0, 0, size, 1, top)
	a.rect(0, 3, size, 1, cOutline)
	return a
}

func waterArt(size, body, foam int, r *rand.Rand) []*art {
	frames := make([]*art, 0, 2)
	for f := 0; f < 2; f++ {
		a := newArt(size, size)
		a.rect(0, 1, size, size-1, body)
		for x := 0; x < size; x++ {
			y := 0
			if (x+f*2)%4 < 2 {
				y = 1
			}
			a.set(x, y, foam)
		}
		for i := 0; i < size/3; i++ {
			a.set(r.Intn(size), 3+r.Intn(size-4), foam)
		}
		frames = append(frames, a)
	}
	return frames
}

func bushArt(size, body, dark int, r *rand.Rand) *art {
	a := newArt(size, size)
	for y := size / 3; y < size; y++ {
		width := size/2 - 1
		if y < size/2 {
			width = size/2 - 3 + r.Intn(2)
		}
		for x := size/2 - width; x < size/2; x++ {
			a.set(x, y, body)
		}
	}
	a.mirror()
	for i := 0; i < size/2; i++ {
		a.set(r.Intn(size), size/2+r.Intn(size/2), dark)
	}
	a.outline()
	return a
}

// starArt is a one-pixel-plus decoration for space backgrounds.
func starArt(size, color int, r *rand.Rand) *art {
	a := newArt(size, size)
	for i := 0; i < 3; i++ {
		a.set(r.Intn(size), r.Intn(size), color)
	}
	c := size / 2
	a.set(c, c, cLight)
	a.set(c-1, c, color)
	a.set(c+1, c, color)
	a.set(c, c-1, color)
	a.set(c, c+1, color)
	return a
}

// pelletArt is the small dot a maze game is full of.
func pelletArt(size, color int) *art {
	a := newArt(size, size)
	c := size / 2
	a.rect(c-1, c-1, 2, 2, color)
	a.outline()
	return a
}

// flagArt marks the end of a level when a door would not suit.
func flagArt(size, pole, cloth int) *art {
	a := newArt(size, size)
	a.rect(size/2-1, 1, 2, size-1, pole)
	for y := 2; y < size/2; y++ {
		w := size/2 - 1 - (y-2)/3
		a.rect(size/2+1, y, w, 1, cloth)
	}
	a.outline()
	return a
}

// powerArt draws an upgrade: a glowing orb with a chevron inside it. Upgrades
// have to read as different from coins at a glance, because picking one up is
// the most important thing that happens in an exploration game.
func powerArt(size, core, glow, shell int) []*art {
	frames := make([]*art, 0, 2)
	for f := 0; f < 2; f++ {
		a := newArt(size, size)
		c := float64(size)/2 - 0.5
		r := float64(size)/2 - 2
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx, dy := float64(x)-c, float64(y)-c
				d := dx*dx + dy*dy
				if d > r*r {
					continue
				}
				if d > (r-1.6)*(r-1.6) {
					a.set(x, y, shell)
				} else {
					a.set(x, y, core)
				}
			}
		}
		// a chevron pointing up, brighter on the second frame
		mid := size / 2
		colour := glow
		if f == 1 {
			colour = cLight
		}
		for i := 0; i < size/4; i++ {
			a.set(mid-1-i, mid+i-1, colour)
			a.set(mid+i, mid+i-1, colour)
		}
		a.outline()
		frames = append(frames, a)
	}
	return frames
}

// keyArt draws a keycard: the other half of a locked door.
func keyArt(size, body, stripe int) *art {
	a := newArt(size, size)
	w, h := size-4, size-7
	x, y := 2, (size-h)/2
	a.rect(x, y, w, h, body)
	a.rect(x, y+1, w, 2, stripe)
	a.rect(x+w-4, y+h-3, 3, 2, stripe)
	a.outline()
	return a
}

// gateArt draws a locked door: a heavy frame with a coloured light on it, so
// which key it wants is visible from across the room.
func gateArt(size, frame, panel, light int) *art {
	a := newArt(size, size)
	a.rect(0, 0, size, size, frame)
	a.rect(2, 1, size-4, size-2, panel)
	for y := 1; y < size-1; y += 3 {
		a.rect(2, y, size-4, 1, frame)
	}
	c := size / 2
	a.rect(c-2, c-2, 4, 4, light)
	a.rect(c-1, c-1, 2, 2, cLight)
	return a
}
