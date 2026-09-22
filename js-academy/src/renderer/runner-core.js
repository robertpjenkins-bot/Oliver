'use strict';

// The lesson test harness. These two functions are shipped as source text
// into a Web Worker (plain lessons) or a sandboxed iframe (DOM lessons), and
// are also required directly by the lesson validator under Node. They must
// therefore be self-contained: no references to anything outside themselves.

function jsaFormat(value, depth, seen) {
  depth = depth || 0;
  seen = seen || [];
  const t = typeof value;
  if (value === null) return 'null';
  if (t === 'undefined') return 'undefined';
  if (t === 'string') return depth === 0 ? value : JSON.stringify(value);
  if (t === 'number' || t === 'boolean') return Object.is(value, -0) ? '-0' : String(value);
  if (t === 'bigint') return value + 'n';
  if (t === 'symbol') return value.toString();
  if (t === 'function') {
    if (/^class\s/.test(Function.prototype.toString.call(value))) return '[class ' + (value.name || 'anonymous') + ']';
    return '[Function: ' + (value.name || 'anonymous') + ']';
  }
  if (seen.indexOf(value) !== -1) return '[Circular]';
  if (depth > 4) return Array.isArray(value) ? '[Array]' : '[Object]';
  const next = seen.concat([value]);
  const f = (v) => jsaFormat(v, depth + 1, next);
  if (value instanceof Error) return value.name + ': ' + value.message;
  if (value instanceof Date) return isNaN(value) ? 'Invalid Date' : value.toISOString();
  if (value instanceof RegExp) return String(value);
  if (typeof Promise !== 'undefined' && value instanceof Promise) return 'Promise { … }';
  if (Array.isArray(value)) {
    if (value.length === 0) return '[]';
    const items = [];
    for (let i = 0; i < value.length; i++) items.push(i in value ? f(value[i]) : '<empty>');
    return '[' + items.join(', ') + ']';
  }
  if (value instanceof Map) {
    const parts = [];
    value.forEach((v, k) => parts.push(f(k) + ' => ' + f(v)));
    return 'Map(' + value.size + ') {' + (parts.length ? ' ' + parts.join(', ') + ' ' : '') + '}';
  }
  if (value instanceof Set) {
    const parts = [];
    value.forEach((v) => parts.push(f(v)));
    return 'Set(' + value.size + ') {' + (parts.length ? ' ' + parts.join(', ') + ' ' : '') + '}';
  }
  if (typeof Element !== 'undefined' && value instanceof Element) {
    return '<' + value.tagName.toLowerCase() + (value.id ? ' id="' + value.id + '"' : '') + '>';
  }
  const ctor = value.constructor && value.constructor !== Object ? value.constructor.name + ' ' : '';
  const keys = Object.keys(value);
  if (keys.length === 0) return ctor + '{}';
  const body = keys.map((k) => (/^[A-Za-z_$][\w$]*$/.test(k) ? k : JSON.stringify(k)) + ': ' + f(value[k]));
  return ctor + '{ ' + body.join(', ') + ' }';
}

// Runs `code`, then evaluates each test expression inside the same scope so
// tests can see the learner's variables and functions. `emit(type, text)`
// receives console output as it happens.
async function jsaRunProgram(code, tests, opts, emit) {
  opts = opts || {};
  const logs = [];
  const MAX_LINES = 10000;
  const make = (type) => function () {
    if (logs.length >= MAX_LINES) {
      throw new Error('Too much output (over ' + MAX_LINES + ' lines), so the program was stopped. Is there a loop that never ends?');
    }
    const text = Array.prototype.map.call(arguments, (a) => jsaFormat(a)).join(' ');
    logs.push(text);
    emit(type, text);
  };
  const fakeConsole = {
    log: make('log'), info: make('info'), debug: make('log'), warn: make('warn'), error: make('error'),
    table: make('log'), dir: make('log'), clear() {}, group: make('log'), groupEnd() {}, time() {}, timeEnd() {},
    assert: function (cond) {
      if (!cond) make('error').apply(null, ['Assertion failed:'].concat(Array.prototype.slice.call(arguments, 1)));
    },
  };
  const out = () => logs.join('\n');
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

  const check = async (evaluate) => {
    if (opts.wait) await sleep(opts.wait);
    const results = [];
    for (const test of tests || []) {
      try {
        const v = await evaluate(test[0]);
        results.push({ m: test[1], pass: !!v });
      } catch (e) {
        results.push({ m: test[1], pass: false, err: (e && e.name ? e.name + ': ' : '') + (e && e.message ? e.message : String(e)) });
      }
    }
    return results;
  };

  const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
  let fn;
  try {
    fn = new AsyncFunction('console', '$logs', '$out', '$src', '$check', code + '\n;return await $check(function (__jsaExpr) { return eval(__jsaExpr); });');
  } catch (e) {
    return { error: e.name + ': ' + e.message, syntax: true, results: [], logs: logs };
  }
  try {
    const results = await fn(fakeConsole, logs, out, code, check);
    // A top-level `return` in the learner's code skips the tests.
    if (!Array.isArray(results)) return { results: [], logs: logs, skipped: true };
    return { results: results, logs: logs };
  } catch (e) {
    let msg;
    if (e && typeof e === 'object' && 'message' in e) msg = (e.name || 'Error') + ': ' + e.message;
    else msg = 'Uncaught ' + jsaFormat(e, 1);
    return { error: msg, results: [], logs: logs };
  }
}

if (typeof module !== 'undefined' && module.exports) module.exports = { jsaFormat, jsaRunProgram };
