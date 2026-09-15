// A headless check of the one mechanic an exploration game stands on: a door
// that will not open until the player has found what opens it.
//
//   node engine_doors.mjs <game.json>
//
// It puts the player against the first area's door, proves it will not let
// them through, hands them the missiles, and proves that it does.
import { readFileSync } from 'node:fs';

function stubContext() {
  const noop = () => {};
  return {
    fillStyle: '#000', globalAlpha: 1, imageSmoothingEnabled: false,
    fillRect: noop, drawImage: noop, clearRect: noop,
    save: noop, restore: noop, translate: noop, scale: noop
  };
}
const document = {
  createElement() {
    const c = { width: 1, height: 1 };
    c.getContext = () => stubContext();
    return c;
  }
};
const win = {
  document,
  addEventListener() {}, removeEventListener() {},
  requestAnimationFrame() { return 0; }, cancelAnimationFrame() {},
  navigator: { getGamepads: () => [] },
  setInterval: () => 0, clearInterval: () => {}
};
globalThis.window = win;
globalThis.document = document;
globalThis.requestAnimationFrame = win.requestAnimationFrame;
globalThis.cancelAnimationFrame = win.cancelAnimationFrame;

const source = readFileSync(new URL('./engine.js', import.meta.url), 'utf8');
new Function('window', 'document', 'requestAnimationFrame', 'cancelAnimationFrame', 'navigator', source)
  (win, document, win.requestAnimationFrame, win.cancelAnimationFrame, win.navigator);

const game = JSON.parse(readFileSync(process.argv[2], 'utf8'));
const runtime = win.OliverEngine.start(document.createElement('canvas'), game, { skipTitle: true });
const tile = game.screen.tile;

function fail(why) { console.error(why); process.exit(1); }

// Find the door and a floor to stand on beside it.
const level = runtime.level;
let door = null;
for (const ch of Object.keys(level.tiles)) {
  if (level.tiles[ch].kind === 'door') {
    for (let y = 0; y < level.height && !door; y++) {
      for (let x = 0; x < level.width && !door; x++) {
        if (level.rows[y][x] === ch) door = { x, y, ch, needs: level.tiles[ch].needs };
      }
    }
  }
}
if (!door) fail('the area has no door to test');

// Stand on the floor on the near side of the door.
let floorY = door.y;
while (floorY < level.height && !level.solidAt(door.x + 2, floorY + 1)) floorY++;
const player = runtime.player;
player.x = (door.x + 2) * tile;
player.y = floorY * tile - player.h;
player.vx = 0; player.vy = 0;

function hold(keys, frames) {
  for (let f = 0; f < frames; f++) {
    runtime.input.down = {};
    runtime.input.pressed = {};
    keys.forEach(k => { runtime.input.down[k] = true; });
    if (f % 20 === 0) keys.forEach(k => { runtime.input.pressed[k] = true; });
    runtime.step(1 / 60);
    runtime.input.endFrame();
  }
}

// Walking into a shut door gets nowhere, and firing without the weapon does
// nothing at all.
hold(['ArrowLeft', 'KeyX'], 180);
const stoppedAt = player.x;
if (stoppedAt < door.x * tile + tile) {
  fail('the player walked through a shut door: x=' + stoppedAt + ' door at ' + (door.x * tile));
}
if (runtime.level.rows[door.y][door.x] !== door.ch) {
  fail('the door opened without the ' + door.needs);
}
const shotsBefore = runtime.entities.filter(e => e.kind === 'projectile').length;
if (shotsBefore > 0) fail('the player fired a weapon they have not found yet');

// Find the missiles, and the same door gives way. Hold on to this area: going
// through the door leads to what is behind it, which may well be the way to
// the next area, and then the level the engine is running is a different one.
const area = runtime.level;
runtime.grant(door.needs, 'test');
hold(['ArrowLeft', 'KeyX'], 240);
if (area.rows[door.y][door.x] === door.ch) {
  fail('the door stayed shut after the player found the ' + door.needs);
}
for (let y = 0; y < area.height; y++) {
  if (area.rows[y][door.x] === door.ch) {
    fail('only part of the door opened: row ' + y + ' is still shut');
  }
}
hold(['ArrowLeft'], 120);
const wentThrough = runtime.level !== area || player.x < door.x * tile;
if (!wentThrough) {
  fail('the door opened but the player could not get through: x=' + player.x);
}

console.log(JSON.stringify({
  door: door.needs,
  blockedAt: Math.round(stoppedAt / tile),
  doorAt: door.x,
  wentThrough: true,
  reached: runtime.level.def.id,
  inventory: Object.keys(runtime.inventory)
}));
