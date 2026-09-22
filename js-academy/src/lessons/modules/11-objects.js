registerModule({
  id: 'objects',
  title: 'Objects',
  icon: '🗂️',
  lessons: [
    {
      t: 'Your First Object',
      b: `An **object** groups related values under names called **keys** (or properties):

\`\`\`js
const dog = {
  name: "Rex",
  age: 4,
  goodBoy: true,
};
\`\`\`

Arrays are for ordered lists; objects are for describing one *thing* with named parts.`,
      code: `// Create an object called book with title "Dune", author "Frank Herbert" and pages 412\n`,
      sol: `const book = {\n  title: "Dune",\n  author: "Frank Herbert",\n  pages: 412,\n};`,
      tests: [['book.title === "Dune"', '`book.title` is `"Dune"`'], ['book.author === "Frank Herbert"', '`book.author` is `"Frank Herbert"`'], ['book.pages === 412', '`book.pages` is `412`']],
    },
    {
      t: 'Dot Notation',
      b: `Read a property with a dot:

\`\`\`js
const car = { make: "Volvo", year: 2019 };
car.make   // "Volvo"
car.color  // undefined — no such key
\`\`\``,
      code: `const user = { username: "coder42", level: 7, country: "NZ" };\n// Print "coder42 is level 7" using the object's properties\n`,
      sol: 'const user = { username: "coder42", level: 7, country: "NZ" };\nconsole.log(`${user.username} is level ${user.level}`);',
      tests: [['$out() === "coder42 is level 7"', 'Print `coder42 is level 7`'], ['$src.includes("user.username") && $src.includes("user.level")', 'Use dot notation']],
    },
    {
      t: 'Bracket Notation',
      b: `Square brackets let you use a key stored in a **variable**, or keys with spaces:

\`\`\`js
const prices = { apple: 1.2, "ice cream": 3 };
const item = "apple";
prices[item]          // 1.2
prices["ice cream"]   // 3
\`\`\``,
      code: `const stock = { apples: 12, pears: 0, plums: 7 };\n// Write howMany(fruit) returning the stock for that fruit name\n`,
      sol: `const stock = { apples: 12, pears: 0, plums: 7 };\nfunction howMany(fruit) {\n  return stock[fruit];\n}`,
      tests: [['howMany("apples") === 12 && howMany("plums") === 7', '`howMany` looks up the right value'], ['$src.includes("stock[fruit]")', 'Use `stock[fruit]`']],
    },
    {
      t: 'Adding and Changing Properties',
      b: `Assign to a property to change it — or to add it if it does not exist yet:

\`\`\`js
const player = { name: "Kai" };
player.score = 0;     // added
player.score += 10;   // changed
\`\`\``,
      code: `const phone = { brand: "Nokia", battery: 40 };\n// Charge the battery to 100 and add a property color set to "blue"\n`,
      sol: `const phone = { brand: "Nokia", battery: 40 };\nphone.battery = 100;\nphone.color = "blue";`,
      tests: [['phone.battery === 100', '`battery` is `100`'], ['phone.color === "blue"', '`color` is `"blue"`']],
    },
    {
      t: 'Deleting Properties',
      b: `\`delete\` removes a property completely:

\`\`\`js
const profile = { name: "Eve", password: "123" };
delete profile.password;
// { name: "Eve" }
\`\`\``,
      code: `const account = { user: "sam", password: "hunter2", email: "sam@x.io" };\n// Remove the password before sharing the account object\n`,
      sol: `const account = { user: "sam", password: "hunter2", email: "sam@x.io" };\ndelete account.password;`,
      tests: [['!("password" in account)', '`password` is gone'], ['account.user === "sam" && account.email === "sam@x.io"', 'The other properties stay']],
    },
    {
      t: 'Methods',
      b: `A function stored in an object is called a **method**:

\`\`\`js
const robot = {
  name: "R2",
  beep() {
    return "Beep boop!";
  },
};
robot.beep();  // "Beep boop!"
\`\`\``,
      code: `// Create calculator with methods add(a, b) and multiply(a, b)\n`,
      sol: `const calculator = {\n  add(a, b) {\n    return a + b;\n  },\n  multiply(a, b) {\n    return a * b;\n  },\n};`,
      tests: [['calculator.add(2, 3) === 5', '`calculator.add(2, 3)` is `5`'], ['calculator.multiply(4, 5) === 20', '`calculator.multiply(4, 5)` is `20`']],
    },
    {
      t: 'this',
      b: `Inside a method, \`this\` refers to the object the method was called on — so methods can use the object's own data:

\`\`\`js
const cat = {
  name: "Tom",
  intro() {
    return "I am " + this.name;
  },
};
cat.intro();   // "I am Tom"
\`\`\``,
      code: `const counter = {\n  count: 0,\n  // add increment(): adds 1 to this.count\n  // add reset(): sets this.count back to 0\n};\n`,
      sol: `const counter = {\n  count: 0,\n  increment() {\n    this.count++;\n  },\n  reset() {\n    this.count = 0;\n  },\n};`,
      tests: [['(counter.increment(), counter.increment(), counter.count === 2)', '`increment()` adds 1'], ['(counter.reset(), counter.count === 0)', '`reset()` sets it to 0'], ['$src.includes("this.count")', 'Use `this.count`']],
    },
    {
      t: 'Nested Objects',
      b: `Property values can be objects (or arrays) themselves. Chain the dots to reach inside:

\`\`\`js
const student = {
  name: "Jo",
  address: { city: "Leeds", postcode: "LS1" },
  grades: [90, 85],
};
student.address.city   // "Leeds"
student.grades[1]      // 85
\`\`\``,
      code: `const game = {\n  title: "Space Quest",\n  settings: { volume: 7, difficulty: "hard" },\n  highScores: [9800, 7200, 5100],\n};\n// Create difficulty and topScore from game, and set the volume to 10\n`,
      sol: `const game = {\n  title: "Space Quest",\n  settings: { volume: 7, difficulty: "hard" },\n  highScores: [9800, 7200, 5100],\n};\nconst difficulty = game.settings.difficulty;\nconst topScore = game.highScores[0];\ngame.settings.volume = 10;`,
      tests: [['difficulty === "hard"', '`difficulty` is `"hard"`'], ['topScore === 9800', '`topScore` is `9800`'], ['game.settings.volume === 10', 'Volume is set to `10`']],
    },
    {
      t: 'Lists of Objects',
      b: `Arrays of objects are everywhere in real programs — lists of users, products, messages…

\`\`\`js
const todos = [
  { text: "Shop", done: true },
  { text: "Code", done: false },
];
todos.filter(t => !t.done)   // what is left to do
\`\`\``,
      code: `const pets = [\n  { name: "Rex", type: "dog", age: 3 },\n  { name: "Tom", type: "cat", age: 5 },\n  { name: "Fido", type: "dog", age: 8 },\n];\n// Create dogNames: the names of all the dogs\n// Create totalAge: the sum of all ages\n`,
      sol: `const pets = [\n  { name: "Rex", type: "dog", age: 3 },\n  { name: "Tom", type: "cat", age: 5 },\n  { name: "Fido", type: "dog", age: 8 },\n];\nconst dogNames = pets.filter((p) => p.type === "dog").map((p) => p.name);\nconst totalAge = pets.reduce((sum, p) => sum + p.age, 0);`,
      tests: [['dogNames.join() === "Rex,Fido"', '`dogNames` is ["Rex", "Fido"]'], ['totalAge === 16', '`totalAge` is `16`']],
    },
    {
      t: 'Listing the Keys',
      b: `\`Object.keys(obj)\` returns an array of an object's keys:

\`\`\`js
Object.keys({ a: 1, b: 2 })   // ["a", "b"]
\`\`\`

Handy for counting properties or looping over them.`,
      code: `const settings = { sound: true, music: false, hints: true, subtitles: true };\n// Create names (the keys) and howMany (the number of settings)\n`,
      sol: `const settings = { sound: true, music: false, hints: true, subtitles: true };\nconst names = Object.keys(settings);\nconst howMany = names.length;`,
      tests: [['names.join() === "sound,music,hints,subtitles"', '`names` holds the keys'], ['howMany === 4', '`howMany` is `4`']],
    },
    {
      t: 'Listing the Values',
      b: `\`Object.values(obj)\` gives the values:

\`\`\`js
Object.values({ a: 1, b: 2 })   // [1, 2]
\`\`\``,
      code: `const expenses = { rent: 800, food: 250, bus: 60, fun: 90 };\n// Create total: the sum of all expenses, using Object.values\n`,
      sol: `const expenses = { rent: 800, food: 250, bus: 60, fun: 90 };\nconst total = Object.values(expenses).reduce((a, b) => a + b, 0);`,
      tests: [['total === 1200', '`total` is `1200`'], ['$src.includes("Object.values")', 'Use `Object.values`']],
    },
    {
      t: 'Keys and Values Together',
      b: `\`Object.entries(obj)\` gives \`[key, value]\` pairs — perfect with \`for…of\`:

\`\`\`js
for (const [key, value] of Object.entries({ x: 1, y: 2 })) {
  console.log(key, value);
}
\`\`\`

\`Object.fromEntries\` turns pairs back into an object.`,
      code: `const inventory = { swords: 2, potions: 5, keys: 1 };\n// Print each item like "swords: 2"\n`,
      sol: 'const inventory = { swords: 2, potions: 5, keys: 1 };\nfor (const [item, count] of Object.entries(inventory)) {\n  console.log(`${item}: ${count}`);\n}',
      tests: [['$out() === "swords: 2\\npotions: 5\\nkeys: 1"', 'Print each item and its count'], ['$src.includes("Object.entries")', 'Use `Object.entries`']],
    },
    {
      t: 'Does It Have That Key?',
      b: `Check whether a property exists with \`in\` or \`Object.hasOwn\`:

\`\`\`js
const car = { wheels: 4, spare: 0 };
"spare" in car            // true
Object.hasOwn(car, "gps") // false
\`\`\`

Why not just \`if (car.spare)\`? Because \`0\` is falsy, even though the key exists!`,
      code: `const answers = { q1: "A", q2: "", q3: "C" };\n// Write answered(q) returning true if the key q exists in answers (even if empty)\n`,
      sol: `const answers = { q1: "A", q2: "", q3: "C" };\nfunction answered(q) {\n  return Object.hasOwn(answers, q);\n}`,
      tests: [['answered("q2") === true', '`answered("q2")` is `true` even though it is empty'], ['answered("q9") === false', '`answered("q9")` is `false`']],
    },
    {
      t: 'Shorthand Properties',
      b: `When a property has the same name as a variable, write it once:

\`\`\`js
const name = "Lia";
const age = 9;
const kid = { name, age };   // same as { name: name, age: age }
\`\`\``,
      code: `function makePoint(x, y) {\n  // return an object with x and y, using shorthand\n}\n`,
      sol: `function makePoint(x, y) {\n  return { x, y };\n}`,
      tests: [['makePoint(3, 4).x === 3 && makePoint(3, 4).y === 4', '`makePoint(3, 4)` is { x: 3, y: 4 }'], ['/\\{\\s*x\\s*,\\s*y\\s*\\}/.test($src)', 'Use shorthand `{ x, y }`']],
    },
    {
      t: 'Computed Keys',
      b: `Put square brackets around a key in an object literal to compute it from an expression:

\`\`\`js
const field = "color";
const style = { [field]: "red" };   // { color: "red" }
\`\`\``,
      code: `// Write pair(key, value) returning an object with that ONE property\n`,
      sol: `function pair(key, value) {\n  return { [key]: value };\n}`,
      tests: [['pair("lang", "js").lang === "js"', '`pair("lang", "js")` is { lang: "js" }'], ['Object.keys(pair("a", 1)).length === 1', 'Only one property']],
    },
    {
      t: 'Copying and Merging',
      b: `Spread \`...\` works on objects too. Later properties win:

\`\`\`js
const defaults = { theme: "light", size: 12 };
const mine = { size: 16 };
const settings = { ...defaults, ...mine };
// { theme: "light", size: 16 }
\`\`\``,
      code: `const defaults = { volume: 5, language: "en", fullscreen: false };\nconst userPrefs = { volume: 9, fullscreen: true };\n// Create config: defaults overridden by userPrefs, without changing either\n`,
      sol: `const defaults = { volume: 5, language: "en", fullscreen: false };\nconst userPrefs = { volume: 9, fullscreen: true };\nconst config = { ...defaults, ...userPrefs };`,
      tests: [['config.volume === 9 && config.language === "en" && config.fullscreen === true', '`config` is merged correctly'], ['defaults.volume === 5', '`defaults` is unchanged']],
    },
    {
      t: 'Objects Are References',
      b: `Like arrays, objects are stored by **reference**. Two variables can point to the same object, and \`===\` checks for the *same object*, not the same contents:

\`\`\`js
const a = { n: 1 };
const b = { n: 1 };
a === b   // false — two different objects
const c = a;
a === c   // true — the same one
\`\`\`

Write a function that checks whether two points have the same contents.`,
      code: `// Write samePoint(p, q) returning true if p.x === q.x and p.y === q.y\n`,
      sol: `function samePoint(p, q) {\n  return p.x === q.x && p.y === q.y;\n}`,
      tests: [['samePoint({ x: 1, y: 2 }, { x: 1, y: 2 }) === true', 'Equal contents → `true`'], ['samePoint({ x: 1, y: 2 }, { x: 2, y: 1 }) === false', 'Different → `false`']],
    },
    {
      t: 'Optional Chaining',
      b: `Reading a property of \`undefined\` crashes. \`?.\` stops safely and gives \`undefined\` instead:

\`\`\`js
const user = { profile: null };
user.profile.name    // TypeError!
user.profile?.name   // undefined
\`\`\`

Combine with \`??\` for a default: \`user.profile?.name ?? "Anonymous"\`.`,
      code: `// Write cityOf(person) returning person.address.city, or "Unknown" if anything is missing\n`,
      sol: `function cityOf(person) {\n  return person?.address?.city ?? "Unknown";\n}`,
      tests: [['cityOf({ address: { city: "Oslo" } }) === "Oslo"', 'Finds the city'], ['cityOf({}) === "Unknown"', 'No address → `"Unknown"`'], ['cityOf(null) === "Unknown"', 'No person → `"Unknown"`']],
    },
    {
      t: 'Freezing Objects',
      b: `\`Object.freeze\` makes an object read-only. Changes are silently ignored (or throw in strict mode):

\`\`\`js
const RULES = Object.freeze({ maxPlayers: 4 });
RULES.maxPlayers = 99;   // ignored
\`\`\``,
      code: `// Create COLORS: a frozen object with red "#f00" and blue "#00f"\n`,
      sol: `const COLORS = Object.freeze({ red: "#f00", blue: "#00f" });`,
      tests: [['COLORS.red === "#f00" && COLORS.blue === "#00f"', '`COLORS` has red and blue'], ['Object.isFrozen(COLORS)', '`COLORS` is frozen']],
    },
    {
      t: 'Project: Inventory Manager',
      b: `Build an \`inventory\` object with an \`items\` property (an object mapping names to quantities, starting empty) and these methods:

- \`add(name, qty)\` — adds to the quantity (creating the item if needed)
- \`remove(name, qty)\` — subtracts; if it drops to 0 or below, delete the item. Return \`false\` if the item did not exist, \`true\` otherwise
- \`total()\` — the total number of all items
- \`list()\` — an array of strings like \`"apples x3"\`, sorted by name`,
      code: `const inventory = {\n  items: {},\n};\n`,
      sol: 'const inventory = {\n  items: {},\n  add(name, qty) {\n    this.items[name] = (this.items[name] ?? 0) + qty;\n  },\n  remove(name, qty) {\n    if (!Object.hasOwn(this.items, name)) return false;\n    this.items[name] -= qty;\n    if (this.items[name] <= 0) delete this.items[name];\n    return true;\n  },\n  total() {\n    return Object.values(this.items).reduce((a, b) => a + b, 0);\n  },\n  list() {\n    return Object.keys(this.items)\n      .sort()\n      .map((k) => `${k} x${this.items[k]}`);\n  },\n};',
      tests: [
        ['(inventory.add("pears", 2), inventory.add("apples", 3), inventory.add("pears", 1), inventory.items.pears === 3)', '`add` creates and increases items'],
        ['inventory.total() === 6', '`total()` is `6`'],
        ['inventory.list().join() === "apples x3,pears x3"', '`list()` is sorted and formatted'],
        ['inventory.remove("apples", 5) === true && !("apples" in inventory.items)', '`remove` deletes items that reach 0'],
        ['inventory.remove("kiwis", 1) === false', '`remove` returns `false` for unknown items'],
      ],
    },
  ],
});
