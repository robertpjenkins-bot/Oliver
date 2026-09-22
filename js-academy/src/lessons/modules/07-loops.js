registerModule({
  id: 'loops',
  title: 'Loops',
  icon: '🔁',
  lessons: [
    {
      t: 'The while Loop',
      b: `A **loop** repeats code. A \`while\` loop keeps going **as long as** its condition is true:

\`\`\`js
let n = 1;
while (n <= 3) {
  console.log(n);
  n++;
}
// 1, 2, 3
\`\`\`

The \`n++\` is essential: without it, \`n\` stays 1 forever and the loop never ends.`,
      code: `// Print the numbers 1 to 5 using a while loop\n`,
      sol: `let n = 1;\nwhile (n <= 5) {\n  console.log(n);\n  n++;\n}`,
      tests: [['$out() === "1\\n2\\n3\\n4\\n5"', 'Print 1 to 5, one per line'], ['$src.includes("while")', 'Use a `while` loop']],
    },
    {
      t: 'The for Loop',
      b: `A \`for\` loop packs the three parts of counting into one line:

\`\`\`js
for (let i = 1; i <= 3; i++) {
  console.log(i);
}
\`\`\`

1. **start**: \`let i = 1\` — runs once
2. **condition**: \`i <= 3\` — checked before each round
3. **update**: \`i++\` — runs after each round

\`i\` is the traditional name for a loop counter.`,
      code: `// Print the numbers 1 to 10 using a for loop\n`,
      sol: `for (let i = 1; i <= 10; i++) {\n  console.log(i);\n}`,
      tests: [['$logs.length === 10 && $logs[0] === "1" && $logs[9] === "10"', 'Print 1 to 10'], ['/for\\s*\\(/.test($src)', 'Use a `for` loop']],
    },
    {
      t: 'Counting Down',
      b: `Loops can count down just as easily — start high, loop while above the end, and use \`i--\`:

\`\`\`js
for (let i = 3; i > 0; i--) {
  console.log(i);
}
\`\`\``,
      code: `// Print 10 down to 1, then "Liftoff!"\n`,
      sol: `for (let i = 10; i >= 1; i--) {\n  console.log(i);\n}\nconsole.log("Liftoff!");`,
      tests: [['$out() === "10\\n9\\n8\\n7\\n6\\n5\\n4\\n3\\n2\\n1\\nLiftoff!"', 'Count down from 10 and print `Liftoff!`'], ['$src.includes("--")', 'Count down with `--`']],
    },
    {
      t: 'Stepping by More Than One',
      b: `The update part can change the counter by any amount:

\`\`\`js
for (let i = 0; i <= 20; i += 5) {
  console.log(i);  // 0, 5, 10, 15, 20
}
\`\`\``,
      code: `// Print the even numbers from 2 to 20 by stepping 2 at a time\n`,
      sol: `for (let i = 2; i <= 20; i += 2) {\n  console.log(i);\n}`,
      tests: [['$out() === "2\\n4\\n6\\n8\\n10\\n12\\n14\\n16\\n18\\n20"', 'Print 2, 4, … 20'], ['$src.includes("+= 2") || $src.includes("+=2")', 'Step with `i += 2`']],
    },
    {
      t: 'Adding Things Up',
      b: `A loop plus an **accumulator** variable can total anything:

\`\`\`js
let total = 0;
for (let i = 1; i <= 4; i++) {
  total += i;
}
// total is 10
\`\`\``,
      code: `// Create sum: 1 + 2 + 3 + … + 100 using a loop\n`,
      sol: `let sum = 0;\nfor (let i = 1; i <= 100; i++) {\n  sum += i;\n}`,
      tests: [['sum === 5050', '`sum` is `5050`'], ['/for|while/.test($src)', 'Use a loop']],
    },
    {
      t: 'Looping Over a String',
      b: `Use the index and \`.length\` to visit every character:

\`\`\`js
const word = "hey";
for (let i = 0; i < word.length; i++) {
  console.log(word[i]);
}
\`\`\`

Note \`i < word.length\` (not \`<=\`), because the last index is \`length - 1\`.`,
      code: `const word = "loop";\n// Print each character of word on its own line, using its index\n`,
      sol: `const word = "loop";\nfor (let i = 0; i < word.length; i++) {\n  console.log(word[i]);\n}`,
      tests: [['$out() === "l\\no\\no\\np"', 'Print l, o, o, p'], ['$src.includes("word.length")', 'Loop up to `word.length`']],
    },
    {
      t: 'Counting Vowels',
      b: `Combine a loop with \`if\` to count things. \`"aeiou".includes(ch)\` is a neat way to check for a vowel.`,
      code: `const text = "programming is fun";\nlet vowels = 0;\n// Count the vowels in text\n`,
      sol: `const text = "programming is fun";\nlet vowels = 0;\nfor (let i = 0; i < text.length; i++) {\n  if ("aeiou".includes(text[i])) {\n    vowels++;\n  }\n}`,
      tests: [['vowels === 5', '`vowels` is `5`']],
    },
    {
      t: 'for…of',
      b: `\`for…of\` gives you each item directly, without managing an index:

\`\`\`js
for (const letter of "abc") {
  console.log(letter);
}
\`\`\`

It works on strings, arrays and many other collections.`,
      code: `const secret = "HELLO";\nlet spaced = "";\n// Using for...of, build spaced = "H E L L O " (each letter followed by a space)\n`,
      sol: `const secret = "HELLO";\nlet spaced = "";\nfor (const ch of secret) {\n  spaced += ch + " ";\n}`,
      tests: [['spaced === "H E L L O "', '`spaced` is `"H E L L O "`'], ['$src.includes(" of ")', 'Use `for…of`']],
    },
    {
      t: 'Stopping Early with break',
      b: `\`break\` exits a loop immediately:

\`\`\`js
for (let i = 1; i <= 100; i++) {
  if (i * i > 50) {
    console.log(i);  // 8
    break;
  }
}
\`\`\``,
      code: `// Find the first number from 1 upwards that is divisible by both 7 and 12.\n// Store it in found and stop the loop with break.\nlet found;\n`,
      sol: `let found;\nfor (let i = 1; i <= 1000; i++) {\n  if (i % 7 === 0 && i % 12 === 0) {\n    found = i;\n    break;\n  }\n}`,
      tests: [['found === 84', '`found` is `84`'], ['$src.includes("break")', 'Use `break`']],
    },
    {
      t: 'Skipping with continue',
      b: `\`continue\` skips the rest of the current round and jumps to the next one:

\`\`\`js
for (let i = 1; i <= 5; i++) {
  if (i === 3) continue;
  console.log(i);   // 1, 2, 4, 5
}
\`\`\``,
      code: `// Print the numbers 1 to 10, skipping multiples of 3, using continue\n`,
      sol: `for (let i = 1; i <= 10; i++) {\n  if (i % 3 === 0) continue;\n  console.log(i);\n}`,
      tests: [['$out() === "1\\n2\\n4\\n5\\n7\\n8\\n10"', 'Print 1 2 4 5 7 8 10'], ['$src.includes("continue")', 'Use `continue`']],
    },
    {
      t: 'do…while',
      b: `A \`do…while\` loop runs its body **first**, then checks the condition. So it always runs at least once:

\`\`\`js
let tries = 0;
do {
  tries++;
} while (tries < 3);
\`\`\``,
      code: `let value = 1;\n// Using do...while, keep doubling value while it is less than 1000\n`,
      sol: `let value = 1;\ndo {\n  value *= 2;\n} while (value < 1000);`,
      tests: [['value === 1024', '`value` ends at `1024`'], ['$src.includes("do") && $src.includes("while")', 'Use `do…while`']],
    },
    {
      t: 'Loops Inside Loops',
      b: `A loop inside a loop runs its inner loop **completely** for every round of the outer one — great for grids and tables:

\`\`\`js
for (let row = 1; row <= 2; row++) {
  for (let col = 1; col <= 3; col++) {
    console.log(row, col);
  }
}
// 6 lines
\`\`\``,
      code: `// Print the 3 times table rows for 1 to 3 like:\n// 1 x 1 = 1\n// 1 x 2 = 2\n// 1 x 3 = 3\n// 2 x 1 = 2 ... up to 3 x 3 = 9\n`,
      sol: 'for (let a = 1; a <= 3; a++) {\n  for (let b = 1; b <= 3; b++) {\n    console.log(`${a} x ${b} = ${a * b}`);\n  }\n}',
      tests: [['$logs.length === 9', 'Print 9 lines'], ['$logs[0] === "1 x 1 = 1" && $logs[5] === "2 x 3 = 6" && $logs[8] === "3 x 3 = 9"', 'Use the format `a x b = product`'], ['($src.match(/for\\s*\\(/g) || []).length >= 2', 'Use two nested loops']],
    },
    {
      t: 'Reversing a String',
      b: `Build a new string by walking **backwards** through the original:

\`\`\`js
for (let i = s.length - 1; i >= 0; i--) { … }
\`\`\``,
      code: `const word = "stressed";\nlet reversed = "";\n// Fill reversed with word backwards, using a loop\n`,
      sol: `const word = "stressed";\nlet reversed = "";\nfor (let i = word.length - 1; i >= 0; i--) {\n  reversed += word[i];\n}`,
      tests: [['reversed === "desserts"', '`reversed` is `"desserts"`']],
    },
    {
      t: 'Avoiding Infinite Loops',
      b: `If a loop's condition never becomes false, it runs **forever** and the program freezes. (Here, the runner stops it after a few seconds.)

Common causes: forgetting to update the counter, or updating it in the wrong direction. This loop is meant to print 5, 4, 3, 2, 1 — find the bug.`,
      code: `for (let i = 5; i > 0; i++) {\n  console.log(i);\n}\n`,
      sol: `for (let i = 5; i > 0; i--) {\n  console.log(i);\n}`,
      tests: [['$out() === "5\\n4\\n3\\n2\\n1"', 'Print 5 down to 1']],
    },
    {
      t: 'FizzBuzz',
      b: `The full classic! For each number from 1 to 15, print:

- \`FizzBuzz\` if divisible by 3 and 5
- \`Fizz\` if divisible by 3
- \`Buzz\` if divisible by 5
- the number otherwise`,
      code: `// FizzBuzz from 1 to 15\n`,
      sol: `for (let i = 1; i <= 15; i++) {\n  if (i % 15 === 0) console.log("FizzBuzz");\n  else if (i % 3 === 0) console.log("Fizz");\n  else if (i % 5 === 0) console.log("Buzz");\n  else console.log(i);\n}`,
      tests: [['$out() === "1\\n2\\nFizz\\n4\\nBuzz\\nFizz\\n7\\n8\\nFizz\\nBuzz\\n11\\nFizz\\n13\\n14\\nFizzBuzz"', 'Print FizzBuzz for 1 to 15']],
    },
    {
      t: 'Factorials',
      b: `The **factorial** of n (written n!) is 1 × 2 × 3 × … × n. For example 5! = 120.

Multiply into an accumulator that starts at **1** (starting at 0 would make everything 0).`,
      code: `const n = 10;\nlet factorial = 1;\n// Compute n! with a loop\n`,
      sol: `const n = 10;\nlet factorial = 1;\nfor (let i = 2; i <= n; i++) {\n  factorial *= i;\n}`,
      tests: [['factorial === 3628800', '`factorial` is `3628800`']],
    },
    {
      t: 'Loops Without a Known End',
      b: `\`while\` is ideal when you do not know in advance how many rounds you need. How many times can you halve 1000 before it drops below 1?`,
      code: `let amount = 1000;\nlet halvings = 0;\n// Keep halving amount (and counting) while amount is at least 1\n`,
      sol: `let amount = 1000;\nlet halvings = 0;\nwhile (amount >= 1) {\n  amount /= 2;\n  halvings++;\n}`,
      tests: [['halvings === 10', '`halvings` is `10`'], ['$src.includes("while")', 'Use `while`']],
    },
    {
      t: 'Drawing a Triangle',
      b: `Loops plus \`repeat\` make console pictures. Row \`i\` of a triangle has \`i\` stars:

\`\`\`
*
**
***
\`\`\``,
      code: `const height = 5;\n// Print a triangle of * with height rows\n`,
      sol: `const height = 5;\nfor (let i = 1; i <= height; i++) {\n  console.log("*".repeat(i));\n}`,
      tests: [['$out() === "*\\n**\\n***\\n****\\n*****"', 'Print a 5-row triangle']],
    },
    {
      t: 'The Collatz Puzzle',
      b: `Start with any positive number. If it is even, halve it; if it is odd, multiply by 3 and add 1. Repeat until you reach 1. Mathematicians believe this **always** reaches 1 — but nobody has proven it!

Count how many steps it takes starting from 27 (it is surprisingly many).`,
      code: `let n = 27;\nlet steps = 0;\n`,
      sol: `let n = 27;\nlet steps = 0;\nwhile (n !== 1) {\n  n = n % 2 === 0 ? n / 2 : 3 * n + 1;\n  steps++;\n}`,
      tests: [['steps === 111', '`steps` is `111`'], ['n === 1', '`n` ends at 1']],
    },
    {
      t: 'Project: Prime Finder',
      b: `A **prime** number is greater than 1 and only divisible by 1 and itself: 2, 3, 5, 7, 11…

Print every prime from 2 to 50, one per line. For each candidate, loop through possible divisors — if any divides it exactly, it is not prime.

Also count them in a variable \`primeCount\`.`,
      code: `let primeCount = 0;\n`,
      sol: `let primeCount = 0;\nfor (let n = 2; n <= 50; n++) {\n  let isPrime = true;\n  for (let d = 2; d * d <= n; d++) {\n    if (n % d === 0) {\n      isPrime = false;\n      break;\n    }\n  }\n  if (isPrime) {\n    console.log(n);\n    primeCount++;\n  }\n}`,
      tests: [['$out() === "2\\n3\\n5\\n7\\n11\\n13\\n17\\n19\\n23\\n29\\n31\\n37\\n41\\n43\\n47"', 'Print the primes up to 50'], ['primeCount === 15', '`primeCount` is `15`']],
      hint: 'You only need to test divisors up to the square root: `d * d <= n`.',
    },
  ],
});
