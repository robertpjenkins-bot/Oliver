registerModule({
  id: 'events',
  title: 'Events & Interactive Pages',
  icon: '🖱️',
  lessons: [
    {
      t: 'Responding to Clicks',
      b: `**Events** are things that happen on a page: clicks, key presses, typing… \`addEventListener\` runs a function whenever an event happens:

\`\`\`js
button.addEventListener("click", () => {
  console.log("Clicked!");
});
\`\`\`

After running your code, click the button in the page preview to try it. (The checks click it for you too.)`,
      dom: `<button id="hello">Say hello</button>\n<p id="output"></p>`,
      code: `// When #hello is clicked, set #output's text to "Hello there!"\n`,
      sol: `document.querySelector("#hello").addEventListener("click", () => {\n  document.querySelector("#output").textContent = "Hello there!";\n});`,
      tests: [['document.querySelector("#output").textContent === ""', 'Nothing happens until the click'], ['(document.querySelector("#hello").click(), document.querySelector("#output").textContent === "Hello there!")', 'Clicking shows `Hello there!`'], ['$src.includes("addEventListener")', 'Use `addEventListener`']],
    },
    {
      t: 'A Click Counter',
      b: `Keep state in a variable, update it in the event handler, and redraw:

\`\`\`js
let clicks = 0;
button.addEventListener("click", () => {
  clicks++;
  label.textContent = clicks;
});
\`\`\``,
      dom: `<button id="plus">+1</button>\n<button id="reset">Reset</button>\n<h1 id="count">0</h1>`,
      code: `// #plus adds one to the count shown in #count; #reset sets it back to 0\n`,
      sol: `let count = 0;\nconst show = () => (document.querySelector("#count").textContent = count);\ndocument.querySelector("#plus").addEventListener("click", () => {\n  count++;\n  show();\n});\ndocument.querySelector("#reset").addEventListener("click", () => {\n  count = 0;\n  show();\n});`,
      tests: [['(() => { const p = document.querySelector("#plus"); p.click(); p.click(); p.click(); return document.querySelector("#count").textContent === "3"; })()', 'Three clicks show 3'], ['(document.querySelector("#reset").click(), document.querySelector("#count").textContent === "0")', 'Reset shows 0'], ['(document.querySelector("#plus").click(), document.querySelector("#count").textContent === "1")', 'Counting starts again from 0']],
    },
    {
      t: 'The Event Object',
      b: `Handlers receive an **event object** describing what happened. \`event.target\` is the element the event happened on:

\`\`\`js
document.addEventListener("click", (event) => {
  console.log("You clicked", event.target.textContent);
});
\`\`\``,
      dom: `<button class="color">Red</button>\n<button class="color">Green</button>\n<button class="color">Blue</button>\n<p>Chosen: <span id="chosen">none</span></p>`,
      code: `// For each .color button, on click put the clicked button's text into #chosen (use event.target)\n`,
      sol: `for (const btn of document.querySelectorAll(".color")) {\n  btn.addEventListener("click", (event) => {\n    document.querySelector("#chosen").textContent = event.target.textContent;\n  });\n}`,
      tests: [['(document.querySelectorAll(".color")[1].click(), document.querySelector("#chosen").textContent === "Green")', 'Clicking Green shows Green'], ['(document.querySelectorAll(".color")[2].click(), document.querySelector("#chosen").textContent === "Blue")', 'Clicking Blue shows Blue'], ['$src.includes("target")', 'Use `event.target`']],
    },
    {
      t: 'Live Typing',
      b: `The \`input\` event fires on every change to a text field — perfect for live previews:

\`\`\`js
field.addEventListener("input", () => {
  preview.textContent = field.value;
});
\`\`\``,
      dom: `<input id="name" placeholder="Your name">\n<h2 id="greeting">Hello, stranger!</h2>`,
      code: `// As the user types in #name, update #greeting to "Hello, <name>!"\n// If the field is empty (after trimming), show "Hello, stranger!"\n`,
      sol: 'const field = document.querySelector("#name");\nfield.addEventListener("input", () => {\n  const name = field.value.trim();\n  document.querySelector("#greeting").textContent = `Hello, ${name || "stranger"}!`;\n});',
      tests: [['(() => { const f = document.querySelector("#name"); f.value = "Sam"; f.dispatchEvent(new Event("input")); return document.querySelector("#greeting").textContent === "Hello, Sam!"; })()', 'Typing "Sam" shows "Hello, Sam!"'], ['(() => { const f = document.querySelector("#name"); f.value = "   "; f.dispatchEvent(new Event("input")); return document.querySelector("#greeting").textContent === "Hello, stranger!"; })()', 'An empty field shows "Hello, stranger!"']],
    },
    {
      t: 'Checkboxes and Selects',
      b: `For checkboxes, radio buttons and dropdowns, listen for \`change\`:

\`\`\`js
checkbox.addEventListener("change", () => {
  console.log(checkbox.checked);
});
\`\`\``,
      dom: `<label><input type="checkbox" id="dark"> Dark mode</label>\n<select id="font"><option value="16">Normal</option><option value="22">Large</option></select>\n<p id="demo">Sample text</p>`,
      code: `// When #dark changes, set document.body.className to "dark" if checked, otherwise ""\n// When #font changes, set #demo's font size to the chosen value + "px"\n`,
      sol: `const dark = document.querySelector("#dark");\ndark.addEventListener("change", () => {\n  document.body.className = dark.checked ? "dark" : "";\n});\nconst font = document.querySelector("#font");\nfont.addEventListener("change", () => {\n  document.querySelector("#demo").style.fontSize = font.value + "px";\n});`,
      tests: [['(() => { const d = document.querySelector("#dark"); d.checked = true; d.dispatchEvent(new Event("change")); return document.body.className === "dark"; })()', 'Checking dark mode adds the class'], ['(() => { const d = document.querySelector("#dark"); d.checked = false; d.dispatchEvent(new Event("change")); return document.body.className === ""; })()', 'Unchecking removes it'], ['(() => { const f = document.querySelector("#font"); f.value = "22"; f.dispatchEvent(new Event("change")); return document.querySelector("#demo").style.fontSize === "22px"; })()', 'Choosing Large sets 22px']],
    },
    {
      t: 'Keyboard Events',
      b: `\`keydown\` fires when a key is pressed. \`event.key\` tells you which: \`"a"\`, \`"Enter"\`, \`"ArrowUp"\`, \`" "\` (space)…

\`\`\`js
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") closeMenu();
});
\`\`\``,
      dom: `<p>Press a key! Last key: <b id="last">none</b></p>\n<p>Enter presses: <b id="enters">0</b></p>`,
      code: `// On keydown anywhere in the document: show event.key in #last,\n// and count presses of "Enter" in #enters\n`,
      sol: `let enters = 0;\ndocument.addEventListener("keydown", (e) => {\n  document.querySelector("#last").textContent = e.key;\n  if (e.key === "Enter") {\n    enters++;\n    document.querySelector("#enters").textContent = enters;\n  }\n});`,
      tests: [['(document.dispatchEvent(new KeyboardEvent("keydown", { key: "q" })), document.querySelector("#last").textContent === "q")', 'Shows the last key'], ['(() => { document.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" })); document.dispatchEvent(new KeyboardEvent("keydown", { key: "x" })); document.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" })); return document.querySelector("#enters").textContent === "2"; })()', 'Counts Enter presses']],
    },
    {
      t: 'Forms and preventDefault',
      b: `Submitting a form normally reloads the page. In JavaScript apps, handle the \`submit\` event and call \`event.preventDefault()\` to stop that:

\`\`\`js
form.addEventListener("submit", (event) => {
  event.preventDefault();
  // handle the data ourselves
});
\`\`\``,
      dom: `<form id="signup">\n  <input id="email" placeholder="Email">\n  <button>Sign up</button>\n</form>\n<p id="result"></p>`,
      code: `// On submit: prevent the reload. If #email contains "@", show "Thanks!" in #result,\n// otherwise show "Please enter a valid email"\n`,
      sol: `document.querySelector("#signup").addEventListener("submit", (event) => {\n  event.preventDefault();\n  const email = document.querySelector("#email").value;\n  document.querySelector("#result").textContent = email.includes("@") ? "Thanks!" : "Please enter a valid email";\n});`,
      tests: [['(() => { document.querySelector("#email").value = "nope"; const e = new Event("submit", { cancelable: true }); document.querySelector("#signup").dispatchEvent(e); return e.defaultPrevented && document.querySelector("#result").textContent === "Please enter a valid email"; })()', 'Invalid emails are rejected, without reloading'], ['(() => { document.querySelector("#email").value = "a@b.co"; const e = new Event("submit", { cancelable: true }); document.querySelector("#signup").dispatchEvent(e); return document.querySelector("#result").textContent === "Thanks!"; })()', 'Valid emails are thanked']],
    },
    {
      t: 'Event Delegation',
      b: `Events **bubble**: a click on a button also counts as a click on its parent, grandparent, and so on up to \`document\`. So one listener on a container can handle all its children — even ones added later:

\`\`\`js
list.addEventListener("click", (e) => {
  const item = e.target.closest("li");
  if (item) item.classList.toggle("done");
});
\`\`\``,
      dom: `<style>.done { text-decoration: line-through; }</style>\n<ul id="list"><li>One</li><li>Two</li></ul>`,
      code: `// Add ONE click listener to #list that toggles the "done" class on the clicked li\n`,
      sol: `document.querySelector("#list").addEventListener("click", (e) => {\n  const item = e.target.closest("li");\n  if (item) item.classList.toggle("done");\n});`,
      tests: [['(document.querySelectorAll("#list li")[1].click(), document.querySelectorAll("#list li")[1].classList.contains("done"))', 'Clicking an item toggles it'], ['(() => { const li = document.createElement("li"); li.textContent = "Three"; document.querySelector("#list").append(li); li.click(); return li.classList.contains("done"); })()', 'Works for items added later'], ['($src.match(/addEventListener/g) || []).length === 1', 'Use just one listener']],
    },
    {
      t: 'Stopping the Bubble',
      b: `Sometimes a click on a child should **not** reach the parent — like a delete button inside a clickable card. \`event.stopPropagation()\` stops the bubbling:

\`\`\`js
deleteBtn.addEventListener("click", (e) => {
  e.stopPropagation();
  card.remove();
});
\`\`\``,
      dom: `<div id="card" style="border:1px solid #999;padding:8px">Open me <button id="close">✕</button></div>\n<p id="log"></p>`,
      code: `const log = document.querySelector("#log");\ndocument.querySelector("#card").addEventListener("click", () => {\n  log.textContent = "card opened";\n});\n// When #close is clicked, set log to "closed" — the card must NOT open\n`,
      sol: `const log = document.querySelector("#log");\ndocument.querySelector("#card").addEventListener("click", () => {\n  log.textContent = "card opened";\n});\ndocument.querySelector("#close").addEventListener("click", (e) => {\n  e.stopPropagation();\n  log.textContent = "closed";\n});`,
      tests: [['(document.querySelector("#close").click(), document.querySelector("#log").textContent === "closed")', 'Clicking ✕ only closes'], ['(document.querySelector("#card").click(), document.querySelector("#log").textContent === "card opened")', 'Clicking the card still opens it']],
    },
    {
      t: 'Listening Once',
      b: `Pass \`{ once: true }\` to run a listener only the first time. Or keep a reference to the function and call \`removeEventListener\` later:

\`\`\`js
button.addEventListener("click", start, { once: true });
// or
button.removeEventListener("click", start);
\`\`\``,
      dom: `<button id="claim">Claim your free prize</button>\n<p id="prizes">0</p>`,
      code: `// Each click on #claim should add 1 to #prizes — but only the FIRST click counts\n`,
      sol: `let prizes = 0;\ndocument.querySelector("#claim").addEventListener(\n  "click",\n  () => {\n    prizes++;\n    document.querySelector("#prizes").textContent = prizes;\n  },\n  { once: true },\n);`,
      tests: [['(() => { const b = document.querySelector("#claim"); b.click(); b.click(); b.click(); return document.querySelector("#prizes").textContent === "1"; })()', 'Only one prize, however many clicks']],
    },
    {
      t: 'Mouse Over and Out',
      b: `\`mouseenter\` and \`mouseleave\` fire when the pointer moves onto and off an element — great for hover effects that CSS cannot do alone:

\`\`\`js
card.addEventListener("mouseenter", () => tip.hidden = false);
card.addEventListener("mouseleave", () => tip.hidden = true);
\`\`\``,
      dom: `<span id="word" style="text-decoration:underline dotted">JavaScript</span>\n<p id="tip" hidden>A programming language for the web.</p>`,
      code: `// Show #tip while the mouse is over #word, and hide it again when it leaves\n`,
      sol: `const word = document.querySelector("#word");\nconst tip = document.querySelector("#tip");\nword.addEventListener("mouseenter", () => (tip.hidden = false));\nword.addEventListener("mouseleave", () => (tip.hidden = true));`,
      tests: [['(document.querySelector("#word").dispatchEvent(new MouseEvent("mouseenter")), document.querySelector("#tip").hidden === false)', 'Hovering shows the tip'], ['(document.querySelector("#word").dispatchEvent(new MouseEvent("mouseleave")), document.querySelector("#tip").hidden === true)', 'Leaving hides it']],
    },
    {
      t: 'Custom Events',
      b: `You can invent your own events with \`CustomEvent\`, carrying data in \`detail\`. Different parts of an app can then talk without knowing about each other:

\`\`\`js
document.addEventListener("level-up", (e) => console.log("Now level", e.detail.level));
document.dispatchEvent(new CustomEvent("level-up", { detail: { level: 5 } }));
\`\`\``,
      dom: `<p>Score: <span id="score">0</span></p>`,
      code: `// Listen on document for a "points" event and add event.detail.amount to the #score shown\n// Then dispatch a "points" event with amount 10\n`,
      sol: `let score = 0;\ndocument.addEventListener("points", (e) => {\n  score += e.detail.amount;\n  document.querySelector("#score").textContent = score;\n});\ndocument.dispatchEvent(new CustomEvent("points", { detail: { amount: 10 } }));`,
      tests: [['document.querySelector("#score").textContent === "10"', 'The score is 10 after your event'], ['(document.dispatchEvent(new CustomEvent("points", { detail: { amount: 5 } })), document.querySelector("#score").textContent === "15")', 'More events add more points']],
    },
    {
      t: 'Character Counter',
      b: `A small but real feature: show how many characters are left in a message box, and warn when the limit is passed.`,
      dom: `<style>.over { color: red; }</style>\n<textarea id="msg" rows="3"></textarea>\n<p id="left">40 characters left</p>\n<button id="send">Send</button>`,
      code: `const LIMIT = 40;\n// On input in #msg: show "N characters left" in #left (N can be negative).\n// When over the limit, add class "over" to #left and disable #send; otherwise remove/enable\n`,
      sol: 'const LIMIT = 40;\nconst msg = document.querySelector("#msg");\nconst left = document.querySelector("#left");\nmsg.addEventListener("input", () => {\n  const remaining = LIMIT - msg.value.length;\n  left.textContent = `${remaining} characters left`;\n  left.classList.toggle("over", remaining < 0);\n  document.querySelector("#send").disabled = remaining < 0;\n});',
      tests: [['(() => { const m = document.querySelector("#msg"); m.value = "hello"; m.dispatchEvent(new Event("input")); return document.querySelector("#left").textContent === "35 characters left" && !document.querySelector("#send").disabled; })()', 'Counts down as you type'], ['(() => { const m = document.querySelector("#msg"); m.value = "x".repeat(45); m.dispatchEvent(new Event("input")); return document.querySelector("#left").textContent === "-5 characters left" && document.querySelector("#left").classList.contains("over") && document.querySelector("#send").disabled; })()', 'Warns and disables when over'], ['(() => { const m = document.querySelector("#msg"); m.value = "ok"; m.dispatchEvent(new Event("input")); return !document.querySelector("#left").classList.contains("over"); })()', 'Recovers when back under the limit']],
    },
    {
      t: 'Tabs',
      b: `A tab bar: clicking a tab shows its panel and hides the rest. Link tabs and panels with a \`data-tab\` attribute.`,
      dom: `<style>.tab.active { font-weight: bold; }</style>\n<button class="tab active" data-tab="home">Home</button>\n<button class="tab" data-tab="news">News</button>\n<button class="tab" data-tab="about">About</button>\n<section class="panel" id="home">Welcome home</section>\n<section class="panel" id="news" hidden>No news is good news</section>\n<section class="panel" id="about" hidden>About us</section>`,
      code: `// Clicking a .tab makes it the only .active tab, and shows only the panel whose id matches its data-tab\n`,
      sol: `const tabs = document.querySelectorAll(".tab");\nfor (const tab of tabs) {\n  tab.addEventListener("click", () => {\n    for (const t of tabs) t.classList.toggle("active", t === tab);\n    for (const panel of document.querySelectorAll(".panel")) {\n      panel.hidden = panel.id !== tab.dataset.tab;\n    }\n  });\n}`,
      tests: [['(document.querySelector("[data-tab=news]").click(), !document.querySelector("#news").hidden && document.querySelector("#home").hidden && document.querySelector("#about").hidden)', 'News shows only the news panel'], ['document.querySelectorAll(".tab.active").length === 1 && document.querySelector(".tab.active").dataset.tab === "news"', 'Only the News tab is active'], ['(document.querySelector("[data-tab=home]").click(), !document.querySelector("#home").hidden && document.querySelector("#news").hidden)', 'Switching back works']],
    },
    {
      t: 'Accordion',
      b: `In an accordion, clicking a heading opens its section — and closes any other open one. Each \`.item\` has a \`.head\` button and a \`.body\`.`,
      dom: `<div class="item"><button class="head">What is JS?</button><div class="body" hidden>A programming language.</div></div>\n<div class="item"><button class="head">Is it hard?</button><div class="body" hidden>Not with practice!</div></div>\n<div class="item"><button class="head">Where does it run?</button><div class="body" hidden>Browsers, servers and apps.</div></div>`,
      code: `// Clicking a .head: if its body is hidden, show it and hide all other bodies; if it is open, close it\n`,
      sol: `const bodies = document.querySelectorAll(".body");\nfor (const head of document.querySelectorAll(".head")) {\n  head.addEventListener("click", () => {\n    const body = head.nextElementSibling;\n    const opening = body.hidden;\n    for (const b of bodies) b.hidden = true;\n    body.hidden = !opening;\n  });\n}`,
      tests: [['(() => { const h = document.querySelectorAll(".head"); h[0].click(); const b = document.querySelectorAll(".body"); return !b[0].hidden && b[1].hidden; })()', 'Opens a section'], ['(() => { const h = document.querySelectorAll(".head"); h[2].click(); const b = document.querySelectorAll(".body"); return b[0].hidden && !b[2].hidden; })()', 'Opening another closes the first'], ['(() => { document.querySelectorAll(".head")[2].click(); return [...document.querySelectorAll(".body")].every((b) => b.hidden); })()', 'Clicking an open one closes it']],
    },
    {
      t: 'A To-Do List',
      b: `Put it together: add items from a form, and remove them with a delete button (event delegation makes that easy).`,
      dom: `<form id="add"><input id="task" placeholder="New task"><button>Add</button></form>\n<ul id="todos"></ul>`,
      code: `// On submit (preventDefault!): if #task is not blank, add <li><span>TEXT</span><button class="del">✕</button></li>\n// to #todos and clear the input. Clicking a .del button removes its li.\n`,
      sol: `const input = document.querySelector("#task");\nconst list = document.querySelector("#todos");\ndocument.querySelector("#add").addEventListener("submit", (e) => {\n  e.preventDefault();\n  const text = input.value.trim();\n  if (!text) return;\n  const li = document.createElement("li");\n  const span = document.createElement("span");\n  span.textContent = text;\n  const del = document.createElement("button");\n  del.className = "del";\n  del.textContent = "✕";\n  li.append(span, del);\n  list.append(li);\n  input.value = "";\n});\nlist.addEventListener("click", (e) => {\n  if (e.target.classList.contains("del")) e.target.closest("li").remove();\n});`,
      tests: [
        ['(() => { const f = document.querySelector("#add"); const i = document.querySelector("#task"); i.value = "Feed cat"; f.dispatchEvent(new Event("submit", { cancelable: true })); i.value = "Nap"; f.dispatchEvent(new Event("submit", { cancelable: true })); return [...document.querySelectorAll("#todos li span")].map((s) => s.textContent).join() === "Feed cat,Nap" && i.value === ""; })()', 'Adds tasks and clears the input'],
        ['(() => { const f = document.querySelector("#add"); document.querySelector("#task").value = "   "; f.dispatchEvent(new Event("submit", { cancelable: true })); return document.querySelectorAll("#todos li").length === 2; })()', 'Ignores blank tasks'],
        ['(() => { document.querySelector("#todos li .del").click(); return [...document.querySelectorAll("#todos li span")].map((s) => s.textContent).join() === "Nap"; })()', 'Deletes a task'],
        ['(() => { const e = new Event("submit", { cancelable: true }); document.querySelector("#add").dispatchEvent(e); return e.defaultPrevented; })()', 'Prevents the page reload'],
      ],
    },
    {
      t: 'A Stopwatch',
      b: `Events plus timers: a stopwatch with Start and Stop buttons. Store the interval id so Stop can clear it, and make sure pressing Start twice does not start two timers!`,
      dom: `<h1 id="time">0</h1>\n<button id="start">Start</button>\n<button id="stop">Stop</button>`,
      code: `// Start: every 20 ms add 1 to the number in #time. Stop: stop counting.\n// Pressing Start while already running must not speed it up.\n`,
      sol: `let ticks = 0;\nlet timer = null;\ndocument.querySelector("#start").addEventListener("click", () => {\n  if (timer) return;\n  timer = setInterval(() => {\n    ticks++;\n    document.querySelector("#time").textContent = ticks;\n  }, 20);\n});\ndocument.querySelector("#stop").addEventListener("click", () => {\n  clearInterval(timer);\n  timer = null;\n});`,
      tests: [
        ['(async () => { document.querySelector("#start").click(); document.querySelector("#start").click(); await new Promise((r) => setTimeout(r, 230)); document.querySelector("#stop").click(); const n = Number(document.querySelector("#time").textContent); return n >= 5 && n <= 13; })()', 'Counts while running (and double Start does not double speed)'],
        ['(async () => { const before = document.querySelector("#time").textContent; await new Promise((r) => setTimeout(r, 100)); return document.querySelector("#time").textContent === before; })()', 'Stop stops it'],
      ],
    },
    {
      t: 'Moving with Arrow Keys',
      b: `Games start here: move a box with the arrow keys by changing its position.

\`\`\`js
box.style.left = x + "px";
box.style.top = y + "px";
\`\`\``,
      dom: `<div style="position:relative;width:200px;height:120px;border:1px solid #999">\n  <div id="box" style="position:absolute;left:0;top:0;width:20px;height:20px;background:tomato"></div>\n</div>`,
      code: `// Arrow keys move #box 10px in that direction (keydown on document). It must not leave the area:\n// x from 0 to 180, y from 0 to 100\n`,
      sol: `let x = 0;\nlet y = 0;\nconst box = document.querySelector("#box");\nconst moves = { ArrowLeft: [-10, 0], ArrowRight: [10, 0], ArrowUp: [0, -10], ArrowDown: [0, 10] };\ndocument.addEventListener("keydown", (e) => {\n  const move = moves[e.key];\n  if (!move) return;\n  x = Math.max(0, Math.min(180, x + move[0]));\n  y = Math.max(0, Math.min(100, y + move[1]));\n  box.style.left = x + "px";\n  box.style.top = y + "px";\n});`,
      tests: [
        ['(() => { const k = (key) => document.dispatchEvent(new KeyboardEvent("keydown", { key })); k("ArrowRight"); k("ArrowRight"); k("ArrowDown"); const b = document.querySelector("#box"); return b.style.left === "20px" && b.style.top === "10px"; })()', 'Moves right and down'],
        ['(() => { const k = (key) => document.dispatchEvent(new KeyboardEvent("keydown", { key })); for (let i = 0; i < 5; i++) k("ArrowUp"); for (let i = 0; i < 30; i++) k("ArrowRight"); const b = document.querySelector("#box"); return b.style.top === "0px" && b.style.left === "180px"; })()', 'Stays inside the area'],
      ],
    },
    {
      t: 'Saving with localStorage',
      b: `\`localStorage\` keeps small bits of text **between visits** to a page (or runs of a packed app!):

\`\`\`js
localStorage.setItem("theme", "dark");
localStorage.getItem("theme");   // "dark" — even after restarting
\`\`\`

Only strings are stored, so use \`JSON.stringify\` / \`JSON.parse\` for anything else.

(In this lesson a pretend \`storage\` object stands in for \`localStorage\` — it works the same way.)`,
      dom: `<input id="note" placeholder="Write a note">\n<button id="save">Save</button>`,
      code: `const storage = { data: {}, getItem(k) { return this.data[k] ?? null; }, setItem(k, v) { this.data[k] = String(v); } };\nstorage.setItem("note", "Remember the milk");\n// 1. When the page starts, put the saved "note" (if any) into #note\n// 2. When #save is clicked, store #note's value under "note"\n`,
      sol: `const storage = { data: {}, getItem(k) { return this.data[k] ?? null; }, setItem(k, v) { this.data[k] = String(v); } };\nstorage.setItem("note", "Remember the milk");\nconst note = document.querySelector("#note");\nnote.value = storage.getItem("note") ?? "";\ndocument.querySelector("#save").addEventListener("click", () => {\n  storage.setItem("note", note.value);\n});`,
      tests: [['document.querySelector("#note").value === "Remember the milk"', 'The saved note is loaded'], ['(() => { document.querySelector("#note").value = "Buy bread"; document.querySelector("#save").click(); return storage.getItem("note") === "Buy bread"; })()', 'Saving stores the new note']],
    },
    {
      t: 'Project: Guess the Number',
      b: `Build a complete little game. The secret number is \`secret\` (1–100).

- On submitting \`#guess-form\` (prevent the reload!), read the number from \`#guess\`
- Show \`"Too low!"\`, \`"Too high!"\` or \`"You got it in N guesses!"\` in \`#message\`
- Count guesses in \`#tries\`
- When won, disable \`#guess\` and the submit button, and show \`#again\`
- Clicking \`#again\` resets everything (the checks keep the same \`secret\`)
- Ignore guesses that are not numbers from 1 to 100 (show \`"Enter 1 to 100"\`, and do not count them)`,
      dom: `<form id="guess-form"><input id="guess" type="number"><button id="go">Guess</button></form>\n<p id="message">Guess a number from 1 to 100</p>\n<p>Tries: <span id="tries">0</span></p>\n<button id="again" hidden>Play again</button>`,
      code: `const secret = 42;\n`,
      sol: 'const secret = 42;\nlet tries = 0;\nconst $ = (s) => document.querySelector(s);\nfunction show(msg) {\n  $("#message").textContent = msg;\n  $("#tries").textContent = tries;\n}\n$("#guess-form").addEventListener("submit", (e) => {\n  e.preventDefault();\n  const n = Number($("#guess").value);\n  if (!Number.isInteger(n) || n < 1 || n > 100 || $("#guess").value.trim() === "") {\n    show("Enter 1 to 100");\n    return;\n  }\n  tries++;\n  if (n < secret) show("Too low!");\n  else if (n > secret) show("Too high!");\n  else {\n    show(`You got it in ${tries} guesses!`);\n    $("#guess").disabled = true;\n    $("#go").disabled = true;\n    $("#again").hidden = false;\n  }\n});\n$("#again").addEventListener("click", () => {\n  tries = 0;\n  show("Guess a number from 1 to 100");\n  $("#guess").disabled = false;\n  $("#go").disabled = false;\n  $("#guess").value = "";\n  $("#again").hidden = true;\n});',
      tests: [
        ['(() => { const g = (v) => { document.querySelector("#guess").value = v; document.querySelector("#guess-form").dispatchEvent(new Event("submit", { cancelable: true })); return document.querySelector("#message").textContent; }; return g("10") === "Too low!" && g("90") === "Too high!"; })()', 'Too low / too high hints'],
        ['(() => { document.querySelector("#guess").value = "500"; document.querySelector("#guess-form").dispatchEvent(new Event("submit", { cancelable: true })); return document.querySelector("#message").textContent === "Enter 1 to 100" && document.querySelector("#tries").textContent === "2"; })()', 'Invalid guesses are rejected and not counted'],
        ['(() => { document.querySelector("#guess").value = "42"; document.querySelector("#guess-form").dispatchEvent(new Event("submit", { cancelable: true })); return document.querySelector("#message").textContent === "You got it in 3 guesses!" && document.querySelector("#guess").disabled && !document.querySelector("#again").hidden; })()', 'Winning ends the game'],
        ['(() => { document.querySelector("#again").click(); return document.querySelector("#tries").textContent === "0" && !document.querySelector("#guess").disabled && document.querySelector("#again").hidden; })()', 'Play again resets'],
      ],
    },
  ],
});
