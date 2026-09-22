registerModule({
  id: 'async',
  title: 'Asynchronous JavaScript',
  icon: '⏳',
  lessons: [
    {
      t: 'Doing Things Later',
      b: `\`setTimeout(fn, ms)\` runs a function **later**, after at least \`ms\` milliseconds:

\`\`\`js
setTimeout(() => console.log("Ding!"), 1000);  // after 1 second
console.log("Waiting…");                        // runs first!
\`\`\`

The rest of your program does not stop and wait — it carries on straight away.`,
      code: `// Print "Ready…" now, and "Go!" after 200 milliseconds using setTimeout\n`,
      sol: `console.log("Ready…");\nsetTimeout(() => console.log("Go!"), 200);`,
      tests: [['$out() === "Ready…\\nGo!"', 'Print `Ready…` then, later, `Go!`'], ['$src.includes("setTimeout")', 'Use `setTimeout`']],
      wait: 400,
    },
    {
      t: 'Who Goes First?',
      b: `Even with a delay of \`0\`, a timeout callback waits until the current code has **completely finished**. JavaScript runs one thing at a time, and scheduled callbacks queue up behind it (this is the **event loop**).

\`\`\`js
setTimeout(() => console.log("B"), 0);
console.log("A");
// A, then B
\`\`\``,
      code: `setTimeout(() => console.log("3"), 0);\nconsole.log("1");\n// Add ONE line so the output is 1, 2, 3 — print "2" in the right place\n`,
      sol: `setTimeout(() => console.log("3"), 0);\nconsole.log("1");\nconsole.log("2");`,
      tests: [['$out() === "1\\n2\\n3"', 'Output is 1, 2, 3']],
      wait: 100,
    },
    {
      t: 'Repeating with setInterval',
      b: `\`setInterval\` calls a function **repeatedly**. It returns an id; pass that to \`clearInterval\` to stop it:

\`\`\`js
let n = 0;
const id = setInterval(() => {
  n++;
  console.log("tick", n);
  if (n === 3) clearInterval(id);
}, 100);
\`\`\``,
      code: `// Using setInterval every 50 ms, print "Beep 1", "Beep 2", "Beep 3", then stop\n`,
      sol: `let n = 0;\nconst id = setInterval(() => {\n  n++;\n  console.log("Beep " + n);\n  if (n === 3) clearInterval(id);\n}, 50);`,
      tests: [['$out() === "Beep 1\\nBeep 2\\nBeep 3"', 'Beep exactly three times'], ['$src.includes("clearInterval")', 'Stop it with `clearInterval`']],
      wait: 500,
    },
    {
      t: 'Callbacks',
      b: `How does code get a result that is only ready **later**? The oldest answer: pass in a **callback** to be called when it is done.

\`\`\`js
function loadUser(id, callback) {
  setTimeout(() => callback({ id, name: "Ada" }), 100);
}
loadUser(1, (user) => console.log(user.name));
\`\`\``,
      code: `// Write cookPasta(callback): after 100 ms, call callback with the string "Pasta is ready"\n// Then use it to print the message\n`,
      sol: `function cookPasta(callback) {\n  setTimeout(() => callback("Pasta is ready"), 100);\n}\ncookPasta((msg) => console.log(msg));`,
      tests: [['$out() === "Pasta is ready"', 'Print `Pasta is ready`'], ['$src.includes("setTimeout")', 'Use `setTimeout` inside `cookPasta`']],
      wait: 300,
    },
    {
      t: 'Callback Chains',
      b: `When each step needs the previous one's result, callbacks nest deeper and deeper — nicknamed **callback hell**:

\`\`\`js
step1((a) => {
  step2(a, (b) => {
    step3(b, (c) => {
      console.log("done", c);
    });
  });
});
\`\`\`

It works, but it is hard to read. Promises (next!) fix this.`,
      code: `function later(value, cb) {\n  setTimeout(() => cb(value), 20);\n}\n// Using later three times in a chain, print "one", then "two", then "three"\n`,
      sol: `function later(value, cb) {\n  setTimeout(() => cb(value), 20);\n}\nlater("one", (a) => {\n  console.log(a);\n  later("two", (b) => {\n    console.log(b);\n    later("three", (c) => console.log(c));\n  });\n});`,
      tests: [['$out() === "one\\ntwo\\nthree"', 'Print one, two, three in order']],
      wait: 300,
    },
    {
      t: 'Promises',
      b: `A **Promise** is an object representing a value that will arrive later. It is *pending*, then either **fulfilled** (with a value) or **rejected** (with an error).

\`\`\`js
const p = new Promise((resolve, reject) => {
  setTimeout(() => resolve("🍕"), 100);
});
p.then((food) => console.log("Got", food));
\`\`\``,
      code: `// Create pizza: a Promise that resolves to "Pepperoni" after 50 ms\n// Then use .then to print the value\n`,
      sol: `const pizza = new Promise((resolve) => {\n  setTimeout(() => resolve("Pepperoni"), 50);\n});\npizza.then((p) => console.log(p));`,
      tests: [['pizza instanceof Promise', '`pizza` is a Promise'], ['pizza.then((v) => v === "Pepperoni")', 'It resolves to "Pepperoni"'], ['$out() === "Pepperoni"', 'Print it with `.then`']],
      wait: 200,
    },
    {
      t: 'A delay Helper',
      b: `Wrapping \`setTimeout\` in a promise is so useful that almost every project has this helper:

\`\`\`js
const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
delay(500).then(() => console.log("half a second later"));
\`\`\``,
      code: `// Write delay(ms) returning a promise that resolves after ms milliseconds\n`,
      sol: `const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));`,
      tests: [['delay(10) instanceof Promise', 'Returns a Promise'], ['(async () => { const t = Date.now(); await delay(80); return Date.now() - t >= 70; })()', 'Waits roughly the right time']],
    },
    {
      t: 'Chaining then',
      b: `\`.then\` returns a **new** promise, so you can chain steps in a flat list. Whatever one step returns is passed to the next:

\`\`\`js
Promise.resolve(2)
  .then((n) => n * 10)
  .then((n) => n + 1)
  .then((n) => console.log(n));   // 21
\`\`\`

If a step returns a promise, the chain waits for it.`,
      code: `// Starting from Promise.resolve("js"), chain three .then steps:\n// uppercase it, add " rocks", then print it\n`,
      sol: `Promise.resolve("js")\n  .then((s) => s.toUpperCase())\n  .then((s) => s + " rocks")\n  .then((s) => console.log(s));`,
      tests: [['$out() === "JS rocks"', 'Print `JS rocks`'], ['($src.match(/\\.then\\(/g) || []).length >= 3', 'Use three `.then` steps']],
      wait: 50,
    },
    {
      t: 'Rejections and catch',
      b: `A promise rejects when something fails. \`.catch\` handles the error, and \`.finally\` runs either way:

\`\`\`js
new Promise((_, reject) => reject(new Error("No signal")))
  .then(() => console.log("never"))
  .catch((err) => console.log("Failed:", err.message))
  .finally(() => console.log("done"));
\`\`\``,
      code: `function checkAge(age) {\n  return new Promise((resolve, reject) => {\n    // resolve with "Welcome" if age >= 18, otherwise reject with new Error("Too young")\n  });\n}\n`,
      sol: `function checkAge(age) {\n  return new Promise((resolve, reject) => {\n    if (age >= 18) resolve("Welcome");\n    else reject(new Error("Too young"));\n  });\n}`,
      tests: [['checkAge(30).then((v) => v === "Welcome")', 'Adults are welcomed'], ['checkAge(10).then(() => false, (e) => e.message === "Too young")', 'Others are rejected']],
    },
    {
      t: 'async Functions',
      b: `An \`async\` function **always returns a promise**. Returning a value fulfils it; throwing rejects it:

\`\`\`js
async function getNumber() {
  return 42;
}
getNumber().then(console.log);   // 42
\`\`\``,
      code: `// Write an async function getGreeting(name) returning "Hi, " + name\n`,
      sol: `async function getGreeting(name) {\n  return "Hi, " + name;\n}`,
      tests: [['getGreeting("Ana") instanceof Promise', 'It returns a Promise'], ['getGreeting("Ana").then((v) => v === "Hi, Ana")', 'It resolves to "Hi, Ana"']],
    },
    {
      t: 'await',
      b: `Inside an \`async\` function, \`await\` pauses until a promise settles, then gives you its value. Async code reads like normal code again:

\`\`\`js
async function brew() {
  console.log("Boiling…");
  await delay(1000);
  console.log("Tea is ready");
}
\`\`\`

(In this editor you can also use \`await\` at the top level.)`,
      code: `const delay = (ms) => new Promise((r) => setTimeout(r, ms));\n// Write async function launch() that prints 3, 2, 1 with 30 ms between each, then "Liftoff!"\n// Then call it with await\n`,
      sol: `const delay = (ms) => new Promise((r) => setTimeout(r, ms));\nasync function launch() {\n  for (let i = 3; i >= 1; i--) {\n    console.log(i);\n    await delay(30);\n  }\n  console.log("Liftoff!");\n}\nawait launch();`,
      tests: [['$out() === "3\\n2\\n1\\nLiftoff!"', 'Count down then lift off'], ['$src.includes("await")', 'Use `await`']],
    },
    {
      t: 'Errors with await',
      b: `With \`await\`, a rejected promise **throws** — so ordinary \`try\`/\`catch\` handles it:

\`\`\`js
try {
  const data = await loadData();
} catch (err) {
  console.log("Could not load:", err.message);
}
\`\`\``,
      code: `async function flakyLoad() {\n  throw new Error("Server is down");\n}\n// Write async safeLoad() that awaits flakyLoad() and returns its result,\n// or returns "offline: " + the error message if it fails\n`,
      sol: `async function flakyLoad() {\n  throw new Error("Server is down");\n}\nasync function safeLoad() {\n  try {\n    return await flakyLoad();\n  } catch (err) {\n    return "offline: " + err.message;\n  }\n}`,
      tests: [['safeLoad().then((v) => v === "offline: Server is down")', '`safeLoad()` resolves to "offline: Server is down"']],
    },
    {
      t: 'All at Once',
      b: `Awaiting things one after another is slow when they do not depend on each other. \`Promise.all\` runs them **in parallel** and waits for all of them:

\`\`\`js
const [user, posts] = await Promise.all([loadUser(), loadPosts()]);
\`\`\`

If any one rejects, \`Promise.all\` rejects.`,
      code: `const delay = (ms, value) => new Promise((r) => setTimeout(() => r(value), ms));\n// Write async loadAll() returning ["a", "b", "c"] from delay(100, "a"), delay(100, "b"), delay(100, "c")\n// It must take about 100 ms in total, not 300\n`,
      sol: `const delay = (ms, value) => new Promise((r) => setTimeout(() => r(value), ms));\nasync function loadAll() {\n  return Promise.all([delay(100, "a"), delay(100, "b"), delay(100, "c")]);\n}`,
      tests: [['(async () => { const t = Date.now(); const r = await loadAll(); return r.join() === "a,b,c" && Date.now() - t < 250; })()', 'Returns all three in about 100 ms']],
    },
    {
      t: 'The Fastest Wins',
      b: `\`Promise.race\` settles as soon as the **first** promise settles. A classic use is a timeout:

\`\`\`js
const timeout = (ms) => new Promise((_, reject) =>
  setTimeout(() => reject(new Error("Too slow")), ms));

await Promise.race([slowThing(), timeout(2000)]);
\`\`\``,
      code: `// Write withTimeout(promise, ms): resolves like promise, unless ms passes first,\n// in which case it rejects with new Error("Timed out")\n`,
      sol: `function withTimeout(promise, ms) {\n  const timer = new Promise((_, reject) => setTimeout(() => reject(new Error("Timed out")), ms));\n  return Promise.race([promise, timer]);\n}`,
      tests: [['withTimeout(new Promise((r) => setTimeout(() => r("fast"), 10)), 200).then((v) => v === "fast")', 'Fast promises win'], ['withTimeout(new Promise((r) => setTimeout(r, 300)), 20).then(() => false, (e) => e.message === "Timed out")', 'Slow ones time out']],
    },
    {
      t: 'Waiting for Everything',
      b: `\`Promise.allSettled\` waits for all promises and **never rejects**. It tells you how each one went:

\`\`\`js
const results = await Promise.allSettled([ok(), fails()]);
// [{ status: "fulfilled", value: … }, { status: "rejected", reason: Error }]
\`\`\``,
      code: `// Write async countSuccesses(promises) returning how many of them fulfilled\n`,
      sol: `async function countSuccesses(promises) {\n  const results = await Promise.allSettled(promises);\n  return results.filter((r) => r.status === "fulfilled").length;\n}`,
      tests: [['countSuccesses([Promise.resolve(1), Promise.reject(new Error("x")), Promise.resolve(3)]).then((n) => n === 2)', 'Counts 2 of 3'], ['$src.includes("allSettled")', 'Use `Promise.allSettled`']],
    },
    {
      t: 'The First Success',
      b: `\`Promise.any\` resolves with the **first promise to succeed**, ignoring failures — it only rejects if **all** fail. Perfect for trying several mirrors:

\`\`\`js
const file = await Promise.any([fromServerA(), fromServerB()]);
\`\`\``,
      code: `const delay = (ms, v, fail) => new Promise((res, rej) => setTimeout(() => (fail ? rej(new Error(v)) : res(v)), ms));\n// Write async firstMirror() returning the first SUCCESSFUL result of:\n// delay(10, "mirror A", true), delay(40, "mirror B"), delay(80, "mirror C")\n`,
      sol: `const delay = (ms, v, fail) => new Promise((res, rej) => setTimeout(() => (fail ? rej(new Error(v)) : res(v)), ms));\nasync function firstMirror() {\n  return Promise.any([delay(10, "mirror A", true), delay(40, "mirror B"), delay(80, "mirror C")]);\n}`,
      tests: [['firstMirror().then((v) => v === "mirror B")', 'Resolves with "mirror B"'], ['$src.includes("Promise.any")', 'Use `Promise.any`']],
    },
    {
      t: 'Talking to a Server',
      b: `Real apps get data with \`fetch(url)\`, which returns a promise of a response; \`await response.json()\` then parses it. Here we use a pretend \`fakeFetch\` so it works offline:

\`\`\`js
const res = await fetch("https://api.example.com/users/1");
if (!res.ok) throw new Error("HTTP " + res.status);
const user = await res.json();
\`\`\``,
      code: `const db = { 1: { name: "Ada" }, 2: { name: "Alan" } };\nfunction fakeFetch(url) {\n  const id = url.split("/").pop();\n  return new Promise((resolve) => setTimeout(() => resolve({\n    ok: id in db,\n    status: id in db ? 200 : 404,\n    json: async () => db[id],\n  }), 20));\n}\n// Write async getUserName(id): fakeFetch("/users/" + id); if !ok throw Error("HTTP " + status);\n// otherwise return the parsed user's name\n`,
      sol: `const db = { 1: { name: "Ada" }, 2: { name: "Alan" } };\nfunction fakeFetch(url) {\n  const id = url.split("/").pop();\n  return new Promise((resolve) => setTimeout(() => resolve({\n    ok: id in db,\n    status: id in db ? 200 : 404,\n    json: async () => db[id],\n  }), 20));\n}\nasync function getUserName(id) {\n  const res = await fakeFetch("/users/" + id);\n  if (!res.ok) throw new Error("HTTP " + res.status);\n  const user = await res.json();\n  return user.name;\n}`,
      tests: [['getUserName(2).then((n) => n === "Alan")', 'Gets user 2\'s name'], ['getUserName(9).then(() => false, (e) => e.message === "HTTP 404")', 'Missing users throw "HTTP 404"']],
    },
    {
      t: 'Loops with await',
      b: `\`await\` inside a \`for…of\` loop processes items **one at a time**, in order — sometimes exactly what you need (e.g. to avoid overloading a server):

\`\`\`js
for (const url of urls) {
  const page = await download(url);
  save(page);
}
\`\`\`

⚠️ \`forEach\` does **not** wait for async callbacks. Use \`for…of\`.`,
      code: `const delay = (ms) => new Promise((r) => setTimeout(r, ms));\nasync function processItem(x) {\n  await delay(10);\n  return x * 2;\n}\n// Write async processAll(items) returning the results IN ORDER, processing one at a time\n`,
      sol: `const delay = (ms) => new Promise((r) => setTimeout(r, ms));\nasync function processItem(x) {\n  await delay(10);\n  return x * 2;\n}\nasync function processAll(items) {\n  const out = [];\n  for (const item of items) {\n    out.push(await processItem(item));\n  }\n  return out;\n}`,
      tests: [['processAll([1, 2, 3]).then((r) => r.join() === "2,4,6")', 'Returns [2, 4, 6]'], ['$src.includes("for (const")', 'Use a `for…of` loop']],
    },
    {
      t: 'Trying Again',
      b: `Networks fail sometimes. A **retry** helper calls an async function again when it fails, up to a limit:

\`\`\`js
for (let attempt = 1; attempt <= times; attempt++) {
  try { return await fn(); }
  catch (err) { if (attempt === times) throw err; }
}
\`\`\``,
      code: `// Write async retry(fn, times): call fn() up to times times; return the first success,\n// or rethrow the last error if every attempt fails\n`,
      sol: `async function retry(fn, times) {\n  for (let attempt = 1; attempt <= times; attempt++) {\n    try {\n      return await fn();\n    } catch (err) {\n      if (attempt === times) throw err;\n    }\n  }\n}`,
      tests: [['(async () => { let n = 0; const r = await retry(async () => { n++; if (n < 3) throw new Error("no"); return "yes"; }, 5); return r === "yes" && n === 3; })()', 'Succeeds on the third try'], ['(async () => { let n = 0; try { await retry(async () => { n++; throw new Error("fail " + n); }, 2); } catch (e) { return e.message === "fail 2" && n === 2; } return false; })()', 'Gives up after the limit']],
    },
    {
      t: 'Project: Download Manager',
      b: `Write \`async downloadAll(urls, download, limit)\`. \`download(url)\` returns a promise. Download every URL, but with **at most \`limit\` running at the same time**. Return the results in the **same order** as \`urls\`.

This is a real-world pattern called a *concurrency pool*. Tip: start \`limit\` "workers" that each keep taking the next index until none are left, and use \`Promise.all\` to wait for the workers.`,
      code: `async function downloadAll(urls, download, limit) {\n}\n`,
      sol: `async function downloadAll(urls, download, limit) {\n  const results = new Array(urls.length);\n  let next = 0;\n  async function worker() {\n    while (next < urls.length) {\n      const i = next++;\n      results[i] = await download(urls[i]);\n    }\n  }\n  const workers = Array.from({ length: Math.min(limit, urls.length) }, worker);\n  await Promise.all(workers);\n  return results;\n}`,
      tests: [
        ['(async () => { const dl = (u) => new Promise((r) => setTimeout(() => r(u.toUpperCase()), 30 - u.length)); const r = await downloadAll(["a", "bb", "ccc", "d"], dl, 2); return r.join() === "A,BB,CCC,D"; })()', 'Results keep the original order'],
        ['(async () => { let running = 0, peak = 0; const dl = async (u) => { running++; peak = Math.max(peak, running); await new Promise((r) => setTimeout(r, 15)); running--; return u; }; await downloadAll([1, 2, 3, 4, 5, 6, 7], dl, 3); return peak === 3; })()', 'Never more than `limit` at once'],
        ['downloadAll([], async (u) => u, 3).then((r) => r.length === 0)', 'Works with no URLs'],
      ],
    },
  ],
});
