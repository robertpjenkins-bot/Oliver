'use strict';

// A small Markdown renderer for lesson text: headings, paragraphs, lists,
// quotes, fenced code (highlighted with CodeMirror's runmode), inline code,
// bold, italics and links. Lesson text is our own, but everything is still
// escaped before formatting is applied.
(function (root) {
  const esc = (s) => s.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]);

  function inline(text) {
    const codes = [];
    let s = text.replace(/`([^`]+)`/g, (_, c) => {
      codes.push(c);
      return '\u0000' + (codes.length - 1) + '\u0000';
    });
    s = esc(s)
      .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
      .replace(/(^|[^\w*])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>')
      .replace(/\[([^\]]+)\]\((https?:[^)\s]+)\)/g, '<a href="$2" target="_blank">$1</a>');
    return s.replace(/\u0000(\d+)\u0000/g, (_, i) => '<code>' + esc(codes[+i]) + '</code>');
  }

  function highlight(code, lang) {
    if (root.CodeMirror && root.CodeMirror.runMode) {
      const el = document.createElement('div');
      const mode = lang === 'html' ? 'htmlmixed' : lang === 'css' ? 'css' : 'javascript';
      root.CodeMirror.runMode(code, mode, el);
      return el.innerHTML;
    }
    return esc(code);
  }

  function render(md) {
    const lines = String(md || '').replace(/\r/g, '').split('\n');
    const out = [];
    let i = 0;
    while (i < lines.length) {
      const line = lines[i];
      const fence = line.match(/^```(\w*)/);
      if (fence) {
        const buf = [];
        i++;
        while (i < lines.length && !/^```/.test(lines[i])) buf.push(lines[i++]);
        i++;
        out.push('<pre class="cm-s-jsa code-block"><code>' + highlight(buf.join('\n'), fence[1]) + '</code></pre>');
        continue;
      }
      const h = line.match(/^(#{1,4})\s+(.*)/);
      if (h) {
        const level = Math.min(h[1].length + 1, 5);
        out.push('<h' + level + '>' + inline(h[2]) + '</h' + level + '>');
        i++;
        continue;
      }
      if (/^\s*[-*]\s+/.test(line)) {
        const items = [];
        while (i < lines.length && /^\s*[-*]\s+/.test(lines[i])) items.push(lines[i++].replace(/^\s*[-*]\s+/, ''));
        out.push('<ul>' + items.map((t) => '<li>' + inline(t) + '</li>').join('') + '</ul>');
        continue;
      }
      if (/^\s*\d+\.\s+/.test(line)) {
        const items = [];
        while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i])) items.push(lines[i++].replace(/^\s*\d+\.\s+/, ''));
        out.push('<ol>' + items.map((t) => '<li>' + inline(t) + '</li>').join('') + '</ol>');
        continue;
      }
      if (/^>\s?/.test(line)) {
        const buf = [];
        while (i < lines.length && /^>\s?/.test(lines[i])) buf.push(lines[i++].replace(/^>\s?/, ''));
        out.push('<blockquote>' + inline(buf.join(' ')) + '</blockquote>');
        continue;
      }
      if (!line.trim()) {
        i++;
        continue;
      }
      const buf = [];
      while (i < lines.length && lines[i].trim() && !/^(```|#{1,4}\s|\s*[-*]\s|\s*\d+\.\s|>)/.test(lines[i])) buf.push(lines[i++]);
      out.push('<p>' + inline(buf.join(' ')) + '</p>');
    }
    return out.join('\n');
  }

  root.renderMarkdown = render;
  root.renderInline = inline;
})(typeof window !== 'undefined' ? window : globalThis);
