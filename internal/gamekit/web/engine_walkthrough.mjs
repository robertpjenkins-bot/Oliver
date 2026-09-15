// Plays a generated exploration map the way a person would, and reports how
// far it got.
//
// A bot that holds one direction proves nothing about a map: it dies to the
// first spike and says the game is broken. This one looks a couple of tiles
// ahead, jumps hazards and walls, shoots what is in the way, climbs the shaft
// by aiming at the next ledge, and walks back along the upper corridor to the
// door. If it can finish an area, a person can.
//
//   node engine_walkthrough.mjs <game.json> [seconds]
import { readFileSync } from 'node:fs';

const noop = () => {};
const document = {
  createElement() {
    const c = { width: 1, height: 1 };
    c.getContext = () => ({
      fillStyle: '', globalAlpha: 1, imageSmoothingEnabled: false,
      fillRect: noop, drawImage: noop, clearRect: noop,
      save: noop, restore: noop, translate: noop, scale: noop
    });
    return c;
  }
};
const win = {
  document, addEventListener: noop, removeEventListener: noop,
  requestAnimationFrame: () => 0, cancelAnimationFrame: noop,
  navigator: { getGamepads: () => [] }, setInterval: () => 0, clearInterval: noop
};
globalThis.window = win;
globalThis.document = document;
globalThis.requestAnimationFrame = win.requestAnimationFrame;
globalThis.cancelAnimationFrame = noop;

const source = readFileSync(new URL('./engine.js', import.meta.url), 'utf8');
new Function('window', 'document', 'requestAnimationFrame', 'cancelAnimationFrame', 'navigator', source)
  (win, document, win.requestAnimationFrame, noop, win.navigator);

const game = JSON.parse(readFileSync(process.argv[2], 'utf8'));
const seconds = parseFloat(process.argv[3] || '120');
const runtime = win.OliverEngine.start(document.createElement('canvas'), game, { skipTitle: true });
const tile = game.screen.tile;

// Where the interesting things are, read off the level rather than assumed.
function survey() {
  const level = runtime.level;
  let door = null, exit = null;
  const ledges = [];
  for (let y = 0; y < level.height; y++) {
    for (let x = 0; x < level.width; x++) {
      const kind = level.kindAt(x, y);
      if (kind === 'door') {
        // A door is a column of tiles; its bottom is the floor it stands on,
        // which is what tells the bot it has reached the corridor it is in.
        if (!door) door = { x, top: y, bottom: y };
        else if (x === door.x) door.bottom = Math.max(door.bottom, y);
      }
      if (kind === 'exit' && !exit) exit = { x, y };
      if (kind === 'platform') ledges.push({ x, y });
    }
  }
  return { door, exit, ledges };
}

let world = survey();
let level = runtime.level.def.id;
const report = { areas: [], finished: false };
let maxX = 0, grabbed = [], frames = 0;

function tileAt(px, py) { return runtime.level.kindAt(Math.floor(px / tile), Math.floor(py / tile)); }

// The next ledge above the player, within the shaft, is what to aim at.
function nextLedge(p) {
  let best = null;
  for (const l of world.ledges) {
    const above = l.y * tile < p.y - 4;
    if (!above) continue;
    if (Math.abs(l.x * tile - p.x) > 10 * tile) continue;
    if (!best || l.y > best.y) best = l;
  }
  return best;
}

// Where the bot is in the route. It has to be remembered rather than worked
// out afresh each frame: at the top of the shaft the player is briefly at the
// height of both corridors, and a bot deciding from height alone turns round
// on every frame and climbs the same shaft for ever.
let reachedTop = false;

function decide(p) {
  const keys = {};
  const dir = p.facing >= 0 ? 1 : -1;
  const footY = p.y + p.h - 2;
  const aheadX = dir > 0 ? p.x + p.w + 6 : p.x - 6;
  const upstairs = world.door && p.y < (world.door.bottom + 1) * tile;
  if (upstairs) reachedTop = true;

  if (reachedTop && !upstairs) {
    // Fell back down the shaft: climb it again.
    const ledge = nextLedge(p);
    if (ledge) {
      const tx = (ledge.x + 1) * tile;
      const dx = tx - (p.x + p.w / 2);
      if (Math.abs(dx) > tile * 0.5) keys[dx > 0 ? 'ArrowRight' : 'ArrowLeft'] = true;
      if (p.grounded && Math.abs(dx) <= tile * 1.5) keys.Space = true;
      return keys;
    }
  }

  if (reachedTop) {
    // Head back along the upper corridor to the door.
    keys.ArrowLeft = true;
  } else {
    // Outbound. Climb when there is something climbable overhead and the
    // ability to reach it; otherwise keep going, which is how the shaft is
    // found in the first place.
    const ledge = runtime.inventory.highjump ? nextLedge(p) : null;
    if (ledge) {
      const tx = (ledge.x + 1) * tile;
      const dx = tx - (p.x + p.w / 2);
      if (Math.abs(dx) > tile * 0.5) keys[dx > 0 ? 'ArrowRight' : 'ArrowLeft'] = true;
      if (p.grounded && Math.abs(dx) <= tile * 1.5) keys.Space = true;
      return keys;
    }
    keys.ArrowRight = true;
  }

  // Jump hazards, walls and anything standing in the way.
  const ahead = tileAt(aheadX, footY);
  if (ahead === 'hazard' || ahead === 'solid' || ahead === 'breakable') keys.Space = true;
  for (const e of runtime.entities) {
    if (!e.alive || e.kind !== 'enemy') continue;
    const gap = (e.x - p.x) * (keys.ArrowLeft ? -1 : 1);
    if (gap > 0 && gap < 3 * tile && Math.abs(e.y - p.y) < 2 * tile) {
      keys.KeyX = true;
      if (gap < 1.6 * tile) keys.Space = true;
    }
  }
  // Shoot the door when we reach it.
  if (world.door && Math.abs(p.x - world.door.x * tile) < 5 * tile) keys.KeyX = true;
  return keys;
}

const total = Math.round(seconds * 60);
let fireToggle = false;
// How long to keep holding jump once one has started. The engine gives a
// shorter hop when the key is released early — good game feel, and a bot that
// taps for one frame never gets off the ground.
let holdJump = 0;
for (let f = 0; f < total; f++) {
  frames = f;
  const p = runtime.player;
  if (!p) break;
  const keys = decide(p);

  runtime.input.down = {};
  runtime.input.pressed = {};
  for (const k of Object.keys(keys)) runtime.input.down[k] = true;
  // A jump is offered on the frame the player is actually standing on
  // something, and then held, so it is a jump rather than a hop.
  if (keys.Space && p.grounded) { runtime.input.pressed.Space = true; holdJump = 14; }
  if (holdJump > 0) { runtime.input.down.Space = true; holdJump--; }
  fireToggle = !fireToggle;
  if (keys.KeyX && fireToggle) runtime.input.pressed.KeyX = true;

  runtime.step(1 / 60);
  runtime.input.endFrame();

  maxX = Math.max(maxX, runtime.player ? runtime.player.x / tile : maxX);
  grabbed = Object.keys(runtime.inventory);

  if (runtime.level.def.id !== level) {
    report.areas.push({ area: level, at: (f / 60).toFixed(1) + 's', carrying: grabbed.slice() });
    level = runtime.level.def.id;
    world = survey();
    reachedTop = false;
    maxX = 0;
  }
  if (runtime.state === 'won') { report.finished = true; break; }
  if (runtime.state === 'gameover') break;
}

report.state = runtime.state;
report.reached = level;
report.seconds = +(frames / 60).toFixed(1);
report.carrying = grabbed;
report.lives = runtime.lives;
report.score = runtime.score;
report.furthest = Math.round(maxX);
console.log(JSON.stringify(report));
