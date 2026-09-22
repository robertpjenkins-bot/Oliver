'use strict';

// Checks every lesson: its solution must pass all of its checks, and its
// starting code must not (otherwise the lesson asks for nothing).
//
//   node scripts/validate-lessons.js            all lessons
//   node scripts/validate-lessons.js 07         only modules whose id/file starts with 07

const fs = require('fs');
const path = require('path');
const { jsaFormat, jsaRunProgram } = require('../src/renderer/runner-core');

const root = path.join(__dirname, '..', 'src');
const indexHtml = fs.readFileSync(path.join(root, 'renderer', 'index.html'), 'utf8');
const files = [...indexHtml.matchAll(/src="\.\.\/lessons\/([^"]+)"/g)].map((m) => path.join(root, 'lessons', m[1]));
const filter = process.argv[2];

for (const f of files) {
  if (!fs.existsSync(f)) {
    console.error(`missing lesson file: ${path.relative(root, f)}`);
    continue;
  }
  require(f);
}
const modules = globalThis.JSA_MODULES || [];

let JSDOM;
function domRunner() {
  if (!JSDOM) ({ JSDOM } = require('jsdom'));
  return async (code, lesson) => {
    const dom = new JSDOM(lesson.dom || '', { runScripts: 'outside-only', pretendToBeVisual: true });
    const w = dom.window;
    const run = w.eval(`(() => {\n${jsaFormat}\n${jsaRunProgram}\nreturn jsaRunProgram;\n})()`);
    const r = await run(code, lesson.tests, { wait: lesson.wait || 0 }, () => {});
    w.close();
    return r;
  };
}

function withTimeout(p, ms) {
  return Promise.race([p, new Promise((resolve) => setTimeout(() => resolve({ error: 'TIMEOUT', results: [] }), ms))]);
}

async function runLesson(code, lesson) {
  if (lesson.dom !== undefined) return withTimeout(domRunner()(code, lesson), 8000);
  return withTimeout(jsaRunProgram(code, lesson.tests, { wait: lesson.wait || 0 }, () => {}), 8000);
}

(async () => {
  let problems = 0;
  let count = 0;
  const seenTitles = new Map();
  for (const mod of modules) {
    if (filter && !mod.id.startsWith(filter) && !mod.file?.startsWith(filter)) continue;
    if (!filter && mod.lessons.length !== 20) {
      console.log(`✗ module ${mod.id} has ${mod.lessons.length} lessons (expected 20)`);
      problems++;
    }
    for (const lesson of mod.lessons) {
      count++;
      const where = `${lesson.id} "${lesson.t}"`;
      const fail = (msg) => {
        problems++;
        console.log(`✗ ${where}: ${msg}`);
      };
      for (const k of ['t', 'b', 'sol']) if (typeof lesson[k] !== 'string' || !lesson[k].trim()) fail(`missing ${k}`);
      if (typeof lesson.code !== 'string') fail('missing starter code');
      if (!Array.isArray(lesson.tests) || !lesson.tests.length) {
        fail('no tests');
        continue;
      }
      const key = lesson.t.toLowerCase();
      if (seenTitles.has(key)) fail(`duplicate title (also ${seenTitles.get(key)})`);
      seenTitles.set(key, lesson.id);

      const good = await runLesson(lesson.sol, lesson);
      if (good.error) fail(`solution throws: ${good.error}`);
      good.results.forEach((r) => {
        if (!r.pass) fail(`solution fails "${r.m}"${r.err ? ' — ' + r.err : ''}`);
      });
      if (good.results.length !== lesson.tests.length && !good.error) fail('not every test ran for the solution');

      const start = await runLesson(lesson.code, lesson);
      if (!start.error && start.results.length && start.results.every((r) => r.pass)) fail('starting code already passes every check');
    }
  }
  console.log(`\n${count} lessons in ${modules.length} modules checked, ${problems} problem(s).`);
  process.exit(problems ? 1 : 0);
})();
