'use strict';

/* global CodeMirror, jsaFormat, jsaRunProgram, renderMarkdown, renderInline, JSA_MODULES */

// ───────────────────────── Running learner code ─────────────────────────
//
// Plain lessons run in a throwaway Web Worker, which can be terminated if the
// code loops forever. Lessons about web pages need a real document, so they
// run in a sandboxed iframe instead.

// Plain, readable defaults for lesson pages that bring no styles of their own.
const DOM_BASE_STYLE = '<style>body{font-family:system-ui,sans-serif;margin:12px;color:#1f2330;line-height:1.4}button{font:inherit;padding:4px 10px;margin:2px}input,select,textarea{font:inherit;padding:4px}</style>';

const JSARunner = (() => {
  const workerSource = [
    jsaFormat.toString(),
    jsaRunProgram.toString(),
    `self.addEventListener('unhandledrejection', (e) => {
       const r = e.reason;
       self.postMessage({ kind: 'log', type: 'error', text: 'Uncaught (in promise) ' + (r && r.message ? (r.name || 'Error') + ': ' + r.message : jsaFormat(r, 1)) });
     });
     self.addEventListener('error', (e) => {
       self.postMessage({ kind: 'log', type: 'error', text: 'Uncaught ' + e.message });
       e.preventDefault();
     });
     let shown = 0;
     const emit = (type, text) => {
       // Only the first lines are sent to the page, so a chatty loop cannot flood it.
       shown++;
       if (shown <= 500) self.postMessage({ kind: 'log', type, text });
       else if (shown === 501) self.postMessage({ kind: 'log', type: 'system', text: '… more output was hidden' });
     };
     self.onmessage = async (e) => {
       const d = e.data;
       const r = await jsaRunProgram(d.code, d.tests, d.opts, emit);
       self.postMessage({ kind: 'done', result: r });
     };`,
  ].join('\n');
  const workerUrl = URL.createObjectURL(new Blob([workerSource], { type: 'text/javascript' }));

  function runInWorker(code, lesson, onLog) {
    return new Promise((resolve) => {
      const worker = new Worker(workerUrl);
      const limit = 5000 + (lesson.wait || 0);
      const timer = setTimeout(() => {
        worker.terminate();
        resolve({ error: `Your code ran for more than ${limit / 1000} seconds, so it was stopped. Is there a loop that never ends?`, results: [], timeout: true });
      }, limit);
      worker.onmessage = (e) => {
        if (e.data.kind === 'log') onLog(e.data.type, e.data.text);
        else {
          clearTimeout(timer);
          // Give timers a moment to log, then stop the worker.
          setTimeout(() => worker.terminate(), 1500);
          resolve(e.data.result);
        }
      };
      worker.postMessage({ code, tests: lesson.tests || [], opts: { wait: lesson.wait || 0 } });
    });
  }

  let frameSeq = 0;
  function runInFrame(code, lesson, onLog, holder) {
    return new Promise((resolve) => {
      const token = 'jsa' + ++frameSeq + Math.random().toString(36).slice(2);
      const payload = JSON.stringify({ code, tests: lesson.tests || [], opts: { wait: lesson.wait || 0 }, token }).replace(/</g, '\\u003c');
      const harness = `<script>
${jsaFormat.toString()}
${jsaRunProgram.toString()}
(function () {
  var d = ${payload};
  var post = function (m) { m.token = d.token; parent.postMessage(m, '*'); };
  var shown = 0;
  window.addEventListener('error', function (e) { post({ kind: 'log', type: 'error', text: 'Uncaught ' + e.message }); });
  window.addEventListener('DOMContentLoaded', function () {
    jsaRunProgram(d.code, d.tests, d.opts, function (type, text) {
      shown++;
      if (shown <= 500) post({ kind: 'log', type: type, text: text });
      else if (shown === 501) post({ kind: 'log', type: 'system', text: '… more output was hidden' });
    })
      .then(function (r) { post({ kind: 'done', result: r }); });
  });
})();
<\/script>`;
      const html = DOM_BASE_STYLE + (lesson.dom || '');
      const doc = /<\/head>/i.test(html) ? html.replace(/<\/head>/i, harness + '</head>') : harness + html;
      const frame = document.createElement('iframe');
      frame.setAttribute('sandbox', 'allow-scripts allow-modals');
      const timer = setTimeout(() => {
        window.removeEventListener('message', onMessage);
        resolve({ error: 'Your code took too long to finish.', results: [], timeout: true });
      }, 6000 + (lesson.wait || 0));
      function onMessage(e) {
        if (!e.data || e.data.token !== token) return;
        if (e.data.kind === 'log') onLog(e.data.type, e.data.text);
        else {
          clearTimeout(timer);
          window.removeEventListener('message', onMessage);
          resolve(e.data.result);
        }
      }
      window.addEventListener('message', onMessage);
      holder.replaceChildren(frame);
      frame.srcdoc = doc;
    });
  }

  return {
    run(code, lesson, onLog, frameHolder) {
      return lesson.dom !== undefined ? runInFrame(code, lesson, onLog, frameHolder) : runInWorker(code, lesson, onLog);
    },
  };
})();

// ───────────────────────── Console panel helper ─────────────────────────

function makeConsole(el) {
  return {
    clear() {
      el.replaceChildren();
    },
    add(type, text) {
      const div = document.createElement('div');
      div.className = 'line ' + type;
      div.textContent = text;
      el.appendChild(div);
      el.scrollTop = el.scrollHeight;
    },
    empty(text) {
      if (!el.children.length) {
        const div = document.createElement('div');
        div.className = 'empty';
        div.textContent = text;
        el.appendChild(div);
      }
    },
  };
}

// ───────────────────────── The Learn view ─────────────────────────

const Learn = (() => {
  const $ = (id) => document.getElementById(id);
  const modules = JSA_MODULES;
  const lessons = modules.flatMap((m) => m.lessons);
  const byId = new Map(lessons.map((l) => [l.id, l]));
  const moduleById = new Map(modules.map((m) => [m.id, m]));

  let progress = { completed: {}, code: {}, current: null };
  let current = null;
  let editor;
  let running = false;
  const out = makeConsole($('lesson-output'));

  let saveTimer;
  function saveSoon() {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(() => window.api.saveProgress(progress), 600);
  }

  function updateOverall() {
    const done = lessons.filter((l) => progress.completed[l.id]).length;
    $('overall-count').textContent = done;
    $('overall-total').textContent = lessons.length;
    $('overall-bar').style.width = (100 * done) / lessons.length + '%';
  }

  // ── sidebar ──
  function buildSidebar() {
    const list = $('module-list');
    list.replaceChildren();
    modules.forEach((mod, mi) => {
      const wrap = document.createElement('div');
      wrap.className = 'module';
      wrap.dataset.module = mod.id;
      const head = document.createElement('button');
      head.className = 'module-head';
      head.innerHTML = `<span class="num">${mi + 1}</span><span class="name"></span><span class="count"></span>`;
      head.querySelector('.name').textContent = `${mod.icon || ''} ${mod.title}`.trim();
      head.onclick = () => wrap.classList.toggle('open');
      const bar = document.createElement('div');
      bar.className = 'module-bar';
      bar.innerHTML = '<div></div>';
      const links = document.createElement('div');
      links.className = 'lesson-links';
      mod.lessons.forEach((lesson, li) => {
        const a = document.createElement('button');
        a.className = 'lesson-link';
        a.dataset.id = lesson.id;
        a.innerHTML = '<span class="tick"></span><span class="t"></span>';
        a.querySelector('.t').textContent = `${li + 1}. ${lesson.t}`;
        a.onclick = () => open(lesson.id);
        links.appendChild(a);
      });
      wrap.append(head, bar, links);
      list.appendChild(wrap);
    });
    refreshSidebar();
  }

  function refreshSidebar() {
    document.querySelectorAll('.module').forEach((wrap) => {
      const mod = moduleById.get(wrap.dataset.module);
      const done = mod.lessons.filter((l) => progress.completed[l.id]).length;
      const count = wrap.querySelector('.count');
      count.textContent = `${done}/${mod.lessons.length}`;
      count.classList.toggle('done', done === mod.lessons.length);
      wrap.querySelector('.module-bar div').style.width = (100 * done) / mod.lessons.length + '%';
    });
    document.querySelectorAll('.lesson-link').forEach((a) => {
      a.querySelector('.tick').textContent = progress.completed[a.dataset.id] ? '✓' : '';
      a.classList.toggle('current', current && a.dataset.id === current.id);
    });
    updateOverall();
  }

  function search(q) {
    q = q.trim().toLowerCase();
    const list = $('module-list');
    list.classList.toggle('searching', !!q);
    document.querySelectorAll('.module').forEach((wrap) => {
      const mod = moduleById.get(wrap.dataset.module);
      let any = false;
      wrap.querySelectorAll('.lesson-link').forEach((a) => {
        const l = byId.get(a.dataset.id);
        const hit = !q || l.t.toLowerCase().includes(q) || (l.b || '').toLowerCase().includes(q) || mod.title.toLowerCase().includes(q);
        a.classList.toggle('hidden', !hit);
        any = any || hit;
      });
      wrap.classList.toggle('hidden', !any);
    });
  }

  // ── lesson page ──
  function open(id) {
    const lesson = byId.get(id) || lessons[0];
    if (current && editor) {
      progress.code[current.id] = editor.getValue();
    }
    current = lesson;
    progress.current = lesson.id;
    saveSoon();

    const mod = moduleById.get(lesson.module);
    const idx = mod.lessons.indexOf(lesson);
    const globalIdx = lessons.indexOf(lesson);
    $('lesson-crumbs').textContent = `Module ${modules.indexOf(mod) + 1} · ${mod.title} · Lesson ${idx + 1} of ${mod.lessons.length}  (#${globalIdx + 1})`;
    $('lesson-title').textContent = lesson.t;
    $('lesson-body').innerHTML = renderMarkdown(lesson.b);
    renderChecks(null);
    $('lesson-hint').hidden = true;
    $('btn-hint').hidden = !lesson.hint;
    $('btn-prev').disabled = globalIdx === 0;
    $('btn-next').disabled = globalIdx === lessons.length - 1;
    $('btn-next').classList.remove('glow');
    setStatus(progress.completed[lesson.id] ? '✓ Completed' : '', progress.completed[lesson.id] ? 'pass' : '');

    editor.setValue(progress.code[lesson.id] ?? lesson.code ?? '');
    editor.clearHistory();
    out.clear();
    out.empty('Press ▶ Run & check (Ctrl+Enter) to run your code.');

    const isDom = lesson.dom !== undefined;
    $('dom-preview').hidden = !isDom;
    if (isDom) {
      const frame = document.createElement('iframe');
      frame.setAttribute('sandbox', '');
      frame.srcdoc = DOM_BASE_STYLE + lesson.dom;
      $('dom-frame-holder').replaceChildren(frame);
    }

    document.querySelectorAll('.module').forEach((w) => {
      if (w.dataset.module === mod.id) w.classList.add('open');
    });
    refreshSidebar();
    const link = document.querySelector(`.lesson-link[data-id="${lesson.id}"]`);
    if (link) link.scrollIntoView({ block: 'nearest' });
    $('lesson-text').scrollTop = 0;
    setTimeout(() => editor.refresh(), 0);
  }

  function renderChecks(results) {
    const ul = $('lesson-checks');
    ul.replaceChildren();
    (current.tests || []).forEach((test, i) => {
      const li = document.createElement('li');
      const r = results && results[i];
      if (r) li.className = r.pass ? 'pass' : 'fail';
      const span = document.createElement('span');
      span.innerHTML = renderInline(test[1]);
      if (r && r.err) {
        const err = document.createElement('span');
        err.className = 'err';
        err.textContent = r.err;
        span.appendChild(err);
      }
      li.appendChild(span);
      ul.appendChild(li);
    });
  }

  function setStatus(text, cls) {
    const s = $('run-status');
    s.textContent = text;
    s.className = 'status ' + (cls || '');
  }

  async function run() {
    if (running) return;
    running = true;
    $('btn-run').disabled = true;
    const lesson = current;
    const code = editor.getValue();
    progress.code[lesson.id] = code;
    saveSoon();
    out.clear();
    setStatus('Running…');
    const result = await JSARunner.run(code, lesson, (type, text) => out.add(type, text), $('dom-frame-holder'));
    running = false;
    $('btn-run').disabled = false;
    if (lesson !== current) return;

    if (result.error) out.add('error', result.error);
    out.empty('(nothing was printed)');
    const results = result.results.length ? result.results : (lesson.tests || []).map(() => ({ pass: false }));
    renderChecks(results);
    const passed = results.filter((r) => r.pass).length;
    if (!result.error && results.length && passed === results.length) {
      const first = !progress.completed[lesson.id];
      progress.completed[lesson.id] = Date.now();
      saveSoon();
      setStatus(first ? '🎉 Lesson complete!' : '✓ All checks pass', 'pass');
      $('btn-next').classList.add('glow');
      refreshSidebar();
    } else if (result.error) {
      setStatus(result.syntax ? 'Syntax error — check the console' : 'Your code threw an error', 'fail');
    } else {
      setStatus(`${passed} of ${results.length} checks pass`, 'fail');
    }
  }

  function step(delta) {
    const i = lessons.indexOf(current) + delta;
    if (i >= 0 && i < lessons.length) open(lessons[i].id);
  }

  async function init() {
    editor = CodeMirror($('lesson-editor'), {
      mode: 'javascript',
      theme: 'jsa',
      lineNumbers: true,
      tabSize: 2,
      indentUnit: 2,
      autoCloseBrackets: true,
      matchBrackets: true,
      styleActiveLine: true,
      extraKeys: {
        'Ctrl-Enter': run,
        'Cmd-Enter': run,
        'Ctrl-/': 'toggleComment',
        'Ctrl-Space': 'autocomplete',
        Tab: (cm) => (cm.somethingSelected() ? cm.indentSelection('add') : cm.replaceSelection('  ')),
      },
    });
    editor.on('change', () => {
      if (current) {
        progress.code[current.id] = editor.getValue();
        saveSoon();
      }
    });

    try {
      progress = Object.assign({ completed: {}, code: {}, current: null }, await window.api.loadProgress());
    } catch (e) {
      console.error(e);
    }
    buildSidebar();
    open(progress.current && byId.has(progress.current) ? progress.current : lessons[0].id);

    $('btn-run').onclick = run;
    $('btn-reset').onclick = () => {
      if (confirm('Replace your code with the starting code for this lesson?')) {
        editor.setValue(current.code || '');
      }
    };
    $('btn-hint').onclick = () => {
      $('lesson-hint').innerHTML = renderMarkdown('**Hint:** ' + current.hint);
      $('lesson-hint').hidden = !$('lesson-hint').hidden;
    };
    $('btn-solution').onclick = () => {
      if (confirm('Show one possible solution? It will replace the code in the editor (Ctrl+Z brings yours back).')) {
        editor.setValue(current.sol || '');
      }
    };
    $('btn-prev').onclick = () => step(-1);
    $('btn-next').onclick = () => step(1);
    $('btn-clear-console').onclick = () => out.clear();
    $('lesson-search').oninput = (e) => search(e.target.value);
    $('btn-to-playground').onclick = () => {
      window.Playground.createFrom({
        name: current.t,
        js: editor.getValue(),
        html: current.dom !== undefined ? current.dom : undefined,
      });
    };
  }

  return { init, refresh: () => editor && editor.refresh(), lessons };
})();

window.Learn = Learn;
window.makeConsole = makeConsole;
