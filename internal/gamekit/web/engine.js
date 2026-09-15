// The Oliver game engine: it plays a project.
//
// One file, no dependencies, no build step. The editor runs it to play-test,
// and an exported game is this file plus the project JSON inlined into a page,
// which is why a game exported from the studio is a single file that works
// offline and forever.
//
//   OliverEngine.start(canvas, game, options) -> runtime
//
// Everything the engine knows about a game comes from the project data. There
// is no per-game code anywhere: a new kind of enemy is a new behaviour here,
// available to every game, rather than a script attached to one.
(function (global) {
  'use strict';

  var TRANSPARENT = '.';
  var KEY_ALIASES = {
    left: ['ArrowLeft', 'KeyA'],
    right: ['ArrowRight', 'KeyD'],
    up: ['ArrowUp', 'KeyW'],
    down: ['ArrowDown', 'KeyS'],
    jump: ['Space', 'ArrowUp', 'KeyW', 'KeyZ'],
    fire: ['KeyX', 'KeyJ', 'ControlLeft', 'Enter'],
    pause: ['KeyP', 'Escape'],
    start: ['Enter', 'Space']
  };

  // ---------------------------------------------------------------- helpers

  function clamp(v, lo, hi) { return v < lo ? lo : (v > hi ? hi : v); }
  function sign(v) { return v < 0 ? -1 : (v > 0 ? 1 : 0); }

  function overlaps(a, b) {
    return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
  }

  // deterministic RNG, so a game that looks a certain way keeps looking that way
  function rng(seed) {
    var s = seed >>> 0 || 1;
    return function () {
      s ^= s << 13; s >>>= 0;
      s ^= s >> 17;
      s ^= s << 5; s >>>= 0;
      return s / 4294967296;
    };
  }

  // ------------------------------------------------------------ sprite bank

  // Sprites are stored as text, one character per pixel. Turning each frame
  // into its own canvas once at load keeps drawing to a single drawImage call
  // per entity per frame, which is what lets hundreds of them run at 60fps in
  // a browser.
  function SpriteBank(game) {
    this.frames = {};
    this.size = {};
    this.fps = {};
    var palette = game.palette || [];
    var sprites = game.sprites || [];
    for (var i = 0; i < sprites.length; i++) {
      var sp = sprites[i];
      var list = [];
      for (var f = 0; f < (sp.frames || []).length; f++) {
        list.push(this.render(sp.frames[f], sp.w, sp.h, palette));
      }
      if (!list.length) list.push(this.render([], sp.w || 8, sp.h || 8, palette));
      this.frames[sp.id] = list;
      this.size[sp.id] = { w: sp.w || 8, h: sp.h || 8 };
      this.fps[sp.id] = sp.fps || 0;
    }
  }

  SpriteBank.prototype.render = function (rows, w, h, palette) {
    var c = document.createElement('canvas');
    c.width = Math.max(1, w); c.height = Math.max(1, h);
    var g = c.getContext('2d');
    for (var y = 0; y < rows.length; y++) {
      var row = rows[y];
      for (var x = 0; x < row.length; x++) {
        var ch = row[x];
        if (ch === TRANSPARENT || ch === ' ') continue;
        var idx = parseInt(ch, 16);
        if (isNaN(idx)) continue;
        g.fillStyle = palette[idx] || '#ff00ff';
        g.fillRect(x, y, 1, 1);
      }
    }
    return c;
  };

  SpriteBank.prototype.get = function (id, frame) {
    var list = this.frames[id];
    if (!list || !list.length) return null;
    return list[((frame | 0) % list.length + list.length) % list.length];
  };

  SpriteBank.prototype.count = function (id) {
    var list = this.frames[id];
    return list ? list.length : 0;
  };

  SpriteBank.prototype.dims = function (id, fallback) {
    return this.size[id] || { w: fallback, h: fallback };
  };

  // ------------------------------------------------------------------ audio

  // Sound is synthesised rather than sampled: a game's whole soundtrack is a
  // few numbers in the project file, so an exported game has nothing to load.
  function Audio(game) {
    this.game = game;
    this.ctx = null;
    this.muted = false;
    this.musicTimer = null;
    this.sounds = {};
    (game.sounds || []).forEach(function (s) { this.sounds[s.id] = s; }, this);
  }

  Audio.prototype.resume = function () {
    if (!this.ctx) {
      var AC = global.AudioContext || global.webkitAudioContext;
      if (!AC) return;
      try { this.ctx = new AC(); } catch (e) { return; }
    }
    if (this.ctx.state === 'suspended') this.ctx.resume();
  };

  Audio.prototype.play = function (id) {
    if (this.muted || !id) return;
    var s = this.sounds[id] || BUILTIN_SOUNDS[id];
    if (!s) return;
    this.resume();
    if (!this.ctx) return;
    var now = this.ctx.currentTime;
    var dur = s.dur || 0.1;
    var gain = this.ctx.createGain();
    gain.gain.setValueAtTime(0.0001, now);
    gain.gain.exponentialRampToValueAtTime(Math.max(0.0002, s.vol || 0.25), now + 0.01);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + dur);
    gain.connect(this.ctx.destination);

    if (s.wave === 'noise') {
      var len = Math.max(1, Math.floor(this.ctx.sampleRate * dur));
      var buf = this.ctx.createBuffer(1, len, this.ctx.sampleRate);
      var data = buf.getChannelData(0);
      for (var i = 0; i < len; i++) data[i] = (Math.random() * 2 - 1) * (1 - i / len);
      var src = this.ctx.createBufferSource();
      src.buffer = buf;
      src.connect(gain);
      src.start(now);
      return;
    }
    var osc = this.ctx.createOscillator();
    osc.type = s.wave || 'square';
    osc.frequency.setValueAtTime(s.freq || 440, now);
    if (s.to) osc.frequency.exponentialRampToValueAtTime(Math.max(20, s.to), now + dur);
    osc.connect(gain);
    osc.start(now);
    osc.stop(now + dur + 0.02);
  };

  // A short looping bassline and arpeggio, seeded by the track name. It is not
  // a composition — it is atmosphere that costs nothing to ship.
  Audio.prototype.music = function (name) {
    this.stopMusic();
    if (!name || this.muted) return;
    this.resume();
    if (!this.ctx) return;
    var seed = 0;
    for (var i = 0; i < name.length; i++) seed = (seed * 31 + name.charCodeAt(i)) >>> 0;
    var rand = rng(seed || 7);
    var scale = [0, 2, 3, 5, 7, 8, 10, 12];
    var pattern = [];
    for (var n = 0; n < 16; n++) {
      pattern.push(scale[Math.floor(rand() * scale.length)]);
    }
    var root = 110 * Math.pow(2, Math.floor(rand() * 3) / 12);
    var step = 0;
    var self = this;
    var beat = 0.16;
    this.musicTimer = setInterval(function () {
      if (self.muted || !self.ctx) return;
      var now = self.ctx.currentTime;
      var semi = pattern[step % pattern.length];
      self.tone(root * Math.pow(2, semi / 12) * 2, beat * 0.9, 'square', 0.045, now);
      if (step % 4 === 0) self.tone(root, beat * 1.6, 'triangle', 0.09, now);
      step++;
    }, beat * 1000);
  };

  Audio.prototype.tone = function (freq, dur, wave, vol, now) {
    if (!this.ctx) return;
    var g = this.ctx.createGain();
    g.gain.setValueAtTime(0.0001, now);
    g.gain.exponentialRampToValueAtTime(vol, now + 0.01);
    g.gain.exponentialRampToValueAtTime(0.0001, now + dur);
    g.connect(this.ctx.destination);
    var o = this.ctx.createOscillator();
    o.type = wave; o.frequency.setValueAtTime(freq, now);
    o.connect(g); o.start(now); o.stop(now + dur + 0.02);
  };

  Audio.prototype.stopMusic = function () {
    if (this.musicTimer) { clearInterval(this.musicTimer); this.musicTimer = null; }
  };

  var BUILTIN_SOUNDS = {
    jump: { wave: 'square', freq: 320, to: 620, dur: 0.12, vol: 0.2 },
    coin: { wave: 'square', freq: 880, to: 1320, dur: 0.1, vol: 0.2 },
    hit: { wave: 'saw', freq: 220, to: 60, dur: 0.22, vol: 0.25 },
    shoot: { wave: 'square', freq: 700, to: 180, dur: 0.09, vol: 0.16 },
    win: { wave: 'square', freq: 520, to: 1040, dur: 0.5, vol: 0.25 },
    lose: { wave: 'saw', freq: 300, to: 50, dur: 0.6, vol: 0.25 },
    step: { wave: 'noise', freq: 200, dur: 0.05, vol: 0.08 },
    break: { wave: 'noise', freq: 300, dur: 0.14, vol: 0.2 }
  };

  // ------------------------------------------------------------------ input

  function Input(canvas) {
    this.down = {};
    this.pressed = {};
    var self = this;
    this.onKeyDown = function (e) {
      if (self.down[e.code] !== true) self.pressed[e.code] = true;
      self.down[e.code] = true;
      // Arrows and space scroll the page, which makes a game unplayable in a
      // browser window that is taller than the screen.
      if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Space'].indexOf(e.code) >= 0) e.preventDefault();
    };
    this.onKeyUp = function (e) { self.down[e.code] = false; };
    global.addEventListener('keydown', this.onKeyDown);
    global.addEventListener('keyup', this.onKeyUp);
    this.touch = {};
    this.bindTouch(canvas);
  }

  Input.prototype.bindTouch = function (canvas) {
    if (!canvas || !('ontouchstart' in global)) return;
    var self = this;
    var track = function (e) {
      e.preventDefault();
      self.touch = {};
      var rect = canvas.getBoundingClientRect();
      for (var i = 0; i < e.touches.length; i++) {
        var t = e.touches[i];
        var fx = (t.clientX - rect.left) / rect.width;
        var fy = (t.clientY - rect.top) / rect.height;
        if (fy > 0.55 && fx < 0.2) self.touch.left = true;
        else if (fy > 0.55 && fx < 0.42) self.touch.right = true;
        else if (fx > 0.72) self.touch.jump = true;
        else self.touch.fire = true;
      }
    };
    canvas.addEventListener('touchstart', track, { passive: false });
    canvas.addEventListener('touchmove', track, { passive: false });
    canvas.addEventListener('touchend', function (e) { e.preventDefault(); self.touch = {}; }, { passive: false });
  };

  Input.prototype.pads = function () {
    if (!global.navigator || !navigator.getGamepads) return [];
    var list = navigator.getGamepads ? navigator.getGamepads() : [];
    var out = [];
    for (var i = 0; i < list.length; i++) if (list[i]) out.push(list[i]);
    return out;
  };

  Input.prototype.held = function (action) {
    var codes = KEY_ALIASES[action] || [];
    for (var i = 0; i < codes.length; i++) if (this.down[codes[i]]) return true;
    if (this.touch[action]) return true;
    var pads = this.pads();
    for (var p = 0; p < pads.length; p++) {
      var pad = pads[p];
      var ax = pad.axes[0] || 0, ay = pad.axes[1] || 0;
      if (action === 'left' && (ax < -0.4 || (pad.buttons[14] && pad.buttons[14].pressed))) return true;
      if (action === 'right' && (ax > 0.4 || (pad.buttons[15] && pad.buttons[15].pressed))) return true;
      if (action === 'up' && (ay < -0.4 || (pad.buttons[12] && pad.buttons[12].pressed))) return true;
      if (action === 'down' && (ay > 0.4 || (pad.buttons[13] && pad.buttons[13].pressed))) return true;
      if ((action === 'jump' || action === 'start') && pad.buttons[0] && pad.buttons[0].pressed) return true;
      if (action === 'fire' && pad.buttons[2] && pad.buttons[2].pressed) return true;
    }
    return false;
  };

  Input.prototype.tapped = function (action) {
    var codes = KEY_ALIASES[action] || [];
    for (var i = 0; i < codes.length; i++) if (this.pressed[codes[i]]) return true;
    return false;
  };

  Input.prototype.endFrame = function () { this.pressed = {}; };

  Input.prototype.destroy = function () {
    global.removeEventListener('keydown', this.onKeyDown);
    global.removeEventListener('keyup', this.onKeyUp);
  };

  // ------------------------------------------------------------------ level

  function LevelState(engine, def) {
    this.def = def;
    this.engine = engine;
    this.width = def.width;
    this.height = def.height;
    this.tile = engine.game.screen.tile;
    this.tiles = {};
    (def.tiles || []).forEach(function (t) { this.tiles[t.char] = t; }, this);
    this.rows = (def.map || []).map(function (r) { return r.split(''); });
    this.time = 0;
    this.shake = 0;
    this.dirty = true;
  }

  LevelState.prototype.at = function (tx, ty) {
    if (ty < 0 || ty >= this.rows.length) return null;
    var row = this.rows[ty];
    if (!row || tx < 0 || tx >= row.length) return null;
    return this.tiles[row[tx]] || null;
  };

  LevelState.prototype.kindAt = function (tx, ty) {
    var t = this.at(tx, ty);
    return t ? (t.kind || 'empty') : 'empty';
  };

  // Outside the map the world is solid at the sides and open at the top, so a
  // player cannot walk off the edge of a level but can still jump above it.
  LevelState.prototype.solidAt = function (tx, ty) {
    if (tx < 0 || tx >= this.width) return true;
    if (ty < 0) return false;
    if (ty >= this.height) return false;
    var kind = this.kindAt(tx, ty);
    // A door is a wall until it is opened, and a locked tile is a wall until a
    // rule turns it into an exit.
    return kind === 'solid' || kind === 'breakable' || kind === 'door' || kind === 'locked';
  };

  LevelState.prototype.platformAt = function (tx, ty) {
    return this.kindAt(tx, ty) === 'platform';
  };

  LevelState.prototype.setTile = function (tx, ty, ch) {
    if (ty < 0 || ty >= this.rows.length) return;
    if (tx < 0 || tx >= this.rows[ty].length) return;
    this.rows[ty][tx] = ch;
    this.dirty = true;
  };

  // findChar returns the first tile character whose kind matches, which is how
  // a rule such as "open the exit" finds something to place.
  LevelState.prototype.charOfKind = function (kind) {
    for (var ch in this.tiles) {
      if (this.tiles[ch].kind === kind) return ch;
    }
    return null;
  };

  // openRun clears every tile joined to this one that is the same kind of
  // tile, so a door of any size opens as one thing.
  LevelState.prototype.openRun = function (tx, ty) {
    var want = this.rows[ty] && this.rows[ty][tx];
    if (!want || want === '.') return;
    var queue = [[tx, ty]];
    while (queue.length) {
      var cell = queue.pop();
      var x = cell[0], y = cell[1];
      if (y < 0 || y >= this.rows.length || x < 0 || x >= this.rows[y].length) continue;
      if (this.rows[y][x] !== want) continue;
      this.setTile(x, y, '.');
      queue.push([x + 1, y], [x - 1, y], [x, y + 1], [x, y - 1]);
    }
  };

  // openExit turns every locked tile in the level into a working exit. It
  // reports whether there was anything to open, so a rule can stay quiet when
  // the level has no door.
  LevelState.prototype.openExit = function () {
    var opened = false;
    for (var ch in this.tiles) {
      if (this.tiles[ch].kind === 'locked') {
        this.tiles[ch].kind = 'exit';
        opened = true;
      }
    }
    if (opened) this.dirty = true;
    return opened;
  };

  // --------------------------------------------------------------- entities

  function Entity(proto, x, y, engine) {
    this.proto = proto;
    this.id = proto.id;
    this.kind = proto.kind;
    this.behavior = proto.behavior;
    var dims = engine.sprites.dims(proto.sprite, engine.game.screen.tile);
    this.w = dims.w;
    this.h = dims.h;
    this.x = x;
    this.y = y;
    this.vx = 0;
    this.vy = 0;
    this.hp = proto.health || 1;
    this.alive = true;
    this.grounded = false;
    this.facing = 1;
    this.dir = 1;
    this.frame = 0;
    this.anim = 0;
    this.cooldown = (proto.fires && proto.fires.every) ? proto.fires.every * Math.random() : 0;
    this.age = 0;
    this.homeX = x;
    this.homeY = y;
    this.flash = 0;
    this.invuln = 0;
  }

  Entity.prototype.rect = function () { return { x: this.x, y: this.y, w: this.w, h: this.h }; };
  Entity.prototype.cx = function () { return this.x + this.w / 2; };
  Entity.prototype.cy = function () { return this.y + this.h / 2; };

  // ----------------------------------------------------------------- engine

  function Engine(canvas, game, options) {
    options = options || {};
    this.canvas = canvas;
    this.game = game;
    this.options = options;
    this.sprites = new SpriteBank(game);
    this.audio = new Audio(game);
    this.input = new Input(canvas);

    var screen = game.screen;
    this.buffer = document.createElement('canvas');
    this.buffer.width = screen.width;
    this.buffer.height = screen.height;
    this.bctx = this.buffer.getContext('2d');
    this.bctx.imageSmoothingEnabled = false;

    canvas.width = screen.width * (options.scale || screen.scale || 3);
    canvas.height = screen.height * (options.scale || screen.scale || 3);
    this.ctx = canvas.getContext('2d');
    this.ctx.imageSmoothingEnabled = false;

    this.state = options.skipTitle ? 'playing' : 'title';
    this.score = 0;
    this.lives = game.settings.lives || 3;
    this.vars = {};
    // What the player has found. Abilities belong to the game rather than to
    // the level, which is the whole point of them: a door you could not open
    // an hour ago is a reason to come back here.
    this.inventory = {};
    this.messages = [];
    this.firedOnce = {};
    this.camera = { x: 0, y: 0 };
    this.running = true;
    this.paused = false;
    this.onEvent = options.onEvent || function () {};

    this.loadLevel(options.level || game.settings.startLevel || (game.levels[0] && game.levels[0].id));
    this.camera = this.cameraFor(this.player, 0);
    if (this.state === 'playing') this.audio.music(this.level.def.music);

    var self = this;
    this.last = 0;
    this.frameRequest = null;
    this.loop = function (t) {
      if (!self.running) return;
      var dt = self.last ? Math.min(0.05, (t - self.last) / 1000) : 1 / 60;
      self.last = t;
      self.step(dt);
      self.draw();
      self.input.endFrame();
      self.frameRequest = requestAnimationFrame(self.loop);
    };
    this.frameRequest = requestAnimationFrame(this.loop);
  }

  Engine.prototype.stop = function () {
    this.running = false;
    if (this.frameRequest) cancelAnimationFrame(this.frameRequest);
    this.audio.stopMusic();
    this.input.destroy();
  };

  Engine.prototype.levelDef = function (id) {
    var levels = this.game.levels || [];
    for (var i = 0; i < levels.length; i++) if (levels[i].id === id) return levels[i];
    return levels[0];
  };

  Engine.prototype.loadLevel = function (id) {
    var def = this.levelDef(id);
    if (!def) return;
    this.level = new LevelState(this, def);
    this.entities = [];
    this.player = null;
    var tile = this.game.screen.tile;
    (def.spawns || []).forEach(function (sp) {
      var proto = this.protoOf(sp.object);
      if (!proto) return;
      var e = this.spawn(proto, sp.x * tile, sp.y * tile);
      if (e && sp.dir) e.dir = sp.dir;
    }, this);
    this.levelTime = 0;
    this.timeLeft = def.timeLimit || 0;
    this.checkpoint = this.player ? { x: this.player.x, y: this.player.y } : null;
    this.firedOnce = {};
    this.messages = [];
    this.intro = def.intro ? 2.2 : 0;
    this.runRules({ event: 'start' });
  };

  Engine.prototype.protoOf = function (id) {
    var objects = this.game.objects || [];
    for (var i = 0; i < objects.length; i++) if (objects[i].id === id) return objects[i];
    return null;
  };

  Engine.prototype.spawn = function (proto, x, y) {
    if (typeof proto === 'string') proto = this.protoOf(proto);
    if (!proto) return null;
    var e = new Entity(proto, x, y, this);
    // A sprite taller than one tile is drawn standing on the tile it was
    // placed in, not hanging from it, which is what a level author means when
    // they put a tall enemy on the ground.
    var tile = this.game.screen.tile;
    if (e.h > tile) e.y -= (e.h - tile);
    this.entities.push(e);
    if (proto.kind === 'player' && !this.player) this.player = e;
    return e;
  };

  // ------------------------------------------------------------------ step

  Engine.prototype.step = function (dt) {
    if (this.state === 'title') {
      if (this.input.tapped('start') || this.input.held('jump')) {
        this.audio.resume();
        this.state = 'playing';
        this.audio.music(this.level.def.music);
      }
      return;
    }
    if (this.state === 'gameover' || this.state === 'won') {
      if (this.input.tapped('start')) this.restart();
      return;
    }
    if (this.input.tapped('pause')) this.paused = !this.paused;
    if (this.paused) return;
    if (this.intro > 0) { this.intro -= dt; return; }

    this.levelTime += dt;
    this.level.time += dt;
    if (this.level.shake > 0) this.level.shake = Math.max(0, this.level.shake - dt * 3);

    if (this.timeLeft > 0) {
      this.timeLeft -= dt;
      if (this.timeLeft <= 0) { this.timeLeft = 0; this.killPlayer(); }
    }

    for (var i = 0; i < this.entities.length; i++) {
      var e = this.entities[i];
      if (!e.alive) continue;
      e.age += dt;
      if (e.flash > 0) e.flash -= dt;
      if (e.invuln > 0) e.invuln -= dt;
      this.moveEntity(e, dt);
      this.animate(e, dt);
      this.shoot(e, dt);
    }

    this.collisions();
    this.timers();
    this.camera = this.cameraFor(this.player, dt);
    // On a level that scrolls itself, being left behind by the screen is as
    // fatal as falling: there is nothing back there to go back to.
    if (this.level.def.scroll === 'auto-right' && this.player && this.player.alive &&
        this.player.x + this.player.w < this.camera.x) {
      this.killPlayer();
    }
    this.entities = this.entities.filter(function (e) { return e.alive; });

    for (var m = this.messages.length - 1; m >= 0; m--) {
      this.messages[m].t -= dt;
      if (this.messages[m].t <= 0) this.messages.splice(m, 1);
    }
  };

  Engine.prototype.gravityOf = function (e) {
    if (e.proto.gravity) return e.proto.gravity;
    if (this.level.def.gravity) return this.level.def.gravity;
    return this.game.settings.gravity || 900;
  };

  Engine.prototype.moveEntity = function (e, dt) {
    var speed = e.proto.speed || 60;
    var input = this.input;
    var isPlayer = e === this.player;

    switch (e.behavior) {
      case 'platformer':
        if (isPlayer) {
          var ax = (input.held('right') ? 1 : 0) - (input.held('left') ? 1 : 0);
          e.vx = ax * speed;
          if (ax) e.facing = ax;
          var onLadder = this.tileKindUnder(e, 'ladder');
          if (onLadder) {
            e.vy = ((input.held('down') ? 1 : 0) - (input.held('up') ? 1 : 0)) * speed;
          } else {
            e.vy += this.gravityOf(e) * dt;
            if (input.tapped('jump') && (e.grounded || e.coyote > 0)) {
              // A high jump found halfway through a game has to reach places
              // the ordinary one cannot, or it is not worth finding.
              e.vy = -(e.proto.jump || 330) * (isPlayer && this.has('highjump') ? 1.5 : 1);
              e.coyote = 0;
              this.audio.play('jump');
            }
            // Releasing the jump key early gives a shorter hop. Without this
            // a platformer feels like a lift rather than a jump.
            if (!input.held('jump') && e.vy < -60) e.vy = -60;
          }
        } else {
          e.vy += this.gravityOf(e) * dt;
        }
        break;
      case 'topdown':
        if (isPlayer) {
          var dx = (input.held('right') ? 1 : 0) - (input.held('left') ? 1 : 0);
          var dy = (input.held('down') ? 1 : 0) - (input.held('up') ? 1 : 0);
          if (dx && dy) { dx *= 0.7071; dy *= 0.7071; }
          e.vx = dx * speed; e.vy = dy * speed;
          if (dx) e.facing = sign(dx);
        }
        break;
      case 'ship':
        if (isPlayer) {
          e.vx = ((input.held('right') ? 1 : 0) - (input.held('left') ? 1 : 0)) * speed;
          e.vy = ((input.held('down') ? 1 : 0) - (input.held('up') ? 1 : 0)) * speed;
        }
        break;
      case 'paddle':
        if (isPlayer) {
          e.vx = ((input.held('right') ? 1 : 0) - (input.held('left') ? 1 : 0)) * speed;
          e.vy = 0;
        }
        break;
      case 'patrol':
        e.vx = e.dir * speed;
        e.vy += this.gravityOf(e) * dt;
        break;
      case 'chase':
      case 'flee':
        if (this.player) {
          var tx = this.player.cx() - e.cx(), ty = this.player.cy() - e.cy();
          var dist = Math.sqrt(tx * tx + ty * ty) || 1;
          var reach = e.proto.range || 0;
          if (!reach || dist < reach) {
            var s = e.behavior === 'chase' ? 1 : -1;
            e.vx = (tx / dist) * speed * s;
            e.vy = (ty / dist) * speed * s;
          } else { e.vx = 0; e.vy = 0; }
          e.facing = sign(e.vx) || e.facing;
        }
        break;
      case 'float':
        e.vx = e.dir * speed;
        e.vy = Math.cos(e.age * 2.4) * speed * 0.6;
        break;
      case 'fall':
        e.vy += this.gravityOf(e) * dt;
        break;
      case 'bounce':
        if (!e.launched) {
          e.vx = speed * (e.dir || 1) * 0.7;
          e.vy = -speed;
          e.launched = true;
        }
        break;
      case 'orbit':
        var r = e.proto.range || 32;
        var w = speed / Math.max(8, r);
        e.x = e.homeX + Math.cos(e.age * w) * r;
        e.y = e.homeY + Math.sin(e.age * w) * r;
        return;
      case 'projectile':
        // velocity was set when it was fired
        break;
      case 'static':
      default:
        return;
    }

    var tile = this.game.screen.tile;
    var maxFall = 1200;
    if (e.vy > maxFall) e.vy = maxFall;

    var wasGrounded = e.grounded;
    e.grounded = false;
    this.moveAxis(e, e.vx * dt, 0);
    this.moveAxis(e, e.vy * dt, 1);
    if (wasGrounded && !e.grounded && e.behavior === 'platformer') {
      e.coyote = 0.09; // a fraction of a second of forgiveness after a ledge
    } else if (e.grounded) {
      e.coyote = 0.09;
    } else if (e.coyote > 0) {
      e.coyote -= dt;
    }

    // Patrolling enemies turn at a wall, and at a ledge so they do not walk
    // into the pit the level author dug for the player.
    if (e.behavior === 'patrol') {
      if (e.bumped) { e.dir = -e.dir; e.bumped = false; }
      else if (e.grounded) {
        var aheadX = e.dir > 0 ? e.x + e.w + 1 : e.x - 1;
        var footY = e.y + e.h + 1;
        if (!this.solidPixel(aheadX, footY) && !this.level.platformAt(Math.floor(aheadX / tile), Math.floor(footY / tile))) {
          e.dir = -e.dir;
        }
      }
      e.facing = e.dir;
    }
    if (e.behavior === 'float' && e.bumped) { e.dir = -e.dir; e.bumped = false; }

    if (e.behavior === 'ship' || e.behavior === 'paddle') {
      // Ships and paddles are held on screen rather than in the world.
      e.x = clamp(e.x, this.camera.x, this.camera.x + this.game.screen.width - e.w);
      e.y = clamp(e.y, 0, Math.max(0, this.level.height * tile - e.h));
    }

    // Anything that leaves the world below it is gone; for the player that is
    // a death, which is the oldest rule in the genre.
    if (e.y > this.level.height * tile + 64) {
      if (e === this.player) this.killPlayer();
      else this.destroy(e);
      return;
    }
    // A shot that flies off any other edge is gone too. Without this a
    // shooter accumulates every bullet ever fired.
    if (e.kind === 'projectile' || e.behavior === 'projectile') {
      if (e.x < -64 || e.x > this.level.width * tile + 64 || e.y < -64) this.destroy(e);
    }
  };

  Engine.prototype.solidPixel = function (px, py) {
    var tile = this.game.screen.tile;
    return this.level.solidAt(Math.floor(px / tile), Math.floor(py / tile));
  };

  // moveAxis moves one axis at a time and pushes the entity back out of
  // whatever it entered. Resolving axes separately is what stops an entity
  // catching on the seam between two floor tiles.
  Engine.prototype.moveAxis = function (e, delta, axis) {
    if (!delta) return;
    var tile = this.game.screen.tile;
    if (axis === 0) e.x += delta; else e.y += delta;

    var x0 = Math.floor(e.x / tile), x1 = Math.floor((e.x + e.w - 1) / tile);
    var y0 = Math.floor(e.y / tile), y1 = Math.floor((e.y + e.h - 1) / tile);

    for (var ty = y0; ty <= y1; ty++) {
      for (var tx = x0; tx <= x1; tx++) {
        var solid = this.level.solidAt(tx, ty);
        var oneWay = false;
        if (!solid && axis === 1 && delta > 0 && this.level.platformAt(tx, ty)) {
          // A one-way platform only stops a fall that started above it.
          var top = ty * tile;
          if (e.y + e.h - delta <= top + 1) { solid = true; oneWay = true; }
        }
        if (!solid) continue;
        if (this.level.kindAt(tx, ty) === 'door' && this.openDoor(tx, ty, e)) {
          if (e.kind === 'projectile' || e.behavior === 'projectile') this.destroy(e);
          continue;
        }
        if (axis === 0) {
          if (delta > 0) e.x = tx * tile - e.w; else e.x = (tx + 1) * tile;
          e.vx = e.behavior === 'bounce' ? -e.vx : 0;
          e.bumped = true;
          if (e.behavior === 'projectile') this.destroy(e);
          if (e.kind === 'projectile' || e.behavior === 'projectile') this.breakTile(tx, ty, e);
          return;
        }
        if (delta > 0) {
          e.y = ty * tile - e.h;
          e.grounded = true;
          if (e.behavior === 'bounce') { e.vy = -Math.abs(e.vy); return; }
          e.vy = 0;
        } else {
          e.y = (ty + 1) * tile;
          if (e.behavior === 'bounce') { e.vy = Math.abs(e.vy); return; }
          e.vy = 0;
          if (!oneWay) this.breakTile(tx, ty, e);
        }
        if (e.behavior === 'projectile') this.destroy(e);
        return;
      }
    }
  };

  // openDoor opens a tile the player has the key for. Doors that want missiles
  // are opened by shooting them, which is why the projectile counts as the
  // player: it is the player's shot.
  Engine.prototype.openDoor = function (tx, ty, by) {
    var def = this.level.at(tx, ty);
    if (!def || def.kind !== 'door') return false;
    var mine = by === this.player || by.owner === 'player';
    if (!mine) return false;
    var needs = def.needs || '';
    var byShot = by.kind === 'projectile' || by.behavior === 'projectile';
    if (needs === 'missiles' && !byShot) return false; // shoot it, do not walk into it
    if (!this.has(needs)) {
      if (by === this.player && !this.doorTold) {
        this.doorTold = true;
        var self = this;
        setTimeout(function () { self.doorTold = false; }, 2000);
        this.messages.push({ text: 'Locked — you need the ' + (def.name || needs), t: 2 });
      }
      return false;
    }
    // The whole door opens, not the tile that was hit: a hole shot through
    // one square of a seven-square door is not what anybody meant.
    this.level.openRun(tx, ty);
    this.audio.play('break');
    this.messages.push({ text: (def.name || 'The door') + ' opens', t: 2 });
    return true;
  };

  Engine.prototype.breakTile = function (tx, ty, by) {
    if (this.level.kindAt(tx, ty) !== 'breakable') return;
    if (by !== this.player && by.kind !== 'projectile' && by.behavior !== 'projectile') return;
    this.level.setTile(tx, ty, '.');
    this.audio.play('break');
    this.score += 5;
  };

  Engine.prototype.tileKindUnder = function (e, kind) {
    var tile = this.game.screen.tile;
    var tx = Math.floor(e.cx() / tile), ty = Math.floor(e.cy() / tile);
    return this.level.kindAt(tx, ty) === kind;
  };

  Engine.prototype.animate = function (e, dt) {
    var fps = this.sprites.fps[e.proto.sprite] || 0;
    if (!fps) { e.frame = 0; return; }
    var moving = Math.abs(e.vx) > 4 || Math.abs(e.vy) > 4 || e.behavior === 'static';
    if (!moving && e.behavior === 'platformer') { e.frame = 0; return; }
    e.anim += dt * fps;
    e.frame = Math.floor(e.anim);
  };

  Engine.prototype.shoot = function (e, dt) {
    var f = e.proto.fires;
    if (!f || !f.object) return;
    // A weapon can be something the player finds rather than starts with.
    if (e.proto.firesNeeds && !this.has(e.proto.firesNeeds)) return;
    // The cooldown runs down for everything that shoots; only what starts a
    // shot differs — a key for the player, the clock for everyone else.
    if (e.cooldown > 0) e.cooldown -= dt;
    if (e === this.player) {
      if (!this.input.tapped('fire') || e.cooldown > 0) return;
      e.cooldown = f.every || 0.25;
    } else {
      if (e.cooldown > 0) return;
      e.cooldown = f.every || 1.5;
    }
    var shot = this.spawn(f.object, e.cx(), e.cy());
    if (!shot) return;
    shot.x -= shot.w / 2; shot.y -= shot.h / 2;
    shot.owner = e === this.player ? 'player' : 'enemy';
    var speed = f.speed || 180;
    switch (f.aim) {
      case 'player':
        if (this.player && e !== this.player) {
          var dx = this.player.cx() - e.cx(), dy = this.player.cy() - e.cy();
          var d = Math.sqrt(dx * dx + dy * dy) || 1;
          shot.vx = dx / d * speed; shot.vy = dy / d * speed;
        } else { shot.vx = e.facing * speed; }
        break;
      case 'down': shot.vx = 0; shot.vy = speed; break;
      case 'up': shot.vx = 0; shot.vy = -speed; break;
      default: shot.vx = (e.facing || 1) * speed; shot.vy = 0;
    }
    this.audio.play(f.sound || 'shoot');
  };

  // ------------------------------------------------------------- collisions

  Engine.prototype.collisions = function () {
    var p = this.player;
    var tile = this.game.screen.tile;
    if (p && p.alive) {
      // Tiles the player is standing in, rather than on: spikes, water, the
      // exit and checkpoints all work this way.
      var tx0 = Math.floor((p.x + 2) / tile), tx1 = Math.floor((p.x + p.w - 3) / tile);
      var ty0 = Math.floor((p.y + 2) / tile), ty1 = Math.floor((p.y + p.h - 3) / tile);
      for (var ty = ty0; ty <= ty1; ty++) {
        for (var tx = tx0; tx <= tx1; tx++) {
          var kind = this.level.kindAt(tx, ty);
          if (kind === 'hazard') { this.hurtPlayer(1); }
          else if (kind === 'exit') { this.runRules({ event: 'levelclear' }); this.nextLevel(); return; }
          else if (kind === 'checkpoint') { this.checkpoint = { x: tx * tile, y: ty * tile }; }
          if (kind !== 'empty') this.runRules({ event: 'tile', a: kind });
        }
      }
    }

    for (var i = 0; i < this.entities.length; i++) {
      var a = this.entities[i];
      if (!a.alive) continue;
      for (var j = i + 1; j < this.entities.length; j++) {
        var b = this.entities[j];
        if (!b.alive) continue;
        if (!overlaps(a.rect(), b.rect())) continue;
        this.touch(a, b);
        this.touch(b, a);
        this.runRules({ event: 'collide', a: a.id, b: b.id });
        this.runRules({ event: 'collide', a: b.id, b: a.id });
      }
    }
  };

  // touch applies the built-in meaning of two things meeting. Rules run as
  // well, and can add to this; they cannot be required for a game to make
  // sense, which is what keeps a freshly generated game playable.
  Engine.prototype.touch = function (a, b) {
    var p = this.player;
    if (a === p) {
      if (b.kind === 'item' || b.proto.collect) {
        this.collect(b);
        return;
      }
      if (b.kind === 'goal') {
        this.runRules({ event: 'levelclear' });
        this.nextLevel();
        return;
      }
      if (b.kind === 'enemy' || b.kind === 'hazard') {
        // Landing on an enemy from above beats it, when the game says so.
        if (b.proto.stompable && a.vy > 40 && a.y + a.h - 8 <= b.y) {
          this.damage(b, 1, a);
          a.vy = -(a.proto.jump || 300) * 0.6;
          return;
        }
        this.hurtPlayer(b.proto.damage || 1);
        return;
      }
      if (b.kind === 'projectile' && b.owner === 'enemy') {
        this.hurtPlayer(b.proto.damage || 1);
        this.destroy(b);
        return;
      }
    }
    if (a.kind === 'projectile') {
      if (a.owner === 'player' && (b.kind === 'enemy' || b.proto.fragile)) {
        this.damage(b, a.proto.damage || 1, a);
        this.destroy(a);
      } else if (b.kind === 'block' && b.proto.solid) {
        this.destroy(a);
      }
    }
    if (a.behavior === 'bounce' && b === p) {
      // A ball bouncing off the paddle takes its angle from where it hit,
      // which is the whole skill of a bat-and-ball game.
      var offset = (a.cx() - b.cx()) / (b.w / 2);
      var speed = Math.sqrt(a.vx * a.vx + a.vy * a.vy) || (a.proto.speed || 140);
      a.vx = clamp(offset, -1, 1) * speed * 0.8;
      a.vy = -Math.abs(a.vy || speed);
      a.y = b.y - a.h - 1;
      this.audio.play('step');
    }
    if (a.behavior === 'bounce' && (b.kind === 'block' || b.proto.fragile)) {
      a.vy = -a.vy;
      this.damage(b, 1, a);
    }
  };

  Engine.prototype.collect = function (e) {
    this.score += e.proto.points || 10;
    this.audio.play(e.proto.sound || 'coin');
    // Picking something up can hand the player an ability that lasts the rest
    // of the game, which is what an exploration map is built out of.
    if (e.proto.grants) this.grant(e.proto.grants, e.proto.announce || e.proto.name);
    e.alive = false;
    this.runRules({ event: 'collect', a: e.id });
    this.runRules({ event: 'destroy', a: e.id });
    this.checkCounts(e.id);
    this.checkScore();
  };

  // grant hands the player an ability and says so. An ability nobody is told
  // about is a bug report waiting to happen.
  Engine.prototype.grant = function (name, announce) {
    if (!name || this.inventory[name]) return;
    this.inventory[name] = true;
    this.audio.play('win');
    this.messages.push({ text: announce || name, t: 3 });
    this.runRules({ event: 'grant', a: name });
  };

  Engine.prototype.has = function (name) {
    return !name || this.inventory[name] === true;
  };

  Engine.prototype.damage = function (e, amount, by) {
    if (e === this.player) { this.hurtPlayer(amount); return; }
    e.hp -= amount;
    e.flash = 0.12;
    if (e.hp > 0) return;
    this.score += e.proto.points || 0;
    this.audio.play(e.proto.sound || 'hit');
    if (e.proto.drops) this.spawn(e.proto.drops, e.x, e.y);
    e.alive = false;
    this.runRules({ event: 'destroy', a: e.id });
    this.checkCounts(e.id);
    this.checkScore();
  };

  Engine.prototype.destroy = function (e) {
    if (!e.alive) return;
    e.alive = false;
    this.runRules({ event: 'destroy', a: e.id });
    this.checkCounts(e.id);
  };

  Engine.prototype.hurtPlayer = function (amount) {
    var p = this.player;
    if (!p || !p.alive || p.invuln > 0) return;
    p.hp -= amount;
    p.invuln = 1.1;
    p.flash = 0.3;
    this.level.shake = 1;
    this.audio.play('hit');
    this.runRules({ event: 'health', value: p.hp });
    if (p.hp <= 0) this.killPlayer();
  };

  Engine.prototype.killPlayer = function () {
    var p = this.player;
    if (!p) return;
    this.lives -= 1;
    this.audio.play('lose');
    if (this.lives <= 0) {
      this.state = 'gameover';
      this.audio.stopMusic();
      this.onEvent({ type: 'gameover', score: this.score });
      return;
    }
    // Respawning rebuilds the level so anything already collected comes back
    // with it; a half-finished level with a dead player is not a state anyone
    // wants to resume into.
    var keep = this.lives, score = this.score;
    this.loadLevel(this.level.def.id);
    this.lives = keep;
    this.score = score;
    if (this.checkpoint && this.player && this.game.settings.respawn === 'checkpoint') {
      this.player.x = this.checkpoint.x;
      this.player.y = this.checkpoint.y;
    }
  };

  Engine.prototype.nextLevel = function () {
    var next = this.level.def.next;
    this.audio.play('win');
    if (!next || next === 'win') { this.win(); return; }
    var def = this.levelDef(next);
    if (!def || def.id === this.level.def.id) { this.win(); return; }
    var keepScore = this.score, keepLives = this.lives;
    this.loadLevel(next);
    this.score = keepScore;
    this.lives = keepLives;
    this.audio.music(this.level.def.music);
    this.onEvent({ type: 'level', level: next });
  };

  Engine.prototype.win = function () {
    this.state = 'won';
    this.audio.stopMusic();
    this.audio.play('win');
    this.onEvent({ type: 'win', score: this.score });
  };

  Engine.prototype.restart = function () {
    this.score = 0;
    this.lives = this.game.settings.lives || 3;
    this.vars = {};
    this.inventory = {};
    this.state = 'playing';
    this.loadLevel(this.game.settings.startLevel || this.game.levels[0].id);
    this.audio.music(this.level.def.music);
  };

  Engine.prototype.countOf = function (id) {
    var n = 0;
    for (var i = 0; i < this.entities.length; i++) {
      if (this.entities[i].alive && this.entities[i].id === id) n++;
    }
    return n;
  };

  Engine.prototype.checkCounts = function (id) {
    if (this.countOf(id) === 0) this.runRules({ event: 'count', a: id, value: 0 });
  };

  Engine.prototype.checkScore = function () {
    this.runRules({ event: 'score', value: this.score });
    var goal = this.game.settings.scoreGoal;
    if (goal && this.score >= goal) this.win();
  };

  Engine.prototype.timers = function () {
    var rules = this.game.rules || [];
    for (var i = 0; i < rules.length; i++) {
      var r = rules[i];
      if (r.when.event !== 'timer') continue;
      var key = 'timer:' + (r.id || i);
      if (this.levelTime >= (r.when.value || 0) && !this.firedOnce[key]) {
        this.firedOnce[key] = true;
        this.apply(r);
      }
    }
    if (this.input.tapped('fire') || this.input.tapped('jump')) {
      this.runRules({ event: 'key', key: this.input.tapped('fire') ? 'fire' : 'jump' });
    }
  };

  // ------------------------------------------------------------------ rules

  Engine.prototype.runRules = function (ev) {
    var rules = this.game.rules || [];
    for (var i = 0; i < rules.length; i++) {
      var r = rules[i];
      if (!r.when || r.when.event !== ev.event) continue;
      if (r.level && r.level !== this.level.def.id) continue;
      if (!this.matches(r.when, ev)) continue;
      if (r.once) {
        var key = 'rule:' + (r.id || i);
        if (this.firedOnce[key]) continue;
        this.firedOnce[key] = true;
      }
      this.apply(r);
    }
  };

  Engine.prototype.matches = function (when, ev) {
    switch (when.event) {
      case 'collide':
        return (!when.a || when.a === ev.a) && (!when.b || when.b === ev.b);
      case 'collect':
      case 'destroy':
      case 'tile':
      case 'grant':
        return !when.a || when.a === ev.a;
      case 'count':
        return when.a === ev.a;
      case 'key':
        return !when.key || when.key === ev.key;
      case 'score':
        return this.score >= (when.value || 0);
      case 'health':
        return this.player && this.player.hp <= (when.value || 0);
      default:
        return true;
    }
  };

  Engine.prototype.apply = function (rule) {
    var actions = rule.do || [];
    for (var i = 0; i < actions.length; i++) {
      var a = actions[i];
      switch (a.action) {
        case 'score': this.score += a.value || 0; this.checkScore(); break;
        case 'health':
          if (this.player) this.player.hp = clamp(this.player.hp + (a.value || 0), 0, 99);
          break;
        case 'lives':
          this.lives += a.value || 0;
          if (this.lives <= 0) {
            this.state = 'gameover';
            this.audio.stopMusic();
            this.audio.play('lose');
            this.onEvent({ type: 'gameover', score: this.score });
            return;
          }
          break;
        case 'spawn':
          var at = this.player ? { x: this.player.x, y: this.player.y - 32 } : { x: 32, y: 32 };
          var tile = this.game.screen.tile;
          this.spawn(a.target, a.value ? a.value * tile : at.x, at.y);
          break;
        case 'destroy':
          this.entities.forEach(function (e) { if (e.id === a.target) e.alive = false; });
          break;
        case 'goto': this.loadLevel(a.target); this.audio.music(this.level.def.music); break;
        case 'win': this.win(); break;
        case 'lose': this.lives = 0; this.killPlayer(); break;
        case 'message': this.messages.push({ text: a.text || '', t: a.value || 2.5 }); break;
        case 'sound': this.audio.play(a.target); break;
        case 'grant': this.grant(a.target, a.text); break;
        case 'setvar': this.vars[a.target] = a.value || 0; break;
        case 'addvar': this.vars[a.target] = (this.vars[a.target] || 0) + (a.value || 0); break;
        case 'speed':
          if (this.player) this.player.proto.speed = Math.max(10, (this.player.proto.speed || 60) + (a.value || 0));
          break;
        case 'shake': this.level.shake = a.value || 1; break;
        case 'teleport':
          if (this.player) {
            var t = this.game.screen.tile;
            this.player.x = (a.value || 1) * t;
            this.player.y = 0;
          }
          break;
        case 'openexit':
          // A locked door is drawn from the start and does nothing until a
          // rule opens it, which is how "collect everything to get out" works
          // without any per-game code.
          if (this.level.openExit()) {
            this.messages.push({ text: a.text || 'The way out is open!', t: 2.5 });
            this.audio.play(a.target || 'win');
          }
          break;
      }
    }
  };

  // ----------------------------------------------------------------- camera

  Engine.prototype.cameraFor = function (p, dt) {
    var sw = this.game.screen.width, sh = this.game.screen.height;
    var tile = this.game.screen.tile;
    var lw = this.level.width * tile, lh = this.level.height * tile;
    var scroll = this.level.def.scroll || 'follow';
    var cam = { x: this.camera.x, y: this.camera.y };
    if (scroll === 'fixed') {
      cam.x = 0; cam.y = 0;
    } else if (scroll === 'auto-right') {
      cam.x += (this.game.screen.width / 5) * (dt || 1 / 60);
    } else if (p) {
      cam.x = p.cx() - sw / 2;
      cam.y = p.cy() - sh / 2;
    }
    cam.x = clamp(cam.x, 0, Math.max(0, lw - sw));
    cam.y = clamp(cam.y, 0, Math.max(0, lh - sh));
    if (lh <= sh) cam.y = 0;
    return cam;
  };

  // ------------------------------------------------------------------ draw

  Engine.prototype.draw = function () {
    var g = this.bctx;
    var sw = this.game.screen.width, sh = this.game.screen.height;
    g.fillStyle = this.level.def.background || this.game.palette[15] || '#000';
    g.fillRect(0, 0, sw, sh);

    var shakeX = 0, shakeY = 0;
    if (this.level.shake > 0) {
      shakeX = (Math.random() - 0.5) * 4 * this.level.shake;
      shakeY = (Math.random() - 0.5) * 4 * this.level.shake;
    }
    var camX = Math.round(this.camera.x + shakeX), camY = Math.round(this.camera.y + shakeY);

    this.drawTiles(g, camX, camY);

    for (var i = 0; i < this.entities.length; i++) {
      var e = this.entities[i];
      if (!e.alive) continue;
      if (e.invuln > 0 && Math.floor(e.invuln * 20) % 2 === 0) continue;
      var img = this.sprites.get(e.proto.sprite, e.frame);
      var x = Math.round(e.x - camX), y = Math.round(e.y - camY);
      if (x + e.w < 0 || x > sw || y + e.h < 0 || y > sh) continue;
      if (!img) {
        g.fillStyle = this.game.palette[4] || '#f00';
        g.fillRect(x, y, e.w, e.h);
        continue;
      }
      if (e.facing < 0) {
        g.save();
        g.translate(x + e.w, y);
        g.scale(-1, 1);
        g.drawImage(img, 0, 0);
        g.restore();
      } else {
        g.drawImage(img, x, y);
      }
      if (e.flash > 0) {
        g.globalAlpha = 0.6;
        g.fillStyle = '#ffffff';
        g.fillRect(x, y, e.w, e.h);
        g.globalAlpha = 1;
      }
    }

    this.drawHUD(g);
    if (this.state === 'title') this.drawTitle(g);
    if (this.state === 'gameover') this.drawBanner(g, 'GAME OVER', 'Score ' + this.score + ' — press Enter');
    if (this.state === 'won') this.drawBanner(g, this.game.settings.ending || 'YOU WIN!', 'Score ' + this.score + ' — press Enter');
    if (this.paused) this.drawBanner(g, 'PAUSED', 'press P');
    if (this.intro > 0 && this.level.def.intro) this.drawBanner(g, this.level.def.name || '', this.level.def.intro);

    this.ctx.imageSmoothingEnabled = false;
    this.ctx.drawImage(this.buffer, 0, 0, this.canvas.width, this.canvas.height);
  };

  Engine.prototype.drawTiles = function (g, camX, camY) {
    var tile = this.game.screen.tile;
    var sw = this.game.screen.width, sh = this.game.screen.height;
    var x0 = Math.floor(camX / tile), x1 = Math.ceil((camX + sw) / tile);
    var y0 = Math.floor(camY / tile), y1 = Math.ceil((camY + sh) / tile);
    for (var ty = y0; ty <= y1; ty++) {
      for (var tx = x0; tx <= x1; tx++) {
        var t = this.level.at(tx, ty);
        if (!t || t.kind === 'empty') continue;
        var px = tx * tile - camX, py = ty * tile - camY;
        var img = t.sprite ? this.sprites.get(t.sprite, 0) : null;
        if (img) {
          g.drawImage(img, Math.round(px), Math.round(py));
        } else {
          g.fillStyle = this.game.palette[2] || '#888';
          g.fillRect(Math.round(px), Math.round(py), tile, tile);
        }
      }
    }
  };

  // The HUD is drawn with the engine's own 3x5 pixel font rather than the
  // browser's, so an exported game looks the same on every machine.
  var FONT = {
    '0': ['111', '101', '101', '101', '111'], '1': ['010', '110', '010', '010', '111'],
    '2': ['111', '001', '111', '100', '111'], '3': ['111', '001', '111', '001', '111'],
    '4': ['101', '101', '111', '001', '001'], '5': ['111', '100', '111', '001', '111'],
    '6': ['111', '100', '111', '101', '111'], '7': ['111', '001', '010', '010', '010'],
    '8': ['111', '101', '111', '101', '111'], '9': ['111', '101', '111', '001', '111'],
    'A': ['111', '101', '111', '101', '101'], 'B': ['110', '101', '110', '101', '110'],
    'C': ['111', '100', '100', '100', '111'], 'D': ['110', '101', '101', '101', '110'],
    'E': ['111', '100', '111', '100', '111'], 'F': ['111', '100', '111', '100', '100'],
    'G': ['111', '100', '101', '101', '111'], 'H': ['101', '101', '111', '101', '101'],
    'I': ['111', '010', '010', '010', '111'], 'J': ['001', '001', '001', '101', '111'],
    'K': ['101', '101', '110', '101', '101'], 'L': ['100', '100', '100', '100', '111'],
    'M': ['101', '111', '111', '101', '101'], 'N': ['101', '111', '111', '111', '101'],
    'O': ['111', '101', '101', '101', '111'], 'P': ['111', '101', '111', '100', '100'],
    'Q': ['111', '101', '101', '111', '011'], 'R': ['111', '101', '111', '110', '101'],
    'S': ['111', '100', '111', '001', '111'], 'T': ['111', '010', '010', '010', '010'],
    'U': ['101', '101', '101', '101', '111'], 'V': ['101', '101', '101', '101', '010'],
    'W': ['101', '101', '111', '111', '101'], 'X': ['101', '101', '010', '101', '101'],
    'Y': ['101', '101', '010', '010', '010'], 'Z': ['111', '001', '010', '100', '111'],
    '.': ['000', '000', '000', '000', '010'], ',': ['000', '000', '000', '010', '100'],
    '!': ['010', '010', '010', '000', '010'], '?': ['111', '001', '011', '000', '010'],
    ':': ['000', '010', '000', '010', '000'], '-': ['000', '000', '111', '000', '000'],
    '+': ['000', '010', '111', '010', '000'], '/': ['001', '001', '010', '100', '100'],
    "'": ['010', '010', '000', '000', '000'], '(': ['001', '010', '010', '010', '001'],
    ')': ['100', '010', '010', '010', '100'], '%': ['101', '001', '010', '100', '101'],
    '=': ['000', '111', '000', '111', '000'], '<': ['001', '010', '100', '010', '001'],
    '>': ['100', '010', '001', '010', '100'], '*': ['101', '010', '101', '000', '000'],
    ' ': ['000', '000', '000', '000', '000']
  };

  function textWidth(s) { return s.length * 4 - 1; }

  Engine.prototype.text = function (g, s, x, y, color) {
    s = String(s).toUpperCase();
    g.fillStyle = color || '#ffffff';
    for (var i = 0; i < s.length; i++) {
      var glyph = FONT[s[i]] || FONT['?'];
      for (var r = 0; r < glyph.length; r++) {
        for (var c = 0; c < 3; c++) {
          if (glyph[r][c] === '1') g.fillRect(x + i * 4 + c, y + r, 1, 1);
        }
      }
    }
  };

  Engine.prototype.textCentred = function (g, s, y, color) {
    this.text(g, s, Math.round((this.game.screen.width - textWidth(String(s))) / 2), y, color);
  };

  Engine.prototype.drawHUD = function (g) {
    var s = this.game.settings;
    var pal = this.game.palette;
    var y = 4;
    if (s.showScore !== false) this.text(g, 'SCORE ' + this.score, 4, y, pal[1] || '#fff');
    if (s.showLives !== false) {
      var lives = 'LIVES ' + Math.max(0, this.lives);
      this.text(g, lives, this.game.screen.width - textWidth(lives) - 4, y, pal[1] || '#fff');
    }
    if (this.player && this.player.proto.health > 1) {
      // A row of hearts is how a player reads three or five; past that it is a
      // wall of pixels across the screen, so it becomes a number. A design can
      // hand out twenty hit points and the HUD still fits.
      var hp = Math.max(0, this.player.hp);
      var hearts = Math.min(hp, 6);
      for (var i = 0; i < hearts; i++) this.heart(g, 4 + i * 8, y + 8, pal[4] || '#f55');
      if (hp > hearts) this.text(g, 'x' + hp, 4 + hearts * 8 + 2, y + 8, pal[4] || '#f55');
    }
    if (this.timeLeft > 0) {
      var t = 'TIME ' + Math.ceil(this.timeLeft);
      this.text(g, t, Math.round((this.game.screen.width - textWidth(t)) / 2), y, pal[6] || '#fd5');
    }
    var found = [];
    for (var item in this.inventory) if (this.inventory[item]) found.push(item);
    if (found.length) {
      // What has been found belongs under the lives, on its own line, where it
      // cannot run into the health.
      var label = found.join(' ');
      var width = textWidth(label);
      var room = this.game.screen.width - 8;
      while (width > room && found.length > 1) {
        found.shift();
        label = '.. ' + found.join(' ');
        width = textWidth(label);
      }
      this.text(g, label, this.game.screen.width - width - 4, y + 8, pal[9] || '#7fd');
    }
    for (var m = 0; m < this.messages.length; m++) {
      this.textCentred(g, this.messages[m].text, this.game.screen.height - 20 - m * 8, pal[6] || '#fd5');
    }
  };

  // A five-by-five heart, drawn rather than written: a row of asterisks reads
  // as damage taken, not as health left.
  var HEART = ['01010', '11111', '11111', '01110', '00100'];

  Engine.prototype.heart = function (g, x, y, color) {
    g.fillStyle = color;
    for (var r = 0; r < HEART.length; r++) {
      for (var c = 0; c < HEART[r].length; c++) {
        if (HEART[r][c] === '1') g.fillRect(x + c, y + r, 1, 1);
      }
    }
  };

  Engine.prototype.drawBanner = function (g, title, sub) {
    var sw = this.game.screen.width, sh = this.game.screen.height;
    g.globalAlpha = 0.72;
    g.fillStyle = '#000000';
    g.fillRect(0, sh / 2 - 26, sw, 52);
    g.globalAlpha = 1;
    this.textCentred(g, title, sh / 2 - 14, this.game.palette[6] || '#fd5');
    if (sub) this.textCentred(g, sub, sh / 2 + 4, this.game.palette[1] || '#fff');
  };

  Engine.prototype.drawTitle = function (g) {
    var sw = this.game.screen.width, sh = this.game.screen.height;
    g.globalAlpha = 0.82;
    g.fillStyle = '#000000';
    g.fillRect(0, 0, sw, sh);
    g.globalAlpha = 1;
    var name = this.game.settings.title || this.game.name;
    this.textCentred(g, name, sh / 2 - 26, this.game.palette[6] || '#fd5');
    if (this.game.tagline) this.textCentred(g, this.game.tagline, sh / 2 - 10, this.game.palette[1] || '#fff');
    this.textCentred(g, this.game.settings.hint || 'ARROWS MOVE   SPACE JUMP   X FIRE', sh / 2 + 10, this.game.palette[3] || '#aaa');
    if (Math.floor(Date.now() / 500) % 2 === 0) {
      this.textCentred(g, 'PRESS ENTER', sh / 2 + 28, this.game.palette[1] || '#fff');
    }
  };

  // ---------------------------------------------------------------- exports

  global.OliverEngine = {
    start: function (canvas, game, options) {
      return new Engine(canvas, game, options || {});
    },
    version: 1
  };
})(typeof window !== 'undefined' ? window : this);
