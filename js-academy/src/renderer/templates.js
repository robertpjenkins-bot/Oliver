'use strict';

// Starting points for new playground projects.
window.JSA_TEMPLATES = [
  {
    id: 'blank',
    name: 'Blank page',
    desc: 'An empty page — start from scratch.',
    html: `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>My App</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <h1>Hello!</h1>

  <script src="script.js"></script>
</body>
</html>
`,
    css: `body {
  font-family: system-ui, sans-serif;
  margin: 2rem;
}
`,
    js: `// Your JavaScript goes here.
console.log('Hello from my app!');
`,
  },
  {
    id: 'counter',
    name: 'Click counter',
    desc: 'A button, a number and an event listener.',
    html: `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Counter</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <main>
    <h1 id="count">0</h1>
    <div class="buttons">
      <button id="minus">−</button>
      <button id="reset">Reset</button>
      <button id="plus">+</button>
    </div>
  </main>
  <script src="script.js"></script>
</body>
</html>
`,
    css: `body {
  margin: 0; min-height: 100vh; display: grid; place-items: center;
  font-family: system-ui, sans-serif; background: #1e1b4b; color: white;
}
h1 { font-size: 8rem; text-align: center; margin: 0; }
.buttons { display: flex; gap: 1rem; }
button {
  font-size: 1.5rem; padding: .6rem 1.4rem; border: 0; border-radius: .6rem;
  background: #8b5cf6; color: white; cursor: pointer;
}
button:hover { background: #a78bfa; }
`,
    js: `let count = 0;
const display = document.querySelector('#count');

function show() {
  display.textContent = count;
  display.style.color = count < 0 ? '#f87171' : 'white';
}

document.querySelector('#plus').addEventListener('click', () => { count++; show(); });
document.querySelector('#minus').addEventListener('click', () => { count--; show(); });
document.querySelector('#reset').addEventListener('click', () => { count = 0; show(); });
`,
  },
  {
    id: 'todo',
    name: 'To-do list',
    desc: 'Add, tick and delete tasks. Saved between runs.',
    html: `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>To-do</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <main>
    <h1>📝 My to-dos</h1>
    <form id="form">
      <input id="input" placeholder="What needs doing?" autocomplete="off">
      <button>Add</button>
    </form>
    <ul id="list"></ul>
    <p id="left"></p>
  </main>
  <script src="script.js"></script>
</body>
</html>
`,
    css: `body { font-family: system-ui, sans-serif; background: #f1f5f9; margin: 0; }
main { max-width: 480px; margin: 3rem auto; background: white; padding: 1.5rem 2rem; border-radius: 1rem; box-shadow: 0 10px 30px #0001; }
form { display: flex; gap: .5rem; }
input { flex: 1; padding: .6rem; font-size: 1rem; border: 1px solid #cbd5e1; border-radius: .4rem; }
button { padding: .6rem 1rem; border: 0; border-radius: .4rem; background: #0ea5e9; color: white; cursor: pointer; }
ul { list-style: none; padding: 0; }
li { display: flex; align-items: center; gap: .6rem; padding: .5rem 0; border-bottom: 1px solid #e2e8f0; }
li.done span { text-decoration: line-through; color: #94a3b8; }
li span { flex: 1; }
li button { background: transparent; color: #ef4444; font-size: 1.1rem; }
#left { color: #64748b; font-size: .9rem; }
`,
    js: `// Tasks are kept in localStorage so they survive restarts.
let todos = JSON.parse(localStorage.getItem('todos') || '[]');

const form = document.querySelector('#form');
const input = document.querySelector('#input');
const list = document.querySelector('#list');

function save() {
  localStorage.setItem('todos', JSON.stringify(todos));
}

function render() {
  list.innerHTML = '';
  for (const todo of todos) {
    const li = document.createElement('li');
    li.className = todo.done ? 'done' : '';

    const box = document.createElement('input');
    box.type = 'checkbox';
    box.checked = todo.done;
    box.addEventListener('change', () => { todo.done = box.checked; save(); render(); });

    const text = document.createElement('span');
    text.textContent = todo.text;

    const del = document.createElement('button');
    del.textContent = '✕';
    del.addEventListener('click', () => { todos = todos.filter(t => t !== todo); save(); render(); });

    li.append(box, text, del);
    list.append(li);
  }
  const left = todos.filter(t => !t.done).length;
  document.querySelector('#left').textContent = left + ' left to do';
}

form.addEventListener('submit', (event) => {
  event.preventDefault();
  const text = input.value.trim();
  if (!text) return;
  todos.push({ text, done: false });
  input.value = '';
  save();
  render();
});

render();
`,
  },
  {
    id: 'game',
    name: 'Canvas game',
    desc: 'Catch the falling stars with the arrow keys.',
    html: `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Star Catcher</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <canvas id="game" width="640" height="480"></canvas>
  <p>← → to move · catch the stars · avoid the rocks</p>
  <script src="script.js"></script>
</body>
</html>
`,
    css: `body { margin: 0; background: #020617; color: #94a3b8; font-family: system-ui, sans-serif;
  display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 100vh; }
canvas { background: linear-gradient(#0f172a, #1e293b); border-radius: 12px; box-shadow: 0 0 40px #38bdf833; }
`,
    js: `const canvas = document.querySelector('#game');
const ctx = canvas.getContext('2d');

const player = { x: 300, y: 440, w: 60, h: 14, speed: 6 };
let things = [];
let score = 0;
let lives = 3;
const keys = {};

addEventListener('keydown', (e) => { keys[e.key] = true; });
addEventListener('keyup', (e) => { keys[e.key] = false; });

function spawn() {
  const rock = Math.random() < 0.3;
  things.push({ x: Math.random() * (canvas.width - 20) + 10, y: -10, speed: 2 + Math.random() * 3, rock });
}

function update() {
  if (keys.ArrowLeft) player.x -= player.speed;
  if (keys.ArrowRight) player.x += player.speed;
  player.x = Math.max(0, Math.min(canvas.width - player.w, player.x));

  if (Math.random() < 0.04) spawn();
  for (const t of things) {
    t.y += t.speed;
    const caught = t.y > player.y - 8 && t.y < player.y + player.h && t.x > player.x && t.x < player.x + player.w;
    if (caught) {
      t.gone = true;
      if (t.rock) lives--; else score++;
    }
    if (t.y > canvas.height) t.gone = true;
  }
  things = things.filter((t) => !t.gone);
}

function draw() {
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = '#38bdf8';
  ctx.fillRect(player.x, player.y, player.w, player.h);
  for (const t of things) {
    ctx.font = '22px sans-serif';
    ctx.fillText(t.rock ? '🪨' : '⭐', t.x - 11, t.y + 8);
  }
  ctx.fillStyle = 'white';
  ctx.font = '18px sans-serif';
  ctx.fillText('Score: ' + score + '   Lives: ' + '❤️'.repeat(lives), 12, 26);
}

function loop() {
  update();
  draw();
  if (lives > 0) {
    requestAnimationFrame(loop);
  } else {
    ctx.fillStyle = '#000a';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.fillStyle = 'white';
    ctx.font = 'bold 40px sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('Game over! Score: ' + score, canvas.width / 2, canvas.height / 2);
    ctx.textAlign = 'start';
  }
}
loop();
`,
  },
  {
    id: 'draw',
    name: 'Drawing pad',
    desc: 'Paint with the mouse; pick colours and sizes.',
    html: `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Drawing Pad</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <div class="tools">
    <input type="color" id="color" value="#e11d48">
    <input type="range" id="size" min="1" max="40" value="6">
    <button id="clear">Clear</button>
  </div>
  <canvas id="pad"></canvas>
  <script src="script.js"></script>
</body>
</html>
`,
    css: `html, body { margin: 0; height: 100%; font-family: system-ui, sans-serif; }
body { display: flex; flex-direction: column; }
.tools { display: flex; gap: 1rem; align-items: center; padding: .6rem 1rem; background: #f1f5f9; border-bottom: 1px solid #cbd5e1; }
canvas { flex: 1; cursor: crosshair; }
`,
    js: `const pad = document.querySelector('#pad');
const ctx = pad.getContext('2d');
let drawing = false;
let last = null;

function fit() {
  const img = pad.width ? ctx.getImageData(0, 0, pad.width, pad.height) : null;
  pad.width = pad.clientWidth;
  pad.height = pad.clientHeight;
  if (img) ctx.putImageData(img, 0, 0);
}
addEventListener('resize', fit);
fit();

pad.addEventListener('pointerdown', (e) => { drawing = true; last = [e.offsetX, e.offsetY]; });
addEventListener('pointerup', () => { drawing = false; });
pad.addEventListener('pointermove', (e) => {
  if (!drawing) return;
  ctx.strokeStyle = document.querySelector('#color').value;
  ctx.lineWidth = document.querySelector('#size').value;
  ctx.lineCap = 'round';
  ctx.beginPath();
  ctx.moveTo(...last);
  ctx.lineTo(e.offsetX, e.offsetY);
  ctx.stroke();
  last = [e.offsetX, e.offsetY];
});
document.querySelector('#clear').addEventListener('click', () => ctx.clearRect(0, 0, pad.width, pad.height));
`,
  },
  {
    id: 'quiz',
    name: 'Quiz app',
    desc: 'Questions from an array, score at the end.',
    html: `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Quiz</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <main id="app"></main>
  <script src="script.js"></script>
</body>
</html>
`,
    css: `body { font-family: system-ui, sans-serif; background: linear-gradient(135deg, #fde68a, #fca5a5); min-height: 100vh; margin: 0; display: grid; place-items: center; }
main { background: white; padding: 2rem; border-radius: 1rem; width: min(90vw, 480px); box-shadow: 0 20px 40px #0002; }
button.answer { display: block; width: 100%; margin: .4rem 0; padding: .8rem; font-size: 1rem; border: 2px solid #e5e7eb; background: white; border-radius: .6rem; cursor: pointer; text-align: left; }
button.answer:hover { border-color: #f59e0b; }
.progress { color: #6b7280; font-size: .9rem; }
`,
    js: `const questions = [
  { q: 'Which keyword declares a variable that cannot be reassigned?', options: ['var', 'let', 'const'], answer: 2 },
  { q: 'What does [1, 2, 3].length give?', options: ['2', '3', '4'], answer: 1 },
  { q: 'Which method adds an item to the end of an array?', options: ['push', 'pop', 'shift'], answer: 0 },
  { q: 'What is typeof "hello"?', options: ['text', 'string', 'word'], answer: 1 },
];

let index = 0;
let score = 0;
const app = document.querySelector('#app');

function show() {
  if (index >= questions.length) {
    app.innerHTML = '<h2>You scored ' + score + ' / ' + questions.length + '!</h2><button class="answer" id="again">Play again</button>';
    document.querySelector('#again').onclick = () => { index = 0; score = 0; show(); };
    return;
  }
  const { q, options, answer } = questions[index];
  app.innerHTML = '<p class="progress">Question ' + (index + 1) + ' of ' + questions.length + '</p><h2></h2>';
  app.querySelector('h2').textContent = q;
  options.forEach((text, i) => {
    const b = document.createElement('button');
    b.className = 'answer';
    b.textContent = text;
    b.onclick = () => {
      if (i === answer) score++;
      index++;
      show();
    };
    app.append(b);
  });
}
show();
`,
  },
];
