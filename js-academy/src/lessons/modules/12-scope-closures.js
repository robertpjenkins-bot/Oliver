registerModule({
  id: 'scope',
  title: 'Scope & Closures',
  icon: '🔒',
  lessons: [
    {
      t: 'Global and Local',
      b: `**Scope** is where a variable can be seen.

- Variables declared at the top level are **global** — visible everywhere.
- Variables declared inside a function are **local** to it.

\`\`\`js
const planet = "Earth";        // global
function show() {
  const moon = "Luna";         // local
  console.log(planet, moon);   // both visible here
}
\`\`\`

Functions can read global variables. The outside world cannot read a function's locals.`,
      code: `let visits = 0;\nfunction visit() {\n  let visits = 0;\n  visits++;\n}\nvisit();\nvisit();\n// visits should be 2, but it stays 0. Fix visit() so it updates the GLOBAL visits\n`,
      sol: `let visits = 0;\nfunction visit() {\n  visits++;\n}\nvisit();\nvisit();`,
      tests: [['visits === 2', '`visits` should be `2`']],
    },
    {
      t: 'Block Scope',
      b: `\`let\` and \`const\` are **block-scoped**: they only exist inside the nearest \`{ }\` — including the blocks of \`if\` and loops.

\`\`\`js
if (true) {
  const inside = "hi";
}
inside;  // ReferenceError

for (let i = 0; i < 3; i++) {}
i;       // ReferenceError
\`\`\``,
      code: `const temperature = 31;\nif (temperature > 30) {\n  const advice = "Stay inside";\n} else {\n  const advice = "Go outside";\n}\n// Fix: make advice usable here, holding the right message\nconsole.log(advice);\n`,
      sol: `const temperature = 31;\nlet advice;\nif (temperature > 30) {\n  advice = "Stay inside";\n} else {\n  advice = "Go outside";\n}\nconsole.log(advice);`,
      tests: [['$out() === "Stay inside"', 'Print `Stay inside`']],
    },
    {
      t: 'Shadowing',
      b: `A local variable with the same name as an outer one **shadows** it: inside, the local wins.

\`\`\`js
const color = "blue";
function paint() {
  const color = "red";  // shadows the outer color
  return color;         // "red"
}
color;                  // still "blue"
\`\`\`

Shadowing is legal but can confuse readers. Predict what gets printed, then store it.`,
      code: `const x = "outer";\nfunction test() {\n  const x = "inner";\n  return x;\n}\n// Create fromFunction = test() and fromOutside = x\n`,
      sol: `const x = "outer";\nfunction test() {\n  const x = "inner";\n  return x;\n}\nconst fromFunction = test();\nconst fromOutside = x;`,
      tests: [['fromFunction === "inner"', '`fromFunction` is `"inner"`'], ['fromOutside === "outer"', '`fromOutside` is `"outer"`']],
    },
    {
      t: 'The Scope Chain',
      b: `Functions can be nested. An inner function can see its own variables, its parent's, its grandparent's… all the way out to the globals. This is the **scope chain**:

\`\`\`js
const a = 1;
function outer() {
  const b = 2;
  function inner() {
    const c = 3;
    return a + b + c;  // 6 — sees all three
  }
  return inner();
}
\`\`\``,
      code: `const base = 100;\nfunction level1() {\n  const bonus = 20;\n  function level2() {\n    const extra = 3;\n    // return base + bonus + extra\n  }\n  return level2();\n}\n`,
      sol: `const base = 100;\nfunction level1() {\n  const bonus = 20;\n  function level2() {\n    const extra = 3;\n    return base + bonus + extra;\n  }\n  return level2();\n}`,
      tests: [['level1() === 123', '`level1()` returns `123`']],
    },
    {
      t: 'Your First Closure',
      b: `Here is the magic: a function **remembers** the variables around it, even after the outer function has finished. That is a **closure**.

\`\`\`js
function makeGreeter(greeting) {
  return function (name) {
    return greeting + ", " + name;
  };
}
const hello = makeGreeter("Hello");
hello("Ana");   // "Hello, Ana" — greeting is remembered!
\`\`\``,
      code: `// Write makeAdder(n) that returns a function adding n to its argument\n`,
      sol: `function makeAdder(n) {\n  return (x) => x + n;\n}`,
      tests: [['makeAdder(5)(10) === 15', '`makeAdder(5)(10)` is `15`'], ['(() => { const add2 = makeAdder(2); return add2(1) === 3 && add2(40) === 42; })()', 'The returned function remembers `n`']],
    },
    {
      t: 'A Private Counter',
      b: `Closures give you **private state**: variables only your returned functions can touch.

\`\`\`js
function makeCounter() {
  let count = 0;
  return () => ++count;
}
const next = makeCounter();
next(); // 1
next(); // 2
\`\`\`

Nothing outside can reset or cheat \`count\`.`,
      code: `// Write makeCounter() returning a function that returns 1, 2, 3, … on each call\n`,
      sol: `function makeCounter() {\n  let count = 0;\n  return () => ++count;\n}`,
      tests: [['(() => { const c = makeCounter(); c(); c(); return c() === 3; })()', 'Counts 1, 2, 3'], ['(() => { const a = makeCounter(), b = makeCounter(); a(); a(); return b() === 1; })()', 'Each counter is independent']],
    },
    {
      t: 'Closures with Several Functions',
      b: `Return an object of functions, and they all share the same private variables:

\`\`\`js
function makeWallet() {
  let coins = 0;
  return {
    add: (n) => { coins += n; },
    balance: () => coins,
  };
}
\`\`\``,
      code: `// Write makeAccount(start) returning { deposit(n), withdraw(n), balance() }\n// withdraw should refuse (return false) if there is not enough money, else return true\n`,
      sol: `function makeAccount(start) {\n  let money = start;\n  return {\n    deposit(n) {\n      money += n;\n    },\n    withdraw(n) {\n      if (n > money) return false;\n      money -= n;\n      return true;\n    },\n    balance() {\n      return money;\n    },\n  };\n}`,
      tests: [['(() => { const a = makeAccount(10); a.deposit(5); return a.balance() === 15; })()', '`deposit` adds money'], ['(() => { const a = makeAccount(10); return a.withdraw(4) === true && a.balance() === 6; })()', '`withdraw` takes money'], ['(() => { const a = makeAccount(10); return a.withdraw(50) === false && a.balance() === 10; })()', 'Cannot overdraw'], ['(() => { const a = makeAccount(10); return a.money === undefined; })()', 'The money is private']],
    },
    {
      t: 'Function Factories',
      b: `A function that makes customised functions is called a **factory**:

\`\`\`js
const makeMultiplier = (factor) => (n) => n * factor;
const double = makeMultiplier(2);
const triple = makeMultiplier(3);
\`\`\`

(Yes — an arrow function returning an arrow function!)`,
      code: `// Write makeFormatter(prefix, suffix) returning a function (text) => prefix + text + suffix\n// Then create bold = makeFormatter("**", "**")\n`,
      sol: `const makeFormatter = (prefix, suffix) => (text) => prefix + text + suffix;\nconst bold = makeFormatter("**", "**");`,
      tests: [['bold("hi") === "**hi**"', '`bold("hi")` is `"**hi**"`'], ['makeFormatter("<", ">")("b") === "<b>"', 'Any prefix and suffix work']],
    },
    {
      t: 'Closures in Loops',
      b: `\`let\` in a \`for\` loop creates a **fresh** variable for every round, so functions made in the loop each remember their own value.

With the old \`var\`, there is only **one** shared variable — a famous bug:

\`\`\`js
const fns = [];
for (var i = 0; i < 3; i++) fns.push(() => i);
fns[0]();  // 3 — not 0!
\`\`\``,
      code: `const makers = [];\nfor (var i = 0; i < 3; i++) {\n  makers.push(() => i * 10);\n}\n// makers[0]() should be 0, makers[1]() 10, makers[2]() 20. Fix the bug.\n`,
      sol: `const makers = [];\nfor (let i = 0; i < 3; i++) {\n  makers.push(() => i * 10);\n}`,
      tests: [['makers[0]() === 0 && makers[1]() === 10 && makers[2]() === 20', 'Each function remembers its own `i`']],
    },
    {
      t: 'Immediately Invoked Functions',
      b: `An **IIFE** (Immediately Invoked Function Expression) is a function that runs the moment it is defined. It creates a private scope:

\`\`\`js
const result = (() => {
  const temp = 6 * 7;
  return temp;
})();
// temp does not leak out
\`\`\``,
      code: `// Create config using an IIFE that computes secondsPerDay = 24 * 60 * 60 internally\n// and returns { secondsPerDay }\n`,
      sol: `const config = (() => {\n  const secondsPerDay = 24 * 60 * 60;\n  return { secondsPerDay };\n})();`,
      tests: [['config.secondsPerDay === 86400', '`config.secondsPerDay` is `86400`'], ['typeof secondsPerDay === "undefined"', '`secondsPerDay` does not leak out'], ['$src.includes(")()")', 'Use an IIFE']],
    },
    {
      t: 'Run Only Once',
      b: `A closure can remember whether something has already happened:

\`\`\`js
function once(fn) {
  let done = false;
  return (...args) => {
    if (done) return;
    done = true;
    return fn(...args);
  };
}
\`\`\`

Useful for "initialise the game" or "submit the form" buttons.`,
      code: `// Write once(fn): returns a function that calls fn the first time only,\n// and returns the FIRST result on every later call\n`,
      sol: `function once(fn) {\n  let done = false;\n  let result;\n  return (...args) => {\n    if (!done) {\n      done = true;\n      result = fn(...args);\n    }\n    return result;\n  };\n}`,
      tests: [['(() => { let calls = 0; const f = once(() => ++calls); f(); f(); f(); return calls === 1; })()', '`fn` runs only once'], ['(() => { const f = once((x) => x * 2); return f(5) === 10 && f(100) === 10; })()', 'Later calls return the first result']],
    },
    {
      t: 'Memoization',
      b: `**Memoization** caches results so slow work is only done once per input. The cache lives in a closure:

\`\`\`js
function memo(fn) {
  const cache = {};
  return (n) => {
    if (!(n in cache)) cache[n] = fn(n);
    return cache[n];
  };
}
\`\`\``,
      code: `let work = 0;\nfunction slowSquare(n) {\n  work++;\n  return n * n;\n}\n// Write memoize(fn) and create fastSquare = memoize(slowSquare)\n`,
      sol: `let work = 0;\nfunction slowSquare(n) {\n  work++;\n  return n * n;\n}\nfunction memoize(fn) {\n  const cache = new Map();\n  return (n) => {\n    if (!cache.has(n)) cache.set(n, fn(n));\n    return cache.get(n);\n  };\n}\nconst fastSquare = memoize(slowSquare);`,
      tests: [['fastSquare(9) === 81', '`fastSquare(9)` is `81`'], ['(() => { work = 0; fastSquare(12); fastSquare(12); fastSquare(12); return work === 1; })()', 'Repeat calls use the cache']],
    },
    {
      t: 'this Depends on the Call',
      b: `For regular functions, \`this\` is decided by **how the function is called**, not where it was written:

\`\`\`js
const dog = {
  name: "Rex",
  speak() { return this.name; },
};
dog.speak();          // "Rex" — called on dog
const f = dog.speak;
f();                  // undefined — called on nothing!
\`\`\`

Losing \`this\` like this is a very common bug when passing methods around.`,
      code: `const timer = {\n  label: "Egg timer",\n  describe() {\n    return "I am " + this.label;\n  },\n};\nconst describeIt = timer.describe;\n// describeIt() loses \`this\`. Create result by calling describe so that this is timer\n`,
      sol: `const timer = {\n  label: "Egg timer",\n  describe() {\n    return "I am " + this.label;\n  },\n};\nconst describeIt = timer.describe;\nconst result = timer.describe();`,
      tests: [['result === "I am Egg timer"', '`result` is `"I am Egg timer"`']],
    },
    {
      t: 'call and apply',
      b: `\`call\` and \`apply\` run a function with a \`this\` you choose:

\`\`\`js
function intro(greeting) {
  return greeting + ", I am " + this.name;
}
intro.call({ name: "Ada" }, "Hi");      // "Hi, I am Ada"
intro.apply({ name: "Bo" }, ["Hey"]);   // arguments as an array
\`\`\``,
      code: `function describe(sound, times) {\n  return this.animal + " says " + sound.repeat(times);\n}\n// Create a (using call) for { animal: "Cow" } with "moo", 2\n// Create b (using apply) for { animal: "Dog" } with ["woof", 3]\n`,
      sol: `function describe(sound, times) {\n  return this.animal + " says " + sound.repeat(times);\n}\nconst a = describe.call({ animal: "Cow" }, "moo", 2);\nconst b = describe.apply({ animal: "Dog" }, ["woof", 3]);`,
      tests: [['a === "Cow says moomoo"', '`a` is `"Cow says moomoo"`'], ['b === "Dog says woofwoofwoof"', '`b` is `"Dog says woofwoofwoof"`'], ['$src.includes(".call(") && $src.includes(".apply(")', 'Use `call` and `apply`']],
    },
    {
      t: 'bind',
      b: `\`bind\` makes a **new** function with \`this\` permanently fixed — the standard fix for losing \`this\`:

\`\`\`js
const speak = dog.speak.bind(dog);
speak();  // "Rex" — works anywhere now
\`\`\`

\`bind\` can also pre-fill arguments: \`add.bind(null, 10)\` makes a function that adds 10.`,
      code: `const player = {\n  name: "Nova",\n  shout() {\n    return this.name.toUpperCase() + "!";\n  },\n};\n// Create shoutLater: player.shout with this bound to player\n`,
      sol: `const player = {\n  name: "Nova",\n  shout() {\n    return this.name.toUpperCase() + "!";\n  },\n};\nconst shoutLater = player.shout.bind(player);`,
      tests: [['shoutLater() === "NOVA!"', '`shoutLater()` is `"NOVA!"`'], ['$src.includes(".bind(")', 'Use `bind`']],
    },
    {
      t: 'Arrow Functions Keep this',
      b: `Arrow functions do **not** have their own \`this\` — they use the \`this\` of the code around them. That makes them perfect **inside** methods:

\`\`\`js
const team = {
  name: "Tigers",
  members: ["Al", "Bea"],
  list() {
    return this.members.map(m => m + " of the " + this.name);
  },
};
\`\`\`

With a regular \`function\` inside \`map\`, \`this.name\` would be lost.`,
      code: `const classroom = {\n  teacher: "Ms Lee",\n  pupils: ["Ari", "Ben"],\n  roll() {\n    return this.pupils.map(function (p) {\n      return p + " (teacher: " + this.teacher + ")";\n    });\n  },\n};\n// Fix roll() so this.teacher works\n`,
      sol: `const classroom = {\n  teacher: "Ms Lee",\n  pupils: ["Ari", "Ben"],\n  roll() {\n    return this.pupils.map((p) => p + " (teacher: " + this.teacher + ")");\n  },\n};`,
      tests: [['classroom.roll().join("|") === "Ari (teacher: Ms Lee)|Ben (teacher: Ms Lee)"', '`roll()` includes the teacher']],
    },
    {
      t: 'The Temporal Dead Zone',
      b: `\`let\` and \`const\` exist from the start of their block, but you cannot touch them until their declaration line runs. That gap is the **temporal dead zone**:

\`\`\`js
console.log(score);  // ReferenceError: Cannot access 'score' before initialization
let score = 10;
\`\`\`

The fix is simply to declare things before you use them.`,
      code: `console.log(greeting + ", " + who);\nconst greeting = "Hello";\nconst who = "world";\n`,
      sol: `const greeting = "Hello";\nconst who = "world";\nconsole.log(greeting + ", " + who);`,
      tests: [['$out() === "Hello, world"', 'Print `Hello, world`']],
    },
    {
      t: 'The Module Pattern',
      b: `Before JavaScript had modules, people built them with closures: an IIFE that keeps helpers private and returns a public API.

\`\`\`js
const Dice = (() => {
  const sides = 6;                 // private
  const roll = () => Math.floor(Math.random() * sides) + 1;
  return { roll };                 // public
})();
\`\`\``,
      code: `// Create a module Temperature with public toF(c) and toC(f)\n// and a PRIVATE helper round1(n) that rounds to 1 decimal place (used by both)\n`,
      sol: `const Temperature = (() => {\n  const round1 = (n) => Math.round(n * 10) / 10;\n  return {\n    toF: (c) => round1(c * 9 / 5 + 32),\n    toC: (f) => round1((f - 32) * 5 / 9),\n  };\n})();`,
      tests: [['Temperature.toF(37) === 98.6', '`toF(37)` is `98.6`'], ['Temperature.toC(100) === 37.8', '`toC(100)` is `37.8`'], ['Temperature.round1 === undefined', '`round1` stays private']],
    },
    {
      t: 'Partial Application',
      b: `**Partial application** fixes some arguments of a function now, and supplies the rest later — closures make it easy:

\`\`\`js
const partial = (fn, ...preset) => (...rest) => fn(...preset, ...rest);
const greet = (greeting, name) => greeting + " " + name;
const hi = partial(greet, "Hi");
hi("Sam");   // "Hi Sam"
\`\`\``,
      code: `// Write partial(fn, ...preset) as described\n// Then create price(currency, amount) returning currency + amount.toFixed(2)\n// and euros = partial(price, "€")\n`,
      sol: `const partial = (fn, ...preset) => (...rest) => fn(...preset, ...rest);\nconst price = (currency, amount) => currency + amount.toFixed(2);\nconst euros = partial(price, "€");`,
      tests: [['euros(5) === "€5.00"', '`euros(5)` is `"€5.00"`'], ['partial((a, b, c) => a + b + c, 1, 2)(3) === 6', '`partial` works with any function']],
    },
    {
      t: 'Project: Quiz Tracker',
      b: `Write \`createQuiz(questions)\` where \`questions\` is an array of \`{ q, answer }\`. It returns an object with:

- \`current()\` — the current question text, or \`null\` when finished
- \`answer(text)\` — checks the answer (ignoring case and outer spaces), moves to the next question, and returns \`true\`/\`false\`
- \`score()\` — how many were right so far

The index and score must be **private** (inside the closure).`,
      code: `function createQuiz(questions) {\n}\n`,
      sol: `function createQuiz(questions) {\n  let index = 0;\n  let right = 0;\n  return {\n    current() {\n      return index < questions.length ? questions[index].q : null;\n    },\n    answer(text) {\n      if (index >= questions.length) return false;\n      const ok = text.trim().toLowerCase() === questions[index].answer.toLowerCase();\n      if (ok) right++;\n      index++;\n      return ok;\n    },\n    score() {\n      return right;\n    },\n  };\n}`,
      tests: [
        ['(() => { const z = createQuiz([{ q: "2+2?", answer: "4" }, { q: "Capital of France?", answer: "Paris" }]); return z.current() === "2+2?"; })()', '`current()` shows the first question'],
        ['(() => { const z = createQuiz([{ q: "2+2?", answer: "4" }, { q: "Capital of France?", answer: "Paris" }]); return z.answer("4") === true && z.current() === "Capital of France?" && z.answer("  paris ") === true && z.current() === null && z.score() === 2; })()', 'Answers are checked and the quiz moves on'],
        ['(() => { const z = createQuiz([{ q: "a", answer: "x" }]); z.answer("y"); return z.score() === 0 && z.index === undefined; })()', 'Wrong answers do not score; state is private'],
      ],
    },
  ],
});
