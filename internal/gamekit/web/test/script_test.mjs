// Tests for a game's own code, run under node against the real engine.
//
// The engine draws to a canvas, so a small stand-in for the browser is set up
// here: enough DOM for it to start, and a clock it is stepped by, so a script
// can be run for a number of frames and its effects checked.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));

// --- the least browser an engine can start in -----------------------------
function fakeContext() {
  const noop = () => {};
  return new Proxy({}, {
    get(_, k) {
      if (k === 'canvas') return { width: 0, height: 0 };
      if (k === 'measureText') return () => ({ width: 0 });
      if (k === 'getImageData') return () => ({ data: [] });
      if (k === 'createLinearGradient') return () => ({ addColorStop: noop });
      return noop;
    },
    set() { return true; }
  });
}
function fakeCanvas() {
  return {
    width: 0, height: 0, style: {}, dataset: {},
    getContext: fakeContext,
    addEventListener() {}, removeEventListener() {},
    focus() {}, getBoundingClientRect: () => ({ left: 0, top: 0, width: 256, height: 224 })
  };
}
let raf = [];
global.window = {
  addEventListener() {}, removeEventListener() {},
  requestAnimationFrame(fn) { raf.push(fn); return raf.length; },
  cancelAnimationFrame() {},
  devicePixelRatio: 1
};
global.document = {
  createElement: () => fakeCanvas(),
  addEventListener() {}, removeEventListener() {}
};
global.requestAnimationFrame = global.window.requestAnimationFrame;
global.cancelAnimationFrame = global.window.cancelAnimationFrame;
global.AudioContext = function () {
  return {
    state: 'running', currentTime: 0, destination: {},
    resume() {}, close() {},
    createOscillator: () => ({ connect() {}, start() {}, stop() {}, frequency: { setValueAtTime() {} }, type: '' }),
    createGain: () => ({ connect() {}, gain: { setValueAtTime() {}, exponentialRampToValueAtTime() {}, linearRampToValueAtTime() {}, value: 0 } })
  };
};
// Node defines navigator read-only, so it is patched rather than replaced.
if (!global.navigator || !global.navigator.getGamepads) {
  Object.defineProperty(global, 'navigator', {
    value: { getGamepads: () => [] }, configurable: true, writable: true
  });
}

// --- load the engine -------------------------------------------------------
const src = readFileSync(join(here, '..', 'engine.js'), 'utf8');
new Function(src).call(global.window);
const Engine = global.window.OliverEngine;

// --- a tiny game to run scripts in ----------------------------------------
function tinyGame(code) {
  const solid = new Array(21).join('#');
  const sky = new Array(21).join('.');
  return {
    format: 'oliver.game/v1', name: 'Test', genre: 'platformer',
    screen: { width: 160, height: 144, tile: 8, scale: 1 },
    palette: ['#000', '#fff', '#f00', '#0f0'],
    settings: { gravity: 600, lives: 3, startLevel: 'l1' },
    sprites: [{ id: 'blob', w: 8, h: 8, fps: 0, frames: [new Array(8).fill('11111111')] }],
    objects: [
      { id: 'hero', kind: 'player', sprite: 'blob', behavior: 'platformer', speed: 6, jump: 12, health: 3 },
      { id: 'baddie', kind: 'enemy', sprite: 'blob', behavior: 'patrol', speed: 2, health: 1, damage: 1 },
      { id: 'coin', kind: 'item', sprite: 'blob', behavior: 'static', collect: true, points: 5 }
    ],
    levels: [{
      id: 'l1', name: 'One', width: 20, height: 10,
      tiles: [{ char: '#', kind: 'solid', sprite: 'blob' }, { char: '^', kind: 'hazard', sprite: 'blob' }],
      map: [sky, sky, sky, sky, sky, sky, sky, sky, sky, solid],
      spawns: [{ object: 'hero', x: 2, y: 8, dir: 1 }],
      next: 'win'
    }, {
      id: 'l2', name: 'Two', width: 20, height: 10,
      tiles: [{ char: '#', kind: 'solid', sprite: 'blob' }],
      map: [sky, sky, sky, sky, sky, sky, sky, sky, sky, solid],
      spawns: [{ object: 'hero', x: 2, y: 8, dir: 1 }],
      next: 'win'
    }],
    rules: [], sounds: [],
    code: code || ''
  };
}

function run(code, frames = 30, dt = 1 / 60) {
  raf = [];
  const engine = Engine.start(fakeCanvas(), tinyGame(code), { skipTitle: true, scale: 1 });
  for (let i = 0; i < frames; i++) engine.step(dt);
  return engine;
}

let ran = 0, failed = 0;
function test(name, fn) {
  ran++;
  try { fn(); } catch (e) { failed++; console.error('FAIL ' + name + '\n  ' + (e.stack || e.message).split('\n').slice(0, 3).join('\n  ')); }
}

// --------------------------------------------------------------------------
test('a script runs at all, and when.start fires', () => {
  const e = run('when.start(function () { set("hello", 1); });', 2);
  assert.deepEqual(e.script.errors, []);
  assert.equal(e.vars.hello, 1);
});

test('when.update runs every frame and is given the time step', () => {
  const e = run('var n = 0, total = 0;\nwhen.update(function (dt) { n++; total += dt; set("n", n); set("t", total); });', 10);
  assert.deepEqual(e.script.errors, []);
  assert.equal(e.vars.n, 10);
  assert.ok(Math.abs(e.vars.t - 10 / 60) < 1e-9, 'dt should add up to the time elapsed');
});

test('every() repeats on a period and after() fires once', () => {
  // An eighth of a second and a half a second are both exact in binary, so
  // this counts ticks rather than accumulated rounding error.
  const e = run('set("rep", 0); set("once", 0);\n' +
    'every(0.5, function () { set("rep", get("rep") + 1); });\n' +
    'after(0.5, function () { set("once", get("once") + 1); });', 16, 0.125);
  assert.equal(e.vars.rep, 4, 'four half-seconds in two seconds');
  assert.equal(e.vars.once, 1, 'after() fires once and never again');
});

test('the player can be read and moved, in tiles', () => {
  const e = run('when.start(function () { player.x = 5; player.y = 3; });', 1, 0);
  assert.equal(e.player.x / 8, 5);
  assert.equal(e.player.y / 8, 3);
  const e2 = run('when.start(function () { set("where", Math.round(player.x)); });', 1, 0);
  assert.equal(e2.vars.where, 2, 'the player starts where the level put it');
});

test('spawn puts a thing into the level and find gets it back', () => {
  const e = run('when.start(function () { spawn("baddie", 9, 8); });', 2, 0);
  assert.deepEqual(e.script.errors, []);
  assert.equal(e.countOf('baddie'), 1);
  const e2 = run('when.start(function () { spawn("baddie", 9, 8); set("found", find("baddie") ? 1 : 0); });', 2, 0);
  assert.equal(e2.vars.found, 1);
});

test('spawning something that does not exist is reported, not thrown', () => {
  const e = run('when.start(function () { spawn("dragon", 1, 1); set("after", 1); });', 2, 0);
  assert.equal(e.vars.after, 1, 'the handler carried on past the bad call');
  assert.ok(e.script.errors.some(t => t.includes('dragon')), e.script.errors.join('; '));
});

test('all() lists things by id or by kind', () => {
  const e = run('when.start(function () {\n' +
    '  spawn("baddie", 6, 8); spawn("baddie", 9, 8); spawn("coin", 12, 8);\n' +
    '  set("baddies", all("baddie").length);\n' +
    '  set("enemies", all("enemy").length);\n' +
    '  set("everything", all().length);\n' +
    '});', 2, 0);
  assert.equal(e.vars.baddies, 2);
  assert.equal(e.vars.enemies, 2);
  assert.equal(e.vars.everything, 4, 'two baddies, a coin and the player');
});

test('score, lives, give and has all work', () => {
  const e = run('when.start(function () {\n' +
    '  score(30); set("s", score());\n' +
    '  lives(5); set("l", lives());\n' +
    '  give("boots"); set("h", has("boots") ? 1 : 0); set("no", has("wings") ? 1 : 0);\n' +
    '});', 2, 0);
  assert.equal(e.vars.s, 30);
  assert.equal(e.vars.l, 5);
  assert.equal(e.vars.h, 1);
  assert.equal(e.vars.no, 0);
});

test('say puts a message on screen', () => {
  const e = run('when.start(function () { say("hello there", 3); });', 1, 0);
  assert.equal(e.messages.length, 1);
  assert.equal(e.messages[0].text, 'hello there');
});

test('tiles can be read and written', () => {
  const e = run('when.start(function () {\n' +
    '  set("floor", tiles.at(5, 9));\n' +
    '  set("air", tiles.at(5, 0));\n' +
    '  tiles.set(5, 0, "solid"); set("now", tiles.at(5, 0));\n' +
    '  tiles.clear(5, 0); set("gone", tiles.at(5, 0));\n' +
    '});', 2, 0);
  assert.equal(e.vars.floor, 'solid');
  assert.equal(e.vars.air, 'empty');
  assert.equal(e.vars.now, 'solid');
  assert.equal(e.vars.gone, 'empty');
});

test('goTo moves to another level and the script survives it', () => {
  const e = run('when.start(function () { set("here", level.id); });\n' +
    'after(0.05, function () { goTo("l2"); });', 30);
  assert.equal(e.level.def.id, 'l2');
  assert.equal(e.vars.here, 'l2', 'when.start fires again for the new level');
});

test('when.collect fires when the player takes something', () => {
  const e = run('when.start(function () { spawn("coin", 2, 8); });\n' +
    'when.collect("coin", function () { set("got", (get("got") || 0) + 1); });', 30);
  assert.equal(e.vars.got, 1, 'the coin spawned on the player should be collected once');
});

test('a mistake in the code is reported and the game keeps running', () => {
  const e = run('this is not javascript at all', 5);
  assert.ok(e.script.errors.length, 'the syntax error should be reported');
  assert.ok(e.script.errors[0].startsWith('your code'), e.script.errors[0]);
  assert.equal(e.running, true, 'the engine should still be running');
});

test('a handler that throws every frame is retired rather than left to fail forever', () => {
  const e = run('when.update(function () { throw new Error("boom"); });', 60);
  assert.ok(e.script.retired.includes('when.update'), 'it should be retired');
  assert.ok(e.script.errors.length <= 2, 'and reported twice at most, not sixty times');
  assert.equal(e.running, true);
});

test('one bad handler does not stop the others', () => {
  const e = run('when.update(function () { throw new Error("boom"); });\n' +
    'when.update(function () { set("ok", (get("ok") || 0) + 1); });', 20);
  assert.equal(e.vars.ok, 20);
});

test('the engine is not reachable from a script', () => {
  const e = run('when.start(function () { set("hasEngine", (typeof engine)); });', 2, 0);
  assert.equal(e.vars.hasEngine, 'undefined', 'no engine handle is in scope');
});

test('a game with no code has no script trouble', () => {
  const e = run('', 10);
  assert.deepEqual(e.script.errors, []);
  assert.equal(e.script.ready, false, 'nothing to compile');
  assert.equal(e.running, true);
});

test('player.jump and push move the player', () => {
  const e = run('when.start(function () { player.jump(10); set("up", player.vy < 0 ? 1 : 0); });', 1, 0);
  assert.equal(e.vars.up, 1);
  const e2 = run('when.start(function () { player.push(3, 0); set("vx", Math.round(player.vx)); });', 1, 0);
  assert.equal(e2.vars.vx, 3);
});

test('distanceTo measures in tiles', () => {
  const e = run('when.start(function () {\n' +
    '  player.x = 0; player.y = 0;\n' +
    '  var b = spawn("baddie", 3, 4);\n' +
    '  set("d", Math.round(player.distanceTo(b)));\n' +
    '});', 2, 0);
  assert.equal(e.vars.d, 5, '3 across and 4 up is 5');
});

console.log(`${ran - failed}/${ran} script tests passed`);
process.exit(failed ? 1 : 0);
