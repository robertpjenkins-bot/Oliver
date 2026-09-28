// Drawing on a grid of characters.
//
// A level map and a sprite frame are the same thing: an array of strings, all
// the same length, one character per cell. A level's characters name tiles and
// a sprite's name colours, but nothing here cares which — so one set of
// operations serves both editors, and a tool written once works in both.
//
// Every operation takes rows and returns new rows. Nothing is edited in place,
// which is what makes undo a matter of keeping the old array rather than
// remembering what each tool did.
(function (root) {
  'use strict';

  function height(rows) { return rows.length; }
  function width(rows) { return rows.length ? rows[0].length : 0; }

  function at(rows, x, y) {
    if (y < 0 || y >= rows.length) return '';
    if (x < 0 || x >= rows[y].length) return '';
    return rows[y][x];
  }

  // set writes one cell, and is the only place a row string is rebuilt.
  function set(rows, x, y, ch) {
    if (y < 0 || y >= rows.length) return rows;
    if (x < 0 || x >= rows[y].length) return rows;
    var out = rows.slice();
    out[y] = rows[y].slice(0, x) + ch + rows[y].slice(x + 1);
    return out;
  }

  // plot writes many cells at once. Tools work out which cells they want and
  // hand the list over, so a stroke of any shape costs one pass.
  function plot(rows, points, ch) {
    if (!points.length) return rows;
    var byRow = {};
    points.forEach(function (p) {
      if (p.y < 0 || p.y >= rows.length) return;
      if (p.x < 0 || p.x >= rows[p.y].length) return;
      (byRow[p.y] = byRow[p.y] || []).push(p.x);
    });
    var out = rows.slice();
    Object.keys(byRow).forEach(function (y) {
      var chars = out[y].split('');
      byRow[y].forEach(function (x) { chars[x] = ch; });
      out[y] = chars.join('');
    });
    return out;
  }

  // line is Bresenham, which is the right answer here for the same reason it
  // was in 1965: every cell it picks is a cell, and none is picked twice.
  function line(x0, y0, x1, y1) {
    var points = [];
    var dx = Math.abs(x1 - x0), dy = Math.abs(y1 - y0);
    var sx = x0 < x1 ? 1 : -1, sy = y0 < y1 ? 1 : -1;
    var err = dx - dy;
    for (;;) {
      points.push({ x: x0, y: y0 });
      if (x0 === x1 && y0 === y1) break;
      var e2 = 2 * err;
      if (e2 > -dy) { err -= dy; x0 += sx; }
      if (e2 < dx) { err += dx; y0 += sy; }
    }
    return points;
  }

  function rect(x0, y0, x1, y1, filled) {
    var lox = Math.min(x0, x1), hix = Math.max(x0, x1);
    var loy = Math.min(y0, y1), hiy = Math.max(y0, y1);
    var points = [];
    for (var y = loy; y <= hiy; y++) {
      for (var x = lox; x <= hix; x++) {
        if (filled || y === loy || y === hiy || x === lox || x === hix) {
          points.push({ x: x, y: y });
        }
      }
    }
    return points;
  }

  // ellipse draws the oval that fits the dragged box. Circles are tedious to
  // paint by hand and common in what people draw, so the tool earns its place.
  function ellipse(x0, y0, x1, y1, filled) {
    var lox = Math.min(x0, x1), hix = Math.max(x0, x1);
    var loy = Math.min(y0, y1), hiy = Math.max(y0, y1);
    var cx = (lox + hix) / 2, cy = (loy + hiy) / 2;
    var rx = Math.max(0.5, (hix - lox) / 2), ry = Math.max(0.5, (hiy - loy) / 2);
    var points = [], seen = {};
    for (var y = loy; y <= hiy; y++) {
      for (var x = lox; x <= hix; x++) {
        var nx = (x - cx) / rx, ny = (y - cy) / ry;
        var d = nx * nx + ny * ny;
        if (d > 1.05) continue;
        // The edge is where a neighbour would fall outside; that is thinner
        // and more even than testing a band of the distance.
        if (!filled) {
          var inside = true;
          [[1, 0], [-1, 0], [0, 1], [0, -1]].forEach(function (o) {
            var ox = (x + o[0] - cx) / rx, oy = (y + o[1] - cy) / ry;
            if (ox * ox + oy * oy > 1.05) inside = false;
          });
          if (inside) continue;
        }
        var key = x + ',' + y;
        if (seen[key]) continue;
        seen[key] = 1;
        points.push({ x: x, y: y });
      }
    }
    return points;
  }

  // fill spreads from one cell over everything of the same character joined to
  // it. Four-way, not eight: a diagonal touch is not a join, and treating it
  // as one leaks paint through corners that look closed.
  function fill(rows, x, y, ch) {
    var from = at(rows, x, y);
    if (from === '' || from === ch) return rows;
    var w = width(rows), h = height(rows);
    var queue = [{ x: x, y: y }], points = [];
    var seen = {};
    seen[x + ',' + y] = 1;
    while (queue.length) {
      var p = queue.pop();
      if (at(rows, p.x, p.y) !== from) continue;
      points.push(p);
      [[1, 0], [-1, 0], [0, 1], [0, -1]].forEach(function (o) {
        var nx = p.x + o[0], ny = p.y + o[1];
        if (nx < 0 || ny < 0 || nx >= w || ny >= h) return;
        var key = nx + ',' + ny;
        if (seen[key]) return;
        seen[key] = 1;
        queue.push({ x: nx, y: ny });
      });
    }
    return plot(rows, points, ch);
  }

  // replace swaps one character for another everywhere. It is what you want
  // when a tile turns out to be the wrong kind after the level is built.
  function replace(rows, from, to) {
    return rows.map(function (row) {
      var out = '';
      for (var i = 0; i < row.length; i++) out += row[i] === from ? to : row[i];
      return out;
    });
  }

  // read lifts a rectangle out as its own little grid, which is all a clipboard
  // needs to be.
  function read(rows, x0, y0, x1, y1) {
    var lox = Math.min(x0, x1), hix = Math.max(x0, x1);
    var loy = Math.min(y0, y1), hiy = Math.max(y0, y1);
    var out = [];
    for (var y = loy; y <= hiy; y++) {
      var row = '';
      for (var x = lox; x <= hix; x++) row += at(rows, x, y) || '.';
      out.push(row);
    }
    return out;
  }

  // write stamps a grid down at a position. Anything hanging off the edge is
  // dropped rather than wrapped: paste near a corner should lose the overhang,
  // not have it reappear on the far side.
  //
  // skip is the character treated as see-through. Pass null to stamp every
  // cell, which is what moving a selection needs.
  function write(rows, patch, x, y, skip) {
    var out = rows.slice();
    for (var r = 0; r < patch.length; r++) {
      var ty = y + r;
      if (ty < 0 || ty >= out.length) continue;
      var chars = out[ty].split('');
      var changed = false;
      for (var c = 0; c < patch[r].length; c++) {
        var tx = x + c;
        if (tx < 0 || tx >= chars.length) continue;
        if (skip != null && patch[r][c] === skip) continue;
        chars[tx] = patch[r][c];
        changed = true;
      }
      if (changed) out[ty] = chars.join('');
    }
    return out;
  }

  // clear fills a rectangle with one character.
  function clear(rows, x0, y0, x1, y1, ch) {
    return plot(rows, rect(x0, y0, x1, y1, true), ch);
  }

  function flipH(rows) {
    return rows.map(function (row) { return row.split('').reverse().join(''); });
  }

  function flipV(rows) { return rows.slice().reverse(); }

  // rotate turns a quarter turn clockwise. The result is as wide as the input
  // was tall, so a caller resizing a sprite has to take the new shape from
  // what comes back rather than assuming it kept its own.
  function rotate(rows) {
    var w = width(rows), h = height(rows);
    var out = [];
    for (var x = 0; x < w; x++) {
      var row = '';
      for (var y = h - 1; y >= 0; y--) row += rows[y][x];
      out.push(row);
    }
    return out;
  }

  // shift moves everything by a whole number of cells, filling what it leaves
  // behind. Wrapping instead of filling is what you want for a repeating
  // background, so both are offered.
  function shift(rows, dx, dy, ch, wrap) {
    var w = width(rows), h = height(rows);
    var out = [];
    for (var y = 0; y < h; y++) {
      var row = '';
      for (var x = 0; x < w; x++) {
        var sx = x - dx, sy = y - dy;
        if (wrap) {
          sx = ((sx % w) + w) % w;
          sy = ((sy % h) + h) % h;
        } else if (sx < 0 || sy < 0 || sx >= w || sy >= h) {
          row += ch;
          continue;
        }
        row += rows[sy][sx];
      }
      out.push(row);
    }
    return out;
  }

  // resize keeps the top-left corner where it is, which is the one anchor that
  // never moves what is already drawn.
  function resize(rows, w, h, ch) {
    var out = [];
    for (var y = 0; y < h; y++) {
      var row = '';
      for (var x = 0; x < w; x++) row += at(rows, x, y) || ch;
      out.push(row);
    }
    return out;
  }

  // outline returns the cells that are empty but touch something drawn. It is
  // how a one-click border round a shape is done.
  function outline(rows, empty) {
    var w = width(rows), h = height(rows);
    var points = [];
    for (var y = 0; y < h; y++) {
      for (var x = 0; x < w; x++) {
        if (rows[y][x] !== empty) continue;
        var touches = false;
        [[1, 0], [-1, 0], [0, 1], [0, -1]].forEach(function (o) {
          var n = at(rows, x + o[0], y + o[1]);
          if (n !== '' && n !== empty) touches = true;
        });
        if (touches) points.push({ x: x, y: y });
      }
    }
    return points;
  }

  // blank makes an empty grid, which every "new" button needs.
  function blank(w, h, ch) {
    var rows = [];
    for (var y = 0; y < h; y++) rows.push(new Array(w + 1).join(ch));
    return rows;
  }

  var ops = {
    width: width, height: height, at: at, set: set, plot: plot,
    line: line, rect: rect, ellipse: ellipse, fill: fill, replace: replace,
    read: read, write: write, clear: clear,
    flipH: flipH, flipV: flipV, rotate: rotate, shift: shift,
    resize: resize, outline: outline, blank: blank
  };

  if (typeof module === 'object' && module.exports) module.exports = ops;
  else root.PaintOps = ops;
})(this);
