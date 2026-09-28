// Tests for the shared grid operations, run under plain node.
//
// These are the parts of the editor with an answer that is right or wrong
// regardless of what the screen looks like, so they are tested here rather
// than through a browser.
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
const ops = createRequire(import.meta.url)('../paintops.js');

let ran = 0, failed = 0;
function test(name, fn) {
  ran++;
  try { fn(); } catch (e) { failed++; console.error('FAIL ' + name + '\n  ' + e.message); }
}
const grid = (...rows) => rows;
const show = rows => rows.join('|');

test('line walks from one end to the other and touches each cell once', () => {
  const pts = ops.line(0, 0, 4, 0);
  assert.equal(pts.length, 5);
  assert.deepEqual(pts.map(p => p.x), [0, 1, 2, 3, 4]);
  const diag = ops.line(0, 0, 3, 3);
  assert.deepEqual(diag.map(p => p.x + ',' + p.y), ['0,0', '1,1', '2,2', '3,3']);
  const seen = new Set(ops.line(2, 7, 9, 1).map(p => p.x + ',' + p.y));
  assert.equal(seen.size, ops.line(2, 7, 9, 1).length, 'a cell was drawn twice');
});

test('a line of no length is still one cell', () => {
  assert.deepEqual(ops.line(3, 3, 3, 3), [{ x: 3, y: 3 }]);
});

test('rect outlines, and filled fills', () => {
  const out = ops.plot(ops.blank(5, 4, '.'), ops.rect(0, 0, 4, 3, false), '#');
  assert.equal(show(out), '#####|#...#|#...#|#####');
  const solid = ops.plot(ops.blank(5, 4, '.'), ops.rect(1, 1, 3, 2, true), '#');
  assert.equal(show(solid), '.....|.###.|.###.|.....');
});

test('a rectangle dragged backwards is the same rectangle', () => {
  const a = ops.plot(ops.blank(4, 4, '.'), ops.rect(3, 3, 1, 1, true), '#');
  const b = ops.plot(ops.blank(4, 4, '.'), ops.rect(1, 1, 3, 3, true), '#');
  assert.equal(show(a), show(b));
});

test('fill spreads over joined cells and stops at a wall', () => {
  const room = grid('#####', '#...#', '#.#.#', '#...#', '#####');
  const done = ops.fill(room, 1, 1, 'o');
  assert.equal(show(done), '#####|#ooo#|#o#o#|#ooo#|#####');
});

test('fill does not leak through a diagonal touch', () => {
  //  a#
  //  #b   — a and b touch only at a corner, so filling a must leave b alone
  const g = grid('.#', '#.');
  const done = ops.fill(g, 0, 0, 'o');
  assert.equal(show(done), 'o#|#.');
});

test('fill on its own colour changes nothing', () => {
  const g = grid('..', '..');
  assert.equal(show(ops.fill(g, 0, 0, '.')), '..|..');
});

test('fill outside the grid changes nothing', () => {
  const g = grid('..', '..');
  assert.equal(show(ops.fill(g, 9, 9, 'x')), '..|..');
});

test('nothing is edited in place', () => {
  const before = grid('..', '..');
  const copy = before.slice();
  ops.fill(before, 0, 0, 'x');
  ops.set(before, 0, 0, 'y');
  ops.plot(before, [{ x: 1, y: 1 }], 'z');
  assert.deepEqual(before, copy, 'an operation wrote to the rows it was given');
});

test('writing off the edge drops the overhang instead of wrapping it', () => {
  const g = ops.blank(4, 4, '.');
  const out = ops.write(g, ['ab', 'cd'], 3, 3, null);
  assert.equal(show(out), '....|....|....|...a');
});

test('write skips the see-through character, and stamps it when told to', () => {
  const g = ops.blank(3, 1, '#');
  assert.equal(show(ops.write(g, ['a.b'], 0, 0, '.')), 'a#b');
  assert.equal(show(ops.write(g, ['a.b'], 0, 0, null)), 'a.b');
});

test('read lifts a region out at the size asked for', () => {
  const g = grid('abcd', 'efgh', 'ijkl');
  assert.equal(show(ops.read(g, 1, 0, 2, 1)), 'bc|fg');
  assert.equal(show(ops.read(g, 2, 1, 1, 0)), 'bc|fg', 'dragged backwards');
});

test('read past the edge pads rather than returning short rows', () => {
  const g = grid('ab', 'cd');
  const out = ops.read(g, 1, 1, 3, 3);
  assert.deepEqual(out.map(r => r.length), [3, 3, 3]);
  assert.equal(show(out), 'd..|...|...');
});

test('copy then paste puts back exactly what was taken', () => {
  const g = grid('#..#', '.##.', '#..#', '.##.');
  const taken = ops.read(g, 0, 0, 3, 3);
  const wiped = ops.clear(g, 0, 0, 3, 3, '.');
  assert.equal(show(wiped), '....|....|....|....');
  assert.equal(show(ops.write(wiped, taken, 0, 0, null)), show(g));
});

test('flips are their own opposite', () => {
  const g = grid('ab', 'cd');
  assert.equal(show(ops.flipH(g)), 'ba|dc');
  assert.equal(show(ops.flipV(g)), 'cd|ab');
  assert.equal(show(ops.flipH(ops.flipH(g))), show(g));
  assert.equal(show(ops.flipV(ops.flipV(g))), show(g));
});

test('four rotations come back to the start', () => {
  const g = grid('abc', 'def');
  assert.equal(show(ops.rotate(g)), 'da|eb|fc');
  let r = g;
  for (let i = 0; i < 4; i++) r = ops.rotate(r);
  assert.equal(show(r), show(g));
});

test('rotating a tall grid returns a wide one', () => {
  const g = ops.blank(2, 5, '.');
  const r = ops.rotate(g);
  assert.equal(ops.width(r), 5);
  assert.equal(ops.height(r), 2);
});

test('shift moves the picture and fills behind it', () => {
  const g = grid('ab', 'cd');
  assert.equal(show(ops.shift(g, 1, 0, '.', false)), '.a|.c');
  assert.equal(show(ops.shift(g, 0, -1, '.', false)), 'cd|..');
});

test('a wrapping shift loses nothing and returns after a full lap', () => {
  const g = grid('abc', 'def');
  assert.equal(show(ops.shift(g, 1, 0, '.', true)), 'cab|fde');
  let r = g;
  for (let i = 0; i < 3; i++) r = ops.shift(r, 1, 0, '.', true);
  assert.equal(show(r), show(g));
});

test('resize keeps the top-left corner and pads the rest', () => {
  const g = grid('ab', 'cd');
  assert.equal(show(ops.resize(g, 3, 3, '.')), 'ab.|cd.|...');
  assert.equal(show(ops.resize(g, 1, 1, '.')), 'a');
});

test('replace swaps one character everywhere', () => {
  assert.equal(show(ops.replace(grid('a.a', '.a.'), 'a', 'b')), 'b.b|.b.');
});

test('outline finds the empty cells touching something drawn', () => {
  const g = grid('...', '.#.', '...');
  const out = ops.plot(g, ops.outline(g, '.'), 'o');
  assert.equal(show(out), '.o.|o#o|.o.');
});

test('ellipse is symmetric and stays inside its box', () => {
  const pts = ops.ellipse(0, 0, 8, 8, false);
  assert.ok(pts.length > 8, 'an ellipse of some size');
  pts.forEach(p => {
    assert.ok(p.x >= 0 && p.x <= 8 && p.y >= 0 && p.y <= 8, 'a point left the box');
  });
  const has = (x, y) => pts.some(p => p.x === x && p.y === y);
  for (const p of pts) assert.ok(has(8 - p.x, p.y), `no mirror for ${p.x},${p.y}`);
  const filled = ops.ellipse(0, 0, 8, 8, true);
  assert.ok(filled.length > pts.length, 'filled should cover more than the edge');
});

test('plot ignores points outside the grid', () => {
  const g = ops.blank(2, 2, '.');
  assert.equal(show(ops.plot(g, [{ x: -1, y: 0 }, { x: 0, y: 9 }, { x: 1, y: 1 }], '#')), '..|.#');
});

test('blank makes the size asked for', () => {
  const g = ops.blank(5, 3, '.');
  assert.equal(ops.width(g), 5);
  assert.equal(ops.height(g), 3);
});

console.log(`${ran - failed}/${ran} grid operation tests passed`);
process.exit(failed ? 1 : 0);
