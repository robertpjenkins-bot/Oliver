'use strict';

/* global CodeMirror, JSA_TEMPLATES, makeConsole */

const Playground = (() => {
  const $ = (id) => document.getElementById(id);
  const out = makeConsole($('pg-output'));
  const modes = { js: 'javascript', html: 'htmlmixed', css: 'css' };

  let editor;
  let docs = {};
  let activeFile = 'js';
  let project = null; // { id, name, html, css, js, width, height, template }
  let saveTimer, runTimer;
  let previewToken = null;

  // ── building the page ──

  // Puts the CSS and JS into the HTML, replacing the <link href="style.css">
  // and <script src="script.js"> tags when present, like a browser would load
  // them. The result is one self-contained page.
  function buildDocument(p, { preview } = {}) {
    let html = p.html || '';
    const style = `<style>\n${p.css || ''}\n</style>`;
    const script = `<script>\n${(p.js || '').replace(/<\/script/gi, '<\\/script')}\n</script>`;

    if (!/<html[\s>]/i.test(html)) html = `<!doctype html><html><head><meta charset="utf-8"><title>${esc(p.name || 'App')}</title></head><body>\n${html}\n</body></html>`;

    const linkRe = /<link[^>]*href=["']?style\.css["']?[^>]*>/i;
    html = linkRe.test(html) ? html.replace(linkRe, () => style) : html.replace(/<\/head>/i, () => style + '\n</head>');

    const scriptRe = /<script[^>]*src=["']?script\.js["']?[^>]*>\s*<\/script>/i;
    html = scriptRe.test(html) ? html.replace(scriptRe, () => script) : html.replace(/<\/body>/i, () => script + '\n</body>');

    if (preview) {
      const hook = `<script>(${previewHook.toString()})(${JSON.stringify(previewToken)});</script>`;
      html = /<head[^>]*>/i.test(html) ? html.replace(/<head[^>]*>/i, (m) => m + hook) : hook + html;
    }
    return html;
  }

  function esc(s) {
    return String(s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]);
  }

  // Runs inside the preview iframe: forwards console output and errors to the
  // app, and stands in for localStorage, which a sandboxed frame cannot use.
  function previewHook(token) {
    const post = (type, args) => {
      const text = Array.prototype.map
        .call(args, (a) => {
          if (typeof a === 'string') return a;
          if (a instanceof Error) return a.name + ': ' + a.message;
          try {
            return JSON.stringify(a);
          } catch (e) {
            return String(a);
          }
        })
        .join(' ');
      parent.postMessage({ token, type, text }, '*');
    };
    ['log', 'info', 'warn', 'error', 'debug'].forEach((type) => {
      const orig = console[type];
      console[type] = function () {
        post(type === 'debug' ? 'log' : type, arguments);
        orig.apply(console, arguments);
      };
    });
    addEventListener('error', (e) => post('error', ['Line ' + e.lineno + ': ' + e.message]));
    addEventListener('unhandledrejection', (e) => post('error', ['Uncaught (in promise) ' + (e.reason && e.reason.message ? e.reason.message : e.reason)]));
    try {
      window.localStorage.getItem('x');
    } catch (e) {
      const store = new Map();
      const mem = {
        getItem: (k) => (store.has(String(k)) ? store.get(String(k)) : null),
        setItem: (k, v) => store.set(String(k), String(v)),
        removeItem: (k) => store.delete(String(k)),
        clear: () => store.clear(),
        key: (i) => Array.from(store.keys())[i] ?? null,
        get length() {
          return store.size;
        },
      };
      Object.defineProperty(window, 'localStorage', { value: mem, configurable: true });
      Object.defineProperty(window, 'sessionStorage', { value: mem, configurable: true });
    }
  }

  function collect() {
    if (!project) return null;
    project.html = docs.html.getValue();
    project.css = docs.css.getValue();
    project.js = docs.js.getValue();
    project.name = $('project-name').value.trim() || 'Untitled';
    return project;
  }

  function run() {
    if (!project) return;
    collect();
    out.clear();
    previewToken = 'pg' + Math.random().toString(36).slice(2);
    const frame = document.createElement('iframe');
    frame.setAttribute('sandbox', 'allow-scripts allow-modals allow-forms allow-pointer-lock allow-popups');
    frame.srcdoc = buildDocument(project, { preview: true });
    $('pg-frame-holder').replaceChildren(frame);
  }

  window.addEventListener('message', (e) => {
    if (e.data && e.data.token && e.data.token === previewToken) out.add(e.data.type, e.data.text);
  });

  // ── saving ──

  function markDirty() {
    $('pg-save-state').textContent = '● unsaved';
    clearTimeout(saveTimer);
    saveTimer = setTimeout(save, 1200);
    if ($('pg-autorun').checked) {
      clearTimeout(runTimer);
      runTimer = setTimeout(run, 900);
    }
  }

  async function save() {
    if (!project) return;
    clearTimeout(saveTimer);
    collect();
    const meta = await window.api.saveProject(project);
    project.id = meta.id;
    project.created = meta.created;
    $('pg-save-state').textContent = '✓ saved';
    renderList();
  }

  // ── project list ──

  async function renderList() {
    const list = await window.api.listProjects();
    const el = $('project-list');
    el.replaceChildren();
    if (!list.length) {
      el.innerHTML = '<div class="project-empty">No projects yet. Make one with <b>New project</b>, or open a lesson in the playground.</div>';
      return;
    }
    for (const p of list) {
      const b = document.createElement('button');
      b.className = 'project-item' + (project && project.id === p.id ? ' current' : '');
      b.innerHTML = '<span class="n"></span><span class="when"></span>';
      b.querySelector('.n').textContent = p.name;
      b.querySelector('.when').textContent = 'edited ' + new Date(p.updated).toLocaleString();
      b.onclick = () => openProject(p.id);
      el.appendChild(b);
    }
  }

  async function openProject(id) {
    if (project) await save();
    const p = await window.api.loadProject(id);
    if (!p) return;
    load(p);
  }

  function load(p) {
    project = p;
    $('project-name').value = p.name;
    for (const f of ['js', 'html', 'css']) {
      docs[f] = CodeMirror.Doc(p[f] || '', modes[f]);
    }
    switchFile(activeFile);
    $('pg-save-state').textContent = p.id ? '✓ saved' : '';
    setEnabled(true);
    run();
    renderList();
  }

  function switchFile(file) {
    activeFile = file;
    document.querySelectorAll('.etab').forEach((t) => t.classList.toggle('active', t.dataset.file === file));
    if (docs[file]) editor.swapDoc(docs[file]);
    editor.focus();
  }

  function setEnabled(on) {
    ['btn-pg-run', 'btn-pg-save', 'btn-pg-export', 'btn-pg-pack', 'btn-pg-delete', 'project-name'].forEach((id) => ($(id).disabled = !on));
  }

  async function create(name, template, overrides = {}) {
    if (project) await save();
    const t = JSA_TEMPLATES.find((x) => x.id === template) || JSA_TEMPLATES[0];
    const p = {
      name: name || 'My App',
      template: t.id,
      html: overrides.html !== undefined ? overrides.html : t.html.replace(/<title>[^<]*<\/title>/, `<title>${esc(name || 'My App')}</title>`),
      css: overrides.css !== undefined ? overrides.css : t.css,
      js: overrides.js !== undefined ? overrides.js : t.js,
      width: 900,
      height: 700,
    };
    const meta = await window.api.saveProject(p);
    Object.assign(p, meta);
    load(p);
  }

  // Called from a lesson's "Open in Playground" button.
  async function createFrom({ name, js, html }) {
    document.querySelector('.tab[data-view="playground"]').click();
    let page;
    if (html !== undefined) {
      page = `<!doctype html>\n<html>\n<head>\n  <meta charset="utf-8">\n  <title>${esc(name)}</title>\n  <link rel="stylesheet" href="style.css">\n</head>\n<body>\n${html.trim()}\n\n  <script src="script.js"></script>\n</body>\n</html>\n`;
    }
    await create(name, 'blank', { js, html: page, css: html !== undefined ? '' : undefined });
    toast('Copied into a new playground project');
  }

  // ── dialogs ──

  let chosenTemplate = 'blank';
  function openNewDialog() {
    const list = $('template-list');
    list.replaceChildren();
    for (const t of JSA_TEMPLATES) {
      const b = document.createElement('button');
      b.className = 'template' + (t.id === chosenTemplate ? ' selected' : '');
      b.innerHTML = '<b></b><span></span>';
      b.querySelector('b').textContent = t.name;
      b.querySelector('span').textContent = t.desc;
      b.onclick = () => {
        chosenTemplate = t.id;
        list.querySelectorAll('.template').forEach((x) => x.classList.toggle('selected', x === b));
        if (/^(My App|Blank page|Click counter|To-do list|Canvas game|Drawing pad|Quiz app)$/.test($('new-name').value)) $('new-name').value = t.name;
      };
      list.appendChild(b);
    }
    $('modal-new').hidden = false;
    $('new-name').select();
  }

  async function exportHtml() {
    await save();
    const r = await window.api.exportHtml(project.name, buildDocument(project));
    if (r.filePath) toast('Saved ' + r.filePath);
  }

  let lastPacked = null;
  async function openPackDialog() {
    await save();
    $('pack-name').textContent = project.name;
    $('pack-width').value = project.width || 900;
    $('pack-height').value = project.height || 700;
    $('pack-log').hidden = true;
    $('pack-log').textContent = '';
    $('btn-pack-show').hidden = true;
    $('btn-pack-go').disabled = false;
    $('modal-pack').hidden = false;
    const avail = await window.api.packAvailable();
    if (!avail.ok) {
      $('pack-log').hidden = false;
      $('pack-log').textContent = avail.reason;
      $('btn-pack-go').disabled = true;
    }
  }

  async function pack() {
    project.width = Math.max(200, Math.min(4000, parseInt($('pack-width').value, 10) || 900));
    project.height = Math.max(200, Math.min(4000, parseInt($('pack-height').value, 10) || 700));
    await save();
    const log = $('pack-log');
    log.hidden = false;
    log.textContent = '';
    $('btn-pack-go').disabled = true;
    $('btn-pack-close').disabled = true;
    const stop = window.api.onPackProgress((msg) => {
      log.textContent += msg + '\n';
      log.scrollTop = log.scrollHeight;
    });
    try {
      const r = await window.api.packAppImage({
        name: project.name,
        document: buildDocument(project),
        width: project.width,
        height: project.height,
      });
      if (r.canceled) {
        log.hidden = !log.textContent;
      } else if (r.error) {
        log.textContent += '\n✗ ' + r.error + '\n';
      } else {
        lastPacked = r.filePath;
        log.textContent += `\n✓ Your app is ready!\n${r.filePath}\n\nDouble-click it to run, or share the file with a friend.\n`;
        $('btn-pack-show').hidden = false;
      }
      log.scrollTop = log.scrollHeight;
    } finally {
      stop();
      $('btn-pack-go').disabled = false;
      $('btn-pack-close').disabled = false;
    }
  }

  async function remove() {
    if (!project || !confirm(`Delete “${project.name}”? This cannot be undone.`)) return;
    clearTimeout(saveTimer);
    if (project.id) await window.api.deleteProject(project.id);
    project = null;
    const list = await window.api.listProjects();
    if (list.length) openProject(list[0].id);
    else {
      for (const f of ['js', 'html', 'css']) docs[f] = CodeMirror.Doc('', modes[f]);
      editor.swapDoc(docs[activeFile]);
      $('project-name').value = '';
      $('pg-frame-holder').replaceChildren();
      out.clear();
      setEnabled(false);
      renderList();
    }
  }

  function toast(text) {
    const t = $('toast');
    t.textContent = text;
    t.hidden = false;
    clearTimeout(toast.timer);
    toast.timer = setTimeout(() => (t.hidden = true), 3500);
  }

  async function init() {
    for (const f of ['js', 'html', 'css']) docs[f] = CodeMirror.Doc('', modes[f]);
    editor = CodeMirror($('pg-editor'), {
      value: docs.js,
      theme: 'jsa',
      lineNumbers: true,
      tabSize: 2,
      indentUnit: 2,
      autoCloseBrackets: true,
      autoCloseTags: true,
      matchBrackets: true,
      styleActiveLine: true,
      extraKeys: {
        'Ctrl-Enter': run,
        'Ctrl-S': save,
        'Ctrl-/': 'toggleComment',
        'Ctrl-Space': 'autocomplete',
        Tab: (cm) => (cm.somethingSelected() ? cm.indentSelection('add') : cm.replaceSelection('  ')),
      },
    });
    editor.on('change', (_cm, change) => {
      if (change.origin !== 'setValue') markDirty();
    });
    setEnabled(false);

    document.querySelectorAll('.etab').forEach((t) => (t.onclick = () => switchFile(t.dataset.file)));
    $('btn-pg-run').onclick = run;
    $('btn-pg-save').onclick = save;
    $('btn-pg-export').onclick = exportHtml;
    $('btn-pg-pack').onclick = openPackDialog;
    $('btn-pack-go').onclick = pack;
    $('btn-pack-show').onclick = () => lastPacked && window.api.showItem(lastPacked);
    $('btn-pg-delete').onclick = remove;
    $('btn-pg-clear').onclick = () => out.clear();
    $('btn-new-project').onclick = openNewDialog;
    $('project-name').oninput = markDirty;
    $('btn-create-project').onclick = () => {
      $('modal-new').hidden = true;
      create($('new-name').value.trim(), chosenTemplate);
    };
    $('new-name').onkeydown = (e) => e.key === 'Enter' && $('btn-create-project').click();
    document.querySelectorAll('[data-close]').forEach((b) => (b.onclick = () => (b.closest('.modal').hidden = true)));
    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') document.querySelectorAll('.modal').forEach((m) => {
        if (!$('btn-pack-close').disabled) m.hidden = true;
      });
    });

    const list = await window.api.listProjects();
    await renderList();
    if (list.length) openProject(list[0].id);
  }

  return { init, createFrom, refresh: () => editor && editor.refresh(), toast, buildDocument };
})();

window.Playground = Playground;
