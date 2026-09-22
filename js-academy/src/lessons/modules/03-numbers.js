registerModule({
  id: 'numbers',
  title: 'Numbers & Math',
  icon: '🔢',
  lessons: [
    {
      t: 'The Remainder Operator',
      b: `\`%\` gives the **remainder** after dividing:

\`\`\`js
17 % 5   // 2   (17 = 3 × 5 + 2)
10 % 2   // 0   (divides exactly)
\`\`\`

It is incredibly handy: \`n % 2 === 0\` tells you whether \`n\` is even, and \`% 60\` turns total seconds into "seconds past the minute".`,
      code: `// Create leftover: the remainder of 29 divided by 4\n// Create isEven: the remainder of 36 divided by 2\n`,
      sol: `let leftover = 29 % 4;\nlet isEven = 36 % 2;`,
      tests: [['leftover === 1', '`leftover` should be `1`'], ['isEven === 0', '`isEven` should be `0`'], ['($src.match(/%/g) || []).length >= 2', 'Use `%` for both']],
    },
    {
      t: 'Powers',
      b: `\`**\` raises a number to a power:

\`\`\`js
2 ** 3    // 8   (2 × 2 × 2)
10 ** 2   // 100
9 ** 0.5  // 3   (a square root!)
\`\`\``,
      code: `// Create cube: 4 to the power of 3\n// Create kilobyte: 2 to the power of 10\n`,
      sol: `let cube = 4 ** 3;\nlet kilobyte = 2 ** 10;`,
      tests: [['cube === 64', '`cube` should be `64`'], ['kilobyte === 1024', '`kilobyte` should be `1024`'], ['$src.includes("**")', 'Use `**`']],
    },
    {
      t: 'Rounding',
      b: `The built-in \`Math\` object has rounding helpers:

\`\`\`js
Math.round(4.5)  // 5  — to the nearest whole number
Math.floor(4.9)  // 4  — always down
Math.ceil(4.1)   // 5  — always up
Math.trunc(-4.7) // -4 — just chop off the decimals
\`\`\``,
      code: `const n = 7.6;\n// Create down (n rounded down), up (n rounded up) and nearest (n rounded to nearest)\n`,
      sol: `const n = 7.6;\nconst down = Math.floor(n);\nconst up = Math.ceil(n);\nconst nearest = Math.round(n);`,
      tests: [['down === 7', '`down` is `7`'], ['up === 8', '`up` is `8`'], ['nearest === 8', '`nearest` is `8`'], ['$src.includes("Math.floor") && $src.includes("Math.ceil") && $src.includes("Math.round")', 'Use `Math.floor`, `Math.ceil` and `Math.round`']],
    },
    {
      t: 'Biggest and Smallest',
      b: `\`Math.max\` and \`Math.min\` accept any number of values:

\`\`\`js
Math.max(3, 9, 2)   // 9
Math.min(3, 9, 2)   // 2
\`\`\`

A neat trick: \`Math.max(0, x)\` makes sure a value never goes below zero.`,
      code: `const a = 42, b = 17, c = 99, d = 3;\n// Create highest and lowest of the four values\n`,
      sol: `const a = 42, b = 17, c = 99, d = 3;\nconst highest = Math.max(a, b, c, d);\nconst lowest = Math.min(a, b, c, d);`,
      tests: [['highest === 99', '`highest` is `99`'], ['lowest === 3', '`lowest` is `3`'], ['$src.includes("Math.max") && $src.includes("Math.min")', 'Use `Math.max` and `Math.min`']],
    },
    {
      t: 'Distance with Math.abs',
      b: `\`Math.abs\` gives the **absolute value** — the number without its sign:

\`\`\`js
Math.abs(-8)  // 8
Math.abs(8)   // 8
\`\`\`

It is perfect for "how far apart are these?", where you do not care which is bigger.`,
      code: `const myGuess = 35;\nconst secret = 52;\n// Create distance: how far apart the guess and the secret are (always positive)\n`,
      sol: `const myGuess = 35;\nconst secret = 52;\nconst distance = Math.abs(myGuess - secret);`,
      tests: [['distance === 17', '`distance` is `17`'], ['$src.includes("Math.abs")', 'Use `Math.abs`']],
    },
    {
      t: 'Circles: PI and Square Roots',
      b: `\`Math.PI\` is π (3.14159…), and \`Math.sqrt\` finds square roots.

\`\`\`js
const r = 2;
const area = Math.PI * r ** 2;  // 12.566…
Math.sqrt(49);                   // 7
\`\`\``,
      code: `const radius = 10;\n// Create circumference (2 × π × radius) and side (the square root of 144)\n`,
      sol: `const radius = 10;\nconst circumference = 2 * Math.PI * radius;\nconst side = Math.sqrt(144);`,
      tests: [['Math.abs(circumference - 62.83185307179586) < 1e-9', '`circumference` is about `62.83`'], ['side === 12 && $src.includes("Math.sqrt")', '`side` is `12`, using `Math.sqrt`']],
    },
    {
      t: 'Random Numbers',
      b: `\`Math.random()\` returns a random decimal from 0 up to (but not including) 1. To roll a dice from 1 to 6:

\`\`\`js
Math.random() * 6                    // 0 … 5.999
Math.floor(Math.random() * 6)        // 0 … 5
Math.floor(Math.random() * 6) + 1    // 1 … 6  ✓
\`\`\``,
      code: `// Create roll: a random whole number from 1 to 20 (like a 20-sided die)\n`,
      sol: `const roll = Math.floor(Math.random() * 20) + 1;`,
      tests: [['Number.isInteger(roll) && roll >= 1 && roll <= 20', '`roll` is a whole number from 1 to 20'], ['$src.includes("Math.random")', 'Use `Math.random()`'], ['$src.includes("20")', 'Scale it by 20']],
    },
    {
      t: 'Decimal Surprises',
      b: `Computers store decimals in binary, so some simple-looking sums come out slightly off:

\`\`\`js
0.1 + 0.2   // 0.30000000000000004
\`\`\`

That is not a JavaScript bug — nearly every language does this. When you display money or measurements, round them. \`toFixed(n)\` returns a **string** with exactly \`n\` decimals:

\`\`\`js
(0.1 + 0.2).toFixed(2)   // "0.30"
\`\`\``,
      code: `const sum = 0.1 + 0.2;\n// Create shown: sum formatted with exactly 2 decimals\n`,
      sol: `const sum = 0.1 + 0.2;\nconst shown = sum.toFixed(2);`,
      tests: [['shown === "0.30"', '`shown` should be the string `"0.30"`'], ['$src.includes("toFixed")', 'Use `toFixed`']],
    },
    {
      t: 'Formatting Prices',
      b: `Because \`toFixed\` returns a string, it is perfect for showing prices:

\`\`\`js
const price = 4.5;
console.log("$" + price.toFixed(2)); // $4.50
\`\`\``,
      code: `const coffee = 3.2;\nconst cake = 2.35;\n// Print the total as a price, like $5.55\n`,
      sol: 'const coffee = 3.2;\nconst cake = 2.35;\nconsole.log("$" + (coffee + cake).toFixed(2));',
      tests: [['$out() === "$5.55"', 'Print `$5.55`'], ['$src.includes("toFixed(2)")', 'Use `toFixed(2)`']],
    },
    {
      t: 'Reading Numbers from Text',
      b: `\`Number("42px")\` is \`NaN\`, because the whole string must be a number. \`parseInt\` and \`parseFloat\` read as much of a number as they can from the **start** of a string:

\`\`\`js
parseInt("42px")      // 42
parseFloat("3.5kg")   // 3.5
parseInt("3.99")      // 3 (whole numbers only)
\`\`\``,
      code: `const width = "250px";\nconst weight = "72.5kg";\n// Create w (250 as a number) and kg (72.5 as a number)\n`,
      sol: `const width = "250px";\nconst weight = "72.5kg";\nconst w = parseInt(width);\nconst kg = parseFloat(weight);`,
      tests: [['w === 250', '`w` is the number `250`'], ['kg === 72.5', '`kg` is the number `72.5`']],
    },
    {
      t: 'Whole Numbers',
      b: `\`Number.isInteger\` checks whether a number has no fractional part:

\`\`\`js
Number.isInteger(5)     // true
Number.isInteger(5.0)   // true — same number
Number.isInteger(5.2)   // false
\`\`\``,
      code: `const people = 7;\nconst pizzaSlices = 24;\n// Create perPerson: slices divided by people\n// Create fair: whether perPerson is a whole number\n`,
      sol: `const people = 7;\nconst pizzaSlices = 24;\nconst perPerson = pizzaSlices / people;\nconst fair = Number.isInteger(perPerson);`,
      tests: [['Math.abs(perPerson - 24 / 7) < 1e-12', '`perPerson` is 24 ÷ 7'], ['fair === false', '`fair` should be `false`'], ['$src.includes("Number.isInteger")', 'Use `Number.isInteger`']],
    },
    {
      t: 'Infinity',
      b: `Dividing a non-zero number by zero does not crash in JavaScript — you get \`Infinity\` (or \`-Infinity\`):

\`\`\`js
1 / 0     // Infinity
-1 / 0    // -Infinity
0 / 0     // NaN
\`\`\`

\`Number.isFinite(x)\` checks for a "normal" number (not \`Infinity\`, not \`NaN\`).`,
      code: `const speed = 100 / 0;\n// Create ok: whether speed is a finite number\n`,
      sol: `const speed = 100 / 0;\nconst ok = Number.isFinite(speed);`,
      tests: [['ok === false', '`ok` should be `false`'], ['$src.includes("Number.isFinite")', 'Use `Number.isFinite`']],
    },
    {
      t: 'Averages',
      b: `To find the average (mean), add the values up and divide by how many there are. Parentheses matter here:

\`\`\`js
(4 + 8) / 2   // 6 ✓
4 + 8 / 2     // 8 ✗ — only 8 was divided
\`\`\``,
      code: `const test1 = 72, test2 = 85, test3 = 91;\n// Create average of the three scores\n`,
      sol: `const test1 = 72, test2 = 85, test3 = 91;\nconst average = (test1 + test2 + test3) / 3;`,
      tests: [['Math.abs(average - 82.66666666666667) < 1e-9', '`average` should be about `82.67`']],
    },
    {
      t: 'Converting Temperatures',
      b: `Formulas translate directly into code. Celsius to Fahrenheit is:

> F = C × 9 / 5 + 32

\`\`\`js
const c = 100;
const f = c * 9 / 5 + 32;  // 212
\`\`\``,
      code: `const celsius = 25;\n// Create fahrenheit using the formula\n// Then create back: fahrenheit converted back to celsius, (F − 32) × 5 / 9\n`,
      sol: `const celsius = 25;\nconst fahrenheit = celsius * 9 / 5 + 32;\nconst back = (fahrenheit - 32) * 5 / 9;`,
      tests: [['fahrenheit === 77', '`fahrenheit` is `77`'], ['back === 25', '`back` is `25`'], ['$src.includes("celsius *") || $src.includes("celsius*")', 'Use the `celsius` variable']],
    },
    {
      t: 'Percentages',
      b: `"20% of x" is \`x * 0.2\` (or \`x * 20 / 100\`). A 20% discount leaves 80%:

\`\`\`js
const price = 50;
const discount = price * 0.2;   // 10
const salePrice = price - discount; // 40
\`\`\``,
      code: `const jacket = 80;\nconst percentOff = 25;\n// Create saving (how much you save) and salePrice\n`,
      sol: `const jacket = 80;\nconst percentOff = 25;\nconst saving = jacket * percentOff / 100;\nconst salePrice = jacket - saving;`,
      tests: [['saving === 20', '`saving` is `20`'], ['salePrice === 60', '`salePrice` is `60`'], ['$src.includes("percentOff")', 'Use `percentOff` in the calculation']],
    },
    {
      t: 'Rounding to Decimal Places',
      b: `\`toFixed\` gives a string. To get a rounded **number**, multiply, round, then divide:

\`\`\`js
const x = 3.14159;
Math.round(x * 100) / 100   // 3.14
Math.round(x * 10) / 10     // 3.1
\`\`\``,
      code: `const g = 9.80665;\n// Create g1 (rounded to 1 decimal) and g3 (rounded to 3 decimals) as numbers\n`,
      sol: `const g = 9.80665;\nconst g1 = Math.round(g * 10) / 10;\nconst g3 = Math.round(g * 1000) / 1000;`,
      tests: [['g1 === 9.8', '`g1` is `9.8`'], ['g3 === 9.807', '`g3` is `9.807`'], ['typeof g1 === "number" && typeof g3 === "number"', 'Both are numbers, not strings']],
    },
    {
      t: 'Big Numbers',
      b: `Long numbers are hard to read. You can put underscores in them — JavaScript ignores them:

\`\`\`js
const population = 8_000_000_000;
\`\`\`

Numbers are exact only up to \`Number.MAX_SAFE_INTEGER\` (9,007,199,254,740,991). Beyond that, whole numbers start to lose precision.`,
      code: `// Create billion = one billion, written with underscores\n// Create isSafe: whether billion * billion is a safe integer (use Number.isSafeInteger)\n`,
      sol: `const billion = 1_000_000_000;\nconst isSafe = Number.isSafeInteger(billion * billion);`,
      tests: [['billion === 1000000000 && $src.includes("_000")', '`billion` written as `1_000_000_000`'], ['isSafe === false && $src.includes("Number.isSafeInteger")', '`isSafe` should be `false`, using `Number.isSafeInteger`']],
    },
    {
      t: 'BigInt',
      b: `For whole numbers of any size, use **BigInt** by adding \`n\` after the number:

\`\`\`js
const huge = 2n ** 100n;
console.log(huge); // 1267650600228229401496703205376n
\`\`\`

BigInts cannot be mixed with normal numbers in maths — \`1n + 1\` is a TypeError. Convert with \`BigInt(1)\`.`,
      code: `// Create grains: 2 to the power of 64, as a BigInt\n`,
      sol: `const grains = 2n ** 64n;`,
      tests: [['typeof grains === "bigint"', '`grains` is a BigInt'], ['grains === 18446744073709551616n', '`grains` is 2⁶⁴']],
    },
    {
      t: 'Hours and Minutes',
      b: `Combining \`Math.floor\` and \`%\` splits a quantity into units:

\`\`\`js
const totalSeconds = 135;
const minutes = Math.floor(totalSeconds / 60); // 2
const seconds = totalSeconds % 60;             // 15
\`\`\``,
      code: `const movieLength = 142; // minutes\n// Create hours and mins, then print "2h 22m" using them\n`,
      sol: 'const movieLength = 142;\nconst hours = Math.floor(movieLength / 60);\nconst mins = movieLength % 60;\nconsole.log(`${hours}h ${mins}m`);',
      tests: [['hours === 2 && mins === 22', '`hours` is `2` and `mins` is `22`'], ['$out() === "2h 22m"', 'Print `2h 22m`']],
    },
    {
      t: 'Project: Tip Calculator',
      b: `A group dinner costs **\`bill\`**. They tip **\`tipPercent\`** percent, and split the total between **\`people\`**.

Create \`tip\`, \`total\` and \`perPerson\` (rounded to 2 decimal places as a **number**), then print:

\`\`\`
Each person pays $26.45
\`\`\`

Make sure your code still works if someone changes \`bill\`, \`tipPercent\` or \`people\`.`,
      code: `const bill = 92;\nconst tipPercent = 15;\nconst people = 4;\n`,
      sol: 'const bill = 92;\nconst tipPercent = 15;\nconst people = 4;\nconst tip = bill * tipPercent / 100;\nconst total = bill + tip;\nconst perPerson = Math.round(total / people * 100) / 100;\nconsole.log(`Each person pays $${perPerson.toFixed(2)}`);',
      tests: [['Math.abs(tip - 13.8) < 1e-9', '`tip` is `13.8`'], ['Math.abs(total - 105.8) < 1e-9', '`total` is `105.8`'], ['perPerson === 26.45', '`perPerson` is `26.45`'], ['$out() === "Each person pays $26.45"', 'Print `Each person pays $26.45`']],
    },
  ],
});
