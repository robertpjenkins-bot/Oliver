registerModule({
  id: 'functional',
  title: 'Functional Programming',
  icon: 'λ',
  lessons: [
    {
      t: 'Functions Are Values',
      b: `In JavaScript, functions are **first-class values**: you can store them in variables, arrays and objects, pass them around and return them — just like numbers or strings.

\`\`\`js
const ops = [(x) => x + 1, (x) => x * 2, Math.sqrt];
ops.map((f) => f(16));   // [17, 32, 4]
\`\`\``,
      code: `// Create transforms: an array of three functions — trim, uppercase, and add "!" to a string\n// Then create result by applying each in turn to "  hello "\n`,
      sol: `const transforms = [(s) => s.trim(), (s) => s.toUpperCase(), (s) => s + "!"];\nlet result = "  hello ";\nfor (const f of transforms) result = f(result);`,
      tests: [['Array.isArray(transforms) && transforms.length === 3 && transforms.every((f) => typeof f === "function")', '`transforms` holds three functions'], ['result === "HELLO!"', '`result` is `"HELLO!"`']],
    },
    {
      t: 'Build Your Own map',
      b: `Higher-order functions like \`map\` are not magic. Write your own to see how simple they are: make a new array, call the function on each item, push the result.`,
      code: `// Write myMap(array, fn) without using .map — fn receives (item, index)\n`,
      sol: `function myMap(array, fn) {\n  const out = [];\n  for (let i = 0; i < array.length; i++) out.push(fn(array[i], i));\n  return out;\n}`,
      tests: [['myMap([1, 2, 3], (x) => x * x).join() === "1,4,9"', 'Squares numbers'], ['myMap(["a", "b"], (x, i) => x + i).join() === "a0,b1"', 'Passes the index'], ['!$src.includes(".map(")', 'Do not use `.map`']],
    },
    {
      t: 'Build Your Own filter',
      b: `Same idea: keep only the items where the test function returns something truthy.`,
      code: `// Write myFilter(array, test) without using .filter\n`,
      sol: `function myFilter(array, test) {\n  const out = [];\n  for (const item of array) if (test(item)) out.push(item);\n  return out;\n}`,
      tests: [['myFilter([1, 2, 3, 4, 5], (x) => x % 2).join() === "1,3,5"', 'Keeps odd numbers'], ['myFilter([], () => true).length === 0', 'Handles empty arrays'], ['!$src.includes(".filter(")', 'Do not use `.filter`']],
    },
    {
      t: 'Build Your Own reduce',
      b: `\`reduce\` keeps a running accumulator. Start with the initial value, and replace it with \`fn(acc, item)\` for each item.`,
      code: `// Write myReduce(array, fn, initial) without using .reduce\n`,
      sol: `function myReduce(array, fn, initial) {\n  let acc = initial;\n  for (const item of array) acc = fn(acc, item);\n  return acc;\n}`,
      tests: [['myReduce([1, 2, 3, 4], (a, b) => a + b, 0) === 10', 'Sums numbers'], ['myReduce(["a", "b", "c"], (acc, x) => x + acc, "") === "cba"', 'Builds strings'], ['!$src.includes(".reduce(")', 'Do not use `.reduce`']],
    },
    {
      t: "Don't Mutate",
      b: `Functional code avoids **mutation** — changing data in place. Instead, it returns new data and leaves the original alone. This prevents a whole family of "who changed my array?!" bugs.

\`\`\`js
// mutating ✗
function addTodo(list, t) { list.push(t); return list; }
// immutable ✓
const addTodo = (list, t) => [...list, t];
\`\`\``,
      code: `function removeAt(list, index) {\n  list.splice(index, 1);\n  return list;\n}\n// Rewrite removeAt so it returns a NEW array and does not change list\n`,
      sol: `const removeAt = (list, index) => list.filter((_, i) => i !== index);`,
      tests: [['removeAt(["a", "b", "c"], 1).join() === "a,c"', 'Removes the item'], ['(() => { const l = [1, 2, 3]; removeAt(l, 0); return l.length === 3; })()', 'Does not change the original']],
    },
    {
      t: 'Updating Nested Data',
      b: `To "change" something deep inside immutable data, copy each level on the way down with spread:

\`\`\`js
const updated = {
  ...state,
  user: { ...state.user, name: "New name" },
};
\`\`\`

Everything not on the path is shared, not copied — so it is cheap.`,
      code: `// Write rename(state, newName) returning a new state with state.player.name changed\n// (state and state.player must not be modified)\n`,
      sol: `const rename = (state, newName) => ({ ...state, player: { ...state.player, name: newName } });`,
      tests: [['(() => { const s = { level: 2, player: { name: "A", hp: 9 } }; const t = rename(s, "B"); return t.player.name === "B" && t.player.hp === 9 && t.level === 2; })()', 'The new state is correct'], ['(() => { const s = { level: 2, player: { name: "A", hp: 9 } }; rename(s, "B"); return s.player.name === "A"; })()', 'The old state is untouched']],
    },
    {
      t: 'Spotting Impure Functions',
      b: `A function is **impure** if it reads or changes anything outside itself (globals, the console, the clock, random numbers, its arguments) — or if the same inputs can give different outputs.

Impure code is not bad — programs must do *something*! — but keeping most functions pure makes the rest easy to test.`,
      code: `// For each function, set the matching variable to true if it is PURE, false if not\nlet total = 0;\nconst a = (x) => x * 2;                    // pureA\nconst b = (x) => { total += x; return total; }; // pureB\nconst c = () => Math.random();            // pureC\nconst d = (arr) => arr.length;            // pureD\n\nconst pureA = null, pureB = null, pureC = null, pureD = null;\n`,
      sol: `let total = 0;\nconst a = (x) => x * 2;\nconst b = (x) => { total += x; return total; };\nconst c = () => Math.random();\nconst d = (arr) => arr.length;\n\nconst pureA = true, pureB = false, pureC = false, pureD = true;`,
      tests: [['pureA === true', '`a` is pure'], ['pureB === false', '`b` changes a global'], ['pureC === false', '`c` is random'], ['pureD === true', '`d` is pure']],
    },
    {
      t: 'Composing Functions',
      b: `**Composition** plugs functions together: the output of one becomes the input of the next.

\`\`\`js
const compose = (f, g) => (x) => f(g(x));
const shout = compose((s) => s + "!", (s) => s.toUpperCase());
shout("hi");   // "HI!"
\`\`\`

Note that \`compose(f, g)\` runs **g first**.`,
      code: `// Write compose(f, g) as above\n// Then create doubleThenIncrement = compose(inc, double) with small helpers inc and double\n`,
      sol: `const compose = (f, g) => (x) => f(g(x));\nconst inc = (x) => x + 1;\nconst double = (x) => x * 2;\nconst doubleThenIncrement = compose(inc, double);`,
      tests: [['doubleThenIncrement(5) === 11', '`doubleThenIncrement(5)` is `11`'], ['compose((s) => s.length, (s) => s.trim())("  ab ") === 2', '`compose` works with any functions']],
    },
    {
      t: 'Pipelines',
      b: `\`pipe\` composes **any number** of functions, left to right — reading like a recipe:

\`\`\`js
const pipe = (...fns) => (x) => fns.reduce((v, f) => f(v), x);

const slugify = pipe(
  (s) => s.trim(),
  (s) => s.toLowerCase(),
  (s) => s.replace(/\\s+/g, "-"),
);
slugify("  Hello World ")   // "hello-world"
\`\`\``,
      code: `// Write pipe(...fns) and use it to make slugify as above\n`,
      sol: `const pipe = (...fns) => (x) => fns.reduce((v, f) => f(v), x);\nconst slugify = pipe(\n  (s) => s.trim(),\n  (s) => s.toLowerCase(),\n  (s) => s.replace(/\\s+/g, "-"),\n);`,
      tests: [['slugify("  Hello Big World ") === "hello-big-world"', '`slugify` works'], ['pipe()(7) === 7', '`pipe()` with no functions returns the input'], ['pipe((x) => x + 1, (x) => x * 10)(1) === 20', '`pipe` runs left to right']],
    },
    {
      t: 'Currying',
      b: `A **curried** function takes its arguments one at a time, returning a new function for each:

\`\`\`js
const add = (a) => (b) => a + b;
add(2)(3);          // 5
const add10 = add(10);
[1, 2].map(add10);  // [11, 12]
\`\`\`

Currying makes it easy to create specialised functions from general ones.`,
      code: `// Write a curried between(min)(max)(n) returning whether min <= n <= max\n// Then create isTeen = between(13)(19)\n`,
      sol: `const between = (min) => (max) => (n) => n >= min && n <= max;\nconst isTeen = between(13)(19);`,
      tests: [['isTeen(15) === true && isTeen(20) === false', '`isTeen` works'], ['[5, 50, 500].filter(between(10)(100)).join() === "50"', 'Works with `filter`']],
    },
    {
      t: 'Small Reusable Helpers',
      b: `Tiny, general helpers combine into powerful expressions:

\`\`\`js
const prop = (key) => (obj) => obj[key];
const not = (fn) => (...args) => !fn(...args);

users.map(prop("name"));
items.filter(not(isEmpty));
\`\`\``,
      code: `// Write prop(key) and not(fn) as above\nconst books = [\n  { title: "Dune", read: true },\n  { title: "Emma", read: false },\n  { title: "Beloved", read: false },\n];\n// Then create unreadTitles using filter(not(prop("read"))) and map(prop("title"))\n`,
      sol: `const prop = (key) => (obj) => obj[key];\nconst not = (fn) => (...args) => !fn(...args);\nconst books = [\n  { title: "Dune", read: true },\n  { title: "Emma", read: false },\n  { title: "Beloved", read: false },\n];\nconst unreadTitles = books.filter(not(prop("read"))).map(prop("title"));`,
      tests: [['unreadTitles.join() === "Emma,Beloved"', '`unreadTitles` is ["Emma", "Beloved"]'], ['not((x) => x > 1)(5) === false', '`not` flips results']],
    },
    {
      t: 'Combining Predicates',
      b: `Predicates (functions returning booleans) can be combined with small helpers:

\`\`\`js
const both = (f, g) => (x) => f(x) && g(x);
const either = (f, g) => (x) => f(x) || g(x);
\`\`\``,
      code: `// Write allOf(...tests) returning a predicate that is true only if every test passes\n// Write anyOf(...tests) returning a predicate that is true if any test passes\n`,
      sol: `const allOf = (...tests) => (x) => tests.every((t) => t(x));\nconst anyOf = (...tests) => (x) => tests.some((t) => t(x));`,
      tests: [['allOf((n) => n > 0, (n) => n % 2 === 0)(4) === true && allOf((n) => n > 0, (n) => n % 2 === 0)(3) === false', '`allOf` works'], ['anyOf((s) => s.startsWith("a"), (s) => s.endsWith("z"))("fizz") === true', '`anyOf` works'], ['allOf()(1) === true', '`allOf()` with no tests is true']],
    },
    {
      t: 'Declarative Style',
      b: `**Imperative** code says *how* (loops, counters, pushes). **Declarative** code says *what* (filter these, map to that). Rewrite this loop declaratively with array methods.`,
      code: `function namesOfAdults(people) {\n  const result = [];\n  for (let i = 0; i < people.length; i++) {\n    if (people[i].age >= 18) {\n      result.push(people[i].name.toUpperCase());\n    }\n  }\n  result.sort();\n  return result;\n}\n`,
      sol: `const namesOfAdults = (people) =>\n  people\n    .filter((p) => p.age >= 18)\n    .map((p) => p.name.toUpperCase())\n    .toSorted();`,
      tests: [['namesOfAdults([{ name: "zoe", age: 30 }, { name: "al", age: 12 }, { name: "bo", age: 18 }]).join() === "BO,ZOE"', 'Same results as before'], ['!/for\\s*\\(/.test($src)', 'No `for` loops']],
    },
    {
      t: 'Writing groupBy',
      b: `Grouping is a perfect \`reduce\`: the accumulator is an object of arrays. Write it from scratch (no \`Object.groupBy\`).`,
      code: `// Write groupBy(array, keyFn) returning an object of arrays, using reduce\n`,
      sol: `const groupBy = (array, keyFn) =>\n  array.reduce((groups, item) => {\n    const key = keyFn(item);\n    (groups[key] ??= []).push(item);\n    return groups;\n  }, {});`,
      tests: [['JSON.stringify(groupBy([1, 2, 3, 4, 5], (n) => (n % 2 ? "odd" : "even"))) === \'{"odd":[1,3,5],"even":[2,4]}\'', 'Groups odd and even'], ['!$src.includes("Object.groupBy")', 'Write it yourself']],
    },
    {
      t: 'Debouncing',
      b: `A **debounced** function waits until calls **stop** for a while before running — ideal for search boxes (search once the user pauses typing, not on every key):

\`\`\`js
function debounce(fn, ms) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), ms);
  };
}
\`\`\``,
      code: `// Write debounce(fn, ms) as above.\n// Then create search = debounce((q) => console.log("Searching for " + q), 50)\n// and call search("j"), search("ja"), search("jav"), search("java") right away\n`,
      sol: `function debounce(fn, ms) {\n  let timer;\n  return (...args) => {\n    clearTimeout(timer);\n    timer = setTimeout(() => fn(...args), ms);\n  };\n}\nconst search = debounce((q) => console.log("Searching for " + q), 50);\nsearch("j");\nsearch("ja");\nsearch("jav");\nsearch("java");`,
      tests: [['$out() === "Searching for java"', 'Only the last search runs']],
      wait: 200,
    },
    {
      t: 'Memoizing Anything',
      b: `A general \`memoize\` works for functions with several arguments by turning the arguments into a cache key, e.g. with \`JSON.stringify(args)\`.`,
      code: `// Write memoize(fn) that caches by JSON.stringify(args)\n`,
      sol: `function memoize(fn) {\n  const cache = new Map();\n  return (...args) => {\n    const key = JSON.stringify(args);\n    if (!cache.has(key)) cache.set(key, fn(...args));\n    return cache.get(key);\n  };\n}`,
      tests: [['(() => { let calls = 0; const add = memoize((a, b) => (calls++, a + b)); add(1, 2); add(1, 2); add(2, 1); return add(1, 2) === 3 && calls === 2; })()', 'Caches per argument list']],
    },
    {
      t: 'Automatic Currying',
      b: `Every function has a \`length\`: the number of parameters it declares. A general \`curry\` can use it to keep collecting arguments until it has enough:

\`\`\`js
const add3 = curry((a, b, c) => a + b + c);
add3(1)(2)(3);   // 6
add3(1, 2)(3);   // 6
\`\`\``,
      code: `// Write curry(fn)\n`,
      sol: `function curry(fn) {\n  return function curried(...args) {\n    if (args.length >= fn.length) return fn(...args);\n    return (...more) => curried(...args, ...more);\n  };\n}`,
      tests: [['curry((a, b, c) => a + b + c)(1)(2)(3) === 6', 'One at a time'], ['curry((a, b, c) => a * b * c)(2, 3)(4) === 24', 'Several at once'], ['curry((a, b) => a - b)(10, 4) === 6', 'All at once']],
    },
    {
      t: 'Functions in a Lookup Table',
      b: `Instead of a big \`switch\`, store the behaviours in an object of functions (the **strategy** pattern). Adding a new case is then just adding a property:

\`\`\`js
const shippingCost = {
  standard: (kg) => 5 + kg,
  express: (kg) => 15 + kg * 2,
};
shippingCost[method](weight);
\`\`\``,
      code: `// Create operations: an object with "+", "-", "*" and "/" functions\n// Write calc(a, op, b) using it, returning NaN for unknown operators\n`,
      sol: `const operations = {\n  "+": (a, b) => a + b,\n  "-": (a, b) => a - b,\n  "*": (a, b) => a * b,\n  "/": (a, b) => a / b,\n};\nconst calc = (a, op, b) => (operations[op] ? operations[op](a, b) : NaN);`,
      tests: [['calc(6, "*", 7) === 42 && calc(9, "-", 4) === 5 && calc(1, "/", 4) === 0.25', 'Calculates'], ['Number.isNaN(calc(1, "%", 2))', 'Unknown operators give NaN'], ['!$src.includes("switch")', 'No `switch`']],
    },
    {
      t: 'Transforming Objects',
      b: `\`Object.entries\` + \`map\` + \`Object.fromEntries\` transforms every value (or key) of an object functionally:

\`\`\`js
const mapValues = (obj, fn) =>
  Object.fromEntries(Object.entries(obj).map(([k, v]) => [k, fn(v)]));
\`\`\``,
      code: `// Write mapValues(obj, fn) and filterValues(obj, test) returning new objects\n`,
      sol: `const mapValues = (obj, fn) => Object.fromEntries(Object.entries(obj).map(([k, v]) => [k, fn(v)]));\nconst filterValues = (obj, test) => Object.fromEntries(Object.entries(obj).filter(([, v]) => test(v)));`,
      tests: [['JSON.stringify(mapValues({ a: 1, b: 2 }, (v) => v * 10)) === \'{"a":10,"b":20}\'', '`mapValues` works'], ['JSON.stringify(filterValues({ a: 1, b: 0, c: 3 }, Boolean)) === \'{"a":1,"c":3}\'', '`filterValues` works']],
    },
    {
      t: 'Project: Sales Report',
      b: `Write \`salesReport(orders)\` using **only** array methods and small pure functions (no \`for\`/\`while\` loops, no mutation of the input).

Each order: \`{ id, customer, items: [{ name, price, qty }], status }\`.

Return:

- \`revenue\` — total of \`price × qty\` for orders whose status is \`"paid"\`, rounded to 2 decimals
- \`topCustomer\` — the customer with the highest paid total
- \`bestSeller\` — the item name with the most units sold across paid orders
- \`refunded\` — ids of orders with status \`"refunded"\``,
      code: `// Sample data to try your function with — the checks use it too\nconst ORDERS = () => [{ id: 1, customer: "ann", status: "paid", items: [{ name: "pen", price: 1.5, qty: 10 }, { name: "book", price: 12, qty: 1 }] }, { id: 2, customer: "bea", status: "paid", items: [{ name: "lamp", price: 30, qty: 1 }, { name: "pen", price: 1.5, qty: 3 }, { name: "mug", price: 4, qty: 2 }] }, { id: 3, customer: "ann", status: "refunded", items: [{ name: "lamp", price: 30, qty: 5 }] }, { id: 4, customer: "cy", status: "pending", items: [{ name: "mug", price: 4, qty: 50 }] }];\n\nfunction salesReport(orders) {\n}\n`,
      sol: `const ORDERS = () => [{ id: 1, customer: "ann", status: "paid", items: [{ name: "pen", price: 1.5, qty: 10 }, { name: "book", price: 12, qty: 1 }] }, { id: 2, customer: "bea", status: "paid", items: [{ name: "lamp", price: 30, qty: 1 }, { name: "pen", price: 1.5, qty: 3 }, { name: "mug", price: 4, qty: 2 }] }, { id: 3, customer: "ann", status: "refunded", items: [{ name: "lamp", price: 30, qty: 5 }] }, { id: 4, customer: "cy", status: "pending", items: [{ name: "mug", price: 4, qty: 50 }] }];\n\nfunction salesReport(orders) {\n  const paid = orders.filter((o) => o.status === "paid");\n  const orderTotal = (o) => o.items.reduce((sum, i) => sum + i.price * i.qty, 0);\n  const sumBy = (pairs) => pairs.reduce((acc, [k, v]) => ({ ...acc, [k]: (acc[k] ?? 0) + v }), {});\n  const maxKey = (obj) => Object.entries(obj).reduce((best, e) => (e[1] > best[1] ? e : best), ["", -Infinity])[0];\n\n  const revenue = Math.round(paid.reduce((sum, o) => sum + orderTotal(o), 0) * 100) / 100;\n  const byCustomer = sumBy(paid.map((o) => [o.customer, orderTotal(o)]));\n  const units = sumBy(paid.flatMap((o) => o.items.map((i) => [i.name, i.qty])));\n  return {\n    revenue,\n    topCustomer: maxKey(byCustomer),\n    bestSeller: maxKey(units),\n    refunded: orders.filter((o) => o.status === "refunded").map((o) => o.id),\n  };\n}`,
      tests: [
        ['(() => { const r = salesReport(ORDERS()); return r.revenue === 69.5; })()', '`revenue` is 69.5'],
        ['salesReport(ORDERS()).topCustomer === "bea"', '`topCustomer` is "bea"'],
        ['salesReport(ORDERS()).bestSeller === "pen"', '`bestSeller` is "pen"'],
        ['salesReport(ORDERS()).refunded.join() === "3"', '`refunded` is [3]'],
        ['(() => { const o = ORDERS(); const before = JSON.stringify(o); salesReport(o); return JSON.stringify(o) === before; })()', 'The input is not changed'],
        ['!/\\b(for|while)\\s*\\(/.test($src)', 'No loops'],
      ],
    },
  ],
});
