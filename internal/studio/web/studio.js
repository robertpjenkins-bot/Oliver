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
    tile: '#', place: null, color: 1, zoom: 2,
    spawn: -1, onion: false
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
      forgetHistory();
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

  // ---------------------------------------------------------------- history
  //
  // Undo covers the whole project rather than the canvas alone. Painting is
  // only one of the ways to change a game, and an editor where the brush can
  // be taken back but renaming a level cannot is an editor you are afraid to
  // touch. Every change goes through change(), so everything is undoable for
  // the same reason and by the same code.
  //
  // What is kept is the project as it was, as text. It is more than a diff
  // would cost and it is worth it: nothing has to describe how to reverse
  // itself, so no edit can ever be undone wrongly.
  var past = [], future = [], HISTORY = 80;

  // change records the state before an edit. Call it, then make the edit.
  //
  // A label that repeats while the same gesture is in progress — one stroke of
  // a brush, one drag of a slider — is folded into the first: a stroke is one
  // thing to take back, not forty.
  var lastLabel = '', lastAt = 0;
  function change(label, coalesce) {
    if (!G) return;
    var now = Date.now();
    if (coalesce && label === lastLabel && now - lastAt < 900 && past.length) {
      lastAt = now;
      future.length = 0;
      return;
    }
    past.push({ label: label, json: JSON.stringify(G) });
    if (past.length > HISTORY) past.shift();
    future.length = 0;
    lastLabel = label;
    lastAt = now;
    updateHistoryButtons();
  }

  function stepTo(from, to, verb) {
    if (!from.length) { status('nothing to ' + verb); return; }
    var entry = from.pop();
    to.push({ label: entry.label, json: JSON.stringify(G) });
    G = JSON.parse(entry.json);
    // A selection or a frame that no longer exists would leave the editor
    // pointing at nothing, so every index is pulled back into range.
    clampSelection();
    selection = null;
    floating = null;
    sel.spawn = -1;
    lastLabel = '';
    renderAll();
    drawLevel();
    drawSpriteEditor();
    updateHistoryButtons();
    markSaved();
    status(verb + ' ' + entry.label);
  }

  function undo() { stepTo(past, future, 'undo'); }

  // forgetHistory is for when the project itself is replaced rather than
  // edited. Undoing across that boundary would restore part of one game into
  // another, which is not an undo by any reading of the word.
  function forgetHistory() {
    past.length = 0;
    future.length = 0;
    floating = null;
    selection = null;
    sel.spawn = -1;
    lastLabel = '';
    updateHistoryButtons();
  }

  function redo() { stepTo(future, past, 'redo'); }

  function clampSelection() {
    sel.level = Math.max(0, Math.min(sel.level, G.levels.length - 1));
    sel.sprite = Math.max(0, Math.min(sel.sprite, G.sprites.length - 1));
    sel.object = Math.max(0, Math.min(sel.object, G.objects.length - 1));
    sel.rule = Math.max(0, Math.min(sel.rule, (G.rules || []).length - 1));
    var s = G.sprites[sel.sprite];
    sel.frame = s ? Math.max(0, Math.min(sel.frame, s.frames.length - 1)) : 0;
  }

  function updateHistoryButtons() {
    $('undo').disabled = !past.length;
    $('redo').disabled = !future.length;
    $('undo').title = past.length ? 'undo ' + past[past.length - 1].label + ' (ctrl+Z)' : 'nothing to undo';
    $('redo').title = future.length ? 'redo ' + future[future.length - 1].label + ' (ctrl+shift+Z)' : 'nothing to redo';
  }

  // markSaved exists so undo does not have to pretend nothing happened: the
  // project on disk is behind again, and saving is how it catches up.
  function markSaved() {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(save, 1200);
  }

  $('undo').addEventListener('click', undo);
  $('redo').addEventListener('click', redo);

  // ------------------------------------------------------------------ tools
  //
  // One set of tools serves both canvases. A level map and a sprite frame are
  // both grids of characters — what differs is only what a character means and
  // how big a cell is drawn — so a surface says how to read and write its
  // rows, and every tool is written once against that.
  var P = window.PaintOps;

  var TOOLS = [
    { id: 'pencil', key: 'b', label: 'pencil', hint: 'drag to draw' },
    { id: 'line', key: 'l', label: 'line', hint: 'drag from one end to the other' },
    { id: 'rect', key: 'r', label: 'box', hint: 'drag a box · hold shift to fill it' },
    { id: 'ellipse', key: 'o', label: 'oval', hint: 'drag a box · hold shift to fill it' },
    { id: 'fill', key: 'g', label: 'fill', hint: 'click an area to flood it' },
    { id: 'pick', key: 'i', label: 'pick', hint: 'click to take what is already there' },
    { id: 'select', key: 'm', label: 'select', hint: 'drag a region · drag inside it to move · ctrl+C, ctrl+V, delete' }
  ];

  var tool = 'pencil';
  var preview = null;     // the shape under the cursor, not yet committed
  var selection = null;   // { x0, y0, x1, y1 } in cells, on the surface named below
  var selectionOn = '';
  var clip = null;        // { rows, from } — what was copied, and from which surface
  var floating = null;    // a selection picked up and being dragged

  function toolById(id) {
    for (var i = 0; i < TOOLS.length; i++) if (TOOLS[i].id === id) return TOOLS[i];
    return TOOLS[0];
  }

  function setTool(id) {
    tool = id;
    dropFloating();
    renderToolbars();
    drawLevel();
    drawSpriteEditor();
  }

  function renderToolbars() {
    ['leveltools', 'spritetools'].forEach(function (where) {
      var box = $(where);
      if (!box) return;
      box.innerHTML = '';
      TOOLS.forEach(function (t) {
        box.appendChild(el('button', {
          class: 'tool' + (tool === t.id ? ' on' : ''),
          text: t.label,
          title: t.label + '  (' + t.key.toUpperCase() + ') — ' + t.hint,
          onclick: function () { setTool(t.id); }
        }));
      });
    });
    var lvbox = $('leveltools');
    if (lvbox) {
      lvbox.appendChild(el('span', { class: 'toolgap' }));
      lvbox.appendChild(el('button', {
        class: 'tool', text: '▶ test',
        title: 'play this level from its own start  (T)',
        onclick: function () { playThisLevel(null); }
      }));
      lvbox.appendChild(el('button', {
        class: 'tool', text: '▶ from here',
        title: 'play this level starting where you last clicked on the map  (shift+T) — the game is not changed',
        onclick: function () { playThisLevel(lastCell); }
      }));
    }

    var t = toolById(tool);
    // The cursor readout is appended to this text, and remembers it to put it
    // back; a new tool means a new sentence to remember.
    if ($('levelhint')) {
      $('levelhint').textContent = t.label + ': ' + t.hint +
        ' · right button erases · middle-drag or shift+drag scrolls';
      delete $('levelhint').dataset.base;
    }
    if ($('spritehint')) {
      $('spritehint').textContent = t.label + ': ' + t.hint + ' · right button erases';
      delete $('spritehint').dataset.base;
    }
  }

  // ------------------------------------------------------------- a surface
  //
  // A surface is everything a tool needs to know about the thing being drawn
  // on: how to read its rows, how to put rows back, what a cell means, and how
  // big a cell is on screen. The level editor and the sprite editor each
  // describe themselves this way, and then share every tool below.
  function levelSurface() {
    var lv = level();
    if (!lv) return null;
    return {
      name: 'level',
      canvas: $('levelcanvas'),
      rows: function () { return lv.map; },
      put: function (rows) { lv.map = rows; },
      cellSize: function () { return G.screen.tile * sel.zoom; },
      paint: function () { return sel.tile; },
      empty: '.',
      w: lv.width, h: lv.height,
      redraw: drawLevel,
      // Picking on a level takes the tile under the cursor, and picking an
      // object spawn selects that instead: both are "what is already here".
      pick: function (x, y) {
        var sp = spawnAt(lv, x, y);
        if (sp >= 0) { sel.spawn = sp; sel.place = lv.spawns[sp].object; renderLevels(); return; }
        var ch = P.at(lv.map, x, y);
        if (ch && ch !== '.') { sel.place = null; sel.tile = ch; }
        renderLevels();
      }
    };
  }

  function spriteSurface() {
    var s = sprite();
    if (!s) return null;
    return {
      name: 'sprite',
      canvas: $('spritecanvas'),
      rows: function () { return s.frames[sel.frame]; },
      put: function (rows) { s.frames[sel.frame] = rows; },
      cellSize: function () { return parseInt($('spritecanvas').dataset.zoom, 10) || 8; },
      paint: function () { return sel.color.toString(16); },
      empty: '.',
      w: s.w, h: s.h,
      redraw: drawSpriteEditor,
      pick: function (x, y) {
        var ch = P.at(s.frames[sel.frame], x, y);
        var idx = parseInt(ch, 16);
        if (!isNaN(idx)) { sel.color = idx; renderSprites(); }
      }
    };
  }

  function cellOf(surface, e) {
    var canvas = surface.canvas;
    var rect = canvas.getBoundingClientRect();
    var size = surface.cellSize();
    return {
      x: Math.floor((e.clientX - rect.left) * (canvas.width / rect.width) / size),
      y: Math.floor((e.clientY - rect.top) * (canvas.height / rect.height) / size)
    };
  }

  function inside(box, x, y) {
    return x >= Math.min(box.x0, box.x1) && x <= Math.max(box.x0, box.x1) &&
      y >= Math.min(box.y0, box.y1) && y <= Math.max(box.y0, box.y1);
  }

  // drawOverlay paints what the tools want shown on top of a surface: the
  // shape being dragged, and the edge of a selection. Both are drawn rather
  // than written, so nothing is committed until the button comes up.
  function drawOverlay(ctx, surface) {
    var size = surface.cellSize();
    if (preview && preview.on === surface.name) {
      ctx.globalAlpha = 0.55;
      preview.points.forEach(function (p) {
        ctx.fillStyle = preview.color;
        ctx.fillRect(p.x * size, p.y * size, size, size);
      });
      ctx.globalAlpha = 1;
    }
    if (selection && selectionOn === surface.name) {
      var x = Math.min(selection.x0, selection.x1) * size;
      var y = Math.min(selection.y0, selection.y1) * size;
      var w = (Math.abs(selection.x1 - selection.x0) + 1) * size;
      var h = (Math.abs(selection.y1 - selection.y0) + 1) * size;
      ctx.save();
      ctx.setLineDash([4, 3]);
      ctx.strokeStyle = '#f2c94c';
      ctx.lineWidth = 1;
      ctx.strokeRect(x + 0.5, y + 0.5, w - 1, h - 1);
      ctx.restore();
    }
  }

  // The colour a preview is drawn in, so a shape can be seen before it lands.
  function previewColor(surface, erase) {
    if (erase) return '#ff6b8a';
    if (surface.name === 'sprite') return G.palette[sel.color] || '#fff';
    return 'rgba(242,201,76,.9)';
  }

  // dropFloating puts a picked-up selection back down wherever it now is.
  function dropFloating() {
    if (!floating) return;
    var surface = floating.on === 'level' ? levelSurface() : spriteSurface();
    if (surface) {
      surface.put(P.write(surface.rows(), floating.rows, floating.x, floating.y, null));
      surface.redraw();
    }
    floating = null;
  }

  // bindPaint wires one canvas to every tool. Which tool is running is decided
  // on the way down and kept for the whole gesture: changing tool mid-drag
  // would finish a shape nobody started.
  function bindPaint(canvasId, surfaceFor) {
    var canvas = $(canvasId);
    var drag = null;

    canvas.addEventListener('contextmenu', function (e) { e.preventDefault(); });

    canvas.addEventListener('mousedown', function (e) {
      if (!G) return;
      var surface = surfaceFor();
      if (!surface) return;

      // Scrolling a map bigger than the window: the middle button, or shift,
      // because a level worth building is wider than the screen.
      if (e.button === 1 || (e.shiftKey && tool !== 'rect' && tool !== 'ellipse')) {
        e.preventDefault();
        startPanning(canvas, e);
        return;
      }
      e.preventDefault();

      var c = cellOf(surface, e);
      var erase = e.button === 2;
      var t = tool;

      if (t === 'pick') { surface.pick(c.x, c.y); return; }

      // Dragging a placed object moves it — but only while objects are what
      // is being placed. With a tile in hand the pencil paints, always, even
      // over an object: otherwise there is no way to lay ground under
      // anything standing on it, which is most of a level.
      if (surface.name === 'level' && t === 'pencil' && !erase && sel.place) {
        var hit = spawnAt(level(), c.x, c.y);
        if (hit >= 0) {
          sel.spawn = hit;
          change('move ' + (objectById(level().spawns[hit].object) || {}).id);
          drag = { kind: 'spawn', index: hit, surface: surface };
          renderLevels();
          return;
        }
      }

      if (t === 'select') {
        if (selection && selectionOn === surface.name && inside(selection, c.x, c.y) && !erase) {
          // Picking a selection up leaves a hole, which is what moving means.
          change('move a region');
          var lox = Math.min(selection.x0, selection.x1), loy = Math.min(selection.y0, selection.y1);
          floating = {
            on: surface.name,
            rows: P.read(surface.rows(), selection.x0, selection.y0, selection.x1, selection.y1),
            x: lox, y: loy
          };
          surface.put(P.clear(surface.rows(), selection.x0, selection.y0, selection.x1, selection.y1, surface.empty));
          drag = { kind: 'float', grabX: c.x - lox, grabY: c.y - loy, surface: surface };
          surface.redraw();
          return;
        }
        dropFloating();
        selection = { x0: c.x, y0: c.y, x1: c.x, y1: c.y };
        selectionOn = surface.name;
        drag = { kind: 'select', surface: surface };
        surface.redraw();
        return;
      }

      dropFloating();

      if (t === 'fill') {
        change('fill');
        surface.put(P.fill(surface.rows(), c.x, c.y, erase ? surface.empty : surface.paint()));
        surface.redraw();
        markDirty(true);
        return;
      }

      if (t === 'pencil') {
        change('draw', true);
        if (surface.name === 'level' && sel.place && !erase) {
          placeSpawn(level(), sel.place, c.x, c.y);
        } else if (surface.name === 'level' && sel.place && erase) {
          eraseSpawn(level(), c.x, c.y);
        } else {
          surface.put(P.set(surface.rows(), c.x, c.y, erase ? surface.empty : surface.paint()));
        }
        drag = { kind: 'pencil', last: c, erase: erase, surface: surface };
        surface.redraw();
        markDirty(true);
        return;
      }

      // Everything else is a shape: remember where it started and show it
      // following the cursor until the button comes up.
      drag = { kind: 'shape', tool: t, from: c, erase: erase, surface: surface };
      preview = { on: surface.name, points: [c], color: previewColor(surface, erase) };
      surface.redraw();
    });

    canvas.addEventListener('mousemove', function (e) {
      if (!G) return;
      var surface = surfaceFor();
      if (!surface) return;
      var c = cellOf(surface, e);
      showCursorAt(surface, c);
      if (!drag) return;

      if (drag.kind === 'pencil') {
        // Between two mouse events the cursor may have crossed several cells,
        // so the gap is drawn in: a fast stroke is a line, not a dotted one.
        var pts = P.line(drag.last.x, drag.last.y, c.x, c.y);
        if (surface.name === 'level' && sel.place) {
          pts.forEach(function (p) {
            if (drag.erase) eraseSpawn(level(), p.x, p.y);
            else if (p.x !== drag.last.x || p.y !== drag.last.y) placeSpawn(level(), sel.place, p.x, p.y);
          });
        } else {
          surface.put(P.plot(surface.rows(), pts, drag.erase ? surface.empty : surface.paint()));
        }
        drag.last = c;
        surface.redraw();
        markDirty(true);
        return;
      }

      if (drag.kind === 'spawn') {
        var sp = level().spawns[drag.index];
        if (sp && (sp.x !== c.x || sp.y !== c.y)) {
          sp.x = Math.max(0, Math.min(c.x, level().width - 1));
          sp.y = Math.max(0, Math.min(c.y, level().height - 1));
          surface.redraw();
          renderSpawnProps();
          markDirty(true);
        }
        return;
      }

      if (drag.kind === 'select') {
        selection.x1 = Math.max(0, Math.min(c.x, surface.w - 1));
        selection.y1 = Math.max(0, Math.min(c.y, surface.h - 1));
        surface.redraw();
        return;
      }

      if (drag.kind === 'float' && floating) {
        floating.x = c.x - drag.grabX;
        floating.y = c.y - drag.grabY;
        selection = {
          x0: floating.x, y0: floating.y,
          x1: floating.x + P.width(floating.rows) - 1,
          y1: floating.y + P.height(floating.rows) - 1
        };
        surface.redraw();
        return;
      }

      if (drag.kind === 'shape') {
        preview = {
          on: surface.name,
          points: shapePoints(drag.tool, drag.from, c, e.shiftKey),
          color: previewColor(surface, drag.erase)
        };
        surface.redraw();
      }
    });

    window.addEventListener('mouseup', function (e) {
      if (!drag) return;
      var surface = drag.surface;
      if (drag.kind === 'shape') {
        var c = cellOf(surface, e);
        var pts = shapePoints(drag.tool, drag.from, c, e.shiftKey);
        change(drag.tool === 'line' ? 'a line' : drag.tool === 'rect' ? 'a box' : 'an oval');
        surface.put(P.plot(surface.rows(), pts, drag.erase ? surface.empty : surface.paint()));
        preview = null;
        surface.redraw();
        markDirty(true);
      }
      if (drag.kind === 'float') {
        // The floating region stays floating until something else happens to
        // it, so it can be nudged again without being cut out twice.
        surface.redraw();
        markDirty(true);
      }
      if (drag.kind === 'select' || drag.kind === 'spawn' || drag.kind === 'pencil') {
        surface.redraw();
      }
      drag = null;
    });

    canvas.addEventListener('mouseleave', function () {
      if (!drag) { preview = null; showCursorAt(null); }
    });
  }

  function shapePoints(which, from, to, shift) {
    if (which === 'line') return P.line(from.x, from.y, to.x, to.y);
    if (which === 'rect') return P.rect(from.x, from.y, to.x, to.y, shift);
    if (which === 'ellipse') return P.ellipse(from.x, from.y, to.x, to.y, shift);
    return [to];
  }

  // Where the cursor is, in cells. Building a level means counting tiles —
  // how wide a gap is, how high a ledge — and counting them by eye is the
  // difference between a jump that works and one that does not.
  var lastCell = { x: 2, y: 2 };

  function showCursorAt(surface, c) {
    if (surface && surface.name === 'level' && c &&
        c.x >= 0 && c.y >= 0 && c.x < surface.w && c.y < surface.h) {
      lastCell = { x: c.x, y: c.y };
    }
    var box = surface && surface.name === 'level' ? $('levelhint') : $('spritehint');
    if (!box) return;
    if (!box.dataset.base) box.dataset.base = box.textContent;
    if (!surface || !c || c.x < 0 || c.y < 0 || c.x >= surface.w || c.y >= surface.h) {
      box.textContent = box.dataset.base;
      return;
    }
    var extra = '';
    if (selection && selectionOn === surface.name) {
      extra = ' · selection ' + (Math.abs(selection.x1 - selection.x0) + 1) +
        '×' + (Math.abs(selection.y1 - selection.y0) + 1);
    }
    box.textContent = box.dataset.base + '  —  ' + c.x + ', ' + c.y + extra;
  }

  function startPanning(canvas, e) {
    var wrap = canvas.parentNode;
    var fromX = e.clientX, fromY = e.clientY;
    var left = wrap.scrollLeft, top = wrap.scrollTop;
    function move(ev) {
      wrap.scrollLeft = left - (ev.clientX - fromX);
      wrap.scrollTop = top - (ev.clientY - fromY);
    }
    function up() {
      window.removeEventListener('mousemove', move);
      window.removeEventListener('mouseup', up);
    }
    window.addEventListener('mousemove', move);
    window.addEventListener('mouseup', up);
  }

  // ------------------------------------------------------- spawns on a map
  //
  // An object on a level is a spawn: which object, and where. They are a list
  // rather than part of the map because two can share a tile and because each
  // carries its own facing.
  function spawnAt(lv, x, y) {
    var list = lv.spawns || [];
    // Last drawn is first picked: what is on top is what you meant to click.
    for (var i = list.length - 1; i >= 0; i--) {
      if (Math.round(list[i].x) === x && Math.round(list[i].y) === y) return i;
    }
    return -1;
  }

  function placeSpawn(lv, id, x, y) {
    if (x < 0 || y < 0 || x >= lv.width || y >= lv.height) return;
    lv.spawns = lv.spawns || [];
    if (spawnAt(lv, x, y) >= 0) return;   // one per cell, or a drag leaves a pile
    var o = objectById(id);
    // A game can only have one player, and placing a second one silently is
    // how a level ends up with two and starts at the wrong one.
    if (o && o.kind === 'player') {
      lv.spawns = lv.spawns.filter(function (sp) {
        var other = objectById(sp.object);
        return !other || other.kind !== 'player';
      });
    }
    lv.spawns.push({ object: id, x: x, y: y, dir: 1 });
    sel.spawn = lv.spawns.length - 1;
    renderSpawnProps();
  }

  function eraseSpawn(lv, x, y) {
    var hit = spawnAt(lv, x, y);
    if (hit < 0) return;
    lv.spawns.splice(hit, 1);
    if (sel.spawn === hit) sel.spawn = -1;
    renderSpawnProps();
  }

  function renderAll() {
    if (!G) return;
    renderToolbars();
    updateHistoryButtons();
    renderLevels();
    renderSprites();
    renderObjects();
    renderRules();
    renderSettings();
    renderCode();
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
      if (b.dataset.tab === 'code') renderCode();
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
        forgetHistory();
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
    change('ask: ' + text.slice(0, 30));
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
    runtime = window.OliverEngine.start(canvas, playableGame(), {
      skipTitle: true,
      scale: scale,
      level: $('playlevel').value || undefined,
      onEvent: function (ev) {
        if (ev && ev.event === 'scripterror') noteScriptTrouble(ev.text);
      }
    });
    canvas.focus();
  }

  // --------------------------------------------------------- try it now
  //
  // Building a level by hand is a loop: move a ledge, see whether the jump
  // works, move it again. Anything between those two halves is friction, so
  // the level being edited can be played from the editor, and played from the
  // exact spot being worked on.
  //
  // Testing from a spot does not move the player in the project. The engine is
  // handed a copy with the player put where you asked, and the game on disk
  // never hears about it: a test is a question, not an edit.
  var testFrom = null;

  function playThisLevel(from) {
    if (!G) return;
    var lv = level();
    if (!lv) return;
    testFrom = from || null;
    $('playlevel').value = lv.id;
    showTab('play');
    Array.prototype.forEach.call(document.querySelectorAll('#tabs button'), function (b) {
      b.classList.toggle('on', b.dataset.tab === 'play');
    });
    startPlaying();
    status(from ? 'testing ' + (lv.name || lv.id) + ' from ' + from.x + ', ' + from.y
      : 'testing ' + (lv.name || lv.id));
  }

  // playableGame is the project as the engine should see it for this run.
  function playableGame() {
    if (!testFrom) return G;
    var copy = JSON.parse(JSON.stringify(G));
    var lv = null;
    for (var i = 0; i < copy.levels.length; i++) {
      if (copy.levels[i].id === level().id) lv = copy.levels[i];
    }
    if (!lv) return G;
    var player = null;
    for (var j = 0; j < copy.objects.length; j++) {
      if (copy.objects[j].kind === 'player') player = copy.objects[j];
    }
    if (!player) return G;
    lv.spawns = (lv.spawns || []).filter(function (sp) { return sp.object !== player.id; });
    lv.spawns.push({ object: player.id, x: testFrom.x, y: testFrom.y, dir: 1 });
    return copy;
  }

  function stopPlaying() {
    if (runtime) { runtime.stop(); runtime = null; }
  }

  // Choosing a level by hand is a fresh start, not a continuation of a test.
  $('playlevel').addEventListener('change', function () { testFrom = null; startPlaying(); });

  $('restart').addEventListener('click', startPlaying);

  // ---------------------------------------------------------------- levels

  function level() { return G.levels[Math.min(sel.level, G.levels.length - 1)]; }

  function renderLevels() {
    var list = $('levellist');
    list.innerHTML = '';
    G.levels.forEach(function (l, i) {
      list.appendChild(el('li', {
        class: i === sel.level ? 'on' : '',
        onclick: function () {
          sel.level = i;
          sel.spawn = -1;
          selection = null;
          dropFloating();
          renderLevels();
          drawLevel();
        }
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

    renderTileDef();
    renderSpawnProps();
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
    if (w === lv.width && h === lv.height) return;
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
    change('add a level');
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
    change('delete a level');
    var gone = G.levels.splice(sel.level, 1)[0];
    G.levels.forEach(function (l, i) {
      if (l.next === gone.id) l.next = (G.levels[i + 1] || {}).id || 'win';
    });
    if (G.settings.startLevel === gone.id) G.settings.startLevel = G.levels[0].id;
    sel.level = Math.max(0, sel.level - 1);
    renderLevels(); renderPlayLevels(); drawLevel(); markDirty();
  }

  $('addlevel').addEventListener('click', duplicateLevel);
  $('addtile').addEventListener('click', addTileDef);

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
    // A region picked up with the select tool rides above the map until it is
    // put down, so it can be seen against what it is being moved onto.
    if (floating && floating.on === 'level') {
      var fdefs = defs;
      for (var fy = 0; fy < floating.rows.length; fy++) {
        for (var fx = 0; fx < floating.rows[fy].length; fx++) {
          var fdef = fdefs[floating.rows[fy][fx]];
          if (!fdef) continue;
          drawSprite(ctx, spriteById(fdef.sprite), 0,
            (floating.x + fx) * tile * z, (floating.y + fy) * tile * z, z);
        }
      }
    }
    (lv.spawns || []).forEach(function (sp, i) {
      var o = objectById(sp.object);
      if (!o) return;
      drawSprite(ctx, spriteById(o.sprite), 0, sp.x * tile * z, sp.y * tile * z, z);
      if (o.kind === 'player') {
        ctx.strokeStyle = '#f2c94c';
        ctx.strokeRect(sp.x * tile * z + 0.5, sp.y * tile * z + 0.5, tile * z - 1, tile * z - 1);
      }
      // Which object is selected has to be visible on the map, not only in a
      // list: it is the one whose facing and patrol the side panel is editing.
      if (i === sel.spawn) {
        ctx.save();
        ctx.strokeStyle = '#7ee0ff';
        ctx.lineWidth = 2;
        ctx.strokeRect(sp.x * tile * z - 1, sp.y * tile * z - 1, tile * z + 2, tile * z + 2);
        ctx.restore();
      }
    });
    // A grid at the tile size, faint enough to read the level through.
    ctx.strokeStyle = 'rgba(255,255,255,.06)';
    ctx.beginPath();
    for (var gx = 0; gx <= lv.width; gx++) { ctx.moveTo(gx * tile * z, 0); ctx.lineTo(gx * tile * z, canvas.height); }
    for (var gy = 0; gy <= lv.height; gy++) { ctx.moveTo(0, gy * tile * z); ctx.lineTo(canvas.width, gy * tile * z); }
    ctx.stroke();
    drawOverlay(ctx, { name: 'level', cellSize: function () { return tile * z; } });
  }

  bindPaint('levelcanvas', levelSurface);

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
      el('button', {
        text: 'duplicate',
        onclick: function () {
          change('duplicate a sprite');
          var copy = JSON.parse(JSON.stringify(s));
          copy.id = uniqueSpriteId(s.id);
          G.sprites.push(copy);
          sel.sprite = G.sprites.length - 1;
          sel.frame = 0;
          renderSprites(); drawSpriteEditor(); markDirty();
        }
      }),
      el('button', { class: 'danger', text: 'delete', onclick: function () { deleteSprite(); } })
    ]));
    renderPaletteEdit();
    renderSpriteShapes();
  }

  function uniqueSpriteId(base) {
    var n = 2;
    while (spriteById(base + n)) n++;
    return base + n;
  }

  function renameSprite(s, name) {
    name = (name || '').trim();
    if (!name || spriteById(name)) { status('that name is taken', 'warn'); return; }
    if (name === s.id) return;
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
    change('clear a frame');
    var s = sprite();
    s.frames[sel.frame] = s.frames[sel.frame].map(function (row) { return row.replace(/./g, '.'); });
    drawSpriteEditor(); markDirty(true);
  }

  function deleteSprite() {
    var s = sprite();
    var used = G.objects.some(function (o) { return o.sprite === s.id; });
    if (used) { status('that sprite is in use by an object', 'warn'); return; }
    change('delete a sprite');
    G.sprites.splice(sel.sprite, 1);
    sel.sprite = Math.max(0, sel.sprite - 1);
    renderSprites(); drawSpriteEditor(); markDirty(true);
  }

  $('addsprite').addEventListener('click', function () {
    change('add a sprite');
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
    // Onion skin: the frame before this one, faintly, so an animation can be
    // drawn against the pose it follows instead of from memory.
    if (sel.onion && s.frames.length > 1) {
      ctx.globalAlpha = 0.28;
      drawSprite(ctx, s, (sel.frame + s.frames.length - 1) % s.frames.length, 0, 0, zoom);
      ctx.globalAlpha = 1;
    }
    drawSprite(ctx, s, sel.frame, 0, 0, zoom);
    if (floating && floating.on === 'sprite') {
      drawSprite(ctx, { w: P.width(floating.rows), h: P.height(floating.rows), frames: [floating.rows] },
        0, floating.x * zoom, floating.y * zoom, zoom);
    }
    ctx.strokeStyle = 'rgba(255,255,255,.07)';
    ctx.beginPath();
    for (var gx = 0; gx <= s.w; gx++) { ctx.moveTo(gx * zoom, 0); ctx.lineTo(gx * zoom, canvas.height); }
    for (var gy = 0; gy <= s.h; gy++) { ctx.moveTo(0, gy * zoom); ctx.lineTo(canvas.width, gy * zoom); }
    ctx.stroke();
    canvas.dataset.zoom = zoom;
    drawOverlay(ctx, { name: 'sprite', cellSize: function () { return zoom; } });
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
        change('add a frame');
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
          change('delete a frame');
          s.frames.splice(sel.frame, 1);
          sel.frame = Math.max(0, sel.frame - 1);
          drawSpriteEditor(); markDirty(true);
        }
      }));
    }
  }

  bindPaint('spritecanvas', spriteSurface);

  // ------------------------------------------------------- tile definitions
  //
  // A tile kind is one character in a map row plus what that character means.
  // Making them editable is what turns the level editor from "arrange what you
  // were given" into "build what you want": a new kind of ground, a hazard
  // that looks like yours, a door that needs the key you invented.
  var TILEKINDS = ['solid', 'platform', 'hazard', 'ladder', 'water', 'exit',
    'breakable', 'checkpoint', 'locked', 'door', 'empty'];

  // Characters a map may use. Dot is empty and cannot be taken, and the rest
  // are chosen to stay readable when a map is looked at as text.
  var TILECHARS = '#=^%~xoTbLdBHWMNSCPRGKJZ0123456789abcefghijklmnpqrstuvwyz'.split('');

  function freeTileChar(lv) {
    var taken = {};
    lv.tiles.forEach(function (t) { taken[t.char] = 1; });
    for (var i = 0; i < TILECHARS.length; i++) {
      if (!taken[TILECHARS[i]]) return TILECHARS[i];
    }
    return '';
  }

  function tileDef(lv, ch) {
    for (var i = 0; i < lv.tiles.length; i++) if (lv.tiles[i].char === ch) return lv.tiles[i];
    return null;
  }

  function renderTileDef() {
    var box = $('tiledef');
    if (!box) return;
    box.innerHTML = '';
    var lv = level();
    if (!lv || sel.place) return;
    var def = tileDef(lv, sel.tile);
    if (!def) return;

    box.appendChild(el('div', { class: 'listhead' }, [el('span', { text: 'this tile' })]));
    box.appendChild(field('name', 'text', def.name || '', function (v) {
      def.name = v; renderLevels(); markDirty();
    }));
    box.appendChild(select_('kind', TILEKINDS, def.kind || 'solid', function (v) {
      def.kind = v; renderLevels(); markDirty(true);
    }));
    box.appendChild(select_('picture', G.sprites.map(function (s) { return s.id; }), def.sprite || '', function (v) {
      def.sprite = v; renderLevels(); drawLevel(); markDirty(true);
    }));
    if (def.kind === 'door') {
      box.appendChild(field('needs', 'text', def.needs || '', function (v) {
        def.needs = v.trim(); markDirty(true);
      }));
    }
    box.appendChild(el('div', { class: 'row' }, [
      el('button', {
        text: 'edit picture', title: 'open this tile’s sprite in the sprite editor',
        onclick: function () { editSprite(def.sprite); }
      }),
      el('button', {
        class: 'danger', text: 'delete',
        title: 'remove this kind of tile, and clear it off the map',
        onclick: function () { deleteTileDef(lv, def); }
      })
    ]));
    var used = countChar(lv.map, def.char);
    box.appendChild(el('p', { class: 'note', text: used + ' on this map · character "' + def.char + '"' }));
  }

  function countChar(rows, ch) {
    var n = 0;
    rows.forEach(function (row) {
      for (var i = 0; i < row.length; i++) if (row[i] === ch) n++;
    });
    return n;
  }

  function addTileDef() {
    var lv = level();
    if (!lv) return;
    var ch = freeTileChar(lv);
    if (!ch) { status('this level has as many kinds of tile as a map can hold', 'warn'); return; }
    change('add a tile');
    lv.tiles.push({
      char: ch, kind: 'solid', name: 'tile ' + (lv.tiles.length + 1),
      sprite: (G.sprites[0] || {}).id || ''
    });
    sel.place = null;
    sel.tile = ch;
    renderLevels();
    markDirty(true);
  }

  function deleteTileDef(lv, def) {
    if (lv.tiles.length < 2) { status('a level needs at least one kind of tile', 'warn'); return; }
    change('delete a tile');
    lv.map = P.replace(lv.map, def.char, '.');
    lv.tiles = lv.tiles.filter(function (t) { return t !== def; });
    sel.tile = lv.tiles[0].char;
    renderLevels();
    drawLevel();
    markDirty(true);
  }

  // editSprite jumps to the sprite editor with a given sprite open. Going
  // looking for it in a list of thirty is the sort of small friction that
  // stops people changing things they should be changing.
  function editSprite(id) {
    for (var i = 0; i < G.sprites.length; i++) {
      if (G.sprites[i].id === id) {
        sel.sprite = i;
        sel.frame = 0;
        showTab('sprites');
        renderSprites();
        drawSpriteEditor();
        return;
      }
    }
    status('that tile has no picture yet — pick one first', 'warn');
  }

  function showTab(name) {
    Array.prototype.forEach.call(document.querySelectorAll('#tabs button'), function (b) {
      b.classList.toggle('on', b.dataset.tab === name);
    });
    Array.prototype.forEach.call(document.querySelectorAll('.tabpage'), function (page) {
      page.hidden = page.id !== 'tab-' + name;
    });
    if (name !== 'play') stopPlaying();
  }

  // -------------------------------------------------------- a placed object
  //
  // Placing an object is half the job; which way it faces and how far it walks
  // is the other half, and until now there was nowhere to say either.
  function renderSpawnProps() {
    var box = $('spawnprops');
    if (!box) return;
    box.innerHTML = '';
    var lv = level();
    if (!lv || sel.spawn < 0 || !lv.spawns || !lv.spawns[sel.spawn]) return;
    var sp = lv.spawns[sel.spawn];
    var o = objectById(sp.object);

    box.appendChild(el('div', { class: 'listhead' }, [
      el('span', { text: 'placed: ' + (o ? (o.name || o.id) : sp.object) })
    ]));
    box.appendChild(field('x', 'number', sp.x, function (v) {
      sp.x = clampNum(v, 0, lv.width - 1); drawLevel(); markDirty(true);
    }));
    box.appendChild(field('y', 'number', sp.y, function (v) {
      sp.y = clampNum(v, 0, lv.height - 1); drawLevel(); markDirty(true);
    }));
    box.appendChild(select_('facing', ['right', 'left'], sp.dir < 0 ? 'left' : 'right', function (v) {
      sp.dir = v === 'left' ? -1 : 1; markDirty(true);
    }));
    box.appendChild(el('div', { class: 'row' }, [
      el('button', { text: 'edit this kind', onclick: function () { editObject(sp.object); } }),
      el('button', {
        class: 'danger', text: 'remove',
        onclick: function () {
          change('remove an object');
          lv.spawns.splice(sel.spawn, 1);
          sel.spawn = -1;
          renderSpawnProps(); drawLevel(); markDirty(true);
        }
      })
    ]));
  }

  function clampNum(v, lo, hi) {
    var n = parseFloat(v);
    if (isNaN(n)) return lo;
    return Math.max(lo, Math.min(hi, n));
  }

  function editObject(id) {
    for (var i = 0; i < G.objects.length; i++) {
      if (G.objects[i].id === id) {
        sel.object = i;
        showTab('objects');
        renderObjects();
        return;
      }
    }
  }

  // ------------------------------------------------------------- the palette
  //
  // Every colour in the game comes from this list, and a sprite stores an
  // index into it rather than a colour. Changing one entry repaints everything
  // drawn in it at once, which is the whole reason it works this way.
  function renderPaletteEdit() {
    var box = $('paletteedit');
    if (!box) return;
    box.innerHTML = '';
    var i = sel.color;
    var current = G.palette[i] || '#000000';
    box.appendChild(el('div', { class: 'listhead' }, [el('span', { text: 'colour ' + i.toString(16) })]));
    box.appendChild(field('colour', 'color', current, function (v) {
      G.palette[i] = v;
      renderSprites(); drawSpriteEditor(); drawLevel(); markDirty(true);
    }));
    box.appendChild(el('p', {
      class: 'note',
      text: 'Every pixel drawn in this colour changes with it, everywhere in the game.'
    }));
    if (G.palette.length < 16) {
      box.appendChild(el('div', { class: 'row' }, [
        el('button', {
          text: '+ colour',
          onclick: function () {
            change('add a colour');
            G.palette.push('#ffffff');
            sel.color = G.palette.length - 1;
            renderSprites(); markDirty(true);
          }
        })
      ]));
    }
  }

  // ---------------------------------------------------------- sprite shapes
  //
  // Turning, flipping and nudging a drawing. Every one of these is minutes of
  // clicking done by hand, and each is a single pass over the rows.
  function renderSpriteShapes() {
    var box = $('spriteshapes');
    if (!box) return;
    box.innerHTML = '';
    var s = sprite();
    if (!s) return;

    function act(label, title, fn) {
      box.appendChild(el('button', { class: 'tool', text: label, title: title, onclick: fn }));
    }
    function onFrame(label, fn) {
      change(label);
      s.frames[sel.frame] = fn(s.frames[sel.frame]);
      drawSpriteEditor(); markDirty(true);
    }

    act('flip ↔', 'mirror this frame left to right', function () {
      onFrame('flip a frame', P.flipH);
    });
    act('flip ↕', 'mirror this frame top to bottom', function () {
      onFrame('flip a frame', P.flipV);
    });
    act('turn ⟳', 'rotate this frame a quarter turn', function () {
      var rows = P.rotate(s.frames[sel.frame]);
      change('turn a frame');
      if (P.width(rows) !== s.w || P.height(rows) !== s.h) {
        // A rotation of a non-square sprite changes its shape, and every frame
        // has to keep the same shape as the rest.
        s.frames = s.frames.map(function (f, i) {
          return i === sel.frame ? rows : P.resize(P.rotate(f), P.width(rows), P.height(rows), '.');
        });
        s.w = P.width(rows);
        s.h = P.height(rows);
      } else {
        s.frames[sel.frame] = rows;
      }
      renderSprites(); drawSpriteEditor(); markDirty(true);
    });
    ['←', '→', '↑', '↓'].forEach(function (arrow, i) {
      var d = [[-1, 0], [1, 0], [0, -1], [0, 1]][i];
      act(arrow, 'nudge this frame one pixel (hold shift to wrap it round)', function (e) {
        onFrame('nudge a frame', function (rows) {
          return P.shift(rows, d[0], d[1], '.', e.shiftKey);
        });
      });
    });
    act('outline', 'draw a border round everything in this frame', function () {
      onFrame('outline a frame', function (rows) {
        return P.plot(rows, P.outline(rows, '.'), sel.color.toString(16));
      });
    });
    act('onion', 'show the frame before this one faintly underneath', function () {
      sel.onion = !sel.onion;
      renderSpriteShapes();
      drawSpriteEditor();
    });
    if (sel.onion) box.lastChild.classList.add('on');
    act('copy ▸', 'copy this frame over the next one', function () {
      if (s.frames.length < 2) { status('there is only one frame', 'warn'); return; }
      change('copy a frame');
      s.frames[(sel.frame + 1) % s.frames.length] = s.frames[sel.frame].slice();
      drawSpriteEditor(); markDirty(true);
    });
  }

  // ------------------------------------------------------ copy, cut, paste
  //
  // A selection can be copied between the two editors, because both hold the
  // same kind of grid. What the characters mean changes, so pasting a sprite
  // into a level would be nonsense — and is refused rather than silently
  // making a map out of colour numbers.
  function activeSurface() {
    if (!$('tab-levels').hidden) return levelSurface();
    if (!$('tab-sprites').hidden) return spriteSurface();
    return null;
  }

  function copySelection(cut) {
    var surface = activeSurface();
    if (!surface || !selection || selectionOn !== surface.name) {
      status('select a region first (the select tool, or press M)', 'warn');
      return;
    }
    clip = {
      from: surface.name,
      rows: P.read(surface.rows(), selection.x0, selection.y0, selection.x1, selection.y1)
    };
    if (cut) {
      change('cut');
      surface.put(P.clear(surface.rows(), selection.x0, selection.y0, selection.x1, selection.y1, surface.empty));
      surface.redraw();
      markDirty(true);
    }
    status((cut ? 'cut ' : 'copied ') + P.width(clip.rows) + '×' + P.height(clip.rows));
  }

  function pasteClip() {
    var surface = activeSurface();
    if (!surface) return;
    if (!clip) { status('nothing copied yet', 'warn'); return; }
    if (clip.from !== surface.name) {
      status('that was copied from the ' + clip.from + ' editor, where the characters mean something else', 'warn');
      return;
    }
    change('paste');
    // Paste lands where the selection is, or at the corner when there is none,
    // and becomes the new selection so it can be dragged into place.
    var x = selection && selectionOn === surface.name ? Math.min(selection.x0, selection.x1) : 0;
    var y = selection && selectionOn === surface.name ? Math.min(selection.y0, selection.y1) : 0;
    surface.put(P.write(surface.rows(), clip.rows, x, y, null));
    selection = { x0: x, y0: y, x1: x + P.width(clip.rows) - 1, y1: y + P.height(clip.rows) - 1 };
    selectionOn = surface.name;
    surface.redraw();
    markDirty(true);
  }

  function deleteSelection() {
    var surface = activeSurface();
    if (!surface || !selection || selectionOn !== surface.name) return;
    change('clear a region');
    surface.put(P.clear(surface.rows(), selection.x0, selection.y0, selection.x1, selection.y1, surface.empty));
    // On a level, clearing a region takes the objects standing in it too:
    // leaving them floating over cleared ground is never what was meant.
    if (surface.name === 'level') {
      var lv = level();
      lv.spawns = (lv.spawns || []).filter(function (sp) {
        return !inside(selection, Math.round(sp.x), Math.round(sp.y));
      });
      sel.spawn = -1;
      renderSpawnProps();
    }
    surface.redraw();
    markDirty(true);
  }

  function selectAll() {
    var surface = activeSurface();
    if (!surface) return;
    dropFloating();
    selection = { x0: 0, y0: 0, x1: surface.w - 1, y1: surface.h - 1 };
    selectionOn = surface.name;
    setTool('select');
    surface.redraw();
  }

  // ---------------------------------------------------------------- keys
  //
  // Everything here has a button somewhere; these are for the hand that is not
  // on the mouse. They are deliberately the keys other pixel editors use, so
  // nobody has to learn a new set for this one.
  document.addEventListener('keydown', function (e) {
    if (!G) return;
    var t = e.target;
    // Never steal a key from something being typed into.
    if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT')) return;

    var mod = e.ctrlKey || e.metaKey;
    if (mod) {
      var k = e.key.toLowerCase();
      if (k === 'z') { e.preventDefault(); e.shiftKey ? redo() : undo(); return; }
      if (k === 'y') { e.preventDefault(); redo(); return; }
      if (k === 'c') { e.preventDefault(); copySelection(false); return; }
      if (k === 'x') { e.preventDefault(); copySelection(true); return; }
      if (k === 'v') { e.preventDefault(); pasteClip(); return; }
      if (k === 'a') { e.preventDefault(); selectAll(); return; }
      if (k === 's') { e.preventDefault(); save(); return; }
      return;
    }

    if (e.key === 'Delete' || e.key === 'Backspace') {
      if (selection) { e.preventDefault(); deleteSelection(); }
      return;
    }
    if (e.key === 'Escape') {
      dropFloating();
      selection = null;
      sel.spawn = -1;
      preview = null;
      drawLevel(); drawSpriteEditor(); renderSpawnProps();
      return;
    }

    if (e.key.toLowerCase() === 't' && !$('tab-levels').hidden) {
      playThisLevel(e.shiftKey ? lastCell : null);
      return;
    }

    // A tool by its letter.
    for (var i = 0; i < TOOLS.length; i++) {
      if (e.key.toLowerCase() === TOOLS[i].key) { setTool(TOOLS[i].id); return; }
    }

    // Square brackets zoom the level, which is how you go from placing a tile
    // to seeing whether a jump looks possible.
    if (!$('tab-levels').hidden) {
      if (e.key === '[') { sel.zoom = Math.max(1, sel.zoom - 1); renderLevelProps(); drawLevel(); return; }
      if (e.key === ']') { sel.zoom = Math.min(6, sel.zoom + 1); renderLevelProps(); drawLevel(); return; }
    }

    // Numbers pick a colour while drawing, or a tile while building.
    if (/^[0-9a-f]$/.test(e.key)) {
      if (!$('tab-sprites').hidden) {
        var idx = parseInt(e.key, 16);
        if (idx < G.palette.length) { sel.color = idx; renderSprites(); }
      } else if (!$('tab-levels').hidden) {
        var lv = level();
        var n = parseInt(e.key, 16);
        if (lv && lv.tiles[n]) { sel.place = null; sel.tile = lv.tiles[n].char; renderLevels(); }
      }
      return;
    }

    // Comma and full stop step through the frames of an animation.
    if (!$('tab-sprites').hidden && (e.key === ',' || e.key === '.')) {
      var s = sprite();
      if (!s) return;
      sel.frame = (sel.frame + (e.key === '.' ? 1 : s.frames.length - 1)) % s.frames.length;
      drawSpriteEditor();
    }
  });

  // --------------------------------------------------------------- widgets

  // Every panel in the studio is built from these three, which is why undo is
  // recorded here rather than at each of the hundred places that change
  // something. A field cannot be edited without the state before it being
  // kept, so there is no way to add a setting that quietly is not undoable.
  function field(label, type, value, onchange) {
    var input = el('input', { type: type, value: value == null ? '' : value });
    input.addEventListener('change', function () {
      change('change ' + label, type === 'color' || type === 'range');
      onchange(type === 'checkbox' ? input.checked : input.value);
    });
    if (type === 'checkbox') input.checked = !!value;
    return el('div', { class: 'field' }, [el('label', { text: label }), input]);
  }

  function check(label, value, onchange) {
    var input = el('input', { type: 'checkbox' });
    input.checked = !!value;
    input.addEventListener('change', function () {
      change('change ' + label);
      onchange(input.checked);
    });
    return el('div', { class: 'field' }, [el('label', { text: label }), input]);
  }

  function select_(label, options, value, onchange) {
    var s = el('select');
    options.forEach(function (o) {
      s.appendChild(el('option', { value: o, text: o, selected: o === value }));
    });
    s.addEventListener('change', function () {
      change('change ' + label);
      onchange(s.value);
    });
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

  // Every shortcut the editor has, written down where it can be found. A key
  // nobody knows about is the same as a key that does not exist.
  var SHORTCUTS = [
    ['B  L  R  O  G  I  M', 'pencil · line · box · oval · fill · pick · select'],
    ['shift while dragging', 'fill the box or oval being drawn'],
    ['right button', 'erase with whatever tool is in hand'],
    ['middle-drag, shift+drag', 'scroll a map wider than the window'],
    ['ctrl+Z · ctrl+shift+Z', 'undo · redo, over anything in the editor'],
    ['ctrl+C · ctrl+X · ctrl+V', 'copy · cut · paste a selected region'],
    ['ctrl+A · delete · escape', 'select everything · clear it · drop the selection'],
    ['0–9, a–f', 'a colour while drawing, a tile while building'],
    ['[  ]', 'zoom the level out and in'],
    [',  .', 'step back and forward through animation frames'],
    ['T · shift+T', 'test this level · test it from where the cursor is'],
    ['ctrl+S', 'save now, rather than waiting for the studio to do it']
  ];

  function renderShortcuts(form) {
    form.appendChild(el('h3', { text: 'keys' }));
    var table = el('div', { class: 'keys' });
    SHORTCUTS.forEach(function (row) {
      table.appendChild(el('kbd', { text: row[0] }));
      table.appendChild(el('span', { text: row[1] }));
    });
    form.appendChild(table);
  }

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
    renderShortcuts(form);
  }

  // ------------------------------------------------------------------ code
  //
  // A game's own script. Rules are the game saying what happens in its own
  // data; this is for the things nobody thought to make a rule for — a boss
  // that changes at half health, a lift waiting on two switches, a counter of
  // your own invention.
  //
  // The reference on the left is the whole API. It is written down here rather
  // than generated because the useful half of a reference is the sentence
  // after the name, and only a person can write that.
  var API = [
    ['when something happens', [
      ['when.start(fn)', 'the level begins, and again for every level after it'],
      ['when.update(fn)', 'every frame — fn(dt) gets the seconds since the last one'],
      ['when.collide(a, b, fn)', 'two things touch, named by id or by kind'],
      ['when.collect(id, fn)', 'the player picks something up'],
      ['when.destroy(id, fn)', 'something is destroyed'],
      ['when.grant(name, fn)', 'the player gains an ability'],
      ['when.tile(kind, fn)', 'the player steps on a kind of tile'],
      ['when.key(name, fn)', 'a key is pressed: jump, fire, up, down, left, right'],
      ['when.count(id, fn)', 'the last one of something is gone'],
      ['when.health(fn)', 'the player is hurt'],
      ['when.levelClear(fn)', 'the level is finished']
    ]],
    ['later', [
      ['every(seconds, fn)', 'run it again and again, this often'],
      ['after(seconds, fn)', 'run it once, this long from now']
    ]],
    ['the player', [
      ['player.x, player.y', 'where, in tiles — set them to move'],
      ['player.vx, player.vy', 'how fast, in tiles a second'],
      ['player.hp', 'health left'],
      ['player.facing', '"left" or "right"'],
      ['player.grounded', 'true when standing on something'],
      ['player.jump(n)', 'send it upward'],
      ['player.push(x, y)', 'add to its speed'],
      ['player.hurt(n)', 'take health off it'],
      ['player.heal(n)', 'put health back'],
      ['player.distanceTo(t)', 'how far away something is, in tiles']
    ]],
    ['things in the level', [
      ['spawn(id, x, y)', 'put one there, and hand it back'],
      ['find(id)', 'the first one of these, or null'],
      ['all(idOrKind)', 'every one of them, as a list'],
      ['count(id)', 'how many are left'],
      ['t.remove()', 'take one out of the game']
    ]],
    ['the game', [
      ['score(n)', 'add to the score — score() reads it'],
      ['lives(n)', 'set lives — lives() reads them'],
      ['give(name)', 'give the player an ability'],
      ['has(name)', 'true if the player has it'],
      ['set(name, v)', 'remember something'],
      ['get(name)', 'read it back'],
      ['goTo(levelId)', 'move to another level'],
      ['win()', 'the game is won'],
      ['lose()', 'the player dies']
    ]],
    ['saying and hearing', [
      ['say(text, seconds)', 'put words on the screen'],
      ['sound(name)', 'play one of the game’s sounds'],
      ['shake(n)', 'shake the screen'],
      ['log(...)', 'print to the browser console, for finding things out']
    ]],
    ['the map', [
      ['tiles.at(x, y)', 'what kind of tile is there'],
      ['tiles.set(x, y, kind)', 'put a tile there'],
      ['tiles.clear(x, y)', 'take it away'],
      ['level.id, level.name', 'which level this is'],
      ['level.width, level.height', 'how big it is, in tiles'],
      ['level.time', 'seconds since it started']
    ]],
    ['odds and ends', [
      ['random(n)', 'a whole number from 0 to n-1'],
      ['pick(list)', 'one of them, at random']
    ]]
  ];

  // Starters are whole small ideas rather than single calls: the point of one
  // is to be changed, so each has something obvious to change in it.
  var SNIPPETS = [
    ['say hello', 'when.start(function () {\n  say("here we go", 2);\n});\n'],
    ['a boss that speeds up', 'when.update(function () {\n  var boss = find("boss");\n  if (!boss) return;\n  if (boss.hp <= 2) boss.vx = boss.vx * 1.01;\n});\n'],
    ['drop something every few seconds', 'every(3, function () {\n  spawn("coin", random(level.width), 1);\n});\n'],
    ['a double jump', 'var jumps = 0;\nwhen.update(function () {\n  if (player.grounded) jumps = 0;\n});\nwhen.key("jump", function () {\n  jumps = jumps + 1;\n  if (jumps === 2) player.jump(11);\n});\n'],
    ['chase the player', 'when.update(function () {\n  all("enemy").forEach(function (foe) {\n    foe.vx = player.x > foe.x ? 3 : -3;\n  });\n});\n'],
    ['a key that opens the way', 'when.collect("key", function () {\n  give("the key");\n  say("the gate is open", 2);\n  tiles.clear(18, 8);\n});\n'],
    ['count what is left', 'when.update(function () {\n  set("left", count("coin"));\n});\nwhen.count("coin", function () {\n  say("all of them!", 2);\n  win();\n});\n'],
    ['hurt the player on a tile', 'when.tile("water", function () {\n  player.hurt(1);\n});\n'],
    ['a shop, sort of', 'when.collide("player", "shopkeeper", function () {\n  if (score() >= 50) {\n    score(-50);\n    give("boots");\n    say("boots bought", 2);\n  } else {\n    say("fifty coins, no less", 2);\n  }\n});\n']
  ];

  function renderCode() {
    if (!G) return;
    var box = $('codetext');
    if (document.activeElement !== box) box.value = G.code || '';

    var list = $('snippets');
    list.innerHTML = '';
    SNIPPETS.forEach(function (row) {
      list.appendChild(el('li', {
        title: 'put this into your code',
        onclick: function () { insertCode('\n' + row[1]); }
      }, [el('span', { text: row[0] })]));
    });

    var ref = $('apiref');
    ref.innerHTML = '';
    API.forEach(function (group) {
      ref.appendChild(el('h4', { text: group[0] }));
      group[1].forEach(function (row) {
        ref.appendChild(el('div', {}, [
          el('code', {
            text: row[0],
            title: 'put this into your code',
            onclick: function () { insertCode(callOf(row[0])); }
          }),
          el('span', { text: '  ' + row[1] })
        ]));
      });
    });
    checkCode(true);
  }

  // callOf turns a reference line into something worth typing: a listener
  // arrives with its function already open, so there is somewhere to write.
  function callOf(signature) {
    var name = signature.split('(')[0];
    if (signature.indexOf('fn)') >= 0) {
      var args = signature.slice(signature.indexOf('(') + 1, signature.lastIndexOf(')'))
        .split(',').map(function (a) { return a.trim(); }).filter(function (a) { return a !== 'fn'; });
      var head = args.length ? args.map(function (a) { return '"' + a + '"'; }).join(', ') + ', ' : '';
      return name + '(' + head + 'function () {\n  \n});\n';
    }
    if (signature.indexOf('(') < 0) return name;
    return signature + ';\n';
  }

  function insertCode(text) {
    var box = $('codetext');
    var at = box.selectionStart == null ? box.value.length : box.selectionStart;
    var end = box.selectionEnd == null ? at : box.selectionEnd;
    change('edit the code', true);
    box.value = box.value.slice(0, at) + text + box.value.slice(end);
    box.focus();
    box.selectionStart = box.selectionEnd = at + text.length;
    saveCode();
  }

  function saveCode() {
    if (!G) return;
    G.code = $('codetext').value;
    checkCode(true);
    markDirty(true);
  }

  // checkCode compiles the script without running it, so a typo is caught
  // here rather than by a black screen in the play tab. The names the engine
  // gives a script are declared as arguments, so a call to one of them reads
  // as a call rather than as an unknown variable.
  var SCRIPT_NAMES = ['when', 'every', 'after', 'spawn', 'find', 'all', 'count',
    'say', 'sound', 'shake', 'score', 'lives', 'give', 'has', 'get', 'set',
    'goTo', 'win', 'lose', 'tiles', 'random', 'pick', 'player', 'level', 'log'];

  function checkCode(quiet) {
    var box = $('codeerr');
    var source = ($('codetext').value || '').trim();
    if (!source) {
      box.hidden = true;
      $('codenote').textContent = 'no code yet — the game runs on its rules alone';
      return true;
    }
    try {
      /* jshint evil:true */
      Function.apply(null, SCRIPT_NAMES.concat(['"use strict";\n' + source]));
      box.hidden = quiet;
      box.className = 'codeerr good';
      box.textContent = 'This reads as valid code. Press "run it" to see what it does.';
      $('codenote').textContent = source.split('\n').length + ' lines';
      return true;
    } catch (err) {
      box.hidden = false;
      box.className = 'codeerr';
      box.textContent = explainCodeError(err, source);
      $('codenote').textContent = 'there is a mistake in it';
      return false;
    }
  }

  // Where a mistake is, rather than only that there is one. The finding is
  // its own file so it can be tested without a browser.
  function explainCodeError(err, source) {
    return window.CodeCheck.explain(err, source);
  }

  $('codetext').addEventListener('input', function () {
    change('edit the code', true);
    saveCode();
  });

  // Tab indents rather than leaving the box, because leaving the box in the
  // middle of writing a function is never what was wanted.
  $('codetext').addEventListener('keydown', function (e) {
    if (e.key !== 'Tab') return;
    e.preventDefault();
    var at = this.selectionStart, end = this.selectionEnd;
    this.value = this.value.slice(0, at) + '  ' + this.value.slice(end);
    this.selectionStart = this.selectionEnd = at + 2;
    saveCode();
  });

  $('checkcode').addEventListener('click', function () {
    if (checkCode(false)) status('the code reads as valid');
    else status('there is a mistake in the code', 'warn');
  });

  $('runcode').addEventListener('click', function () {
    if (!checkCode(false)) { status('fix the mistake first', 'warn'); return; }
    scriptTrouble = [];
    playThisLevel(null);
  });

  // What the script complained about while the game was running. A mistake
  // that only shows up on the twentieth frame is the hardest kind to find, so
  // it is carried back to the editor rather than left in the console.
  var scriptTrouble = [];

  function noteScriptTrouble(text) {
    if (scriptTrouble.indexOf(text) >= 0) return;
    scriptTrouble.push(text);
    var box = $('codeerr');
    box.hidden = false;
    box.className = 'codeerr';
    box.textContent = 'While the game was running:\n\n' + scriptTrouble.join('\n');
    status('the code ran into trouble — see the code tab', 'warn');
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
    // A Claude Code nobody has signed into cannot design anything, so the
    // studio does not get to call itself connected to it. Saying otherwise
    // is how a game gets made offline while the badge insists Claude made it.
    var unusable = info.claudeCode && info.signedIn === 'no' &&
      !!(info.source && info.source.indexOf('Claude Code') >= 0);
    var working = info.online && !unusable;

    var badge = $('assistant');
    badge.textContent = working ? info.assistant : (unusable ? 'not signed in' : 'offline designer');
    badge.className = 'badge' + (working ? '' : ' offline');
    badge.title = working
      ? 'designing with ' + info.assistant + ' (' + (info.source || 'connected') + ') — click to change'
      : (unusable ? 'Claude Code is installed but not signed in — click to sign in'
        : 'click to design with Claude instead');
    $('connect').hidden = false;
    $('connectnote').textContent = unusable
      ? 'Claude Code is on this machine but not signed in, so designs are still being made offline.'
      : (info.online
        ? 'Designing with ' + info.assistant + ', using ' + (info.source || 'a saved key') + '.'
        : 'The studio designs offline on its own. Give it a key and Claude does the designing instead — better games, and it understands what you ask for.');
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
    // Installed and signed in are two different problems with two different
    // buttons, and offering "use it" to a Claude Code that will refuse is how
    // somebody ends up staring at a design that quietly happened offline.
    var needsSignIn = info.claudeCode && info.signedIn === 'no';
    $('signinhelp').hidden = !needsSignIn;
    $('clirow').hidden = !info.claudeCode || needsSignIn;
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
  $('recheck2').addEventListener('click', function () { connect({ use: 'claude-code' }); });
  $('signin').addEventListener('click', function () {
    var button = this;
    button.disabled = true;
    clearAlert();
    status('opening the sign-in…');
    api('/api/signin', { method: 'POST', body: '{}' })
      .then(function (res) {
        showAlert(res.says);
        status('finish signing in, then press "check again"');
      })
      .catch(function (e) { showAlert(e.message); status('could not open the sign-in', 'err'); })
      .then(function () { button.disabled = false; });
  });
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
