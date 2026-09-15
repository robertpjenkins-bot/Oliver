// A headless run of the engine, used by the Go test suite.
//
// It stands in for a browser with the smallest possible canvas and timing
// stubs, loads a project, and plays it for a few thousand frames with the
// controls held down. What it proves is narrow but valuable: that a generated
// game boots, steps, draws and ends up in a sane state without throwing —
// which is the failure a person would otherwise only meet in the browser.
//
//   node engine_smoke.mjs <game.json> [frames]
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

const listeners = {};
const win = {
  document,
  addEventListener(name, fn) { (listeners[name] = listeners[name] || []).push(fn); },
  removeEventListener() {},
  requestAnimationFrame() { return 0; },
  cancelAnimationFrame() {},
  navigator: { getGamepads: () => [] },
  setInterval: () => 0,
  clearInterval: () => {}
};
globalThis.window = win;
globalThis.document = document;
globalThis.requestAnimationFrame = win.requestAnimationFrame;
globalThis.cancelAnimationFrame = win.cancelAnimationFrame;
// Node defines navigator as a getter, so the engine gets its stub through the
// window object it is handed rather than through the global.

const source = readFileSync(new URL('./engine.js', import.meta.url), 'utf8');
// The engine is a plain script, not a module: run it against the stubs above.
new Function('window', 'document', 'requestAnimationFrame', 'cancelAnimationFrame', 'navigator', source)
  (win, document, win.requestAnimationFrame, win.cancelAnimationFrame, win.navigator);

const Engine = win.OliverEngine;
if (!Engine) { console.error('the engine did not load'); process.exit(1); }

const game = JSON.parse(readFileSync(process.argv[2], 'utf8'));
const frames = parseInt(process.argv[3] || '2400', 10);

const canvas = document.createElement('canvas');
const runtime = Engine.start(canvas, game, { skipTitle: true });

const startX = runtime.player ? runtime.player.x : 0;
const startY = runtime.player ? runtime.player.y : 0;
let moved = false;
let drew = 0;

for (let f = 0; f < frames; f++) {
  // Hold right, jump and fire in a repeating pattern: enough to exercise
  // movement, collision, shooting and the rules that hang off them.
  runtime.input.down = {
    ArrowRight: (f % 240) < 200,
    ArrowLeft: (f % 240) >= 200,
    ArrowDown: (f % 180) > 150,
    Space: (f % 45) === 0,
    KeyX: (f % 30) === 0
  };
  runtime.input.pressed = {};
  if (f % 45 === 0) runtime.input.pressed.Space = true;
  if (f % 30 === 0) runtime.input.pressed.KeyX = true;

  runtime.step(1 / 60);
  runtime.draw();
  drew++;
  runtime.input.endFrame();

  if (runtime.player && (Math.abs(runtime.player.x - startX) > 4 || Math.abs(runtime.player.y - startY) > 4)) {
    moved = true;
  }
  if (runtime.entities.length > 4000) {
    console.error('runaway spawning: ' + runtime.entities.length + ' entities at frame ' + f);
    process.exit(1);
  }
}

const report = {
  frames: drew,
  state: runtime.state,
  score: runtime.score,
  lives: runtime.lives,
  entities: runtime.entities.length,
  level: runtime.level.def.id,
  moved
};
if (!moved) { console.error('the player never moved: ' + JSON.stringify(report)); process.exit(1); }
if (['playing', 'gameover', 'won'].indexOf(runtime.state) < 0) {
  console.error('unexpected state: ' + JSON.stringify(report)); process.exit(1);
}
console.log(JSON.stringify(report));
