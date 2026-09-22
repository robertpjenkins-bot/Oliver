registerModule({
  id: 'generators',
  title: 'Iterators & Generators',
  icon: '♾️',
  lessons: [
    {
      t: 'What Is Iterable?',
      b: `\`for…of\` works on arrays, strings, Maps and Sets because they are **iterable**: they know how to hand out their values one at a time.

Under the hood, an iterable has a method stored under the special key \`Symbol.iterator\`:

\`\`\`js
typeof [][Symbol.iterator]      // "function"
typeof ({})[Symbol.iterator]    // "undefined" — plain objects are not iterable
\`\`\``,
      code: `// Write isIterable(x) returning true if x is not null/undefined and has a Symbol.iterator function\n`,
      sol: `function isIterable(x) {\n  return x != null && typeof x[Symbol.iterator] === "function";\n}`,
      tests: [['isIterable([1]) && isIterable("abc") && isIterable(new Map())', 'Arrays, strings and Maps are iterable'], ['!isIterable({}) && !isIterable(42) && !isIterable(null)', 'Objects, numbers and null are not']],
    },
    {
      t: 'Stepping Through by Hand',
      b: `Calling \`[Symbol.iterator]()\` gives an **iterator**. Each \`next()\` returns \`{ value, done }\`:

\`\`\`js
const it = ["a", "b"][Symbol.iterator]();
it.next();   // { value: "a", done: false }
it.next();   // { value: "b", done: false }
it.next();   // { value: undefined, done: true }
\`\`\`

This is exactly what \`for…of\` does for you.`,
      code: `const colors = ["red", "green", "blue"];\nconst it = colors[Symbol.iterator]();\n// Call it.next() three times, storing the VALUES in a, b, c\n// Then call it once more and store whether it is done in finished\n`,
      sol: `const colors = ["red", "green", "blue"];\nconst it = colors[Symbol.iterator]();\nconst a = it.next().value;\nconst b = it.next().value;\nconst c = it.next().value;\nconst finished = it.next().done;`,
      tests: [['a === "red" && b === "green" && c === "blue"', 'The three values come out in order'], ['finished === true', 'Then it is done']],
    },
    {
      t: 'A Custom Iterable',
      b: `Give any object a \`[Symbol.iterator]\` method returning an iterator, and it works with \`for…of\` and spread:

\`\`\`js
const three = {
  [Symbol.iterator]() {
    let n = 0;
    return { next: () => n < 3 ? { value: ++n, done: false } : { value: undefined, done: true } };
  },
};
[...three]   // [1, 2, 3]
\`\`\``,
      code: `// Create countdown: an iterable object that produces 3, 2, 1\n`,
      sol: `const countdown = {\n  [Symbol.iterator]() {\n    let n = 4;\n    return {\n      next() {\n        n--;\n        return n > 0 ? { value: n, done: false } : { value: undefined, done: true };\n      },\n    };\n  },\n};`,
      tests: [['[...countdown].join() === "3,2,1"', '`[...countdown]` is [3, 2, 1]'], ['[...countdown].length === 3', 'It can be iterated more than once']],
    },
    {
      t: 'Generator Functions',
      b: `Writing iterators by hand is fiddly. A **generator function** (\`function*\`) makes it easy: each \`yield\` hands out one value, then the function **pauses** until the next value is requested.

\`\`\`js
function* greetings() {
  yield "hello";
  yield "hola";
  yield "bonjour";
}
[...greetings()]   // ["hello", "hola", "bonjour"]
\`\`\``,
      code: `// Write a generator function* seasons() yielding "spring", "summer", "autumn", "winter"\n`,
      sol: `function* seasons() {\n  yield "spring";\n  yield "summer";\n  yield "autumn";\n  yield "winter";\n}`,
      tests: [['[...seasons()].join() === "spring,summer,autumn,winter"', 'Yields the four seasons'], ['$src.includes("function*") && $src.includes("yield")', 'Use `function*` and `yield`']],
    },
    {
      t: 'Generators with Loops',
      b: `\`yield\` inside a loop produces a whole sequence:

\`\`\`js
function* evens(limit) {
  for (let i = 0; i <= limit; i += 2) yield i;
}
[...evens(6)]   // [0, 2, 4, 6]
\`\`\``,
      code: `// Write function* range(start, end, step = 1) yielding start, start+step, … while below end\n`,
      sol: `function* range(start, end, step = 1) {\n  for (let i = start; i < end; i += step) yield i;\n}`,
      tests: [['[...range(0, 5)].join() === "0,1,2,3,4"', '`range(0, 5)` is 0…4'], ['[...range(10, 30, 5)].join() === "10,15,20,25"', '`range(10, 30, 5)` steps by 5']],
    },
    {
      t: 'Pausing and Resuming',
      b: `A generator's code only runs when someone asks for the next value. Between \`yield\`s, it is frozen — local variables and all:

\`\`\`js
function* demo() {
  console.log("start");
  yield 1;
  console.log("middle");
  yield 2;
}
const g = demo();   // nothing printed yet!
g.next();           // prints "start"
\`\`\``,
      code: `function* steps() {\n  console.log("A");\n  yield 1;\n  console.log("B");\n  yield 2;\n  console.log("C");\n}\nconst g = steps();\n// Call g.next() exactly TWICE so that only A and B are printed\n`,
      sol: `function* steps() {\n  console.log("A");\n  yield 1;\n  console.log("B");\n  yield 2;\n  console.log("C");\n}\nconst g = steps();\ng.next();\ng.next();`,
      tests: [['$out() === "A\\nB"', 'Only A and B are printed']],
    },
    {
      t: 'Infinite Sequences',
      b: `Because generators are lazy, they can go on **forever** — you just stop asking:

\`\`\`js
function* naturals() {
  let n = 1;
  while (true) yield n++;
}
\`\`\`

Never spread an infinite generator (\`[...naturals()]\` would never finish)! Take only what you need.`,
      code: `function* naturals() {\n  let n = 1;\n  while (true) yield n++;\n}\n// Write take(n, iterable) returning an array of the first n values\n`,
      sol: `function* naturals() {\n  let n = 1;\n  while (true) yield n++;\n}\nfunction take(n, iterable) {\n  const out = [];\n  if (n <= 0) return out;\n  for (const value of iterable) {\n    out.push(value);\n    if (out.length === n) break;\n  }\n  return out;\n}`,
      tests: [['take(5, naturals()).join() === "1,2,3,4,5"', '`take(5, naturals())` is 1…5'], ['take(0, naturals()).length === 0', '`take(0, …)` is empty'], ['take(3, "hello").join("") === "hel"', 'Works on any iterable']],
    },
    {
      t: 'Fibonacci Forever',
      b: `Generators are a lovely way to express endless sequences. Keep two variables and update them after each \`yield\`.`,
      code: `// Write an infinite generator function* fibonacci() yielding 0, 1, 1, 2, 3, 5, 8, …\n`,
      sol: `function* fibonacci() {\n  let [a, b] = [0, 1];\n  while (true) {\n    yield a;\n    [a, b] = [b, a + b];\n  }\n}`,
      tests: [['(() => { const it = fibonacci(); const out = []; for (let i = 0; i < 10; i++) out.push(it.next().value); return out.join() === "0,1,1,2,3,5,8,13,21,34"; })()', 'The first 10 values are correct']],
    },
    {
      t: 'Lazy map',
      b: `A generator can transform another iterable **lazily** — each value is processed only when requested:

\`\`\`js
function* map(iterable, fn) {
  for (const x of iterable) yield fn(x);
}
\`\`\``,
      code: `// Write generator functions map(iterable, fn) and filter(iterable, test)\n`,
      sol: `function* map(iterable, fn) {\n  for (const x of iterable) yield fn(x);\n}\nfunction* filter(iterable, test) {\n  for (const x of iterable) if (test(x)) yield x;\n}`,
      tests: [['[...map([1, 2, 3], (x) => x * 10)].join() === "10,20,30"', '`map` works'], ['[...filter("a1b2c3", (c) => c >= "0" && c <= "9")].join("") === "123"', '`filter` works'], ['(() => { let calls = 0; const it = map([1, 2, 3], (x) => (calls++, x)); it.next(); return calls === 1; })()', '`map` is lazy']],
    },
    {
      t: 'Delegating with yield*',
      b: `\`yield*\` hands over to another iterable, yielding all its values:

\`\`\`js
function* both() {
  yield* [1, 2];
  yield* "ab";
}
[...both()]   // [1, 2, "a", "b"]
\`\`\``,
      code: `// Write function* chain(...iterables) yielding every value of each iterable in turn, using yield*\n`,
      sol: `function* chain(...iterables) {\n  for (const it of iterables) yield* it;\n}`,
      tests: [['[...chain([1, 2], new Set([3]), "45")].join() === "1,2,3,4,5"', 'Chains arrays, sets and strings'], ['$src.includes("yield*")', 'Use `yield*`']],
    },
    {
      t: 'Walking Trees with Generators',
      b: `\`yield*\` plus recursion walks a whole tree, one node at a time:

\`\`\`js
function* walk(node) {
  yield node.name;
  for (const child of node.children) yield* walk(child);
}
\`\`\``,
      code: `const menu = {\n  label: "File",\n  items: [\n    { label: "New", items: [] },\n    { label: "Open Recent", items: [{ label: "a.txt", items: [] }, { label: "b.txt", items: [] }] },\n    { label: "Quit", items: [] },\n  ],\n};\n// Write function* labels(node) yielding every label in the menu tree\n`,
      sol: `const menu = {\n  label: "File",\n  items: [\n    { label: "New", items: [] },\n    { label: "Open Recent", items: [{ label: "a.txt", items: [] }, { label: "b.txt", items: [] }] },\n    { label: "Quit", items: [] },\n  ],\n};\nfunction* labels(node) {\n  yield node.label;\n  for (const child of node.items) yield* labels(child);\n}`,
      tests: [['[...labels(menu)].join() === "File,New,Open Recent,a.txt,b.txt,Quit"', 'Yields every label in order']],
    },
    {
      t: 'Iterable Classes',
      b: `Give a class a generator method called \`*[Symbol.iterator]()\` and its instances work with \`for…of\`:

\`\`\`js
class Team {
  constructor(...members) { this.members = members; }
  *[Symbol.iterator]() {
    yield* this.members;
  }
}
\`\`\``,
      code: `// Write class Grid(width, height) that iterates over every [x, y] pair, row by row\n`,
      sol: `class Grid {\n  constructor(width, height) {\n    this.width = width;\n    this.height = height;\n  }\n  *[Symbol.iterator]() {\n    for (let y = 0; y < this.height; y++) {\n      for (let x = 0; x < this.width; x++) yield [x, y];\n    }\n  }\n}`,
      tests: [['[...new Grid(2, 2)].map((p) => p.join(":")).join() === "0:0,1:0,0:1,1:1"', 'A 2×2 grid gives four cells, row by row'], ['[...new Grid(3, 1)].length === 3', 'A 3×1 grid gives three cells']],
    },
    {
      t: 'Generators Are Iterators',
      b: `A generator object is both an **iterator** (it has \`next()\`) and **iterable** — but only **once**. After it finishes, it stays finished:

\`\`\`js
const g = [1, 2].values();
[...g]   // [1, 2]
[...g]   // [] — already used up!
\`\`\`

To iterate again, call the generator function again.`,
      code: `function* abc() {\n  yield "a";\n  yield "b";\n  yield "c";\n}\nconst g = abc();\nconst first = [...g];\n// Create second: the three letters again (it should NOT be empty)\n`,
      sol: `function* abc() {\n  yield "a";\n  yield "b";\n  yield "c";\n}\nconst g = abc();\nconst first = [...g];\nconst second = [...abc()];`,
      tests: [['first.join("") === "abc"', '`first` is a, b, c'], ['second.join("") === "abc"', '`second` is a, b, c too']],
    },
    {
      t: 'Generators Can Return',
      b: `A \`return\` inside a generator finishes it. The returned value shows up with \`done: true\`, and is **not** included by \`for…of\` or spread:

\`\`\`js
function* g() { yield 1; return "end"; }
const it = g();
it.next();   // { value: 1, done: false }
it.next();   // { value: "end", done: true }
[...g()]     // [1]
\`\`\``,
      code: `// Write function* upTo(limit, items) that yields items until it has yielded limit of them,\n// then returns "limit reached". If items run out first, it just ends.\n`,
      sol: `function* upTo(limit, items) {\n  let count = 0;\n  for (const item of items) {\n    if (count === limit) return "limit reached";\n    yield item;\n    count++;\n  }\n}`,
      tests: [['[...upTo(2, ["a", "b", "c"])].join() === "a,b"', 'Stops after 2'], ['(() => { const it = upTo(1, [7, 8]); it.next(); const r = it.next(); return r.done && r.value === "limit reached"; })()', 'Returns "limit reached"'], ['[...upTo(5, [1, 2])].join() === "1,2"', 'Ends early if items run out']],
    },
    {
      t: 'Talking Back to a Generator',
      b: `\`next(value)\` can **send a value in**: it becomes the result of the paused \`yield\` expression.

\`\`\`js
function* chat() {
  const name = yield "What is your name?";
  yield "Hello, " + name;
}
const c = chat();
c.next().value;        // "What is your name?"
c.next("Ada").value;   // "Hello, Ada"
\`\`\``,
      code: `// Write function* runningTotal() that yields the total so far, starting at 0,\n// and adds each value sent in with next(value)\n`,
      sol: `function* runningTotal() {\n  let total = 0;\n  while (true) {\n    const n = yield total;\n    total += n;\n  }\n}`,
      tests: [['(() => { const t = runningTotal(); t.next(); t.next(5); t.next(10); return t.next(1).value === 16; })()', 'Keeps a running total'], ['runningTotal().next().value === 0', 'Starts at 0']],
    },
    {
      t: 'Iterators to Arrays',
      b: `Turn any finite iterable into an array with spread or \`Array.from\`. \`Array.from\` can also map as it goes:

\`\`\`js
Array.from(new Set([1, 2]), x => x * 2)   // [2, 4]
Array.from(map.keys())
\`\`\``,
      code: `const scores = new Map([["ann", 7], ["bob", 12], ["cy", 9]]);\n// Create names (array of keys) and doubled (values times 2) using Array.from\n`,
      sol: `const scores = new Map([["ann", 7], ["bob", 12], ["cy", 9]]);\nconst names = Array.from(scores.keys());\nconst doubled = Array.from(scores.values(), (v) => v * 2);`,
      tests: [['names.join() === "ann,bob,cy"', '`names` is ["ann", "bob", "cy"]'], ['doubled.join() === "14,24,18"', '`doubled` is [14, 24, 18]']],
    },
    {
      t: 'Iterator Helpers',
      b: `Modern JavaScript gives iterators their own lazy \`map\`, \`filter\`, \`take\` and \`toArray\` methods — perfect with infinite generators:

\`\`\`js
naturals()
  .filter(n => n % 3 === 0)
  .map(n => n * n)
  .take(3)
  .toArray();   // [9, 36, 81]
\`\`\``,
      code: `function* naturals() {\n  let n = 1;\n  while (true) yield n++;\n}\n// Create firstSquares: the first 5 square numbers that are EVEN, using iterator helpers\n`,
      sol: `function* naturals() {\n  let n = 1;\n  while (true) yield n++;\n}\nconst firstSquares = naturals()\n  .map((n) => n * n)\n  .filter((sq) => sq % 2 === 0)\n  .take(5)\n  .toArray();`,
      tests: [['firstSquares.join() === "4,16,36,64,100"', '`firstSquares` is [4, 16, 36, 64, 100]'], ['$src.includes(".take(")', 'Use `.take()`']],
    },
    {
      t: 'Chunking',
      b: `A generator can group values as they stream past. Collect into a buffer, and yield it whenever it is full (plus any leftover at the end).`,
      code: `// Write function* chunk(iterable, size) yielding arrays of up to size items\n`,
      sol: `function* chunk(iterable, size) {\n  let buffer = [];\n  for (const x of iterable) {\n    buffer.push(x);\n    if (buffer.length === size) {\n      yield buffer;\n      buffer = [];\n    }\n  }\n  if (buffer.length) yield buffer;\n}`,
      tests: [['JSON.stringify([...chunk([1, 2, 3, 4, 5], 2)]) === "[[1,2],[3,4],[5]]"', 'Chunks into pairs with a leftover'], ['JSON.stringify([...chunk("abcdef", 3)]) === \'[["a","b","c"],["d","e","f"]]\'', 'Works on strings'], ['[...chunk([], 3)].length === 0', 'Empty input gives nothing']],
    },
    {
      t: 'Zipping',
      b: `"Zipping" pairs up items from two sequences: first with first, second with second… stopping when either runs out. Use the iterators' \`next()\` directly.`,
      code: `// Write function* zip(a, b) yielding [itemA, itemB] pairs until either iterable is done\n`,
      sol: `function* zip(a, b) {\n  const ia = a[Symbol.iterator]();\n  const ib = b[Symbol.iterator]();\n  while (true) {\n    const x = ia.next();\n    const y = ib.next();\n    if (x.done || y.done) return;\n    yield [x.value, y.value];\n  }\n}`,
      tests: [['JSON.stringify([...zip([1, 2, 3], "ab")]) === \'[[1,"a"],[2,"b"]]\'', 'Pairs up until the shorter one ends']],
    },
    {
      t: 'Project: Prime Stream',
      b: `Build a lazy prime-number pipeline:

- \`function* primes()\` — yields 2, 3, 5, 7, 11, … forever
- \`primesBetween(lo, hi)\` — returns an array of primes with \`lo <= p <= hi\`. It must **stop** consuming the infinite generator once past \`hi\`
- \`nthPrime(n)\` — the nth prime (1-based: \`nthPrime(1)\` is 2)`,
      code: `function* primes() {\n}\n\nfunction primesBetween(lo, hi) {\n}\n\nfunction nthPrime(n) {\n}\n`,
      sol: `function* primes() {\n  const found = [];\n  for (let n = 2; ; n++) {\n    if (found.every((p) => p * p > n || n % p !== 0)) {\n      found.push(n);\n      yield n;\n    }\n  }\n}\n\nfunction primesBetween(lo, hi) {\n  const out = [];\n  for (const p of primes()) {\n    if (p > hi) break;\n    if (p >= lo) out.push(p);\n  }\n  return out;\n}\n\nfunction nthPrime(n) {\n  let i = 0;\n  for (const p of primes()) {\n    if (++i === n) return p;\n  }\n}`,
      tests: [['(() => { const it = primes(); const out = []; for (let i = 0; i < 6; i++) out.push(it.next().value); return out.join() === "2,3,5,7,11,13"; })()', '`primes()` starts 2, 3, 5, 7, 11, 13'], ['primesBetween(10, 30).join() === "11,13,17,19,23,29"', '`primesBetween(10, 30)` is correct'], ['nthPrime(1) === 2 && nthPrime(100) === 541', '`nthPrime` works']],
    },
  ],
});
