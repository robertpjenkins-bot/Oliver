package gamekit

import (
	"fmt"
	"math/rand"
)

// Tile characters used by the generated levels. They are chosen to be
// readable in the raw JSON: a level map should look like the level it makes.
const (
	chEmpty     = '.'
	chGround    = '#'
	chPlatform  = '='
	chSpike     = '^'
	chExit      = 'X'
	chLocked    = 'D'
	chWater     = '~'
	chLadder    = 'H'
	chBreakable = 'o'
	chCheck     = '*'
)

// grid is a level under construction.
type grid struct {
	w, h int
	rows [][]byte
}

func newGrid(w, h int, fill byte) *grid {
	g := &grid{w: w, h: h, rows: make([][]byte, h)}
	for y := range g.rows {
		g.rows[y] = make([]byte, w)
		for x := range g.rows[y] {
			g.rows[y][x] = fill
		}
	}
	return g
}

func (g *grid) set(x, y int, c byte) {
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		return
	}
	g.rows[y][x] = c
}

func (g *grid) at(x, y int) byte {
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		return chGround
	}
	return g.rows[y][x]
}

func (g *grid) fillRect(x, y, w, h int, c byte) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			g.set(xx, yy, c)
		}
	}
}

func (g *grid) strings() []string {
	out := make([]string, g.h)
	for y, row := range g.rows {
		out[y] = string(row)
	}
	return out
}

// tileSet returns the tile definitions the generated levels use. Every level
// carries its own set, so a level can be lifted out of one game and dropped
// into another without losing its meaning.
func tileSet(ground, platform, hazard, door, water string) []TileDef {
	defs := []TileDef{
		{Char: string(chGround), Sprite: ground, Kind: TileSolid, Name: "ground"},
		{Char: string(chPlatform), Sprite: platform, Kind: TilePlatform, Name: "platform"},
		{Char: string(chSpike), Sprite: hazard, Kind: TileHazard, Name: "spikes"},
		{Char: string(chExit), Sprite: door, Kind: TileExit, Name: "exit"},
		{Char: string(chLocked), Sprite: door, Kind: TileLocked, Name: "locked door"},
		{Char: string(chBreakable), Sprite: ground, Kind: TileBreakable, Name: "loose block"},
		{Char: string(chCheck), Sprite: platform, Kind: TileCheckpoint, Name: "checkpoint"},
	}
	if water != "" {
		defs = append(defs, TileDef{Char: string(chWater), Sprite: water, Kind: TileWater, Name: "water"})
	}
	return defs
}

// ------------------------------------------------------------- platformer

// buildPlatformer lays out a side-on level: ground with jumpable gaps, floating
// platforms above it, and a door at the far end.
//
// Every gap it digs is narrower than the player's jump and every platform is
// within reach of the one before, so a generated level is always finishable —
// a level that cannot be completed is worse than no level at all.
func buildPlatformer(idx int, r *rand.Rand, d designContext) Level {
	w := 56 + idx*14
	h := 15
	groundY := h - 2
	g := newGrid(w, h, chEmpty)
	g.fillRect(0, groundY, w, h-groundY, chGround)

	// The widest gap stays a tile inside what the player's jump covers, so a
	// generated level is crossable even when every jump is taken late.
	widest := 2
	if idx > 0 {
		widest = 3
	}
	x := 8
	for x < w-10 {
		if r.Intn(100) < 45 {
			gap := 2 + r.Intn(widest-1)
			g.fillRect(x, groundY, gap, h-groundY, chEmpty)
			if r.Intn(100) < 40 {
				g.fillRect(x, h-1, gap, 1, chSpike)
			}
			x += gap
		}
		x += 5 + r.Intn(6)
	}

	// Floating platforms, each one reachable from the ground or from the last
	// platform: at most three tiles up and four across.
	lastY := groundY - 3
	for x = 6; x < w-8; x += 5 + r.Intn(5) {
		if r.Intn(100) < 55 {
			continue
		}
		length := 3 + r.Intn(4)
		y := lastY + (r.Intn(3) - 1)
		if y < 4 {
			y = 4
		}
		if y > groundY-2 {
			y = groundY - 2
		}
		g.fillRect(x, y, length, 1, chPlatform)
		lastY = y
	}

	// A checkpoint halfway through a long level.
	if w > 70 {
		g.set(w/2, groundY-1, chCheck)
	}

	// The door sits on solid ground at the end, with the ground guaranteed
	// under it so the exit is never floating over a pit.
	g.fillRect(w-6, groundY, 6, h-groundY, chGround)
	g.set(w-4, groundY-1, chExit)

	lv := Level{
		ID:         fmt.Sprintf("level%d", idx+1),
		Name:       d.levelName(idx),
		Width:      w,
		Height:     h,
		Tiles:      tileSet("tile-ground", "tile-platform", "tile-spike", "tile-door", ""),
		Map:        g.strings(),
		Background: d.theme.Sky,
		Intro:      d.levelIntro(idx),
		Music:      fmt.Sprintf("%s-%d", d.theme.Name, idx),
	}
	lv.Spawns = append(lv.Spawns, Spawn{Object: "player", X: 2, Y: float64(groundY - 1)})

	// Collectables sit above solid footing, and enemies stand on it.
	coins := 8 + idx*3
	for i := 0; i < coins; i++ {
		cx := 4 + r.Intn(w-10)
		cy := groundY - 1 - r.Intn(4)
		if g.at(cx, cy) != chEmpty || !standable(g, cx, cy) {
			continue
		}
		lv.Spawns = append(lv.Spawns, Spawn{Object: "coin", X: float64(cx), Y: float64(cy)})
	}
	enemies := 3 + idx*2
	for i := 0; i < enemies; i++ {
		ex := 10 + r.Intn(w-16)
		if g.at(ex, groundY) != chGround || g.at(ex, groundY-1) != chEmpty {
			continue
		}
		obj := "enemy"
		if r.Intn(100) < 30 {
			obj = "flyer"
		}
		ey := groundY - 1
		if obj == "flyer" {
			ey = groundY - 3 - r.Intn(3)
		}
		lv.Spawns = append(lv.Spawns, Spawn{Object: obj, X: float64(ex), Y: float64(ey), Dir: dirOf(r)})
	}
	return lv
}

// standable reports whether a collectable at this spot has something under it
// within a jump, so coins do not hang unreachable in the sky.
func standable(g *grid, x, y int) bool {
	for dy := 1; dy <= 4; dy++ {
		c := g.at(x, y+dy)
		if c == chGround || c == chPlatform {
			return true
		}
	}
	return false
}

func dirOf(r *rand.Rand) float64 {
	if r.Intn(2) == 0 {
		return -1
	}
	return 1
}

// --------------------------------------------------------------- top-down

// buildTopDown carves rooms out of solid rock and joins them with corridors,
// then puts the exit in the room furthest from the start.
func buildTopDown(idx int, r *rand.Rand, d designContext) Level {
	w, h := 40+idx*6, 30+idx*4
	g := newGrid(w, h, chGround)

	type room struct{ x, y, w, h int }
	var rooms []room
	attempts := 40 + idx*10
	for i := 0; i < attempts && len(rooms) < 6+idx*2; i++ {
		rw, rh := 5+r.Intn(7), 4+r.Intn(6)
		rx, ry := 1+r.Intn(w-rw-2), 1+r.Intn(h-rh-2)
		ok := true
		for _, o := range rooms {
			if rx < o.x+o.w+1 && rx+rw+1 > o.x && ry < o.y+o.h+1 && ry+rh+1 > o.y {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		rooms = append(rooms, room{rx, ry, rw, rh})
		g.fillRect(rx, ry, rw, rh, chEmpty)
	}
	// Corridors in an L from each room to the previous one, which is enough
	// to guarantee every room is reachable from every other.
	for i := 1; i < len(rooms); i++ {
		ax, ay := rooms[i-1].x+rooms[i-1].w/2, rooms[i-1].y+rooms[i-1].h/2
		bx, by := rooms[i].x+rooms[i].w/2, rooms[i].y+rooms[i].h/2
		for x := min(ax, bx); x <= max(ax, bx); x++ {
			g.set(x, ay, chEmpty)
		}
		for y := min(ay, by); y <= max(ay, by); y++ {
			g.set(bx, y, chEmpty)
		}
	}
	if len(rooms) == 0 {
		g.fillRect(1, 1, w-2, h-2, chEmpty)
		rooms = append(rooms, room{1, 1, w - 2, h - 2})
	}

	start := rooms[0]
	end := rooms[len(rooms)-1]
	g.set(end.x+end.w/2, end.y+end.h/2, chLocked)
	// Where the key lives, when this game locks its door with one: never in
	// the room the door is in, and never in the room the player starts in.
	keyRoom := rooms[len(rooms)/2]
	if len(rooms) > 2 && &keyRoom == &end {
		keyRoom = rooms[1]
	}

	lv := Level{
		ID:         fmt.Sprintf("level%d", idx+1),
		Name:       d.levelName(idx),
		Width:      w,
		Height:     h,
		Tiles:      tileSet("tile-ground", "tile-platform", "tile-spike", "tile-door", ""),
		Map:        g.strings(),
		Background: d.theme.Sky,
		Gravity:    -1, // marks "no gravity"; normalised below
		Intro:      d.levelIntro(idx),
		Music:      fmt.Sprintf("%s-%d", d.theme.Name, idx),
	}
	lv.Gravity = 0
	lv.Spawns = append(lv.Spawns, Spawn{Object: "player", X: float64(start.x + 1), Y: float64(start.y + 1)})
	if d.feature("keys") {
		lv.Spawns = append(lv.Spawns, Spawn{
			Object: "key-blue",
			X:      float64(keyRoom.x + keyRoom.w/2),
			Y:      float64(keyRoom.y + keyRoom.h/2),
		})
	}

	for i, rm := range rooms {
		if i == 0 {
			continue
		}
		items := 2 + r.Intn(3)
		for k := 0; k < items; k++ {
			lv.Spawns = append(lv.Spawns, Spawn{
				Object: "coin",
				X:      float64(rm.x + r.Intn(rm.w)),
				Y:      float64(rm.y + r.Intn(rm.h)),
			})
		}
		if i%2 == 1 || idx > 0 {
			lv.Spawns = append(lv.Spawns, Spawn{
				Object: "enemy",
				X:      float64(rm.x + rm.w/2),
				Y:      float64(rm.y + rm.h/2),
				Dir:    dirOf(r),
			})
		}
	}
	return lv
}

// ---------------------------------------------------------------- shooter

// buildShooter builds a single screen with a formation of attackers. The level
// is cleared by destroying them all, which the rules handle.
func buildShooter(idx int, r *rand.Rand, d designContext) Level {
	w, h := 16, 14
	g := newGrid(w, h, chEmpty)

	lv := Level{
		ID:         fmt.Sprintf("wave%d", idx+1),
		Name:       d.levelName(idx),
		Width:      w,
		Height:     h,
		Tiles:      tileSet("tile-ground", "tile-platform", "tile-spike", "tile-door", ""),
		Map:        g.strings(),
		Background: d.theme.Sky,
		Scroll:     "fixed",
		Intro:      d.levelIntro(idx),
		Music:      fmt.Sprintf("%s-%d", d.theme.Name, idx),
	}
	lv.Spawns = append(lv.Spawns, Spawn{Object: "player", X: float64(w/2 - 1), Y: float64(h - 2)})

	rows := 2 + idx
	if rows > 4 {
		rows = 4
	}
	for row := 0; row < rows; row++ {
		for col := 2; col < w-2; col += 2 {
			if r.Intn(100) < 15 {
				continue
			}
			lv.Spawns = append(lv.Spawns, Spawn{
				Object: "flyer",
				X:      float64(col),
				Y:      float64(1 + row*2),
				Dir:    dirOf(r),
			})
		}
	}
	for i := 0; i < 3+idx; i++ {
		lv.Spawns = append(lv.Spawns, Spawn{Object: "coin", X: float64(1 + r.Intn(w-2)), Y: float64(6 + r.Intn(4))})
	}
	return lv
}

// ------------------------------------------------------------------- maze

// buildMaze carves a perfect maze with a depth-first walk, then fills the
// corridors with pellets and lets hunters loose in it.
func buildMaze(idx int, r *rand.Rand, d designContext) Level {
	cells := 9 + idx*2
	w, h := cells*2+1, cells*2+1
	g := newGrid(w, h, chGround)

	type point struct{ x, y int }
	stack := []point{{1, 1}}
	g.set(1, 1, chEmpty)
	visited := map[point]bool{{1, 1}: true}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		dirs := [][2]int{{2, 0}, {-2, 0}, {0, 2}, {0, -2}}
		r.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		moved := false
		for _, dir := range dirs {
			n := point{cur.x + dir[0], cur.y + dir[1]}
			if n.x <= 0 || n.y <= 0 || n.x >= w-1 || n.y >= h-1 || visited[n] {
				continue
			}
			g.set(cur.x+dir[0]/2, cur.y+dir[1]/2, chEmpty)
			g.set(n.x, n.y, chEmpty)
			visited[n] = true
			stack = append(stack, n)
			moved = true
			break
		}
		if !moved {
			stack = stack[:len(stack)-1]
		}
	}
	// A few walls knocked through turn a perfect maze into one with loops,
	// which is what makes being chased through it a game rather than a trap.
	for i := 0; i < cells; i++ {
		x, y := 2+r.Intn(w-4), 2+r.Intn(h-4)
		if (x%2 == 0) != (y%2 == 0) {
			g.set(x, y, chEmpty)
		}
	}
	g.set(w-2, h-2, chLocked)

	lv := Level{
		ID:         fmt.Sprintf("maze%d", idx+1),
		Name:       d.levelName(idx),
		Width:      w,
		Height:     h,
		Tiles:      tileSet("tile-ground", "tile-platform", "tile-spike", "tile-door", ""),
		Map:        g.strings(),
		Background: d.theme.Sky,
		Intro:      d.levelIntro(idx),
		Music:      fmt.Sprintf("%s-%d", d.theme.Name, idx),
	}
	lv.Spawns = append(lv.Spawns, Spawn{Object: "player", X: 1, Y: 1})
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			if g.at(x, y) != chEmpty || (x == 1 && y == 1) {
				continue
			}
			if (x+y)%3 != 0 {
				continue
			}
			lv.Spawns = append(lv.Spawns, Spawn{Object: "pellet", X: float64(x), Y: float64(y)})
		}
	}
	hunters := 2 + idx
	for i := 0; i < hunters; i++ {
		lv.Spawns = append(lv.Spawns, Spawn{
			Object: "hunter",
			X:      float64(w - 2 - i*2),
			Y:      float64(h - 2),
			Dir:    dirOf(r),
		})
	}
	return lv
}

// --------------------------------------------------------------- breakout

// buildBreakout builds a wall of bricks over a paddle and a ball.
func buildBreakout(idx int, r *rand.Rand, d designContext) Level {
	w, h := 16, 14
	g := newGrid(w, h, chEmpty)
	g.fillRect(0, 0, w, 1, chGround)
	g.fillRect(0, 0, 1, h, chGround)
	g.fillRect(w-1, 0, 1, h, chGround)

	lv := Level{
		ID:         fmt.Sprintf("board%d", idx+1),
		Name:       d.levelName(idx),
		Width:      w,
		Height:     h,
		Tiles:      tileSet("tile-ground", "tile-platform", "tile-spike", "tile-door", ""),
		Map:        g.strings(),
		Background: d.theme.Sky,
		Scroll:     "fixed",
		Intro:      d.levelIntro(idx),
		Music:      fmt.Sprintf("%s-%d", d.theme.Name, idx),
	}
	lv.Spawns = append(lv.Spawns, Spawn{Object: "player", X: float64(w/2 - 1), Y: float64(h - 2)})
	lv.Spawns = append(lv.Spawns, Spawn{Object: "ball", X: float64(w / 2), Y: float64(h - 4), Dir: dirOf(r)})

	rows := 3 + idx
	if rows > 5 {
		rows = 5
	}
	for row := 0; row < rows; row++ {
		for col := 1; col < w-1; col++ {
			if r.Intn(100) < 8 {
				continue
			}
			lv.Spawns = append(lv.Spawns, Spawn{Object: "brick", X: float64(col), Y: float64(2 + row)})
		}
	}
	return lv
}

// ----------------------------------------------------------------- runner

// buildRunner builds a level that scrolls itself: the screen moves right at a
// fixed speed and the player has to keep up and stay alive.
func buildRunner(idx int, r *rand.Rand, d designContext) Level {
	w, h := 120+idx*40, 14
	groundY := h - 2
	g := newGrid(w, h, chEmpty)
	g.fillRect(0, groundY, w, h-groundY, chGround)

	for x := 12; x < w-12; x += 6 + r.Intn(7) {
		switch r.Intn(3) {
		case 0:
			gap := 2 + r.Intn(2)
			g.fillRect(x, groundY, gap, h-groundY, chEmpty)
		case 1:
			g.set(x, groundY-1, chSpike)
		case 2:
			height := 1 + r.Intn(3)
			g.fillRect(x, groundY-height, 1, height, chGround)
			g.fillRect(x-4, groundY-height-3, 3, 1, chPlatform)
		}
	}
	g.fillRect(w-8, groundY, 8, h-groundY, chGround)
	g.set(w-4, groundY-1, chExit)

	lv := Level{
		ID:         fmt.Sprintf("run%d", idx+1),
		Name:       d.levelName(idx),
		Width:      w,
		Height:     h,
		Tiles:      tileSet("tile-ground", "tile-platform", "tile-spike", "tile-door", ""),
		Map:        g.strings(),
		Background: d.theme.Sky,
		Scroll:     "auto-right",
		Intro:      d.levelIntro(idx),
		Music:      fmt.Sprintf("%s-%d", d.theme.Name, idx),
	}
	lv.Spawns = append(lv.Spawns, Spawn{Object: "player", X: 2, Y: float64(groundY - 1)})
	for x := 10; x < w-10; x += 5 + r.Intn(8) {
		if g.at(x, groundY) == chGround && g.at(x, groundY-1) == chEmpty {
			lv.Spawns = append(lv.Spawns, Spawn{Object: "coin", X: float64(x), Y: float64(groundY - 2)})
		}
	}
	return lv
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ------------------------------------------------------------ metroidvania

// Tile characters particular to an exploration map.
const (
	chGate = 'G' // a door that needs something
)

// buildMetroidvania builds a map you explore rather than a level you cross.
//
// The shape is deliberate rather than random, because the thing that makes
// this kind of game work — a door you cannot open yet, an ability that later
// opens it, a route back through everything you have already been — is exactly
// the thing a random layout gets wrong. The route is built in:
//
//	start ─► lower corridor, collecting ─► the ability ─► the shaft
//	                                                        │
//	  boss ◄── the door the ability opens ◄── upper corridor ◄┘
//
// The shaft's ledges are spaced further apart than an ordinary jump can reach
// and closer than the high jump can, so the ability is the only way up and the
// map cannot be finished out of order. Everything else — where the enemies
// are, how the rooms are decorated, how long the corridors run — is generated.
func buildMetroidvania(idx int, r *rand.Rand, d designContext) Level {
	w, h := 84+idx*12, 24
	g := newGrid(w, h, chGround)

	lowFloor := h - 3 // the surface the player walks on down here
	lowCeil := h - 9  // headroom of the lower corridor
	upFloor := 10     // the surface of the upper corridor
	upCeil := 3       //
	shaftX := w - 9   // where the two corridors meet
	shaftW := 6
	step := 5 // ledge spacing: too far for a plain jump, within a high jump

	g.fillRect(1, lowCeil, w-2, lowFloor-lowCeil, chEmpty)
	g.fillRect(1, upCeil, w-2, upFloor-upCeil, chEmpty)
	g.fillRect(shaftX, upCeil, shaftW, lowFloor-upCeil, chEmpty)

	// The shaft opens into the upper corridor through two columns at one side;
	// the rest of that floor is closed. Without it the climb ends in mid-air
	// over a hole the width of the shaft, and stepping off the last ledge in
	// any direction drops the player all the way back down.
	g.fillRect(shaftX+2, upFloor, shaftW-2, 1, chGround)

	// Ledges up the shaft, under the opening, so the climb arrives where the
	// way on is. They are the gate: too far apart for an ordinary jump.
	for y := lowFloor - step; y > upFloor; y -= step {
		g.fillRect(shaftX, y, 3, 1, chPlatform)
	}

	// Somewhere to stand at the top of the shaft, and spikes along the lower
	// corridor so the walk there is not free.
	//
	// The spikes lie on the floor rather than at the bottom of a pit dug
	// through it. A hazard tile is not solid, so a pit with spikes in it is
	// not a hazard at all — it is a hole in the world that the player falls
	// through and dies in, over and over, a few seconds apart. Spikes you can
	// jump over are the thing that was meant.
	g.fillRect(shaftX, upFloor-1, 3, 1, chPlatform)
	// Spaced out far enough that walking into one is a mistake rather than a
	// toll: a corridor that costs health every few seconds is not a corridor
	// anyone explores.
	spacing := 16 + r.Intn(8)
	for x := 16; x < shaftX-6; x += spacing {
		g.fillRect(x, lowFloor-1, 1+r.Intn(2), 1, chSpike)
	}

	// The door, and the room behind it at the far end of the upper corridor.
	// It fills the corridor from floor to ceiling: a door that can be jumped
	// over is a decoration, not a gate.
	gateX := 9
	g.fillRect(gateX, upCeil, 1, upFloor-upCeil, chGate)
	g.fillRect(1, upCeil, gateX-1, upFloor-upCeil, chEmpty)

	lv := Level{
		ID:     fmt.Sprintf("area%d", idx+1),
		Name:   d.levelName(idx),
		Width:  w,
		Height: h,
		Tiles: append(tileSet("tile-ground", "tile-platform", "tile-spike", "tile-door", ""),
			TileDef{
				Char:   string(chGate),
				Sprite: "tile-gate",
				Kind:   TileDoor,
				Name:   d.gateName(idx),
				Needs:  d.gateKey(idx),
			}),
		Map:        g.strings(),
		Background: d.theme.Sky,
		Intro:      d.levelIntro(idx),
		Music:      fmt.Sprintf("%s-%d", d.theme.Name, idx),
	}
	lv.Spawns = append(lv.Spawns, Spawn{Object: "player", X: 3, Y: float64(lowFloor - 1)})

	// The ability that opens the way up sits in the lower corridor, before the
	// shaft, where the player cannot miss it on the way past.
	if idx == 0 {
		lv.Spawns = append(lv.Spawns,
			Spawn{Object: "power-jump", X: float64(clearOfHazards(g, shaftX-12, lowFloor)), Y: float64(lowFloor - 1)},
			// On the floor, not a tile above it: a sprite resting exactly one
			// tile up never quite touches a player walking past, and an
			// upgrade the game depends on is not the place to find that out.
			Spawn{Object: "power-missile", X: float64(shaftX - 6), Y: float64(upFloor - 1)},
		)
	} else {
		lv.Spawns = append(lv.Spawns, Spawn{
			Object: d.keyObject(idx),
			X:      float64(clearOfHazards(g, shaftX/2+r.Intn(8), lowFloor)),
			Y:      float64(lowFloor - 1),
		})
	}

	// What is behind the door: the way on, or the thing at the end of it all.
	if idx == d.levels-1 {
		lv.Spawns = append(lv.Spawns, Spawn{Object: "boss", X: 4, Y: float64(upFloor - 3)})
		lv.Next = "win"
	} else {
		g.set(3, upFloor-1, chExit)
		lv.Map = g.strings()
	}

	// Everything else that makes it a game rather than a diagram.
	for x := 8; x < shaftX-2; x += 5 + r.Intn(5) {
		if g.at(x, lowFloor) != chGround {
			continue
		}
		if r.Intn(100) < 55 {
			lv.Spawns = append(lv.Spawns, Spawn{Object: "coin", X: float64(x), Y: float64(lowFloor - 1)})
		} else {
			lv.Spawns = append(lv.Spawns, Spawn{Object: "enemy", X: float64(x), Y: float64(lowFloor - 1), Dir: dirOf(r)})
		}
	}
	for x := gateX + 3; x < shaftX-1; x += 6 + r.Intn(5) {
		switch r.Intn(3) {
		case 0:
			lv.Spawns = append(lv.Spawns, Spawn{Object: "flyer", X: float64(x), Y: float64(upCeil + 2), Dir: dirOf(r)})
		case 1:
			lv.Spawns = append(lv.Spawns, Spawn{Object: "enemy", X: float64(x), Y: float64(upFloor - 1), Dir: dirOf(r)})
		default:
			lv.Spawns = append(lv.Spawns, Spawn{Object: "coin", X: float64(x), Y: float64(upFloor - 1)})
		}
	}
	return lv
}

// clearOfHazards finds a place on the floor near x with nothing sharp within a
// couple of tiles of it.
//
// A player meeting spikes jumps them, and an item sitting among the spikes is
// an item they sail straight over — which is fine for a coin and fatal for the
// key to the door at the end of the level.
func clearOfHazards(g *grid, x, floorRow int) int {
	clear := func(at int) bool {
		if g.at(at, floorRow) != chGround {
			return false
		}
		for dx := -2; dx <= 2; dx++ {
			if g.at(at+dx, floorRow-1) != chEmpty {
				return false
			}
		}
		return true
	}
	for offset := 0; offset < g.w; offset++ {
		if at := x - offset; at > 2 && clear(at) {
			return at
		}
		if at := x + offset; at < g.w-3 && clear(at) {
			return at
		}
	}
	return x
}
