registerModule({
  id: 'arraymethods',
  title: 'Array Superpowers',
  icon: '⚡',
  lessons: [
    {
      t: 'forEach',
      b: `\`forEach\` calls a function once for every item. The function receives the item (and its index):

\`\`\`js
["a", "b"].forEach((letter, i) => {
  console.log(i, letter);
});
\`\`\`

It is a tidy replacement for many \`for\` loops.`,
      code: `const heroes = ["Ada", "Alan", "Grace"];\n// Using forEach, print "Hero: <name>" for each\n`,
      sol: `const heroes = ["Ada", "Alan", "Grace"];\nheroes.forEach((h) => console.log("Hero: " + h));`,
      tests: [['$out() === "Hero: Ada\\nHero: Alan\\nHero: Grace"', 'Print each hero'], ['$src.includes("forEach")', 'Use `forEach`']],
    },
    {
      t: 'map: Transform Every Item',
      b: `\`map\` builds a **new** array by transforming each item with your function:

\`\`\`js
const prices = [10, 20, 30];
const doubled = prices.map(p => p * 2);  // [20, 40, 60]
\`\`\`

The new array always has the same length as the original.`,
      code: `const celsius = [0, 20, 100];\n// Create fahrenheit using map (F = C * 9/5 + 32)\n`,
      sol: `const celsius = [0, 20, 100];\nconst fahrenheit = celsius.map((c) => c * 9 / 5 + 32);`,
      tests: [['fahrenheit.join() === "32,68,212"', '`fahrenheit` is [32, 68, 212]'], ['$src.includes(".map(")', 'Use `map`']],
    },
    {
      t: 'map with Strings',
      b: `\`map\` works with any kind of value, and the callback also gets the index as its second argument:

\`\`\`js
["a", "b"].map((x, i) => i + ":" + x)   // ["0:a", "1:b"]
\`\`\``,
      code: `const names = ["zoe", "yuki", "xavi"];\n// Create labels like ["1. Zoe", "2. Yuki", "3. Xavi"] using map\n`,
      sol: 'const names = ["zoe", "yuki", "xavi"];\nconst labels = names.map((n, i) => `${i + 1}. ${n[0].toUpperCase()}${n.slice(1)}`);',
      tests: [['labels.join("|") === "1. Zoe|2. Yuki|3. Xavi"', '`labels` is correct'], ['$src.includes(".map(")', 'Use `map`']],
    },
    {
      t: 'filter: Keep Some Items',
      b: `\`filter\` keeps only the items for which your function returns \`true\`:

\`\`\`js
const ages = [5, 17, 22, 40];
const adults = ages.filter(a => a >= 18);   // [22, 40]
\`\`\``,
      code: `const numbers = [3, 8, 11, 14, 20, 27];\n// Create evens using filter\n`,
      sol: `const numbers = [3, 8, 11, 14, 20, 27];\nconst evens = numbers.filter((n) => n % 2 === 0);`,
      tests: [['evens.join() === "8,14,20"', '`evens` is [8, 14, 20]'], ['$src.includes(".filter(")', 'Use `filter`']],
    },
    {
      t: 'find: The First Match',
      b: `\`find\` returns the **first** item that matches, or \`undefined\` if none does:

\`\`\`js
const nums = [4, 9, 16, 25];
nums.find(n => n > 10)   // 16
nums.find(n => n > 99)   // undefined
\`\`\``,
      code: `const words = ["tree", "apple", "sky", "orange", "egg"];\n// Create firstVowelWord: the first word that starts with a vowel\n`,
      sol: `const words = ["tree", "apple", "sky", "orange", "egg"];\nconst firstVowelWord = words.find((w) => "aeiou".includes(w[0]));`,
      tests: [['firstVowelWord === "apple"', '`firstVowelWord` is `"apple"`'], ['$src.includes(".find(")', 'Use `find`']],
    },
    {
      t: 'findIndex',
      b: `\`findIndex\` is like \`find\` but returns the **position** (or \`-1\`):

\`\`\`js
[5, 12, 8].findIndex(n => n > 10)   // 1
\`\`\``,
      code: `const temps = [12, 15, 19, 24, 28, 22];\n// Create firstHotDay: the index of the first temperature above 20\n`,
      sol: `const temps = [12, 15, 19, 24, 28, 22];\nconst firstHotDay = temps.findIndex((t) => t > 20);`,
      tests: [['firstHotDay === 3', '`firstHotDay` is `3`'], ['$src.includes("findIndex")', 'Use `findIndex`']],
    },
    {
      t: 'some: Any Match?',
      b: `\`some\` returns \`true\` if **at least one** item passes your test:

\`\`\`js
[1, 3, 6].some(n => n % 2 === 0)   // true
\`\`\``,
      code: `const passwords = ["abc", "hunter2", "letmein"];\n// Create anyHasDigit: whether some password contains a digit\n// (tip: /\\d/.test(p) checks for a digit)\n`,
      sol: `const passwords = ["abc", "hunter2", "letmein"];\nconst anyHasDigit = passwords.some((p) => /\\d/.test(p));`,
      tests: [['anyHasDigit === true', '`anyHasDigit` is `true`'], ['$src.includes(".some(")', 'Use `some`']],
    },
    {
      t: 'every: All Match?',
      b: `\`every\` returns \`true\` only if **all** items pass:

\`\`\`js
[2, 4, 6].every(n => n % 2 === 0)   // true
[2, 5, 6].every(n => n % 2 === 0)   // false
\`\`\``,
      code: `const answers = ["yes", "yes", "YES", "yes"];\n// Create unanimous: whether every answer is "yes", ignoring case\n`,
      sol: `const answers = ["yes", "yes", "YES", "yes"];\nconst unanimous = answers.every((a) => a.toLowerCase() === "yes");`,
      tests: [['unanimous === true', '`unanimous` is `true`'], ['$src.includes(".every(")', 'Use `every`']],
    },
    {
      t: 'reduce: Boil It Down',
      b: `\`reduce\` combines all items into **one** value. It passes a running total (the **accumulator**) and each item to your function:

\`\`\`js
const total = [1, 2, 3].reduce((sum, n) => sum + n, 0);  // 6
\`\`\`

The \`0\` at the end is the starting value for the accumulator.`,
      code: `const cart = [4.5, 12, 3.25];\n// Create total using reduce\n`,
      sol: `const cart = [4.5, 12, 3.25];\nconst total = cart.reduce((sum, p) => sum + p, 0);`,
      tests: [['total === 19.75', '`total` is `19.75`'], ['$src.includes(".reduce(")', 'Use `reduce`']],
    },
    {
      t: 'reduce into an Object',
      b: `The accumulator can be anything — even an object. Counting things is a classic:

\`\`\`js
const votes = ["cat", "dog", "cat"];
const tally = votes.reduce((acc, v) => {
  acc[v] = (acc[v] || 0) + 1;
  return acc;
}, {});
// { cat: 2, dog: 1 }
\`\`\`

Do not forget to \`return acc\`!`,
      code: `const letters = ["a", "b", "a", "c", "b", "a"];\n// Create counts: an object counting each letter, using reduce\n`,
      sol: `const letters = ["a", "b", "a", "c", "b", "a"];\nconst counts = letters.reduce((acc, l) => {\n  acc[l] = (acc[l] || 0) + 1;\n  return acc;\n}, {});`,
      tests: [['counts.a === 3 && counts.b === 2 && counts.c === 1', '`counts` is { a: 3, b: 2, c: 1 }'], ['$src.includes(".reduce(")', 'Use `reduce`']],
    },
    {
      t: 'Sorting Words',
      b: `\`sort()\` sorts an array **in place**. With no arguments it sorts as text (alphabetically):

\`\`\`js
const pets = ["dog", "cat", "bird"];
pets.sort();   // ["bird", "cat", "dog"]
\`\`\`

For a case-insensitive sort, use \`localeCompare\`:

\`\`\`js
names.sort((a, b) => a.localeCompare(b));
\`\`\``,
      code: `const cities = ["paris", "Berlin", "amsterdam", "Cairo"];\n// Sort cities alphabetically ignoring case, using localeCompare\n`,
      sol: `const cities = ["paris", "Berlin", "amsterdam", "Cairo"];\ncities.sort((a, b) => a.localeCompare(b, undefined, { sensitivity: "base" }));`,
      tests: [['cities.join() === "amsterdam,Berlin,Cairo,paris"', '`cities` is sorted ignoring case'], ['$src.includes("localeCompare")', 'Use `localeCompare`']],
    },
    {
      t: 'Sorting Numbers',
      b: `Watch out! Plain \`sort()\` compares as **text**, so \`[10, 9, 1].sort()\` gives \`[1, 10, 9]\`.

For numbers, pass a **compare function**. Return a negative number if \`a\` should come first:

\`\`\`js
nums.sort((a, b) => a - b);  // ascending
nums.sort((a, b) => b - a);  // descending
\`\`\``,
      code: `const scores = [40, 100, 1, 5, 25, 10];\n// Sort scores from highest to lowest\n`,
      sol: `const scores = [40, 100, 1, 5, 25, 10];\nscores.sort((a, b) => b - a);`,
      tests: [['scores.join() === "100,40,25,10,5,1"', '`scores` is sorted high to low']],
    },
    {
      t: 'Sorting by a Property',
      b: `Compare functions can look inside objects:

\`\`\`js
people.sort((a, b) => a.age - b.age);
\`\`\``,
      code: `const racers = [\n  { name: "Kit", time: 12.4 },\n  { name: "Lou", time: 11.9 },\n  { name: "Max", time: 12.1 },\n];\n// Sort racers by time, fastest first, then create winner (the fastest racer's name)\n`,
      sol: `const racers = [\n  { name: "Kit", time: 12.4 },\n  { name: "Lou", time: 11.9 },\n  { name: "Max", time: 12.1 },\n];\nracers.sort((a, b) => a.time - b.time);\nconst winner = racers[0].name;`,
      tests: [['racers.map((r) => r.name).join() === "Lou,Max,Kit"', '`racers` sorted by time'], ['winner === "Lou"', '`winner` is `"Lou"`']],
    },
    {
      t: 'Chaining Methods',
      b: `Because \`map\` and \`filter\` return arrays, you can **chain** them:

\`\`\`js
const result = [1, 2, 3, 4]
  .filter(n => n % 2 === 0)   // [2, 4]
  .map(n => n * 10);          // [20, 40]
\`\`\``,
      code: `const products = [\n  { name: "Pen", price: 2, inStock: true },\n  { name: "Lamp", price: 30, inStock: false },\n  { name: "Mug", price: 8, inStock: true },\n];\n// Create available: names of in-stock products, in capitals, using filter then map\n`,
      sol: `const products = [\n  { name: "Pen", price: 2, inStock: true },\n  { name: "Lamp", price: 30, inStock: false },\n  { name: "Mug", price: 8, inStock: true },\n];\nconst available = products.filter((p) => p.inStock).map((p) => p.name.toUpperCase());`,
      tests: [['available.join() === "PEN,MUG"', '`available` is ["PEN", "MUG"]'], ['$src.includes(".filter(") && $src.includes(".map(")', 'Chain `filter` and `map`']],
    },
    {
      t: 'Flattening',
      b: `\`flat()\` squashes nested arrays one level; \`flatMap\` maps and then flattens:

\`\`\`js
[[1, 2], [3], [4, 5]].flat()          // [1, 2, 3, 4, 5]
["a b", "c"].flatMap(s => s.split(" "))  // ["a", "b", "c"]
\`\`\``,
      code: `const sentences = ["the cat sat", "on the mat"];\n// Create words: every word from every sentence in one array, using flatMap\n`,
      sol: `const sentences = ["the cat sat", "on the mat"];\nconst words = sentences.flatMap((s) => s.split(" "));`,
      tests: [['words.join() === "the,cat,sat,on,the,mat"', '`words` has all six words'], ['$src.includes("flatMap")', 'Use `flatMap`']],
    },
    {
      t: 'Building Arrays',
      b: `Quick ways to make arrays:

\`\`\`js
Array(3).fill(0)                       // [0, 0, 0]
Array.from({ length: 4 }, (_, i) => i) // [0, 1, 2, 3]
Array.from("hey")                      // ["h", "e", "y"]
\`\`\``,
      code: `// Create squares: [1, 4, 9, …, 100] (the first 10 square numbers) using Array.from\n`,
      sol: `const squares = Array.from({ length: 10 }, (_, i) => (i + 1) ** 2);`,
      tests: [['squares.join() === "1,4,9,16,25,36,49,64,81,100"', '`squares` is correct'], ['$src.includes("Array.from")', 'Use `Array.from`']],
    },
    {
      t: 'Removing Duplicates',
      b: `A clever \`filter\`: keep an item only if this is the **first** place it appears.

\`\`\`js
arr.filter((x, i) => arr.indexOf(x) === i)
\`\`\`

(Later you will see an even shorter way with \`Set\`.)`,
      code: `const tags = ["js", "web", "js", "css", "web", "html"];\n// Create unique: tags without duplicates, keeping first-seen order, using filter\n`,
      sol: `const tags = ["js", "web", "js", "css", "web", "html"];\nconst unique = tags.filter((t, i) => tags.indexOf(t) === i);`,
      tests: [['unique.join() === "js,web,css,html"', '`unique` is js, web, css, html'], ['$src.includes(".filter(")', 'Use `filter`']],
    },
    {
      t: 'Non-Mutating Versions',
      b: `\`sort\`, \`reverse\` and \`splice\` change the original array. Newer JavaScript has safe versions that return a **new** array instead:

\`\`\`js
const nums = [3, 1, 2];
const sorted = nums.toSorted((a, b) => a - b); // [1, 2, 3]
nums;                                          // still [3, 1, 2]
nums.toReversed();   // [2, 1, 3]
nums.with(0, 99);    // [99, 1, 2]
\`\`\``,
      code: `const original = [5, 2, 9, 1];\n// Create ascending (sorted copy) and backwards (reversed copy), leaving original unchanged\n`,
      sol: `const original = [5, 2, 9, 1];\nconst ascending = original.toSorted((a, b) => a - b);\nconst backwards = original.toReversed();`,
      tests: [['ascending.join() === "1,2,5,9"', '`ascending` is [1, 2, 5, 9]'], ['backwards.join() === "1,9,2,5"', '`backwards` is [1, 9, 2, 5]'], ['original.join() === "5,2,9,1"', '`original` is unchanged']],
    },
    {
      t: 'Picking the Right Method',
      b: `A quick cheat sheet:

- transform every item → \`map\`
- keep some items → \`filter\`
- one item → \`find\`
- yes/no → \`some\` / \`every\`
- one combined value → \`reduce\`
- just do something → \`forEach\`

Use the right ones to complete these three.`,
      code: `const words = ["kiwi", "banana", "fig", "cherry"];\n// lengths: the length of each word\n// long: the words longer than 4 letters\n// totalLetters: the sum of all lengths\n`,
      sol: `const words = ["kiwi", "banana", "fig", "cherry"];\nconst lengths = words.map((w) => w.length);\nconst long = words.filter((w) => w.length > 4);\nconst totalLetters = lengths.reduce((a, b) => a + b, 0);`,
      tests: [['lengths.join() === "4,6,3,6"', '`lengths` is [4, 6, 3, 6]'], ['long.join() === "banana,cherry"', '`long` is ["banana", "cherry"]'], ['totalLetters === 19', '`totalLetters` is `19`']],
    },
    {
      t: 'Project: Leaderboard',
      b: `Write \`leaderboard(players)\`. Each player looks like \`{ name: "Ivy", points: 320 }\`.

Return an array of strings for the **top 3** players, highest first, formatted like:

\`\`\`
["🥇 Ivy (320)", "🥈 Rex (290)", "🥉 Zia (150)"]
\`\`\`

Do **not** change the array you were given. If there are fewer than 3 players, return as many as there are.`,
      code: `function leaderboard(players) {\n}\n`,
      sol: 'function leaderboard(players) {\n  const medals = ["🥇", "🥈", "🥉"];\n  return players\n    .toSorted((a, b) => b.points - a.points)\n    .slice(0, 3)\n    .map((p, i) => `${medals[i]} ${p.name} (${p.points})`);\n}',
      tests: [
        ['(() => { const ps = [{ name: "Zia", points: 150 }, { name: "Ivy", points: 320 }, { name: "Bo", points: 40 }, { name: "Rex", points: 290 }]; return leaderboard(ps).join("|") === "🥇 Ivy (320)|🥈 Rex (290)|🥉 Zia (150)"; })()', 'Returns the top 3, formatted'],
        ['(() => { const ps = [{ name: "A", points: 1 }, { name: "B", points: 2 }]; leaderboard(ps); return ps[0].name === "A"; })()', 'Does not reorder the original array'],
        ['leaderboard([{ name: "Solo", points: 9 }]).join() === "🥇 Solo (9)"', 'Works with one player'],
      ],
    },
  ],
});
