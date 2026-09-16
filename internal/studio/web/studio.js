// The studio: an editor for Oliver game projects.
//
// Everything on screen edits one object — the project — and every panel is a
// view of part of it. Saving writes that object to a file; playing hands the
// same object to the engine; exporting inlines it into a page. There is no
// separate "build" of a game, which is why what is played here is exactly what
// is played by whoever it is given to.
(function () {
  'use strict';

  var G = null;          // the project being edited
  var NAME = '';         // the file it is stored under
  var saveTimer = null;
  var runtime = null;    // the running engine, when the play tab is open
  var sel = {
    level: 0, sprite: 0, frame: 0, object: 0, rule: 0,
    tile: '#', place: null, color: 1, zoom: 2
  };

  // ------------------------------------------------------------------ util

  function $(id) { return document.getElementById(id); }

  function el(tag, attrs, kids) {
    var node = document.createElement(tag);
    attrs = attrs || {};
    Object.keys(attrs).forEach(function (k) {
      if (k === 'class') node.className = attrs[k];
      else if (k === 'text') node.textContent = attrs[k];
      else if (k.slice(0, 2) === 'on') node.addEventListener(k.slice(2), attrs[k]);
      else if (attrs[k] === true) node.setAttribute(k, '');
      else if (attrs[k] !== false && attrs[k] != null) node.setAttribute(k, attrs[k]);
    });
    (kids || []).forEach(function (k) { if (k) node.appendChild(k); });
    return node;
  }

  function api(path, options) {
    options = options || {};
    options.headers = Object.assign({
      'X-Oliver-Studio': '1',
      'Content-Type': 'application/json'
    }, options.headers || {});
    return fetch(path, options).then(function (r) {
      return r.json().catch(function () { return {}; }).then(function (body) {
        if (!r.ok) throw new Error(body.error || (r.status + ' ' + r.statusText));
        return body;
      });
    });
  }

  function status(text, level) {
    // The status bar is one line tall. Anything longer — a list of places
    // looked in, a paragraph about what to do next — goes above it, where it
    // can be read and dismissed.
    var lines = String(text).split('\n');
    $('statustext').textContent = lines[0];
    $('status').className = level || '';
    if (lines.length > 1) showAlert(text);
  }

  function showAlert(text) {
    $('alerttext').textContent = text;
    $('alert').hidden = false;
  }

  function clearAlert() { $('alert').hidden = true; }

  function problemsOf(list) {
    if (!list || !list.length) return null;
    status(list.length + ' problem' + (list.length > 1 ? 's' : '') + ': ' + list[0], 'warn');
    return list;
  }

  // ------------------------------------------------------------ persistence

  function markDirty(redrawPlay) {
    if (!G) return;
    status('unsaved changes…');
    clearTimeout(saveTimer);
    saveTimer = setTimeout(save, 1200);
    if (redrawPlay && runtime) startPlaying();
  }

  function save() {
    if (!G) return Promise.resolve();
    clearTimeout(saveTimer);
    return api('/api/game?name=' + encodeURIComponent(NAME), {
      method: 'PUT',
      body: JSON.stringify(G)
    }).then(function (res) {
      NAME = res.name || NAME;
      if (!problemsOf(res.problems)) status('saved ' + NAME);
      refreshGameList();
    }).catch(function (e) { status(e.message, 'err'); });
  }

  function loadGame(name) {
    return api('/api/game?name=' + encodeURIComponent(name)).then(function (game) {
      G = game;
      NAME = name;
      sel.level = 0; sel.sprite = 0; sel.frame = 0; sel.object = 0; sel.rule = 0;
      sel.tile = (G.levels[0] && G.levels[0].tiles[0] && G.levels[0].tiles[0].char) || '#';
      sel.place = null;
      renderAll();
      status('opened ' + G.name);
    }).catch(function (e) { status(e.message, 'err'); });
  }

  function refreshGameList() {
    return api('/api/games').then(function (list) {
      var select = $('gamelist');
      select.innerHTML = '';
      list.forEach(function (info) {
        select.appendChild(el('option', {
          value: info.name,
          text: info.title + '  (' + info.genre + ', ' + info.levels + ' levels)',
          selected: info.name === NAME
        }));
      });
      if (!list.length) select.appendChild(el('option', { value: '', text: 'no games yet' }));
      return list;
    });
  }

  // ------------------------------------------------------------- rendering

  // Sprites are stored as text; this is the one place the studio turns that
  // text into pixels, shared by every panel that shows a sprite.
  function drawSprite(ctx, sprite, frame, x, y, scale) {
    if (!sprite) return;
    var rows = sprite.frames[Math.min(frame || 0, sprite.frames.length - 1)] || [];
    for (var r = 0; r < rows.length; r++) {
      for (var c = 0; c < rows[r].length; c++) {
        var ch = rows[r][c];
        if (ch === '.' || ch === ' ') continue;
        var idx = parseInt(ch, 16);
        if (isNaN(idx)) continue;
        ctx.fillStyle = G.palette[idx] || '#f0f';
        ctx.fillRect(x + c * scale, y + r * scale, scale, scale);
      }
    }
  }

  function spriteById(id) {
    for (var i = 0; i < G.sprites.length; i++) if (G.sprites[i].id === id) return G.sprites[i];
    return null;
  }

  function objectById(id) {
    for (var i = 0; i < G.objects.length; i++) if (G.objects[i].id === id) return G.objects[i];
    return null;
  }

  function thumb(sprite, scale) {
    var c = el('canvas');
    var size = sprite ? Math.max(sprite.w, sprite.h) : 8;
    c.width = size * (scale || 1);
    c.height = size * (scale || 1);
    drawSprite(c.getContext('2d'), sprite, 0, 0, 0, scale || 1);
    return c;
  }

  function renderAll() {
    if (!G) return;
    renderLevels();
    renderSprites();
    renderObjects();
    renderRules();
    renderSettings();
    renderPlayLevels();
  }

  // ------------------------------------------------------------------ tabs

  Array.prototype.forEach.call(document.querySelectorAll('#tabs button'), function (b) {
    b.addEventListener('click', function () {
      Array.prototype.forEach.call(document.querySelectorAll('#tabs button'), function (o) {
        o.classList.toggle('on', o === b);
      });
      Array.prototype.forEach.call(document.querySelectorAll('.tabpage'), function (page) {
        page.hidden = page.id !== 'tab-' + b.dataset.tab;
      });
      if (b.dataset.tab === 'play') startPlaying();
      else stopPlaying();
      if (b.dataset.tab === 'levels') drawLevel();
      if (b.dataset.tab === 'sprites') drawSpriteEditor();
    });
  });

  // ------------------------------------------------------------- assistant

  // A note may carry more than a label: when Claude could not be reached the
  // first line says so in the log and the rest — what to actually do about it
  // — belongs in the alert, where there is room to read it.
  function noteEntry(note, text, cls) {
    note = note || 'done';
    var lines = note.split('\n');
    logEntry(lines[0], text, cls);
    if (lines.length > 1) showAlert(note);
  }

  function logEntry(who, text, cls) {
    $('log').insertBefore(
      el('div', { class: 'entry ' + (cls || '') }, [
        el('div', { class: 'who', text: who }),
        el('div', { text: text })
      ]),
      $('log').firstChild
    );
  }

  // Say how the description is being read while it is being written. "It did
  // not know what a Metroid game is" should be something you find out before
  // you wait for a game, not after playing one.
  var readTimer = null;
  $('prompt').addEventListener('input', function () {
    var text = this.value.trim();
    clearTimeout(readTimer);
    if (!text) { $('designnote').textContent = ''; return; }
    readTimer = setTimeout(function () {
      api('/api/read?prompt=' + encodeURIComponent(text))
        .then(function (res) { $('designnote').textContent = res.says || ''; })
        .catch(function () {});
    }, 350);
  });

  $('design').addEventListener('click', function () {
    var prompt = $('prompt').value.trim();
    if (!prompt) { status('describe the game you want first', 'warn'); return; }
    var button = this;
    button.disabled = true;
    status('designing…');
    logEntry('you', prompt, 'you');
    api('/api/design', { method: 'POST', body: JSON.stringify({ prompt: prompt }) })
      .then(function (res) {
        G = res.game; NAME = res.name;
        sel.level = 0; sel.sprite = 0; sel.object = 0; sel.rule = 0;
        sel.tile = (G.levels[0] && G.levels[0].tiles[0] && G.levels[0].tiles[0].char) || '#';
        renderAll();
        refreshGameList();
        if (res.read) logEntry('understood', res.read);
        noteEntry(res.note, G.name + ' — ' + (G.tagline || '') + ' · ' +
          G.levels.length + ' levels, ' + G.objects.length + ' objects');
        // A game made offline when Claude was meant to make it is still a
        // game, but saying only "made it" would hide that the thing you
        // connected never ran.
        status('made ' + G.name, res.note.indexOf('\n') < 0 ? '' : 'warn');
      })
      .catch(function (e) { logEntry('failed', e.message, 'err'); status(e.message, 'err'); })
      .then(function () { button.disabled = false; });
  });

  function applyInstruction(text) {
    if (!G) { status('open or make a game first', 'warn'); return; }
    if (!text) return;
    status('working…');
    logEntry('you', text, 'you');
    api('/api/assist', {
      method: 'POST',
      body: JSON.stringify({ name: NAME, instruction: text, game: G })
    }).then(function (res) {
      G = res.game; NAME = res.name;
      renderAll();
      noteEntry(res.note, describeGame());
      if (!problemsOf(res.problems)) status('done');
      refreshGameList();
    }).catch(function (e) { logEntry('failed', e.message, 'err'); status(e.message, 'err'); });
  }

  function describeGame() {
    return G.name + ' · ' + G.genre + ' · ' + G.levels.length + ' levels · ' +
      G.objects.length + ' objects · ' + G.rules.length + ' rules';
  }

  $('apply').addEventListener('click', function () {
    applyInstruction($('instruction').value.trim());
    $('instruction').value = '';
  });
  $('instruction').addEventListener('keydown', function (e) {
    if (e.key === 'Enter') { applyInstruction(this.value.trim()); this.value = ''; }
  });

  ['make it harder', 'make it easier', 'add a boss', 'add another level',
   'more enemies', 'more coins', 'give the player a gun', 'add a time limit',
   'repaint it as ice'].forEach(function (s) {
    $('suggestions').appendChild(el('button', { text: s, onclick: function () { applyInstruction(s); } }));
  });

  // ------------------------------------------------------------------ play

  function renderPlayLevels() {
    var select = $('playlevel');
    select.innerHTML = '';
    G.levels.forEach(function (l) {
      select.appendChild(el('option', { value: l.id, text: l.name || l.id }));
    });
    select.onchange = startPlaying;
  }

  function startPlaying() {
    if (!G) return;
    stopPlaying();
    var canvas = $('screen');
    var scale = Math.max(1, Math.min(3, Math.floor((window.innerWidth - 320) / G.screen.width)));
    runtime = window.OliverEngine.start(canvas, G, {
      skipTitle: true,
      scale: scale,
      level: $('playlevel').value || undefined
    });
    canvas.focus();
  }

  function stopPlaying() {
    if (runtime) { runtime.stop(); runtime = null; }
  }

  $('restart').addEventListener('click', startPlaying);

  // ---------------------------------------------------------------- levels

  function level() { return G.levels[Math.min(sel.level, G.levels.length - 1)]; }

  function renderLevels() {
    var list = $('levellist');
    list.innerHTML = '';
    G.levels.forEach(function (l, i) {
      list.appendChild(el('li', {
        class: i === sel.level ? 'on' : '',
        onclick: function () { sel.level = i; renderLevels(); drawLevel(); }
      }, [
        el('span', { text: l.name || l.id }),
        el('small', { text: l.width + '×' + l.height })
      ]));
    });

    var tiles = $('tilepalette');
    tiles.innerHTML = '';
    var lv = level();
    if (!lv) return;
    var eraser = el('li', {
      class: sel.tile === '.' && !sel.place ? 'on' : '',
      onclick: function () { sel.place = null; sel.tile = '.'; renderLevels(); }
    }, [el('span', { text: 'empty' })]);
    tiles.appendChild(eraser);
    lv.tiles.forEach(function (t) {
      tiles.appendChild(el('li', {
        class: (!sel.place && sel.tile === t.char) ? 'on' : '',
        title: t.kind,
        onclick: function () { sel.place = null; sel.tile = t.char; renderLevels(); }
      }, [thumb(spriteById(t.sprite), 1), el('span', { text: t.name || t.kind })]));
    });

    var objects = $('objectpalette');
    objects.innerHTML = '';
    G.objects.forEach(function (o) {
      objects.appendChild(el('li', {
        class: sel.place === o.id ? 'on' : '',
        title: o.kind + ' · ' + o.behavior,
        onclick: function () { sel.place = (sel.place === o.id ? null : o.id); renderLevels(); }
      }, [thumb(spriteById(o.sprite), 1), el('span', { text: o.name || o.id })]));
    });

    renderLevelProps();
  }

  function renderLevelProps() {
    var lv = level();
    var box = $('levelprops');
    box.innerHTML = '';
    if (!lv) return;
    box.appendChild(el('div', { class: 'listhead' }, [el('span', { text: 'level' })]));
    box.appendChild(field('name', 'text', lv.name, function (v) { lv.name = v; renderLevels(); markDirty(); }));
    box.appendChild(field('width', 'number', lv.width, function (v) { resizeLevel(lv, parseInt(v, 10), lv.height); }));
    box.appendChild(field('height', 'number', lv.height, function (v) { resizeLevel(lv, lv.width, parseInt(v, 10)); }));
    box.appendChild(field('background', 'color', lv.background || '#101020', function (v) { lv.background = v; drawLevel(); markDirty(); }));
    box.appendChild(select_('scroll', ['follow', 'fixed', 'auto-right'], lv.scroll || 'follow', function (v) { lv.scroll = v; markDirty(); }));
    box.appendChild(select_('next', G.levels.map(function (l) { return l.id; }).concat(['win']), lv.next || 'win', function (v) { lv.next = v; markDirty(); }));
    box.appendChild(field('time limit', 'number', lv.timeLimit || 0, function (v) { lv.timeLimit = parseInt(v, 10) || 0; markDirty(); }));
    box.appendChild(field('intro text', 'text', lv.intro || '', function (v) { lv.intro = v; markDirty(); }));
    box.appendChild(select_('zoom', ['1', '2', '3'], String(sel.zoom), function (v) { sel.zoom = parseInt(v, 10); drawLevel(); }));
    box.appendChild(el('div', { class: 'row' }, [
      el('button', { text: 'duplicate', onclick: function () { duplicateLevel(); } }),
      el('button', { class: 'danger', text: 'delete', onclick: function () { deleteLevel(); } })
    ]));
  }

  function resizeLevel(lv, w, h) {
    if (!w || !h || w < 4 || h < 4 || w > 400 || h > 400) return;
    var rows = [];
    for (var y = 0; y < h; y++) {
      var old = lv.map[y] || '';
      var row = '';
      for (var x = 0; x < w; x++) row += old[x] || '.';
      rows.push(row);
    }
    lv.map = rows; lv.width = w; lv.height = h;
    renderLevels(); drawLevel(); markDirty();
  }

  function duplicateLevel() {
    var copy = JSON.parse(JSON.stringify(level()));
    copy.id = 'level' + (G.levels.length + 1);
    copy.name = (copy.name || copy.id) + ' again';
    copy.next = 'win';
    if (G.levels.length) G.levels[G.levels.length - 1].next = copy.id;
    G.levels.push(copy);
    sel.level = G.levels.length - 1;
    renderLevels(); renderPlayLevels(); drawLevel(); markDirty();
  }

  function deleteLevel() {
    if (G.levels.length < 2) { status('a game needs at least one level', 'warn'); return; }
    var gone = G.levels.splice(sel.level, 1)[0];
    G.levels.forEach(function (l, i) {
      if (l.next === gone.id) l.next = (G.levels[i + 1] || {}).id || 'win';
    });
    if (G.settings.startLevel === gone.id) G.settings.startLevel = G.levels[0].id;
    sel.level = Math.max(0, sel.level - 1);
    renderLevels(); renderPlayLevels(); drawLevel(); markDirty();
  }

  $('addlevel').addEventListener('click', duplicateLevel);

  function drawLevel() {
    var lv = level();
    if (!lv) return;
    var tile = G.screen.tile, z = sel.zoom;
    var canvas = $('levelcanvas');
    canvas.width = lv.width * tile * z;
    canvas.height = lv.height * tile * z;
    var ctx = canvas.getContext('2d');
    ctx.imageSmoothingEnabled = false;
    ctx.fillStyle = lv.background || '#101020';
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    var defs = {};
    lv.tiles.forEach(function (t) { defs[t.char] = t; });
    for (var y = 0; y < lv.height; y++) {
      for (var x = 0; x < lv.width; x++) {
        var def = defs[lv.map[y][x]];
        if (!def) continue;
        var sprite = spriteById(def.sprite);
        if (sprite) drawSprite(ctx, sprite, 0, x * tile * z, y * tile * z, z);
        else {
          ctx.fillStyle = '#666';
          ctx.fillRect(x * tile * z, y * tile * z, tile * z, tile * z);
        }
      }
    }
    (lv.spawns || []).forEach(function (sp) {
      var o = objectById(sp.object);
      if (!o) return;
      drawSprite(ctx, spriteById(o.sprite), 0, sp.x * tile * z, sp.y * tile * z, z);
      if (o.kind === 'player') {
        ctx.strokeStyle = '#f2c94c';
        ctx.strokeRect(sp.x * tile * z + 0.5, sp.y * tile * z + 0.5, tile * z - 1, tile * z - 1);
      }
    });
    // A grid at the tile size, faint enough to read the level through.
    ctx.strokeStyle = 'rgba(255,255,255,.06)';
    ctx.beginPath();
    for (var gx = 0; gx <= lv.width; gx++) { ctx.moveTo(gx * tile * z, 0); ctx.lineTo(gx * tile * z, canvas.height); }
    for (var gy = 0; gy <= lv.height; gy++) { ctx.moveTo(0, gy * tile * z); ctx.lineTo(canvas.width, gy * tile * z); }
    ctx.stroke();
  }

  (function bindLevelCanvas() {
    var canvas = $('levelcanvas');
    var painting = 0;
    function cellOf(e) {
      var rect = canvas.getBoundingClientRect();
      var tile = G.screen.tile * sel.zoom;
      return {
        x: Math.floor((e.clientX - rect.left) * (canvas.width / rect.width) / tile),
        y: Math.floor((e.clientY - rect.top) * (canvas.height / rect.height) / tile)
      };
    }
    function paint(e, erase) {
      var lv = level();
      var c = cellOf(e);
      if (c.x < 0 || c.y < 0 || c.x >= lv.width || c.y >= lv.height) return;
      if (sel.place && !erase) {
        lv.spawns = lv.spawns || [];
        lv.spawns.push({ object: sel.place, x: c.x, y: c.y, dir: 1 });
      } else if (sel.place && erase) {
        lv.spawns = (lv.spawns || []).filter(function (sp) {
          return Math.abs(sp.x - c.x) > 0.9 || Math.abs(sp.y - c.y) > 0.9;
        });
      } else {
        var row = lv.map[c.y];
        lv.map[c.y] = row.slice(0, c.x) + (erase ? '.' : sel.tile) + row.slice(c.x + 1);
      }
      drawLevel();
      markDirty();
    }
    canvas.addEventListener('contextmenu', function (e) { e.preventDefault(); });
    canvas.addEventListener('mousedown', function (e) {
      if (!G) return;
      painting = e.button === 2 ? 2 : 1;
      paint(e, painting === 2);
    });
    canvas.addEventListener('mousemove', function (e) {
      if (painting) paint(e, painting === 2);
    });
    window.addEventListener('mouseup', function () { painting = 0; });
  })();

  // --------------------------------------------------------------- sprites

  function sprite() { return G.sprites[Math.min(sel.sprite, G.sprites.length - 1)]; }

  function renderSprites() {
    var list = $('spritelist');
    list.innerHTML = '';
    G.sprites.forEach(function (s, i) {
      list.appendChild(el('li', {
        class: i === sel.sprite ? 'on' : '',
        onclick: function () { sel.sprite = i; sel.frame = 0; renderSprites(); drawSpriteEditor(); }
      }, [
        el('span', {}, [thumb(s, 1)]),
        el('span', { text: s.id }),
        el('small', { text: s.w + '×' + s.h + (s.frames.length > 1 ? ' ×' + s.frames.length : '') })
      ]));
    });

    var swatches = $('swatches');
    swatches.innerHTML = '';
    G.palette.forEach(function (color, i) {
      swatches.appendChild(el('li', {
        class: i === sel.color ? 'on' : '',
        title: i.toString(16) + ' ' + color,
        onclick: function () { sel.color = i; renderSprites(); }
      }, [
        el('span', { class: 'swatch', style: 'background:' + color }),
        el('span', { text: i.toString(16) })
      ]));
    });

    var props = $('spriteprops');
    props.innerHTML = '';
    var s = sprite();
    if (!s) return;
    props.appendChild(el('div', { class: 'listhead' }, [el('span', { text: 'sprite' })]));
    props.appendChild(field('id', 'text', s.id, function (v) { renameSprite(s, v); }));
    props.appendChild(field('width', 'number', s.w, function (v) { resizeSprite(s, parseInt(v, 10), s.h); }));
    props.appendChild(field('height', 'number', s.h, function (v) { resizeSprite(s, s.w, parseInt(v, 10)); }));
    props.appendChild(field('fps', 'number', s.fps || 0, function (v) { s.fps = parseFloat(v) || 0; markDirty(true); }));
    props.appendChild(el('div', { class: 'row' }, [
      el('button', { text: 'clear', onclick: function () { clearFrame(); } }),
      el('button', { class: 'danger', text: 'delete', onclick: function () { deleteSprite(); } })
    ]));
  }

  function renameSprite(s, name) {
    name = (name || '').trim();
    if (!name || spriteById(name)) { status('that name is taken', 'warn'); return; }
    var old = s.id;
    s.id = name;
    G.objects.forEach(function (o) { if (o.sprite === old) o.sprite = name; });
    G.levels.forEach(function (l) {
      l.tiles.forEach(function (t) { if (t.sprite === old) t.sprite = name; });
    });
    renderSprites(); markDirty(true);
  }

  function resizeSprite(s, w, h) {
    if (!w || !h || w < 1 || h < 1 || w > 64 || h > 64) return;
    s.frames = s.frames.map(function (rows) {
      var out = [];
      for (var y = 0; y < h; y++) {
        var old = rows[y] || '';
        var row = '';
        for (var x = 0; x < w; x++) row += old[x] || '.';
        out.push(row);
      }
      return out;
    });
    s.w = w; s.h = h;
    renderSprites(); drawSpriteEditor(); markDirty(true);
  }

  function clearFrame() {
    var s = sprite();
    s.frames[sel.frame] = s.frames[sel.frame].map(function (row) { return row.replace(/./g, '.'); });
    drawSpriteEditor(); markDirty(true);
  }

  function deleteSprite() {
    var s = sprite();
    var used = G.objects.some(function (o) { return o.sprite === s.id; });
    if (used) { status('that sprite is in use by an object', 'warn'); return; }
    G.sprites.splice(sel.sprite, 1);
    sel.sprite = Math.max(0, sel.sprite - 1);
    renderSprites(); drawSpriteEditor(); markDirty(true);
  }

  $('addsprite').addEventListener('click', function () {
    var size = G.screen.tile;
    var rows = [];
    for (var y = 0; y < size; y++) rows.push(new Array(size + 1).join('.'));
    var id = 'sprite' + (G.sprites.length + 1);
    G.sprites.push({ id: id, w: size, h: size, fps: 0, frames: [rows] });
    sel.sprite = G.sprites.length - 1;
    sel.frame = 0;
    renderSprites(); drawSpriteEditor(); markDirty();
  });

  function drawSpriteEditor() {
    var s = sprite();
    if (!s) return;
    var canvas = $('spritecanvas');
    var zoom = Math.max(4, Math.min(20, Math.floor(360 / Math.max(s.w, s.h))));
    canvas.width = s.w * zoom;
    canvas.height = s.h * zoom;
    var ctx = canvas.getContext('2d');
    ctx.fillStyle = '#101018';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    // A chequerboard behind the drawing, so transparent is visibly nothing.
    ctx.fillStyle = '#16161f';
    for (var y = 0; y < s.h; y++) {
      for (var x = 0; x < s.w; x++) {
        if ((x + y) % 2) ctx.fillRect(x * zoom, y * zoom, zoom, zoom);
      }
    }
    drawSprite(ctx, s, sel.frame, 0, 0, zoom);
    ctx.strokeStyle = 'rgba(255,255,255,.07)';
    ctx.beginPath();
    for (var gx = 0; gx <= s.w; gx++) { ctx.moveTo(gx * zoom, 0); ctx.lineTo(gx * zoom, canvas.height); }
    for (var gy = 0; gy <= s.h; gy++) { ctx.moveTo(0, gy * zoom); ctx.lineTo(canvas.width, gy * zoom); }
    ctx.stroke();
    canvas.dataset.zoom = zoom;
    renderFrames();
  }

  function renderFrames() {
    var s = sprite();
    var box = $('frames');
    box.innerHTML = '';
    s.frames.forEach(function (rows, i) {
      var c = thumb({ w: s.w, h: s.h, frames: [rows] }, 3);
      c.className = i === sel.frame ? 'on' : '';
      c.addEventListener('click', function () { sel.frame = i; drawSpriteEditor(); });
      box.appendChild(c);
    });
    box.appendChild(el('button', {
      text: '+ frame',
      onclick: function () {
        s.frames.push(s.frames[sel.frame].slice());
        sel.frame = s.frames.length - 1;
        if (!s.fps) s.fps = 8;
        drawSpriteEditor(); markDirty(true);
      }
    }));
    if (s.frames.length > 1) {
      box.appendChild(el('button', {
        class: 'danger', text: '− frame',
        onclick: function () {
          s.frames.splice(sel.frame, 1);
          sel.frame = Math.max(0, sel.frame - 1);
          drawSpriteEditor(); markDirty(true);
        }
      }));
    }
  }

  (function bindSpriteCanvas() {
    var canvas = $('spritecanvas');
    var drawing = 0;
    function paint(e, erase) {
      var s = sprite();
      var zoom = parseInt(canvas.dataset.zoom, 10) || 8;
      var rect = canvas.getBoundingClientRect();
      var x = Math.floor((e.clientX - rect.left) * (canvas.width / rect.width) / zoom);
      var y = Math.floor((e.clientY - rect.top) * (canvas.height / rect.height) / zoom);
      if (x < 0 || y < 0 || x >= s.w || y >= s.h) return;
      var row = s.frames[sel.frame][y];
      var ch = erase ? '.' : sel.color.toString(16);
      s.frames[sel.frame][y] = row.slice(0, x) + ch + row.slice(x + 1);
      drawSpriteEditor();
      markDirty(true);
    }
    canvas.addEventListener('contextmenu', function (e) { e.preventDefault(); });
    canvas.addEventListener('mousedown', function (e) {
      if (!G) return;
      drawing = e.button === 2 ? 2 : 1;
      paint(e, drawing === 2);
    });
    canvas.addEventListener('mousemove', function (e) { if (drawing) paint(e, drawing === 2); });
    window.addEventListener('mouseup', function () { drawing = 0; });
  })();

  // --------------------------------------------------------------- widgets

  function field(label, type, value, onchange) {
    var input = el('input', { type: type, value: value == null ? '' : value });
    input.addEventListener('change', function () {
      onchange(type === 'checkbox' ? input.checked : input.value);
    });
    if (type === 'checkbox') input.checked = !!value;
    return el('div', { class: 'field' }, [el('label', { text: label }), input]);
  }

  function check(label, value, onchange) {
    var input = el('input', { type: 'checkbox' });
    input.checked = !!value;
    input.addEventListener('change', function () { onchange(input.checked); });
    return el('div', { class: 'field' }, [el('label', { text: label }), input]);
  }

  function select_(label, options, value, onchange) {
    var s = el('select');
    options.forEach(function (o) {
      s.appendChild(el('option', { value: o, text: o, selected: o === value }));
    });
    s.addEventListener('change', function () { onchange(s.value); });
    return el('div', { class: 'field' }, [el('label', { text: label }), s]);
  }

  // --------------------------------------------------------------- objects

  var KINDS = ['player', 'enemy', 'item', 'hazard', 'block', 'projectile', 'goal', 'decor', 'npc'];
  var BEHAVIORS = ['platformer', 'topdown', 'ship', 'paddle', 'patrol', 'chase', 'flee',
    'float', 'fall', 'bounce', 'orbit', 'projectile', 'static'];

  function renderObjects() {
    var list = $('objectlist');
    list.innerHTML = '';
    G.objects.forEach(function (o, i) {
      list.appendChild(el('li', {
        class: i === sel.object ? 'on' : '',
        onclick: function () { sel.object = i; renderObjects(); }
      }, [
        el('span', {}, [thumb(spriteById(o.sprite), 1)]),
        el('span', { text: o.name || o.id }),
        el('small', { text: o.kind })
      ]));
    });

    var form = $('objectform');
    form.innerHTML = '';
    var o = G.objects[Math.min(sel.object, G.objects.length - 1)];
    if (!o) return;
    var ids = G.sprites.map(function (s) { return s.id; });
    var objectIds = G.objects.map(function (x) { return x.id; });

    form.appendChild(el('h3', { text: 'object' }));
    form.appendChild(field('id', 'text', o.id, function (v) { renameObject(o, v); }));
    form.appendChild(field('name', 'text', o.name || '', function (v) { o.name = v; renderObjects(); markDirty(); }));
    form.appendChild(select_('sprite', ids, o.sprite, function (v) { o.sprite = v; renderObjects(); markDirty(true); }));
    form.appendChild(select_('kind', KINDS, o.kind, function (v) { o.kind = v; renderObjects(); markDirty(true); }));
    form.appendChild(select_('behaviour', BEHAVIORS, o.behavior, function (v) { o.behavior = v; markDirty(true); }));
    form.appendChild(field('speed', 'number', o.speed || 0, function (v) { o.speed = parseFloat(v) || 0; markDirty(true); }));
    form.appendChild(field('jump', 'number', o.jump || 0, function (v) { o.jump = parseFloat(v) || 0; markDirty(true); }));
    form.appendChild(field('health', 'number', o.health || 0, function (v) { o.health = parseInt(v, 10) || 0; markDirty(true); }));
    form.appendChild(field('damage', 'number', o.damage || 0, function (v) { o.damage = parseInt(v, 10) || 0; markDirty(true); }));
    form.appendChild(field('points', 'number', o.points || 0, function (v) { o.points = parseInt(v, 10) || 0; markDirty(true); }));
    form.appendChild(field('range', 'number', o.range || 0, function (v) { o.range = parseFloat(v) || 0; markDirty(true); }));
    form.appendChild(check('solid', o.solid, function (v) { o.solid = v; markDirty(true); }));
    form.appendChild(check('can be collected', o.collect, function (v) { o.collect = v; markDirty(true); }));
    form.appendChild(check('can be stomped', o.stompable, function (v) { o.stompable = v; markDirty(true); }));
    form.appendChild(check('breaks when hit', o.fragile, function (v) { o.fragile = v; markDirty(true); }));
    form.appendChild(select_('drops', [''].concat(objectIds), o.drops || '', function (v) { o.drops = v; markDirty(true); }));

    form.appendChild(el('h3', { text: 'shooting' }));
    form.appendChild(check('this object shoots', !!o.fires, function (v) {
      o.fires = v ? { object: objectIds[0], every: 1, speed: 180, aim: 'forward' } : undefined;
      renderObjects(); markDirty(true);
    }));
    if (o.fires) {
      form.appendChild(select_('fires', objectIds, o.fires.object, function (v) { o.fires.object = v; markDirty(true); }));
      form.appendChild(field('every (s)', 'number', o.fires.every, function (v) { o.fires.every = parseFloat(v) || 1; markDirty(true); }));
      form.appendChild(field('shot speed', 'number', o.fires.speed, function (v) { o.fires.speed = parseFloat(v) || 180; markDirty(true); }));
      form.appendChild(select_('aim', ['forward', 'player', 'up', 'down'], o.fires.aim, function (v) { o.fires.aim = v; markDirty(true); }));
    }
    form.appendChild(el('div', { class: 'row' }, [
      el('button', { class: 'danger', text: 'delete object', onclick: function () { deleteObject(o); } })
    ]));
  }

  function renameObject(o, name) {
    name = (name || '').trim();
    if (!name || objectById(name)) { status('that id is taken', 'warn'); return; }
    var old = o.id;
    o.id = name;
    G.levels.forEach(function (l) {
      (l.spawns || []).forEach(function (s) { if (s.object === old) s.object = name; });
    });
    G.objects.forEach(function (x) {
      if (x.fires && x.fires.object === old) x.fires.object = name;
      if (x.drops === old) x.drops = name;
    });
    G.rules.forEach(function (r) {
      if (r.when.a === old) r.when.a = name;
      if (r.when.b === old) r.when.b = name;
      (r.do || []).forEach(function (a) { if (a.target === old) a.target = name; });
    });
    renderObjects(); markDirty(true);
  }

  function deleteObject(o) {
    if (o.kind === 'player') { status('a game needs a player', 'warn'); return; }
    G.objects = G.objects.filter(function (x) { return x !== o; });
    G.levels.forEach(function (l) {
      l.spawns = (l.spawns || []).filter(function (s) { return s.object !== o.id; });
    });
    sel.object = Math.max(0, sel.object - 1);
    renderObjects(); renderLevels(); drawLevel(); markDirty(true);
  }

  $('addobject').addEventListener('click', function () {
    G.objects.push({
      id: 'object' + (G.objects.length + 1), name: 'New thing',
      sprite: G.sprites[0].id, kind: 'decor', behavior: 'static', speed: 0
    });
    sel.object = G.objects.length - 1;
    renderObjects(); renderLevels(); markDirty();
  });

  // ----------------------------------------------------------------- rules

  var EVENTS = ['start', 'collide', 'collect', 'destroy', 'tile', 'key', 'timer',
    'score', 'count', 'health', 'grant', 'levelclear'];
  var ACTIONS = ['score', 'spawn', 'destroy', 'goto', 'win', 'lose', 'message', 'sound',
    'health', 'lives', 'grant', 'setvar', 'addvar', 'speed', 'shake', 'teleport', 'openexit'];

  function renderRules() {
    var list = $('rulelist');
    list.innerHTML = '';
    G.rules.forEach(function (r, i) {
      list.appendChild(el('li', {
        class: i === sel.rule ? 'on' : '',
        onclick: function () { sel.rule = i; renderRules(); }
      }, [
        el('span', { text: r.name || r.id || ('rule ' + (i + 1)) }),
        el('small', { text: r.when.event })
      ]));
    });

    var form = $('ruleform');
    form.innerHTML = '';
    var r = G.rules[Math.min(sel.rule, G.rules.length - 1)];
    if (!r) return;
    var objectIds = G.objects.map(function (o) { return o.id; });

    form.appendChild(el('h3', { text: 'when' }));
    form.appendChild(field('name', 'text', r.name || '', function (v) { r.name = v; renderRules(); markDirty(); }));
    form.appendChild(select_('event', EVENTS, r.when.event, function (v) { r.when.event = v; renderRules(); markDirty(true); }));
    form.appendChild(select_('object', [''].concat(objectIds), r.when.a || '', function (v) { r.when.a = v; markDirty(true); }));
    if (r.when.event === 'collide') {
      form.appendChild(select_('and object', [''].concat(objectIds), r.when.b || '', function (v) { r.when.b = v; markDirty(true); }));
    }
    form.appendChild(field('value', 'number', r.when.value || 0, function (v) { r.when.value = parseFloat(v) || 0; markDirty(true); }));
    form.appendChild(select_('only in level', [''].concat(G.levels.map(function (l) { return l.id; })), r.level || '', function (v) { r.level = v; markDirty(true); }));
    form.appendChild(check('only once', r.once, function (v) { r.once = v; markDirty(true); }));

    form.appendChild(el('h3', { text: 'then' }));
    (r.do || []).forEach(function (a, i) {
      var actionSelect = el('select');
      ACTIONS.forEach(function (name) {
        actionSelect.appendChild(el('option', { value: name, text: name, selected: name === a.action }));
      });
      actionSelect.addEventListener('change', function () { a.action = actionSelect.value; markDirty(true); });

      var target = el('input', { value: a.target || '', placeholder: 'target' });
      target.addEventListener('change', function () { a.target = target.value; markDirty(true); });
      var value = el('input', { type: 'number', value: a.value || 0 });
      value.addEventListener('change', function () { a.value = parseFloat(value.value) || 0; markDirty(true); });
      var text = el('input', { value: a.text || '', placeholder: 'message' });
      text.addEventListener('change', function () { a.text = text.value; markDirty(true); });

      form.appendChild(el('div', { class: 'actionrow' }, [
        actionSelect, target, value, text,
        el('button', {
          class: 'danger', text: '×',
          onclick: function () { r.do.splice(i, 1); renderRules(); markDirty(true); }
        })
      ]));
    });
    form.appendChild(el('div', { class: 'row' }, [
      el('button', {
        text: '+ action',
        onclick: function () { r.do = r.do || []; r.do.push({ action: 'score', value: 10 }); renderRules(); markDirty(true); }
      }),
      el('button', {
        class: 'danger', text: 'delete rule',
        onclick: function () {
          G.rules.splice(sel.rule, 1);
          sel.rule = Math.max(0, sel.rule - 1);
          renderRules(); markDirty(true);
        }
      })
    ]));
  }

  $('addrule').addEventListener('click', function () {
    G.rules.push({
      id: 'rule' + (G.rules.length + 1),
      name: 'New rule',
      when: { event: 'collect', a: G.objects[0].id },
      do: [{ action: 'score', value: 10 }]
    });
    sel.rule = G.rules.length - 1;
    renderRules(); markDirty();
  });

  // -------------------------------------------------------------- settings

  function renderSettings() {
    var form = $('settingsform');
    form.innerHTML = '';
    form.appendChild(el('h3', { text: 'game' }));
    form.appendChild(field('name', 'text', G.name, function (v) { G.name = v; markDirty(); }));
    form.appendChild(field('tagline', 'text', G.tagline || '', function (v) { G.tagline = v; markDirty(true); }));
    form.appendChild(field('author', 'text', G.author || '', function (v) { G.author = v; markDirty(); }));
    form.appendChild(select_('genre', ['platformer', 'metroidvania', 'topdown', 'shooter', 'maze', 'breakout', 'runner'],
      G.genre, function (v) { G.genre = v; markDirty(true); }));

    form.appendChild(el('h3', { text: 'screen' }));
    form.appendChild(field('width', 'number', G.screen.width, function (v) { G.screen.width = parseInt(v, 10) || 256; markDirty(true); }));
    form.appendChild(field('height', 'number', G.screen.height, function (v) { G.screen.height = parseInt(v, 10) || 224; markDirty(true); }));
    form.appendChild(field('tile size', 'number', G.screen.tile, function (v) { G.screen.tile = parseInt(v, 10) || 16; markDirty(true); }));
    form.appendChild(field('scale', 'number', G.screen.scale, function (v) { G.screen.scale = parseInt(v, 10) || 3; markDirty(true); }));

    form.appendChild(el('h3', { text: 'play' }));
    form.appendChild(field('gravity', 'number', G.settings.gravity, function (v) { G.settings.gravity = parseFloat(v) || 0; markDirty(true); }));
    form.appendChild(field('lives', 'number', G.settings.lives, function (v) { G.settings.lives = parseInt(v, 10) || 1; markDirty(true); }));
    form.appendChild(select_('starts at', G.levels.map(function (l) { return l.id; }),
      G.settings.startLevel || G.levels[0].id, function (v) { G.settings.startLevel = v; markDirty(true); }));
    form.appendChild(select_('respawn', ['level', 'checkpoint'], G.settings.respawn || 'level',
      function (v) { G.settings.respawn = v; markDirty(true); }));
    form.appendChild(field('score to win', 'number', G.settings.scoreGoal || 0, function (v) { G.settings.scoreGoal = parseInt(v, 10) || 0; markDirty(true); }));
    form.appendChild(check('show score', G.settings.showScore, function (v) { G.settings.showScore = v; markDirty(true); }));
    form.appendChild(check('show lives', G.settings.showLives, function (v) { G.settings.showLives = v; markDirty(true); }));

    form.appendChild(el('h3', { text: 'words' }));
    form.appendChild(field('title screen', 'text', G.settings.title || '', function (v) { G.settings.title = v; markDirty(true); }));
    form.appendChild(field('controls hint', 'text', G.settings.hint || '', function (v) { G.settings.hint = v; markDirty(true); }));
    form.appendChild(field('ending', 'text', G.settings.ending || '', function (v) { G.settings.ending = v; markDirty(true); }));

    form.appendChild(el('h3', { text: 'palette' }));
    var palette = el('div', { class: 'row' });
    G.palette.forEach(function (color, i) {
      var input = el('input', { type: 'color', value: color, title: 'colour ' + i.toString(16) });
      input.style.width = '38px';
      input.addEventListener('change', function () {
        G.palette[i] = input.value;
        renderAll();
        markDirty(true);
      });
      palette.appendChild(input);
    });
    form.appendChild(palette);
  }

  // ------------------------------------------------------------------ boot

  $('save').addEventListener('click', function () { save(); });
  $('gamelist').addEventListener('change', function () { if (this.value) loadGame(this.value); });
  $('open').addEventListener('click', function () {
    if (!NAME) return;
    save().then(function () { window.open('/play/' + encodeURIComponent(NAME), '_blank'); });
  });
  $('export').addEventListener('click', function () {
    if (!NAME) return;
    save().then(function () {
      return api('/api/export', { method: 'POST', body: JSON.stringify({ name: NAME }) });
    }).then(function (res) {
      status('exported to ' + res.path);
      logEntry('exported', res.path);
    }).catch(function (e) { status(e.message, 'err'); });
  });

  window.addEventListener('beforeunload', function () {
    if (saveTimer) { navigator.sendBeacon && save(); }
  });

  // The studio stops itself when nothing has used it for a while, which is how
  // a window opened from a desktop icon ends when the window does. This says
  // the window is still open.
  setInterval(function () {
    api('/api/status').catch(function () {
      status('the studio has stopped — start it again to keep working', 'err');
    });
  }, 20000);

  // Connecting Claude, from inside the studio: an AppImage opened from a
  // desktop icon never sees the environment variables a terminal would set, so
  // asking for the key here is the only way it can ever be given one.
  function showStatus(info) {
    var badge = $('assistant');
    badge.textContent = info.online ? info.assistant : 'offline designer';
    badge.className = 'badge' + (info.online ? '' : ' offline');
    badge.title = info.online
      ? 'designing with ' + info.assistant + ' (' + (info.source || 'connected') + ') — click to change'
      : 'click to design with Claude instead';
    $('connect').hidden = false;
    $('connectnote').textContent = info.online
      ? 'Designing with ' + info.assistant + ', using ' + (info.source || 'a saved key') + '.'
      : 'The studio designs offline on its own. Give it a key and Claude does the designing instead — better games, and it understands what you ask for.';
    $('connectbtn').textContent = info.online ? 'use a key instead' : 'connect with a key';
    // The machine may already have a Claude that needs no key at all.
    $('clirow').hidden = !info.claudeCode;
    // Not finding it is usually a window that cannot see as much as a
    // terminal can, not a machine without it, so say where to look.
    $('clihelp').hidden = !!info.claudeCode;
    if (!info.claudeCode && info.searched) {
      var list = $('clisearched');
      list.innerHTML = '';
      info.searched.forEach(function (p) { list.appendChild(el('li', { text: p })); });
    }
    if (info.claudeCode) {
      $('usecli').textContent = info.source && info.source.indexOf('Claude Code') >= 0
        ? 'designing with Claude Code on this machine'
        : 'use the Claude Code on this machine — no key';
      $('usecli').disabled = !!(info.source && info.source.indexOf('Claude Code') >= 0);
    }
    if (info.endpoint) $('endpoint').value = info.endpoint;
    var select = $('model');
    if (select && !select.options.length && info.models) {
      info.models.forEach(function (m) {
        select.appendChild(el('option', { value: m, text: m, selected: m === info.model }));
      });
    }
    var existing = document.getElementById('disconnectbtn');
    if (info.online && !existing) {
      $('connectbtn').parentNode.appendChild(el('button', {
        id: 'disconnectbtn', text: 'disconnect',
        onclick: function () { connect({ disconnect: true }); }
      }));
    } else if (!info.online && existing) {
      existing.parentNode.removeChild(existing);
    }
  }

  function connect(body) {
    clearAlert();
    status(body.disconnect ? 'disconnecting…' : (body.use === 'claude-code'
      ? 'looking for Claude Code…' : 'checking the key with Anthropic…'));
    api('/api/connect', { method: 'POST', body: JSON.stringify(body) })
      .then(function (res) {
        $('apikey').value = '';
        return api('/api/status');
      })
      .then(function (info) {
        showStatus(info);
        status(info.online ? 'designing with ' + info.assistant : 'designing offline');
        logEntry('assistant', info.online
          ? 'Connected. Designs now come from ' + info.assistant + '.'
          : 'Disconnected. The studio is designing on its own again.');
      })
      .catch(function (e) { status(e.message, 'err'); });
  }

  $('connectbtn').addEventListener('click', function () {
    connect({ key: $('apikey').value.trim(), model: $('model').value, use: 'api' });
  });
  $('usecli').addEventListener('click', function () { connect({ use: 'claude-code' }); });
  $('setclibin').addEventListener('click', function () {
    connect({ use: 'claude-code', bin: $('clibin').value.trim() });
  });
  $('recheck').addEventListener('click', function () { connect({ use: 'claude-code' }); });
  $('alertclose').addEventListener('click', clearAlert);
  $('setendpoint').addEventListener('click', function () {
    connect({ url: $('endpoint').value.trim(), key: $('apikey').value.trim() });
  });
  $('apikey').addEventListener('keydown', function (e) {
    if (e.key === 'Enter') connect({ key: this.value.trim(), model: $('model').value });
  });
  $('assistant').addEventListener('click', function () {
    Array.prototype.forEach.call(document.querySelectorAll('#tabs button'), function (b) {
      if (b.dataset.tab === 'assistant') b.click();
    });
    $('apikey').focus();
  });

  api('/api/status').then(function (info) {
    showStatus(info);
    status('games in ' + info.dir);
  });

  refreshGameList().then(function (list) {
    if (list && list.length) return loadGame(list[0].name);
    status('describe a game and press "make the game"');
  });
})();
