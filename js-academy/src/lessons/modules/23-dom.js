registerModule({
  id: 'dom',
  title: 'The DOM: Changing Web Pages',
  icon: '🌐',
  lessons: [
    {
      t: 'Meet the DOM',
      b: `When a browser loads a web page, it turns the HTML into a tree of objects called the **DOM** (Document Object Model). JavaScript can read and change that tree — and the page updates instantly.

The whole page is available as \`document\`:

\`\`\`js
document.title = "New tab title";
\`\`\`

These lessons run in a real page: see the **Page preview** below your code.`,
      dom: `<h1>My page</h1>\n<p>Change my title!</p>`,
      code: `// Set the document's title to "Hello DOM"\n`,
      sol: `document.title = "Hello DOM";`,
      tests: [['document.title === "Hello DOM"', 'The title is `"Hello DOM"`']],
    },
    {
      t: 'Finding an Element',
      b: `\`document.querySelector(selector)\` finds the **first** element matching a CSS selector. \`#id\` selects by id:

\`\`\`js
const heading = document.querySelector("#main-title");
heading.textContent = "Welcome!";
\`\`\`

\`textContent\` is the text inside an element.`,
      dom: `<h1 id="greeting">Loading…</h1>`,
      code: `// Find the element with id "greeting" and change its text to "Hello, world!"\n`,
      sol: `document.querySelector("#greeting").textContent = "Hello, world!";`,
      tests: [['document.querySelector("#greeting").textContent === "Hello, world!"', 'The heading says `Hello, world!`']],
    },
    {
      t: 'Selecting by Class and Tag',
      b: `Any CSS selector works:

\`\`\`js
document.querySelector("p")          // first <p>
document.querySelector(".warning")   // first element with class="warning"
document.querySelector("ul li")      // first <li> inside a <ul>
\`\`\``,
      dom: `<p>Normal paragraph</p>\n<p class="note">Old note</p>\n<ul><li>First</li><li>Second</li></ul>`,
      code: `// Change the .note paragraph's text to "Updated note"\n// Change the first list item's text to "Number one"\n`,
      sol: `document.querySelector(".note").textContent = "Updated note";\ndocument.querySelector("ul li").textContent = "Number one";`,
      tests: [['document.querySelector(".note").textContent === "Updated note"', 'The note is updated'], ['document.querySelector("li").textContent === "Number one"', 'The first item is updated'], ['document.querySelectorAll("p")[0].textContent === "Normal paragraph"', 'The other paragraph is untouched']],
    },
    {
      t: 'Selecting Many Elements',
      b: `\`querySelectorAll\` returns **all** matches as a list you can loop over:

\`\`\`js
for (const item of document.querySelectorAll("li")) {
  item.textContent = item.textContent.toUpperCase();
}
\`\`\``,
      dom: `<ul>\n  <li class="price">4.5</li>\n  <li class="price">10</li>\n  <li class="price">3.25</li>\n</ul>`,
      code: `// Format every .price as dollars with 2 decimals, e.g. "$4.50"\n`,
      sol: `for (const el of document.querySelectorAll(".price")) {\n  el.textContent = "$" + Number(el.textContent).toFixed(2);\n}`,
      tests: [['[...document.querySelectorAll(".price")].map((e) => e.textContent).join() === "$4.50,$10.00,$3.25"', 'All prices are formatted'], ['$src.includes("querySelectorAll")', 'Use `querySelectorAll`']],
    },
    {
      t: 'textContent vs innerHTML',
      b: `\`innerHTML\` reads and writes **HTML**, so tags become real elements:

\`\`\`js
box.innerHTML = "<b>Bold!</b>";   // shows bold text
box.textContent = "<b>Bold!</b>"; // shows the literal characters <b>
\`\`\`

⚠️ Never put text typed by users into \`innerHTML\` — they could inject their own HTML and scripts. Use \`textContent\` for text.`,
      dom: `<div id="box"></div>\n<div id="safe"></div>`,
      code: `const userInput = "<img src=x onerror=alert(1)>";\n// Put "<em>Welcome</em>" into #box as HTML (so it is emphasised)\n// Put userInput into #safe as plain TEXT\n`,
      sol: `const userInput = "<img src=x onerror=alert(1)>";\ndocument.querySelector("#box").innerHTML = "<em>Welcome</em>";\ndocument.querySelector("#safe").textContent = userInput;`,
      tests: [['document.querySelector("#box em") !== null', '#box contains a real `<em>` element'], ['document.querySelector("#safe img") === null && document.querySelector("#safe").textContent === "<img src=x onerror=alert(1)>"', 'The user input is shown as text, not HTML']],
    },
    {
      t: 'Changing Styles',
      b: `Every element has a \`style\` object. CSS property names are written in camelCase:

\`\`\`js
el.style.color = "crimson";
el.style.backgroundColor = "gold";   // background-color
el.style.fontSize = "24px";
\`\`\``,
      dom: `<p id="alert">Warning: low battery</p>`,
      code: `// Make #alert red text ("red") on a yellow background ("yellow"), bold ("bold")\n`,
      sol: `const alertBox = document.querySelector("#alert");\nalertBox.style.color = "red";\nalertBox.style.backgroundColor = "yellow";\nalertBox.style.fontWeight = "bold";`,
      tests: [['document.querySelector("#alert").style.color === "red"', 'Red text'], ['document.querySelector("#alert").style.backgroundColor === "yellow"', 'Yellow background'], ['document.querySelector("#alert").style.fontWeight === "bold"', 'Bold']],
    },
    {
      t: 'Adding and Removing Classes',
      b: `Changing styles one by one gets messy. Usually it is better to define CSS classes and switch them with \`classList\`:

\`\`\`js
el.classList.add("active");
el.classList.remove("hidden");
el.classList.toggle("open");       // add if missing, remove if present
el.classList.contains("active");   // true / false
\`\`\``,
      dom: `<style>.done { text-decoration: line-through; color: gray; } .urgent { color: red; }</style>\n<ul>\n  <li id="t1" class="urgent">Pay bills</li>\n  <li id="t2">Walk dog</li>\n</ul>`,
      code: `// Mark #t1 as done: add the class "done" and remove "urgent"\n// Toggle "urgent" on #t2\n`,
      sol: `const t1 = document.querySelector("#t1");\nt1.classList.add("done");\nt1.classList.remove("urgent");\ndocument.querySelector("#t2").classList.toggle("urgent");`,
      tests: [['document.querySelector("#t1").className === "done"', '#t1 has only "done"'], ['document.querySelector("#t2").classList.contains("urgent")', '#t2 is urgent'], ['$src.includes("classList")', 'Use `classList`']],
    },
    {
      t: 'Attributes',
      b: `Attributes like \`src\`, \`href\` and \`alt\` can be read and changed — directly as properties, or with \`getAttribute\` / \`setAttribute\`:

\`\`\`js
link.href = "https://example.com";
img.setAttribute("alt", "A cat");
img.getAttribute("alt");   // "A cat"
\`\`\``,
      dom: `<a id="docs" href="#">Documentation</a>\n<img id="logo" src="" alt="">`,
      code: `// Point #docs at "https://developer.mozilla.org"\n// Set #logo's alt text to "Company logo" using setAttribute\n`,
      sol: `document.querySelector("#docs").href = "https://developer.mozilla.org";\ndocument.querySelector("#logo").setAttribute("alt", "Company logo");`,
      tests: [['document.querySelector("#docs").getAttribute("href").startsWith("https://developer.mozilla.org")', 'The link points to MDN'], ['document.querySelector("#logo").getAttribute("alt") === "Company logo"', 'The logo has alt text'], ['$src.includes("setAttribute")', 'Use `setAttribute`']],
    },
    {
      t: 'Creating Elements',
      b: `Build new elements with \`document.createElement\`, fill them in, then add them to the page with \`append\`:

\`\`\`js
const p = document.createElement("p");
p.textContent = "I am new here";
document.body.append(p);
\`\`\``,
      dom: `<div id="messages"></div>`,
      code: `// Create a <p> with class "msg" and text "You have mail", and append it to #messages\n`,
      sol: `const p = document.createElement("p");\np.className = "msg";\np.textContent = "You have mail";\ndocument.querySelector("#messages").append(p);`,
      tests: [['document.querySelector("#messages p.msg")?.textContent === "You have mail"', 'The message is in #messages'], ['$src.includes("createElement")', 'Use `createElement`']],
    },
    {
      t: 'Lists from Data',
      b: `Combine loops and \`createElement\` to turn data into page content — the heart of every web app:

\`\`\`js
for (const name of names) {
  const li = document.createElement("li");
  li.textContent = name;
  list.append(li);
}
\`\`\``,
      dom: `<h2>Planets</h2>\n<ol id="planets"></ol>`,
      code: `const planets = ["Mercury", "Venus", "Earth", "Mars"];\n// Add an <li> to #planets for each planet\n`,
      sol: `const planets = ["Mercury", "Venus", "Earth", "Mars"];\nconst list = document.querySelector("#planets");\nfor (const planet of planets) {\n  const li = document.createElement("li");\n  li.textContent = planet;\n  list.append(li);\n}`,
      tests: [['[...document.querySelectorAll("#planets li")].map((li) => li.textContent).join() === "Mercury,Venus,Earth,Mars"', 'Each planet is listed in order']],
    },
    {
      t: 'Removing Elements',
      b: `Call \`remove()\` on an element to take it off the page:

\`\`\`js
document.querySelector(".ad").remove();
\`\`\``,
      dom: `<ul id="list">\n  <li>Keep</li>\n  <li class="spam">Buy now!!!</li>\n  <li>Keep too</li>\n  <li class="spam">Click here</li>\n</ul>`,
      code: `// Remove every .spam item\n`,
      sol: `for (const el of document.querySelectorAll(".spam")) el.remove();`,
      tests: [['document.querySelectorAll(".spam").length === 0', 'No spam left'], ['document.querySelectorAll("#list li").length === 2', 'The other items stay']],
    },
    {
      t: 'Inserting in the Right Place',
      b: `\`append\` adds at the end. Other options:

\`\`\`js
list.prepend(item);        // at the start
ref.before(item);          // just before ref
ref.after(item);           // just after ref
\`\`\``,
      dom: `<ol id="steps">\n  <li id="mix">Mix</li>\n  <li id="bake">Bake</li>\n</ol>`,
      code: `// Add "Preheat oven" at the start of #steps, and "Pour into tin" between Mix and Bake\n`,
      sol: `const make = (text) => {\n  const li = document.createElement("li");\n  li.textContent = text;\n  return li;\n};\ndocument.querySelector("#steps").prepend(make("Preheat oven"));\ndocument.querySelector("#mix").after(make("Pour into tin"));`,
      tests: [['[...document.querySelectorAll("#steps li")].map((li) => li.textContent).join() === "Preheat oven,Mix,Pour into tin,Bake"', 'The steps are in the right order']],
    },
    {
      t: 'Moving Around the Tree',
      b: `Elements know their relatives:

\`\`\`js
el.parentElement       // the element containing it
el.children            // its child elements
el.firstElementChild
el.nextElementSibling
el.closest(".card")    // nearest ancestor (or itself) matching a selector
\`\`\``,
      dom: `<div class="card" id="c1"><h3>Title</h3><p><span id="deep">Details</span></p></div>`,
      code: `const deep = document.querySelector("#deep");\n// Create card: the nearest .card containing #deep (use closest)\n// Create childCount: how many child elements card has\n`,
      sol: `const deep = document.querySelector("#deep");\nconst card = deep.closest(".card");\nconst childCount = card.children.length;`,
      tests: [['card === document.querySelector("#c1")', '`card` is the #c1 element'], ['childCount === 2', '`childCount` is `2`']],
    },
    {
      t: 'Data Attributes',
      b: `Custom \`data-*\` attributes store extra information on elements. Read them through \`dataset\` (camelCased):

\`\`\`html
<button data-product-id="42" data-price="9.99">Buy</button>
\`\`\`
\`\`\`js
button.dataset.productId   // "42"
button.dataset.price       // "9.99" (always strings)
\`\`\``,
      dom: `<ul>\n  <li data-price="2.5" data-qty="4">Apples</li>\n  <li data-price="1.2" data-qty="10">Eggs</li>\n  <li data-price="3" data-qty="1">Bread</li>\n</ul>\n<p>Total: <span id="total"></span></p>`,
      code: `// Put the total cost (price × qty for every li) into #total with 2 decimals\n`,
      sol: `let total = 0;\nfor (const li of document.querySelectorAll("li")) {\n  total += Number(li.dataset.price) * Number(li.dataset.qty);\n}\ndocument.querySelector("#total").textContent = total.toFixed(2);`,
      tests: [['document.querySelector("#total").textContent === "25.00"', '#total shows `25.00`'], ['$src.includes("dataset")', 'Use `dataset`']],
    },
    {
      t: 'Reading Form Fields',
      b: `Form fields expose what is in them through properties:

\`\`\`js
input.value          // the text in an <input> or <textarea>
checkbox.checked     // true / false
select.value         // the chosen <option>'s value
\`\`\``,
      dom: `<input id="name" value="  Robin  ">\n<input id="agree" type="checkbox" checked>\n<select id="size"><option value="s">Small</option><option value="m" selected>Medium</option></select>\n<p id="summary"></p>`,
      code: `// Put "Robin ordered size m (agreed: true)" into #summary, built from the fields (trim the name)\n`,
      sol: 'const name = document.querySelector("#name").value.trim();\nconst agreed = document.querySelector("#agree").checked;\nconst size = document.querySelector("#size").value;\ndocument.querySelector("#summary").textContent = `${name} ordered size ${size} (agreed: ${agreed})`;',
      tests: [['document.querySelector("#summary").textContent === "Robin ordered size m (agreed: true)"', 'The summary is correct'], ['$src.includes(".value") && $src.includes(".checked")', 'Read `.value` and `.checked`']],
    },
    {
      t: 'Building a Table',
      b: `Tables are just nested elements: \`<tr>\` rows containing \`<td>\` cells. \`insertRow\` and \`insertCell\` are handy shortcuts:

\`\`\`js
const row = table.insertRow();
row.insertCell().textContent = "Ada";
\`\`\``,
      dom: `<table id="scores" border="1"><tr><th>Player</th><th>Score</th></tr></table>`,
      code: `const players = [\n  { name: "Kai", score: 320 },\n  { name: "Mia", score: 450 },\n  { name: "Leo", score: 280 },\n];\n// Add a row for each player to #scores, highest score first\n`,
      sol: `const players = [\n  { name: "Kai", score: 320 },\n  { name: "Mia", score: 450 },\n  { name: "Leo", score: 280 },\n];\nconst table = document.querySelector("#scores");\nfor (const p of players.toSorted((a, b) => b.score - a.score)) {\n  const row = table.insertRow();\n  row.insertCell().textContent = p.name;\n  row.insertCell().textContent = p.score;\n}`,
      tests: [['document.querySelectorAll("#scores tr").length === 4', 'Three rows are added'], ['[...document.querySelectorAll("#scores tr")].slice(1).map((r) => r.cells[0].textContent + r.cells[1].textContent).join() === "Mia450,Kai320,Leo280"', 'Rows are sorted by score']],
    },
    {
      t: 'Showing and Hiding',
      b: `The \`hidden\` property hides an element completely (like \`display: none\`):

\`\`\`js
spinner.hidden = true;    // gone
message.hidden = false;   // visible
\`\`\``,
      dom: `<p id="loading">Loading…</p>\n<div id="content" hidden>Here is your content!</div>`,
      code: `// Hide #loading and show #content\n`,
      sol: `document.querySelector("#loading").hidden = true;\ndocument.querySelector("#content").hidden = false;`,
      tests: [['document.querySelector("#loading").hidden === true', '#loading is hidden'], ['document.querySelector("#content").hidden === false', '#content is visible']],
    },
    {
      t: 'Templates',
      b: `A \`<template>\` holds HTML that is not shown. Clone it to stamp out copies — much tidier than building everything with \`createElement\`:

\`\`\`js
const tpl = document.querySelector("#card-template");
const copy = tpl.content.cloneNode(true);
copy.querySelector(".title").textContent = "Hello";
container.append(copy);
\`\`\``,
      dom: `<template id="contact"><div class="contact"><b class="name"></b> — <span class="email"></span></div></template>\n<div id="contacts"></div>`,
      code: `const people = [\n  { name: "Ada", email: "ada@math.org" },\n  { name: "Alan", email: "alan@bletchley.uk" },\n];\n// For each person, clone the #contact template, fill in .name and .email, and add it to #contacts\n`,
      sol: `const people = [\n  { name: "Ada", email: "ada@math.org" },\n  { name: "Alan", email: "alan@bletchley.uk" },\n];\nconst tpl = document.querySelector("#contact");\nfor (const p of people) {\n  const copy = tpl.content.cloneNode(true);\n  copy.querySelector(".name").textContent = p.name;\n  copy.querySelector(".email").textContent = p.email;\n  document.querySelector("#contacts").append(copy);\n}`,
      tests: [['document.querySelectorAll("#contacts .contact").length === 2', 'Two contacts are shown'], ['document.querySelectorAll("#contacts .name")[1].textContent === "Alan" && document.querySelectorAll("#contacts .email")[0].textContent === "ada@math.org"', 'They are filled in'], ['$src.includes("cloneNode")', 'Clone the template']],
    },
    {
      t: 'Reading the Page',
      b: `The DOM is also great for **reading** information: counting, collecting text, checking state. Here, work out statistics about a to-do list already on the page.`,
      dom: `<ul id="todos">\n  <li class="done">Buy milk</li>\n  <li>Write essay</li>\n  <li class="done">Call gran</li>\n  <li>Fix bike</li>\n  <li>Water plants</li>\n</ul>\n<p id="stats"></p>`,
      code: `// Put "2 of 5 done" into #stats (count the li elements with class "done")\n`,
      sol: 'const all = document.querySelectorAll("#todos li").length;\nconst done = document.querySelectorAll("#todos li.done").length;\ndocument.querySelector("#stats").textContent = `${done} of ${all} done`;',
      tests: [['document.querySelector("#stats").textContent === "2 of 5 done"', '#stats shows `2 of 5 done`'], ['!$src.includes("\\"2 of 5")', 'Count them — do not type the numbers']],
    },
    {
      t: 'Project: Product Grid',
      b: `Render a shop from data. For each product, add to \`#grid\` a \`<div class="card">\` containing:

- an \`<h3>\` with the name
- a \`<p class="price">\` with the price like \`$12.00\`
- if \`stock\` is 0, add the class \`sold-out\` to the card and a \`<span class="badge">\` saying \`Sold out\`

Finally set \`#count\` to \`"3 products"\` (the number of cards).`,
      dom: `<style>\n#grid { display: flex; gap: 8px; }\n.card { border: 1px solid #ccc; border-radius: 8px; padding: 8px; }\n.sold-out { opacity: .5; }\n.badge { background: crimson; color: white; padding: 2px 6px; border-radius: 4px; }\n</style>\n<p id="count"></p>\n<div id="grid"></div>`,
      code: `const products = [\n  { name: "Headphones", price: 59, stock: 4 },\n  { name: "Keyboard", price: 12, stock: 0 },\n  { name: "Mouse", price: 25.5, stock: 11 },\n];\n`,
      sol: 'const products = [\n  { name: "Headphones", price: 59, stock: 4 },\n  { name: "Keyboard", price: 12, stock: 0 },\n  { name: "Mouse", price: 25.5, stock: 11 },\n];\nconst grid = document.querySelector("#grid");\nfor (const p of products) {\n  const card = document.createElement("div");\n  card.className = "card";\n  const h3 = document.createElement("h3");\n  h3.textContent = p.name;\n  const price = document.createElement("p");\n  price.className = "price";\n  price.textContent = "$" + p.price.toFixed(2);\n  card.append(h3, price);\n  if (p.stock === 0) {\n    card.classList.add("sold-out");\n    const badge = document.createElement("span");\n    badge.className = "badge";\n    badge.textContent = "Sold out";\n    card.append(badge);\n  }\n  grid.append(card);\n}\ndocument.querySelector("#count").textContent = `${grid.children.length} products`;',
      tests: [
        ['document.querySelectorAll("#grid .card").length === 3', 'Three cards'],
        ['[...document.querySelectorAll("#grid .card h3")].map((h) => h.textContent).join() === "Headphones,Keyboard,Mouse"', 'Each has a name'],
        ['[...document.querySelectorAll("#grid .price")].map((p) => p.textContent).join() === "$59.00,$12.00,$25.50"', 'Prices are formatted'],
        ['document.querySelectorAll(".sold-out").length === 1 && document.querySelector(".sold-out .badge")?.textContent === "Sold out"', 'The keyboard is marked sold out'],
        ['document.querySelector("#count").textContent === "3 products"', '#count says `3 products`'],
      ],
    },
  ],
});
