// Finding the line a mistake is on.
//
// Browsers describe a syntax error tersely and without saying where: "missing
// ) after argument list" and nothing else. Where is most of the help, so it is
// worked out here instead, by reading the code and counting what is opened
// against what is closed.
//
// This is not a parser and does not try to be. Nearly every syntax error in
// hand-written game code is a bracket or a quote left open, a closer too many,
// or two that do not match — and those can be found exactly, with a line
// number, by keeping a stack. Anything subtler is left to the browser's own
// message, which is shown either way.
(function (root) {
  'use strict';

  var OPENERS = { '(': ')', '[': ']', '{': '}' };
  var CLOSERS = { ')': '(', ']': '[', '}': '{' };
  var NAMES = { '(': 'a bracket', '[': 'a square bracket', '{': 'a curly bracket' };

  // whereItWentWrong returns { line, why } for the first thing that does not
  // add up, or null when everything balances.
  function whereItWentWrong(source) {
    var open = [];
    var line = 1;
    var quote = null, quoteLine = 0;
    var comment = null;

    for (var i = 0; i < source.length; i++) {
      var ch = source[i], next = source[i + 1];

      if (ch === '\n') {
        line++;
        if (comment === '//') comment = null;
        // A plain quote does not survive the end of a line. Saying so here is
        // more useful than following it to the end of the file and blaming
        // whatever it happens to run into.
        if (quote && quote !== '`') {
          return { line: quoteLine, why: 'a quote was opened and never closed' };
        }
        continue;
      }
      if (comment === '/*') {
        if (ch === '*' && next === '/') { comment = null; i++; }
        continue;
      }
      if (comment) continue;

      if (quote) {
        if (ch === '\\') { i++; continue; }
        if (ch === quote) quote = null;
        continue;
      }

      if (ch === '/' && next === '/') { comment = '//'; i++; continue; }
      if (ch === '/' && next === '*') { comment = '/*'; i++; continue; }
      if (ch === '"' || ch === "'" || ch === '`') { quote = ch; quoteLine = line; continue; }

      if (OPENERS[ch]) { open.push({ ch: ch, line: line }); continue; }
      if (CLOSERS[ch]) {
        if (!open.length) {
          return { line: line, why: 'there is one "' + ch + '" too many' };
        }
        var last = open.pop();
        if (last.ch !== CLOSERS[ch]) {
          // The line reported is the one that opened, not the one that
          // clashed: the closer is usually innocent and the fix goes where
          // the opener is. Both are named, so neither has to be hunted for.
          return {
            line: last.line,
            why: NAMES[last.ch] + ' opened here is closed by a "' + ch +
              '" on line ' + line + ', which does not match it'
          };
        }
      }
    }

    if (quote) return { line: quoteLine, why: 'a quote was opened and never closed' };
    if (open.length) {
      // The outermost unclosed one, not the innermost: it is the one whose
      // closer is actually missing.
      return { line: open[0].line, why: NAMES[open[0].ch] + ' was opened and never closed' };
    }
    return null;
  }

  // explain turns what the browser said, plus what was found above, into
  // something worth putting in front of somebody.
  function explain(err, source) {
    var message = err && err.message ? err.message : String(err);
    var found = whereItWentWrong(source);
    if (!found) return message;
    var out = message + '\n\n' + found.why + ' — line ' + found.line;
    var row = source.split('\n')[found.line - 1];
    if (row != null) out += '\n  ' + row.trim();
    return out;
  }

  var api = { whereItWentWrong: whereItWentWrong, explain: explain };
  if (typeof module === 'object' && module.exports) module.exports = api;
  else root.CodeCheck = api;
})(this);
