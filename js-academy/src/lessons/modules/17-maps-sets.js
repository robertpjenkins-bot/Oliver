registerModule({
  id: 'collections',
  title: 'Maps, Sets & Symbols',
  icon: '🧺',
  lessons: [
    {
      t: 'Sets: Unique Values',
      b: `A **Set** is a collection where every value appears **at most once**:

\`\`\`js
const s = new Set([1, 2, 2, 3, 3, 3]);
s.size   // 3
\`\`\`

Adding something that is already there does nothing.`,
      code: `const visitors = ["ana", "ben", "ana", "cy", "ben", "ana"];\n// Create uniqueVisitors (a Set of visitors) and count (how many unique visitors)\n`,
      sol: `const visitors = ["ana", "ben", "ana", "cy", "ben", "ana"];\nconst uniqueVisitors = new Set(visitors);\nconst count = uniqueVisitors.size;`,
      tests: [['uniqueVisitors instanceof Set', '`uniqueVisitors` is a Set'], ['count === 3', '`count` is `3`']],
    },
    {
      t: 'Adding, Checking, Removing',
      b: `Set methods:

\`\`\`js
const tags = new Set();
tags.add("js");
tags.add("css");
tags.has("js");     // true — very fast, even for huge sets
tags.delete("css");
tags.size;          // 1
\`\`\``,
      code: `const collected = new Set(["red gem"]);\n// Add "blue gem" and "gold key", remove "red gem"\n// Create hasKey: whether collected has "gold key"\n`,
      sol: `const collected = new Set(["red gem"]);\ncollected.add("blue gem");\ncollected.add("gold key");\ncollected.delete("red gem");\nconst hasKey = collected.has("gold key");`,
      tests: [['collected.size === 2 && collected.has("blue gem") && !collected.has("red gem")', 'The set is updated'], ['hasKey === true', '`hasKey` is `true`']],
    },
    {
      t: 'Unique Arrays in One Line',
      b: `Combine a Set with spread to remove duplicates from an array:

\`\`\`js
const unique = [...new Set(array)];
\`\`\`

It keeps the order in which values first appeared.`,
      code: `// Write unique(arr) returning a new array without duplicates, using a Set\n`,
      sol: `const unique = (arr) => [...new Set(arr)];`,
      tests: [['unique([3, 1, 3, 2, 1]).join() === "3,1,2"', '`unique([3, 1, 3, 2, 1])` is [3, 1, 2]'], ['$src.includes("new Set")', 'Use a Set']],
    },
    {
      t: 'Looping Over a Set',
      b: `Sets work with \`for…of\` and \`forEach\`, in insertion order:

\`\`\`js
for (const colour of new Set(["red", "blue"])) {
  console.log(colour);
}
\`\`\``,
      code: `const letters = new Set("mississippi");\n// Print each unique letter, in the order they first appear, on its own line\n`,
      sol: `const letters = new Set("mississippi");\nfor (const l of letters) {\n  console.log(l);\n}`,
      tests: [['$out() === "m\\ni\\ns\\np"', 'Print m, i, s, p']],
    },
    {
      t: 'Union',
      b: `The **union** of two sets holds everything in either:

\`\`\`js
const union = new Set([...a, ...b]);
\`\`\`

(Newer engines also have \`a.union(b)\`.)`,
      code: `// Write union(a, b) returning a new Set with everything from both sets\n`,
      sol: `function union(a, b) {\n  return new Set([...a, ...b]);\n}`,
      tests: [['[...union(new Set([1, 2]), new Set([2, 3]))].sort().join() === "1,2,3"', 'union of {1,2} and {2,3} is {1,2,3}'], ['union(new Set(), new Set()).size === 0', 'Empty sets give an empty set']],
    },
    {
      t: 'Intersection',
      b: `The **intersection** holds only what is in **both** sets. Filter one set by membership in the other:

\`\`\`js
[...a].filter(x => b.has(x))
\`\`\``,
      code: `const alice = new Set(["pizza", "sushi", "tacos", "curry"]);\nconst bob = new Set(["burgers", "tacos", "curry", "salad"]);\n// Create bothLike: a Set of foods both alice and bob like\n`,
      sol: `const alice = new Set(["pizza", "sushi", "tacos", "curry"]);\nconst bob = new Set(["burgers", "tacos", "curry", "salad"]);\nconst bothLike = new Set([...alice].filter((f) => bob.has(f)));`,
      tests: [['bothLike instanceof Set && bothLike.size === 2 && bothLike.has("tacos") && bothLike.has("curry")', '`bothLike` is {tacos, curry}']],
    },
    {
      t: 'Difference',
      b: `The **difference** A − B holds what is in A but **not** in B.`,
      code: `const required = new Set(["passport", "ticket", "charger", "toothbrush"]);\nconst packed = new Set(["ticket", "toothbrush", "snacks"]);\n// Create missing: an array of required items not yet packed (in the order of required)\n`,
      sol: `const required = new Set(["passport", "ticket", "charger", "toothbrush"]);\nconst packed = new Set(["ticket", "toothbrush", "snacks"]);\nconst missing = [...required].filter((item) => !packed.has(item));`,
      tests: [['missing.join() === "passport,charger"', '`missing` is ["passport", "charger"]']],
    },
    {
      t: 'Maps: Key → Value',
      b: `A **Map** stores key/value pairs, like an object, but designed for it:

\`\`\`js
const ages = new Map();
ages.set("Ana", 31);
ages.set("Ben", 27);
ages.get("Ana");   // 31
ages.get("Zed");   // undefined
\`\`\``,
      code: `// Create a Map called capitals with France → "Paris" and Japan → "Tokyo"\n// Then create japan = the capital of Japan\n`,
      sol: `const capitals = new Map();\ncapitals.set("France", "Paris");\ncapitals.set("Japan", "Tokyo");\nconst japan = capitals.get("Japan");`,
      tests: [['capitals instanceof Map && capitals.get("France") === "Paris"', '`capitals` is a Map with France'], ['japan === "Tokyo"', '`japan` is `"Tokyo"`']],
    },
    {
      t: 'Any Key You Like',
      b: `Object keys are always strings. Map keys can be **anything** — numbers, booleans, even objects:

\`\`\`js
const clicks = new Map();
const button = { id: "save" };
clicks.set(button, 3);    // the object itself is the key
clicks.get(button);       // 3
\`\`\``,
      code: `const player1 = { name: "Ann" };\nconst player2 = { name: "Bob" };\n// Create scores: a Map using the player OBJECTS as keys, with 10 and 7\n`,
      sol: `const player1 = { name: "Ann" };\nconst player2 = { name: "Bob" };\nconst scores = new Map([\n  [player1, 10],\n  [player2, 7],\n]);`,
      tests: [['scores.get(player1) === 10 && scores.get(player2) === 7', 'Scores are stored by player object'], ['scores.size === 2', 'Two entries']],
    },
    {
      t: 'Map Methods',
      b: `\`has\`, \`delete\`, \`size\` and \`clear\` work like on Sets:

\`\`\`js
map.has("key");
map.delete("key");
map.size;
\`\`\`

\`set\` returns the map, so calls can be chained: \`m.set("a", 1).set("b", 2)\`.`,
      code: `const stock = new Map([["apples", 5], ["pears", 0], ["plums", 12]]);\n// Delete every item whose stock is 0, then create kinds = the number of items left\n`,
      sol: `const stock = new Map([["apples", 5], ["pears", 0], ["plums", 12]]);\nfor (const [item, qty] of stock) {\n  if (qty === 0) stock.delete(item);\n}\nconst kinds = stock.size;`,
      tests: [['!stock.has("pears")', '"pears" is removed'], ['kinds === 2', '`kinds` is `2`']],
    },
    {
      t: 'Looping Over a Map',
      b: `Maps remember insertion order. Loop over entries, keys or values:

\`\`\`js
for (const [key, value] of map) { … }
map.keys();
map.values();
map.forEach((value, key) => { … });
\`\`\``,
      code: `const menu = new Map([["tea", 2.5], ["cake", 3.75], ["soup", 5]]);\n// Print each line like "tea: $2.50"\n`,
      sol: 'const menu = new Map([["tea", 2.5], ["cake", 3.75], ["soup", 5]]);\nfor (const [item, price] of menu) {\n  console.log(`${item}: $${price.toFixed(2)}`);\n}',
      tests: [['$out() === "tea: $2.50\\ncake: $3.75\\nsoup: $5.00"', 'Print the menu']],
    },
    {
      t: 'Counting with a Map',
      b: `Maps are great counters:

\`\`\`js
counts.set(word, (counts.get(word) ?? 0) + 1);
\`\`\``,
      code: `// Write wordCounts(text) returning a Map from each lowercase word to how often it appears\n// (split on spaces)\n`,
      sol: `function wordCounts(text) {\n  const counts = new Map();\n  for (const word of text.toLowerCase().split(" ")) {\n    counts.set(word, (counts.get(word) ?? 0) + 1);\n  }\n  return counts;\n}`,
      tests: [['(() => { const m = wordCounts("The cat and the hat and THE bat"); return m instanceof Map && m.get("the") === 3 && m.get("and") === 2 && m.get("cat") === 1; })()', 'Counts words, ignoring case']],
    },
    {
      t: 'Converting Maps and Objects',
      b: `Maps and objects convert easily through entries:

\`\`\`js
const map = new Map(Object.entries(obj));
const obj2 = Object.fromEntries(map);
\`\`\`

Useful because \`JSON.stringify\` cannot save a Map directly.`,
      code: `const prices = { bread: 2, milk: 1.5 };\n// Create priceMap (a Map from prices), add eggs → 3 to it, then create backToObject\n`,
      sol: `const prices = { bread: 2, milk: 1.5 };\nconst priceMap = new Map(Object.entries(prices));\npriceMap.set("eggs", 3);\nconst backToObject = Object.fromEntries(priceMap);`,
      tests: [['priceMap instanceof Map && priceMap.get("milk") === 1.5', '`priceMap` is a Map of the prices'], ['JSON.stringify(backToObject) === \'{"bread":2,"milk":1.5,"eggs":3}\'', '`backToObject` includes eggs']],
    },
    {
      t: 'Grouping into a Map',
      b: `\`Map.groupBy\` works like \`Object.groupBy\`, but gives a Map — so group keys can be any type:

\`\`\`js
Map.groupBy([1, 2, 3, 4], n => n % 2 === 0);
// Map { false => [1, 3], true => [2, 4] }
\`\`\``,
      code: `const people = [\n  { name: "Ana", age: 12 },\n  { name: "Ben", age: 34 },\n  { name: "Cy", age: 9 },\n  { name: "Di", age: 51 },\n];\n// Create byAdult: Map.groupBy by whether age >= 18 (true/false keys)\n`,
      sol: `const people = [\n  { name: "Ana", age: 12 },\n  { name: "Ben", age: 34 },\n  { name: "Cy", age: 9 },\n  { name: "Di", age: 51 },\n];\nconst byAdult = Map.groupBy(people, (p) => p.age >= 18);`,
      tests: [['byAdult instanceof Map', '`byAdult` is a Map'], ['byAdult.get(true).map((p) => p.name).join() === "Ben,Di"', 'Adults are Ben and Di'], ['byAdult.get(false).length === 2', 'Two children']],
    },
    {
      t: 'WeakMaps',
      b: `A **WeakMap** only accepts objects as keys, and does not keep them alive: when nothing else refers to the object, its entry disappears automatically. It is ideal for attaching extra data to objects you do not own:

\`\`\`js
const visits = new WeakMap();
visits.set(user, 3);
\`\`\`

WeakMaps cannot be looped over or sized — you can only \`get\`, \`set\`, \`has\` and \`delete\`.`,
      code: `// Write countClick(element) that uses a WeakMap to count and return how many times\n// it was called with that element\n`,
      sol: `const clickCounts = new WeakMap();\nfunction countClick(element) {\n  const n = (clickCounts.get(element) ?? 0) + 1;\n  clickCounts.set(element, n);\n  return n;\n}`,
      tests: [['(() => { const a = {}, b = {}; countClick(a); countClick(a); countClick(b); return countClick(a) === 3 && countClick(b) === 2; })()', 'Counts per object'], ['$src.includes("WeakMap")', 'Use a WeakMap']],
    },
    {
      t: 'Symbols',
      b: `A **Symbol** is a unique value — no two symbols are ever equal, even with the same description:

\`\`\`js
const a = Symbol("id");
const b = Symbol("id");
a === b   // false
\`\`\`

Symbols are useful as special keys that can never clash with normal ones.`,
      code: `// Create two symbols first and second, both with description "token"\n// Create same: whether they are equal\n`,
      sol: `const first = Symbol("token");\nconst second = Symbol("token");\nconst same = first === second;`,
      tests: [['typeof first === "symbol" && first.description === "token"', '`first` is a Symbol described "token"'], ['same === false', '`same` is `false`']],
    },
    {
      t: 'Hidden Keys',
      b: `Symbol keys do not show up in \`Object.keys\`, \`for…in\` or \`JSON.stringify\`. That makes them good for internal data:

\`\`\`js
const secret = Symbol("secret");
const obj = { name: "box", [secret]: 42 };
Object.keys(obj)     // ["name"]
obj[secret]          // 42
\`\`\``,
      code: `const internal = Symbol("internal");\n// Create item = { name: "lamp" } plus a symbol-keyed property internal → { cost: 7 }\n`,
      sol: `const internal = Symbol("internal");\nconst item = { name: "lamp", [internal]: { cost: 7 } };`,
      tests: [['item[internal].cost === 7', 'The symbol property holds { cost: 7 }'], ['JSON.stringify(item) === \'{"name":"lamp"}\'', 'It is hidden from JSON']],
    },
    {
      t: 'First Repeat',
      b: `Sets make "have I seen this before?" instant. Walk through the items; if the set already has one, you found a repeat; otherwise add it.`,
      code: `// Write firstRepeat(s) returning the first character that appears a second time, or null\n`,
      sol: `function firstRepeat(s) {\n  const seen = new Set();\n  for (const ch of s) {\n    if (seen.has(ch)) return ch;\n    seen.add(ch);\n  }\n  return null;\n}`,
      tests: [['firstRepeat("abcdbea") === "b"', '`firstRepeat("abcdbea")` is `"b"`'], ['firstRepeat("xyz") === null', 'No repeat → `null`']],
    },
    {
      t: 'Two-Sum with a Map',
      b: `A famous interview puzzle: find two numbers in an array that add up to a target. Checking every pair is slow. With a Map, one pass is enough: for each number, check whether its **partner** (\`target − n\`) was seen earlier.`,
      code: `// Write twoSum(nums, target) returning the INDEXES [i, j] (i < j) of two numbers adding to target, or null\n`,
      sol: `function twoSum(nums, target) {\n  const seen = new Map();\n  for (let j = 0; j < nums.length; j++) {\n    const need = target - nums[j];\n    if (seen.has(need)) return [seen.get(need), j];\n    seen.set(nums[j], j);\n  }\n  return null;\n}`,
      tests: [['JSON.stringify(twoSum([2, 7, 11, 15], 9)) === "[0,1]"', '`twoSum([2, 7, 11, 15], 9)` is [0, 1]'], ['JSON.stringify(twoSum([3, 2, 4], 6)) === "[1,2]"', '`twoSum([3, 2, 4], 6)` is [1, 2]'], ['twoSum([1, 2], 10) === null', 'No pair → `null`']],
    },
    {
      t: 'Project: Friend Network',
      b: `Build a small social network with a Map from each name to a **Set** of friends. Write a class \`Network\` with:

- \`addFriendship(a, b)\` — friendship goes both ways
- \`friendsOf(name)\` — a sorted array of their friends (empty if unknown)
- \`mutual(a, b)\` — a sorted array of friends they have in common
- \`suggest(name)\` — friends-of-friends who are not already friends (and not themselves), sorted`,
      code: `class Network {\n}\n`,
      sol: `class Network {\n  #friends = new Map();\n  #setOf(name) {\n    if (!this.#friends.has(name)) this.#friends.set(name, new Set());\n    return this.#friends.get(name);\n  }\n  addFriendship(a, b) {\n    this.#setOf(a).add(b);\n    this.#setOf(b).add(a);\n  }\n  friendsOf(name) {\n    return [...(this.#friends.get(name) ?? [])].sort();\n  }\n  mutual(a, b) {\n    const other = new Set(this.friendsOf(b));\n    return this.friendsOf(a).filter((f) => other.has(f));\n  }\n  suggest(name) {\n    const mine = new Set(this.friendsOf(name));\n    const out = new Set();\n    for (const f of mine) {\n      for (const ff of this.friendsOf(f)) {\n        if (ff !== name && !mine.has(ff)) out.add(ff);\n      }\n    }\n    return [...out].sort();\n  }\n}`,
      tests: [
        ['(() => { const n = new Network(); n.addFriendship("ann", "bob"); n.addFriendship("ann", "cat"); return n.friendsOf("ann").join() === "bob,cat" && n.friendsOf("bob").join() === "ann" && n.friendsOf("zed").length === 0; })()', '`friendsOf` works both ways'],
        ['(() => { const n = new Network(); n.addFriendship("ann", "bob"); n.addFriendship("ann", "cat"); n.addFriendship("dan", "bob"); n.addFriendship("dan", "cat"); return n.mutual("ann", "dan").join() === "bob,cat"; })()', '`mutual` finds shared friends'],
        ['(() => { const n = new Network(); n.addFriendship("ann", "bob"); n.addFriendship("bob", "cat"); n.addFriendship("bob", "dan"); n.addFriendship("ann", "dan"); return n.suggest("ann").join() === "cat"; })()', '`suggest` finds friends of friends'],
      ],
    },
  ],
});
