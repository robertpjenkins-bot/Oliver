registerModule({
  id: 'modern',
  title: 'Modern Syntax',
  icon: '✨',
  lessons: [
    {
      t: 'Array Destructuring',
      b: `**Destructuring** unpacks values from an array into variables in one go:

\`\`\`js
const rgb = [255, 128, 0];
const [red, green, blue] = rgb;
// red = 255, green = 128, blue = 0
\`\`\``,
      code: `const podium = ["Gold Team", "Silver Team", "Bronze Team"];\n// Using destructuring, create first, second and third\n`,
      sol: `const podium = ["Gold Team", "Silver Team", "Bronze Team"];\nconst [first, second, third] = podium;`,
      tests: [['first === "Gold Team" && second === "Silver Team" && third === "Bronze Team"', 'All three variables are set'], ['/\\[\\s*first\\s*,\\s*second\\s*,\\s*third\\s*\\]\\s*=/.test($src)', 'Use array destructuring']],
    },
    {
      t: 'Skipping and Defaults',
      b: `Leave a gap to skip an item, and use \`=\` for a default when an item is missing:

\`\`\`js
const [, second] = ["a", "b", "c"];       // "b"
const [x, y, z = 0] = [1, 2];             // z = 0
\`\`\``,
      code: `const parts = "2025-07".split("-");\n// Destructure year and month from parts, and day with a default of "01"\n`,
      sol: `const parts = "2025-07".split("-");\nconst [year, month, day = "01"] = parts;`,
      tests: [['year === "2025" && month === "07"', '`year` and `month` are set'], ['day === "01"', '`day` defaults to `"01"`']],
    },
    {
      t: 'Swapping the Easy Way',
      b: `Remember swapping with a temporary variable? Destructuring does it in one line:

\`\`\`js
let a = 1, b = 2;
[a, b] = [b, a];
// a = 2, b = 1
\`\`\``,
      code: `let player1 = "X";\nlet player2 = "O";\n// Swap them with destructuring (no temp variable)\n`,
      sol: `let player1 = "X";\nlet player2 = "O";\n[player1, player2] = [player2, player1];`,
      tests: [['player1 === "O" && player2 === "X"', 'The values are swapped'], ['!/temp/i.test($src)', 'No temporary variable']],
    },
    {
      t: 'Object Destructuring',
      b: `Objects destructure by **key name** (order does not matter):

\`\`\`js
const user = { name: "Ivy", age: 30, city: "Lima" };
const { name, city } = user;
// name = "Ivy", city = "Lima"
\`\`\``,
      code: `const movie = { title: "Up", year: 2009, rating: 8.3, studio: "Pixar" };\n// Destructure title and rating\n`,
      sol: `const movie = { title: "Up", year: 2009, rating: 8.3, studio: "Pixar" };\nconst { title, rating } = movie;`,
      tests: [['title === "Up" && rating === 8.3', '`title` and `rating` are set'], ['/\\{[^}]*title[^}]*\\}\\s*=\\s*movie/.test($src)', 'Use object destructuring']],
    },
    {
      t: 'Renaming While Destructuring',
      b: `Use \`key: newName\` to store a property under a different variable name:

\`\`\`js
const { name: userName } = { name: "Kit" };
// userName = "Kit"
\`\`\`

Handy when the key is vague, or clashes with an existing variable.`,
      code: `const response = { data: [1, 2, 3], status: 200 };\n// Destructure data into a variable called items, and status into code\n`,
      sol: `const response = { data: [1, 2, 3], status: 200 };\nconst { data: items, status: code } = response;`,
      tests: [['items.join() === "1,2,3"', '`items` is [1, 2, 3]'], ['code === 200', '`code` is `200`']],
    },
    {
      t: 'Defaults for Missing Keys',
      b: `Defaults work in object destructuring too:

\`\`\`js
const { theme = "light", size = 14 } = { size: 18 };
// theme = "light", size = 18
\`\`\``,
      code: `const options = { width: 800 };\n// Destructure width (default 640), height (default 480) and title (default "Untitled")\n`,
      sol: `const options = { width: 800 };\nconst { width = 640, height = 480, title = "Untitled" } = options;`,
      tests: [['width === 800', '`width` is `800`'], ['height === 480 && title === "Untitled"', '`height` and `title` use their defaults']],
    },
    {
      t: 'Nested Destructuring',
      b: `Mirror the shape of the data to reach inside:

\`\`\`js
const order = { id: 7, customer: { name: "Ana", address: { city: "Rio" } } };
const { customer: { address: { city } } } = order;
// city = "Rio"
\`\`\``,
      code: `const tweet = {\n  text: "Hello!",\n  author: { handle: "@ada", followers: 1200 },\n  tags: ["js", "code"],\n};\n// In one destructuring, get handle, followers and firstTag (the first tag)\n`,
      sol: `const tweet = {\n  text: "Hello!",\n  author: { handle: "@ada", followers: 1200 },\n  tags: ["js", "code"],\n};\nconst { author: { handle, followers }, tags: [firstTag] } = tweet;`,
      tests: [['handle === "@ada" && followers === 1200', '`handle` and `followers` are set'], ['firstTag === "js"', '`firstTag` is `"js"`']],
    },
    {
      t: 'Destructuring Parameters',
      b: `Functions that take an options object can destructure it right in the parameter list — with defaults:

\`\`\`js
function makeButton({ label, color = "blue" }) {
  return label + " (" + color + ")";
}
makeButton({ label: "OK" });   // "OK (blue)"
\`\`\``,
      code: `// Write describe({ name, species, legs = 4 }) returning "<name> the <species> has <legs> legs"\n`,
      sol: 'function describe({ name, species, legs = 4 }) {\n  return `${name} the ${species} has ${legs} legs`;\n}',
      tests: [['describe({ name: "Rex", species: "dog" }) === "Rex the dog has 4 legs"', 'Uses the default for `legs`'], ['describe({ name: "Polly", species: "parrot", legs: 2 }) === "Polly the parrot has 2 legs"', 'Uses the given `legs`']],
    },
    {
      t: 'Rest in Arrays',
      b: `\`...rest\` in destructuring collects **everything that is left**:

\`\`\`js
const [head, ...tail] = [1, 2, 3, 4];
// head = 1, tail = [2, 3, 4]
\`\`\``,
      code: `const runners = ["Zara", "Yves", "Xena", "Will", "Vic"];\n// Destructure winner (the first) and others (the rest)\n`,
      sol: `const runners = ["Zara", "Yves", "Xena", "Will", "Vic"];\nconst [winner, ...others] = runners;`,
      tests: [['winner === "Zara"', '`winner` is `"Zara"`'], ['others.join() === "Yves,Xena,Will,Vic"', '`others` has the rest']],
    },
    {
      t: 'Rest in Objects',
      b: `Object rest collects the remaining properties — a tidy way to **leave one out**:

\`\`\`js
const { password, ...safeUser } = user;
// safeUser has everything except password
\`\`\``,
      code: `const user = { id: 1, name: "Mo", password: "s3cret", token: "abc" };\n// Create publicUser: user without password and token, using object rest\n`,
      sol: `const user = { id: 1, name: "Mo", password: "s3cret", token: "abc" };\nconst { password, token, ...publicUser } = user;`,
      tests: [['JSON.stringify(publicUser) === \'{"id":1,"name":"Mo"}\'', '`publicUser` is { id: 1, name: "Mo" }'], ['user.password === "s3cret"', '`user` is unchanged']],
    },
    {
      t: 'Spreading Arguments',
      b: `Spread an array into separate arguments of a function call:

\`\`\`js
const nums = [5, 1, 9];
Math.max(...nums);   // 9 — same as Math.max(5, 1, 9)
\`\`\``,
      code: `const heights = [152, 171, 164, 180, 158];\n// Create tallest and shortest using Math.max / Math.min and spread\n`,
      sol: `const heights = [152, 171, 164, 180, 158];\nconst tallest = Math.max(...heights);\nconst shortest = Math.min(...heights);`,
      tests: [['tallest === 180 && shortest === 152', '`tallest` is 180, `shortest` is 152'], ['$src.includes("...heights")', 'Spread `heights`']],
    },
    {
      t: 'Spreading Strings',
      b: `Spreading a string gives you its characters — and, unlike \`split("")\`, it handles emoji correctly:

\`\`\`js
[..."hi"]      // ["h", "i"]
[..."👍🎉"]    // ["👍", "🎉"]
"👍🎉".split("")  // 4 broken halves!
\`\`\``,
      code: `const party = "🎈🎂🎉";\n// Create emojis: an array of the three emoji, and count: how many there are\n`,
      sol: `const party = "🎈🎂🎉";\nconst emojis = [...party];\nconst count = emojis.length;`,
      tests: [['count === 3', '`count` is `3`'], ['emojis[1] === "🎂"', '`emojis[1]` is 🎂']],
    },
    {
      t: 'Optional Calls and Indexes',
      b: `Optional chaining also works for calling functions and reading indexes that might not exist:

\`\`\`js
obj.method?.()   // call only if method exists
arr?.[0]         // first item only if arr exists
\`\`\``,
      code: `// Write firstTag(post) returning the first item of post.tags, or "none"\n// (post.tags might be missing)\n// Write notify(settings) calling settings.onSave?.() and returning "done"\n`,
      sol: `function firstTag(post) {\n  return post.tags?.[0] ?? "none";\n}\nfunction notify(settings) {\n  settings.onSave?.();\n  return "done";\n}`,
      tests: [['firstTag({ tags: ["news"] }) === "news"', 'Finds the first tag'], ['firstTag({}) === "none"', 'No tags → `"none"`'], ['notify({}) === "done"', '`notify` works without `onSave`'], ['(() => { let hit = false; notify({ onSave: () => (hit = true) }); return hit; })()', '`notify` calls `onSave` when present']],
    },
    {
      t: 'Logical Assignment',
      b: `Three shortcuts for "assign only if…":

\`\`\`js
a ||= b;   // a = a || b   — assign if a is falsy
a &&= b;   // a = a && b   — assign if a is truthy
a ??= b;   // a = a ?? b   — assign if a is null/undefined
\`\`\``,
      code: `const config = { retries: 0, name: null, debug: true };\n// Using ??=, give retries a default of 3 and name a default of "app"\n// Using &&=, set debug to "verbose" only if it is already truthy\n`,
      sol: `const config = { retries: 0, name: null, debug: true };\nconfig.retries ??= 3;\nconfig.name ??= "app";\nconfig.debug &&= "verbose";`,
      tests: [['config.retries === 0', '`retries` keeps its `0`'], ['config.name === "app"', '`name` becomes `"app"`'], ['config.debug === "verbose"', '`debug` becomes `"verbose"`'], ['$src.includes("??=") && $src.includes("&&=")', 'Use `??=` and `&&=`']],
    },
    {
      t: 'Tagged Templates',
      b: `Put a function name before a template literal and the function receives the pieces: the fixed strings and the inserted values. It can then build whatever it likes.

\`\`\`js
function loud(strings, ...values) {
  return strings.reduce((out, s, i) =>
    out + s + (i < values.length ? String(values[i]).toUpperCase() : ""), "");
}
loud\`hi \${"sam"}!\`   // "hi SAM!"
\`\`\``,
      code: `// Write a tag function money that formats every inserted number with toFixed(2)\n// money\`Pay \${5} now\` → "Pay 5.00 now"\n`,
      sol: 'function money(strings, ...values) {\n  return strings.reduce((out, s, i) => out + s + (i < values.length ? values[i].toFixed(2) : ""), "");\n}',
      tests: [['money`Pay ${5} now` === "Pay 5.00 now"', 'One value works'], ['money`${1.5} + ${2} = ${3.5}` === "1.50 + 2.00 = 3.50"', 'Several values work']],
    },
    {
      t: 'Destructuring in Loops',
      b: `Destructure right inside \`for…of\` and callback parameters:

\`\`\`js
for (const [key, value] of Object.entries(prices)) { … }
people.map(({ name }) => name);
\`\`\``,
      code: `const scores = { ana: 12, ben: 7, cal: 15 };\n// Create winners: names with a score above 10, using Object.entries,\n// filter(([name, score]) => …) and map(([name]) => …)\n`,
      sol: `const scores = { ana: 12, ben: 7, cal: 15 };\nconst winners = Object.entries(scores)\n  .filter(([name, score]) => score > 10)\n  .map(([name]) => name);`,
      tests: [['winners.join() === "ana,cal"', '`winners` is ["ana", "cal"]'], ['$src.includes("([")', 'Destructure in the callbacks']],
    },
    {
      t: 'Grouping Data',
      b: `\`Object.groupBy\` sorts items into groups by whatever your function returns:

\`\`\`js
const animals = [{ n: "cat", legs: 4 }, { n: "hen", legs: 2 }, { n: "dog", legs: 4 }];
Object.groupBy(animals, a => a.legs);
// { 2: [hen], 4: [cat, dog] }
\`\`\``,
      code: `const words = ["apple", "avocado", "banana", "blueberry", "cherry"];\n// Create byLetter: words grouped by their first letter, using Object.groupBy\n`,
      sol: `const words = ["apple", "avocado", "banana", "blueberry", "cherry"];\nconst byLetter = Object.groupBy(words, (w) => w[0]);`,
      tests: [['byLetter.a.join() === "apple,avocado" && byLetter.b.length === 2 && byLetter.c[0] === "cherry"', '`byLetter` groups the words'], ['$src.includes("Object.groupBy")', 'Use `Object.groupBy`']],
    },
    {
      t: 'Deep Copies',
      b: `Spread only copies **one level** deep — nested objects are still shared. \`structuredClone\` makes a full, deep copy:

\`\`\`js
const original = { pos: { x: 1 } };
const shallow = { ...original };
shallow.pos.x = 99;          // original.pos.x is now 99 too!
const deep = structuredClone(original);
deep.pos.x = 5;              // original is safe
\`\`\``,
      code: `const level = { name: "Cave", enemies: [{ type: "bat", hp: 3 }] };\nconst hardLevel = { ...level };\nhardLevel.enemies[0].hp = 10;\n// Fix it so changing hardLevel does NOT change level\n`,
      sol: `const level = { name: "Cave", enemies: [{ type: "bat", hp: 3 }] };\nconst hardLevel = structuredClone(level);\nhardLevel.enemies[0].hp = 10;`,
      tests: [['level.enemies[0].hp === 3', '`level` is unchanged'], ['hardLevel.enemies[0].hp === 10', '`hardLevel` has the tougher bat']],
    },
    {
      t: 'Getters and Setters',
      b: `\`get\` and \`set\` make properties that run code when read or written:

\`\`\`js
const temp = {
  celsius: 20,
  get fahrenheit() { return this.celsius * 9 / 5 + 32; },
  set fahrenheit(f) { this.celsius = (f - 32) * 5 / 9; },
};
temp.fahrenheit;       // 68 — no () needed
temp.fahrenheit = 212; // celsius becomes 100
\`\`\``,
      code: `const person = {\n  first: "Ada",\n  last: "Lovelace",\n  // add a getter fullName returning "first last"\n  // add a setter fullName that splits a string into first and last\n};\n`,
      sol: 'const person = {\n  first: "Ada",\n  last: "Lovelace",\n  get fullName() {\n    return `${this.first} ${this.last}`;\n  },\n  set fullName(value) {\n    [this.first, this.last] = value.split(" ");\n  },\n};',
      tests: [['person.fullName === "Ada Lovelace"', 'The getter works'], ['(person.fullName = "Alan Turing", person.first === "Alan" && person.last === "Turing")', 'The setter works']],
    },
    {
      t: 'Project: Settings Loader',
      b: `Write \`loadSettings(raw)\` that takes a possibly-incomplete settings object and returns a complete one:

- \`volume\` default \`5\` (keep \`0\` if given!)
- \`theme\` default \`"dark"\`
- \`player\` default \`{ name: "Guest", color: "blue" }\`, but if a partial \`player\` is given, fill in only its missing fields
- any unknown extra keys should be **dropped**
- \`raw\` may be \`undefined\``,
      code: `function loadSettings(raw) {\n}\n`,
      sol: `function loadSettings(raw) {\n  const { volume = 5, theme = "dark", player = {} } = raw ?? {};\n  const { name = "Guest", color = "blue" } = player;\n  return { volume, theme, player: { name, color } };\n}`,
      tests: [
        ['JSON.stringify(loadSettings()) === \'{"volume":5,"theme":"dark","player":{"name":"Guest","color":"blue"}}\'', '`loadSettings()` returns all defaults'],
        ['loadSettings({ volume: 0 }).volume === 0', 'A volume of `0` is kept'],
        ['JSON.stringify(loadSettings({ player: { name: "Zed" } }).player) === \'{"name":"Zed","color":"blue"}\'', 'A partial player is completed'],
        ['!("hack" in loadSettings({ hack: true }))', 'Unknown keys are dropped'],
      ],
    },
  ],
});
