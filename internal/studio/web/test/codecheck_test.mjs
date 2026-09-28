// Tests for finding the line a mistake in a game's code is on.
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
const check = createRequire(import.meta.url)('../codecheck.js');

let ran = 0, failed = 0;
function test(name, fn) {
  ran++;
  try { fn(); } catch (e) { failed++; console.error('FAIL ' + name + '\n  ' + e.message); }
}
const at = src => check.whereItWentWrong(src);

test('a bracket left open is found, on the line it was opened', () => {
  const found = at('when.start(function () {\n  say("one"\n});\n');
  assert.ok(found, 'nothing was found');
  assert.equal(found.line, 2, 'the line to fix is the one that opened');
  assert.match(found.why, /line 3/, 'and the clashing line is named too');
});

test('a curly left open points at the line that opened it', () => {
  const found = at('when.start(function () {\n  say("hi");\n');
  assert.equal(found.line, 1);
});

test('one closer too many is found where it is', () => {
  const found = at('when.start(function () {\n  say("hi");\n}));\n');
  assert.equal(found.line, 3);
  assert.match(found.why, /one "\)" too many/);
});

test('two that do not match say so, and name both lines', () => {
  const found = at('var a = [1, 2);\n');
  assert.equal(found.line, 1);
  assert.match(found.why, /square bracket.*closed by a "\)" on line 1/);
});

test('a quote left open is found on its own line, not at the end of the file', () => {
  const found = at('say("never ends);\nsay("ok");\nsay("also ok");\n');
  assert.equal(found.line, 1);
  assert.match(found.why, /quote/);
});

test('brackets inside strings are not counted', () => {
  assert.equal(at('say("a ( in a string");\n'), null);
  assert.equal(at("say('a ) in a string');\n"), null);
});

test('brackets inside comments are not counted', () => {
  assert.equal(at('// a ) in a comment\nsay("hi");\n'), null);
  assert.equal(at('/* a ( in a\n   block comment */\nsay("hi");\n'), null);
});

test('an escaped quote does not end the string', () => {
  assert.equal(at('say("she said \\"hello\\" loudly");\n'), null);
  assert.equal(at("var s = 'it\\'s fine';\n"), null);
});

test('an apostrophe inside double quotes is just a letter', () => {
  assert.equal(at('say("it\'s fine");\n'), null);
});

test('a template literal may run over several lines', () => {
  assert.equal(at('var s = `one\ntwo\nthree`;\n'), null);
});

test('code that balances is reported as fine', () => {
  const good = [
    '',
    'when.start(function () { say("hi"); });\n',
    'every(2, function () {\n  spawn("coin", random(10), 1);\n});\n',
    'var a = [1, 2, 3].map(function (n) { return { n: n }; });\n'
  ];
  good.forEach(src => assert.equal(at(src), null, JSON.stringify(src)));
});

test('explain keeps what the browser said and adds where', () => {
  const out = check.explain(new SyntaxError('missing ) after argument list'),
    'when.start(function () {\n  say("one"\n});\n');
  assert.match(out, /missing \) after argument list/);
  assert.match(out, /line 2/);
  assert.match(out, /say\("one"/, 'the offending line itself should be shown');
});

test('explain on balanced code says only what the browser said', () => {
  const out = check.explain(new SyntaxError('Unexpected token'), 'var = 1;\n');
  assert.equal(out, 'Unexpected token');
});

console.log(`${ran - failed}/${ran} code check tests passed`);
process.exit(failed ? 1 : 0);
