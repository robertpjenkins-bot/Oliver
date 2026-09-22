registerModule({
  id: 'arrays',
  title: 'Arrays',
  icon: '📚',
  lessons: [
    {
      t: 'Making a List',
      b: `An **array** holds an ordered list of values, written in square brackets:

\`\`\`js
const fruits = ["apple", "banana", "cherry"];
const scores = [90, 72, 85];
const mixed = ["hi", 42, true];   // any types
const empty = [];
\`\`\``,
      code: `// Create an array called planets with "Mercury", "Venus" and "Earth"\n`,
      sol: `const planets = ["Mercury", "Venus", "Earth"];`,
      tests: [['Array.isArray(planets)', '`planets` is an array'], ['planets.join() === "Mercury,Venus,Earth"', 'It holds Mercury, Venus, Earth in order']],
    },
    {
      t: 'Reading Items',
      b: `Get an item by its **index** in square brackets. Like strings, arrays start counting at **0**:

\`\`\`js
const colors = ["red", "green", "blue"];
colors[0]      // "red"
colors[2]      // "blue"
colors[3]      // undefined — nothing there
colors.at(-1)  // "blue" — the last one
\`\`\``,
      code: `const podium = ["Ana", "Ben", "Cleo", "Dev"];\n// Create gold (first item), bronze (third item) and lastPlace (last item)\n`,
      sol: `const podium = ["Ana", "Ben", "Cleo", "Dev"];\nconst gold = podium[0];\nconst bronze = podium[2];\nconst lastPlace = podium.at(-1);`,
      tests: [['gold === "Ana"', '`gold` is `"Ana"`'], ['bronze === "Cleo"', '`bronze` is `"Cleo"`'], ['lastPlace === "Dev"', '`lastPlace` is `"Dev"`'], ['!/["\']Dev["\']/.test($src.split(";").slice(1).join(";"))', 'Read the values from the array']],
    },
    {
      t: 'Changing Items',
      b: `Unlike strings, arrays can be changed. Assign to an index:

\`\`\`js
const pets = ["cat", "dog"];
pets[1] = "parrot";   // ["cat", "parrot"]
\`\`\`

Even a \`const\` array can have its contents changed — \`const\` only stops you pointing the name at a *different* array.`,
      code: `const week = ["Mon", "Tue", "Wednesday", "Thu"];\n// Change "Wednesday" to "Wed"\n`,
      sol: `const week = ["Mon", "Tue", "Wednesday", "Thu"];\nweek[2] = "Wed";`,
      tests: [['week.join() === "Mon,Tue,Wed,Thu"', '`week` is Mon, Tue, Wed, Thu'], ['/week\\[2\\]\\s*=/.test($src)', 'Change it by index']],
    },
    {
      t: 'How Many Items?',
      b: `\`.length\` tells you how many items an array has. The last index is always \`length - 1\`:

\`\`\`js
const nums = [4, 8, 15];
nums.length                 // 3
nums[nums.length - 1]       // 15
\`\`\``,
      code: `const guests = ["Kai", "Mia", "Noor", "Oto", "Pia"];\n// Create count (number of guests) and lastGuest using length\n`,
      sol: `const guests = ["Kai", "Mia", "Noor", "Oto", "Pia"];\nconst count = guests.length;\nconst lastGuest = guests[guests.length - 1];`,
      tests: [['count === 5', '`count` is `5`'], ['lastGuest === "Pia"', '`lastGuest` is `"Pia"`'], ['$src.includes("length - 1")', 'Use `length - 1`']],
    },
    {
      t: 'Adding to the End',
      b: `\`push\` adds items to the **end** of an array (and returns the new length):

\`\`\`js
const queue = ["A"];
queue.push("B");        // ["A", "B"]
queue.push("C", "D");   // ["A", "B", "C", "D"]
\`\`\``,
      code: `const cart = ["milk"];\n// Add "eggs" and then "bread" to the end of cart\n`,
      sol: `const cart = ["milk"];\ncart.push("eggs");\ncart.push("bread");`,
      tests: [['cart.join() === "milk,eggs,bread"', '`cart` is milk, eggs, bread'], ['$src.includes("push")', 'Use `push`']],
    },
    {
      t: 'Removing from the End',
      b: `\`pop\` removes the **last** item and returns it:

\`\`\`js
const stack = [1, 2, 3];
const top = stack.pop();  // top is 3, stack is [1, 2]
\`\`\`

An array used with push/pop is called a **stack** — like a pile of plates.`,
      code: `const plates = ["blue", "red", "green"];\n// Take the top plate off with pop and store it in taken\n`,
      sol: `const plates = ["blue", "red", "green"];\nconst taken = plates.pop();`,
      tests: [['taken === "green"', '`taken` is `"green"`'], ['plates.join() === "blue,red"', '`plates` is now blue, red']],
    },
    {
      t: 'Working at the Front',
      b: `\`unshift\` adds to the **front**, and \`shift\` removes from the front:

\`\`\`js
const line = ["B", "C"];
line.unshift("A");        // ["A", "B", "C"]
const first = line.shift(); // "A", line is ["B", "C"]
\`\`\`

push + shift makes a **queue** — first in, first out.`,
      code: `const queue = ["Sam", "Tia"];\n// "Uma" joins at the back. Then the person at the front is served:\n// remove them and store them in served\n`,
      sol: `const queue = ["Sam", "Tia"];\nqueue.push("Uma");\nconst served = queue.shift();`,
      tests: [['served === "Sam"', '`served` is `"Sam"`'], ['queue.join() === "Tia,Uma"', '`queue` is Tia, Uma']],
    },
    {
      t: 'Looping Through an Array',
      b: `The classic way to visit every item:

\`\`\`js
const nums = [3, 6, 9];
for (let i = 0; i < nums.length; i++) {
  console.log(i, nums[i]);
}
\`\`\``,
      code: `const animals = ["ant", "bee", "cat"];\n// Print each animal like "1. ant", "2. bee", "3. cat"\n`,
      sol: 'const animals = ["ant", "bee", "cat"];\nfor (let i = 0; i < animals.length; i++) {\n  console.log(`${i + 1}. ${animals[i]}`);\n}',
      tests: [['$out() === "1. ant\\n2. bee\\n3. cat"', 'Print a numbered list']],
    },
    {
      t: 'for…of with Arrays',
      b: `When you do not need the index, \`for…of\` is cleaner:

\`\`\`js
for (const name of ["Ali", "Bea"]) {
  console.log("Hi " + name);
}
\`\`\``,
      code: `const words = ["sun", "moon", "star"];\n// Using for...of, print each word in capitals\n`,
      sol: `const words = ["sun", "moon", "star"];\nfor (const w of words) {\n  console.log(w.toUpperCase());\n}`,
      tests: [['$out() === "SUN\\nMOON\\nSTAR"', 'Print SUN, MOON, STAR'], ['$src.includes(" of ")', 'Use `for…of`']],
    },
    {
      t: 'Is It in There?',
      b: `\`includes\` checks if an array contains a value; \`indexOf\` tells you where (or \`-1\`):

\`\`\`js
const tags = ["js", "css", "html"];
tags.includes("css")   // true
tags.indexOf("html")   // 2
tags.indexOf("php")    // -1
\`\`\``,
      code: `const invited = ["Ivy", "Jon", "Kim"];\n// Create jonInvited (is "Jon" in the list?) and zedInvited (is "Zed"?)\n// Create kimPosition: the index of "Kim"\n`,
      sol: `const invited = ["Ivy", "Jon", "Kim"];\nconst jonInvited = invited.includes("Jon");\nconst zedInvited = invited.includes("Zed");\nconst kimPosition = invited.indexOf("Kim");`,
      tests: [['jonInvited === true && zedInvited === false', '`jonInvited` true, `zedInvited` false'], ['kimPosition === 2', '`kimPosition` is `2`']],
    },
    {
      t: 'Totalling an Array',
      b: `Summing a list is a loop plus an accumulator:

\`\`\`js
let total = 0;
for (const n of [5, 10, 15]) total += n;
// 30
\`\`\``,
      code: `// Write sumArray(numbers) returning the sum of an array of numbers\n`,
      sol: `function sumArray(numbers) {\n  let total = 0;\n  for (const n of numbers) total += n;\n  return total;\n}`,
      tests: [['sumArray([1, 2, 3, 4]) === 10', '`sumArray([1, 2, 3, 4])` is `10`'], ['sumArray([]) === 0', '`sumArray([])` is `0`']],
    },
    {
      t: 'Finding the Largest',
      b: `To find the biggest item yourself: start with the first item as the "best so far", then compare each other item against it.`,
      code: `// Write largest(numbers) returning the biggest number, without Math.max\n`,
      sol: `function largest(numbers) {\n  let best = numbers[0];\n  for (const n of numbers) {\n    if (n > best) best = n;\n  }\n  return best;\n}`,
      tests: [['largest([3, 17, 8]) === 17', '`largest([3, 17, 8])` is `17`'], ['largest([-5, -2, -9]) === -2', 'Works with negative numbers'], ['!$src.includes("Math.max")', 'Do not use `Math.max`']],
      hint: 'Starting with `best = 0` breaks for all-negative arrays. Start with `numbers[0]`.',
    },
    {
      t: 'Slicing Arrays',
      b: `\`slice(start, end)\` copies part of an array into a **new** array — the original is untouched:

\`\`\`js
const letters = ["a", "b", "c", "d", "e"];
letters.slice(1, 3)   // ["b", "c"]
letters.slice(-2)     // ["d", "e"]
letters.slice()       // a full copy
\`\`\``,
      code: `const top10 = ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j"];\n// Create top3 (first three) and bottom2 (last two)\n`,
      sol: `const top10 = ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j"];\nconst top3 = top10.slice(0, 3);\nconst bottom2 = top10.slice(-2);`,
      tests: [['top3.join() === "a,b,c"', '`top3` is a, b, c'], ['bottom2.join() === "i,j"', '`bottom2` is i, j'], ['top10.length === 10', '`top10` is unchanged']],
    },
    {
      t: 'Splicing',
      b: `\`splice(start, deleteCount, ...items)\` changes an array **in place**: it can remove items, insert items, or both.

\`\`\`js
const a = ["x", "y", "z"];
a.splice(1, 1);          // removes "y"  → ["x", "z"]
a.splice(1, 0, "NEW");   // inserts      → ["x", "NEW", "z"]
\`\`\``,
      code: `const song = ["do", "re", "oops", "fa", "so"];\n// Replace "oops" with "mi" using splice\n`,
      sol: `const song = ["do", "re", "oops", "fa", "so"];\nsong.splice(2, 1, "mi");`,
      tests: [['song.join() === "do,re,mi,fa,so"', '`song` is do, re, mi, fa, so'], ['$src.includes("splice")', 'Use `splice`']],
    },
    {
      t: 'Joining Arrays Together',
      b: `Combine arrays with \`concat\`, or with the **spread** syntax \`...\`, which unpacks an array's items:

\`\`\`js
const a = [1, 2], b = [3, 4];
a.concat(b)     // [1, 2, 3, 4]
[...a, ...b]    // [1, 2, 3, 4]
[0, ...a, 99]   // [0, 1, 2, 99]
\`\`\``,
      code: `const morning = ["wake", "eat"];\nconst evening = ["read", "sleep"];\n// Create day: "start", then morning, then "work", then evening — using spread\n`,
      sol: `const morning = ["wake", "eat"];\nconst evening = ["read", "sleep"];\nconst day = ["start", ...morning, "work", ...evening];`,
      tests: [['day.join() === "start,wake,eat,work,read,sleep"', '`day` has everything in order'], ['$src.includes("...")', 'Use spread `...`']],
    },
    {
      t: 'Array to String',
      b: `\`join(separator)\` glues all the items into one string (the opposite of \`split\`):

\`\`\`js
["a", "b", "c"].join("-")   // "a-b-c"
["a", "b", "c"].join("")    // "abc"
["a", "b", "c"].join(", ")  // "a, b, c"
\`\`\``,
      code: `const ingredients = ["flour", "sugar", "eggs"];\n// Create sentence: "You need flour, sugar, eggs."\n`,
      sol: 'const ingredients = ["flour", "sugar", "eggs"];\nconst sentence = `You need ${ingredients.join(", ")}.`;',
      tests: [['sentence === "You need flour, sugar, eggs."', '`sentence` is correct'], ['$src.includes("join")', 'Use `join`']],
    },
    {
      t: 'Reversing',
      b: `\`reverse()\` flips an array **in place** (it changes the original!). To keep the original, copy first:

\`\`\`js
const nums = [1, 2, 3];
const backwards = [...nums].reverse();  // nums is still [1, 2, 3]
\`\`\`

Combine with split and join to reverse a string: \`s.split("").reverse().join("")\`.`,
      code: `// Write reverseWord(word) using split, reverse and join\n`,
      sol: `function reverseWord(word) {\n  return word.split("").reverse().join("");\n}`,
      tests: [['reverseWord("drawer") === "reward"', '`reverseWord("drawer")` is `"reward"`'], ['$src.includes("reverse()")', 'Use `reverse()`']],
    },
    {
      t: 'Grids: Arrays of Arrays',
      b: `An array can contain arrays. That is how you model a grid, like a game board:

\`\`\`js
const board = [
  ["X", "O", "X"],
  [" ", "X", "O"],
  ["O", " ", "X"],
];
board[1][2]   // "O" — row 1, column 2
\`\`\``,
      code: `const board = [\n  ["X", "O", "X"],\n  ["O", "X", "O"],\n  ["O", "X", " "],\n];\n// Create centre (the middle cell) and\n// put "X" in the empty bottom-right cell\n`,
      sol: `const board = [\n  ["X", "O", "X"],\n  ["O", "X", "O"],\n  ["O", "X", " "],\n];\nconst centre = board[1][1];\nboard[2][2] = "X";`,
      tests: [['centre === "X"', '`centre` is `"X"`'], ['board[2][2] === "X"', 'The bottom-right cell is `"X"`']],
    },
    {
      t: 'Copies and References',
      b: `Assigning an array to another variable does **not** copy it — both names point at the **same** array:

\`\`\`js
const a = [1, 2];
const b = a;
b.push(3);
console.log(a);  // [1, 2, 3]  — a changed too!
\`\`\`

To make a real copy use \`[...a]\` or \`a.slice()\`. The code below accidentally changes \`original\`. Fix it.`,
      code: `const original = ["red", "green"];\nconst edited = original;\nedited.push("blue");\n`,
      sol: `const original = ["red", "green"];\nconst edited = [...original];\nedited.push("blue");`,
      tests: [['original.join() === "red,green"', '`original` stays red, green'], ['edited.join() === "red,green,blue"', '`edited` is red, green, blue']],
    },
    {
      t: 'Project: Grade Book',
      b: `Write \`report(scores)\` that takes an array of test scores and returns a **string** like:

\`\`\`
Average: 78.5, Best: 94, Passed: 3/4
\`\`\`

- average rounded to 1 decimal place (as a number, so \`80\` not \`80.0\`)
- a pass is **60 or more**

Use loops — no array methods like \`reduce\` yet.`,
      code: `function report(scores) {\n}\n`,
      sol: 'function report(scores) {\n  let total = 0;\n  let best = scores[0];\n  let passed = 0;\n  for (const s of scores) {\n    total += s;\n    if (s > best) best = s;\n    if (s >= 60) passed++;\n  }\n  const average = Math.round((total / scores.length) * 10) / 10;\n  return `Average: ${average}, Best: ${best}, Passed: ${passed}/${scores.length}`;\n}',
      tests: [['report([94, 70, 55, 95]) === "Average: 78.5, Best: 95, Passed: 3/4"', '`report([94, 70, 55, 95])` is correct'], ['report([80, 80]) === "Average: 80, Best: 80, Passed: 2/2"', '`report([80, 80])` is correct'], ['report([10, 20, 31]) === "Average: 20.3, Best: 31, Passed: 0/3"', '`report([10, 20, 31])` is correct']],
    },
  ],
});
