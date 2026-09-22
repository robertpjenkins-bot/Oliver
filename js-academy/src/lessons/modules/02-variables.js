registerModule({
  id: 'vars',
  title: 'Variables & Types',
  icon: '📦',
  lessons: [
    {
      t: 'Creating a Variable with let',
      b: `A **variable** stores a value under a name. Create one with \`let\`:

\`\`\`js
let score = 10;
console.log(score); // 10
\`\`\`

Read \`=\` as "gets": *score gets 10*. It is **assignment**, not "equals" like in maths.`,
      code: `// Create a variable called lives holding the number 3, then print it\n`,
      sol: `let lives = 3;\nconsole.log(lives);`,
      tests: [['lives === 3', 'Create `lives` holding `3`'], ['$out() === "3"', 'Print it']],
    },
    {
      t: 'Changing a Variable',
      b: `A \`let\` variable can be given a new value later. Do not write \`let\` again — that would try to create a second variable with the same name:

\`\`\`js
let mood = "sleepy";
mood = "awake";       // no let here!
console.log(mood);    // awake
\`\`\``,
      code: `let weather = "rainy";\n// Change weather to "sunny" on the next line\n\nconsole.log(weather);\n`,
      sol: `let weather = "rainy";\nweather = "sunny";\nconsole.log(weather);`,
      tests: [['weather === "sunny"', '`weather` should end up as `"sunny"`'], ['$out() === "sunny"', 'Print `sunny`'], ['($src.match(/\\blet\\b/g) || []).length === 1', 'Only use `let` once']],
    },
    {
      t: 'Constants with const',
      b: `Use \`const\` for values that should **never** be reassigned:

\`\`\`js
const daysInWeek = 7;
daysInWeek = 8; // TypeError: Assignment to constant variable.
\`\`\`

A good habit: use \`const\` by default, and switch to \`let\` only when you know the value will change.

The code below crashes. Because \`lives\` has to change, it should not be a constant.`,
      code: `const lives = 3;\nlives = 2;\nconsole.log(lives);\n`,
      sol: `let lives = 3;\nlives = 2;\nconsole.log(lives);`,
      tests: [['$out() === "2"', 'The program should print `2`'], ['/let\\s+lives/.test($src)', 'Declare `lives` with `let`']],
    },
    {
      t: 'Naming Variables',
      b: `Variable names can contain letters, digits, \`_\` and \`$\`, but **cannot start with a digit** and cannot contain spaces or dashes.

JavaScript programmers write multi-word names in **camelCase**: the first word lowercase, each following word capitalised.

\`\`\`js
let firstName = "Ana";   // ✓ camelCase
let highScore2 = 900;    // ✓
let 2ndPlace = "Bo";     // ✗ starts with a digit
let user-name = "x";     // ✗ dash
\`\`\``,
      code: `// Create a variable holding "gold" named with camelCase from the words: first place\n`,
      sol: `let firstPlace = "gold";`,
      tests: [['firstPlace === "gold"', 'Create `firstPlace` holding `"gold"`']],
    },
    {
      t: 'Math with Variables',
      b: `Variables can hold the result of a calculation, and be used in calculations:

\`\`\`js
let width = 5;
let height = 4;
let area = width * height;
console.log(area); // 20
\`\`\``,
      code: `let price = 4;\nlet quantity = 3;\n// Create total = price times quantity, and print it\n`,
      sol: `let price = 4;\nlet quantity = 3;\nlet total = price * quantity;\nconsole.log(total);`,
      tests: [['total === 12', '`total` should be `12`'], ['/total\\s*=\\s*price\\s*\\*\\s*quantity|total\\s*=\\s*quantity\\s*\\*\\s*price/.test($src)', 'Calculate it from `price` and `quantity`'], ['$out() === "12"', 'Print `total`']],
    },
    {
      t: 'Shorthand Assignment',
      b: `Updating a variable using its own value is so common that there are shortcuts:

\`\`\`js
let coins = 10;
coins += 5;  // same as coins = coins + 5  → 15
coins -= 3;  // → 12
coins *= 2;  // → 24
coins /= 4;  // → 6
\`\`\``,
      code: `let points = 100;\n// Using shorthand operators: add 50, then subtract 30, then double it\n\nconsole.log(points);\n`,
      sol: `let points = 100;\npoints += 50;\npoints -= 30;\npoints *= 2;\nconsole.log(points);`,
      tests: [['points === 240', '`points` should end at `240`'], ['$src.includes("+=") && $src.includes("-=") && $src.includes("*=")', 'Use `+=`, `-=` and `*=`']],
    },
    {
      t: 'Counting Up and Down',
      b: `Adding or subtracting exactly 1 has its own operators: \`++\` (increment) and \`--\` (decrement).

\`\`\`js
let steps = 0;
steps++;   // 1
steps++;   // 2
steps--;   // 1
\`\`\``,
      code: `let level = 1;\n// Increase level three times with ++, then decrease it once with --\n\nconsole.log(level);\n`,
      sol: `let level = 1;\nlevel++;\nlevel++;\nlevel++;\nlevel--;\nconsole.log(level);`,
      tests: [['level === 3', '`level` should end at `3`'], ['($src.match(/\\+\\+/g) || []).length === 3 && $src.includes("--")', 'Use `++` three times and `--` once']],
    },
    {
      t: 'undefined',
      b: `A variable declared without a value holds the special value \`undefined\` — "nothing has been put here yet":

\`\`\`js
let answer;
console.log(answer);        // undefined
console.log(typeof answer); // "undefined"
answer = 42;                // now it has a value
\`\`\``,
      code: `// Declare a variable called prize WITHOUT giving it a value, then print it\n`,
      sol: `let prize;\nconsole.log(prize);`,
      tests: [['typeof prize === "undefined"', 'Declare `prize` without a value'], ['$out() === "undefined"', 'Print it']],
    },
    {
      t: 'null',
      b: `\`null\` means "intentionally empty". While \`undefined\` usually means *not set yet*, \`null\` is a value **you** choose to say "there is nothing here":

\`\`\`js
let selectedItem = null;   // nothing selected on purpose
\`\`\`

Fun fact: \`typeof null\` is \`"object"\` — a famous bug from 1995 that can never be fixed without breaking the web.`,
      code: `let winner = "Sam";\n// The race was cancelled — set winner to null and print it\n`,
      sol: `let winner = "Sam";\nwinner = null;\nconsole.log(winner);`,
      tests: [['winner === null', '`winner` should be `null`'], ['$out() === "null"', 'Print it']],
    },
    {
      t: 'Building a Full Name',
      b: `Combine string variables with \`+\`. Remember to add spaces yourself:

\`\`\`js
let greeting = "Good";
let time = "morning";
let message = greeting + " " + time;  // "Good morning"
\`\`\``,
      code: `let firstName = "Grace";\nlet lastName = "Hopper";\n// Create fullName: first name, a space, last name\n`,
      sol: `let firstName = "Grace";\nlet lastName = "Hopper";\nlet fullName = firstName + " " + lastName;`,
      tests: [['fullName === "Grace Hopper"', '`fullName` should be `"Grace Hopper"`'], ['$src.includes("firstName +") || $src.includes("${firstName}")', 'Build it from the two variables']],
    },
    {
      t: 'Template Literals',
      b: `Joining with \`+\` gets messy. **Template literals** use backticks and let you drop values straight into the text with \`\${…}\`:

\`\`\`js
let pet = "cat";
let age = 3;
console.log(\`My \${pet} is \${age} years old\`);
// My cat is 3 years old
\`\`\`

You can put any expression inside \`\${}\`, like \`\${age * 7}\`.`,
      code: `let item = "pizza";\nlet slices = 8;\n// Print "I ate 8 slices of pizza" using a template literal\n`,
      sol: 'let item = "pizza";\nlet slices = 8;\nconsole.log(`I ate ${slices} slices of ${item}`);',
      tests: [['$out() === "I ate 8 slices of pizza"', 'Print `I ate 8 slices of pizza`'], ['$src.includes("`") && $src.includes("${")', 'Use a template literal with `${}`']],
    },
    {
      t: 'Swapping Two Variables',
      b: `A classic puzzle: swap the values of two variables. If you write \`a = b\` first, the old value of \`a\` is lost! The trick is a temporary variable:

\`\`\`js
let temp = a;
a = b;
b = temp;
\`\`\``,
      code: `let left = "apple";\nlet right = "banana";\n// Swap them so left is "banana" and right is "apple"\n`,
      sol: `let left = "apple";\nlet right = "banana";\nlet temp = left;\nleft = right;\nright = temp;`,
      tests: [['left === "banana" && right === "apple"', 'Swap the values'], ['!/left\\s*=\\s*["\']banana/.test($src)', 'Swap the variables — do not just type the strings again']],
    },
    {
      t: 'The Old var Keyword',
      b: `Older code uses \`var\` to create variables. It works, but has confusing rules (it ignores blocks, and can be used before it is declared). Modern JavaScript uses \`let\` and \`const\` instead.

When you meet \`var\` in old code, you can almost always replace it with \`let\` or \`const\`.`,
      code: `var planet = "Mars";\nvar moons = 2;\nmoons = moons + 0;\nconsole.log(planet, moons);\n`,
      sol: `const planet = "Mars";\nlet moons = 2;\nmoons = moons + 0;\nconsole.log(planet, moons);`,
      tests: [['!/\\bvar\\b/.test($src)', 'Replace every `var` with `let` or `const`'], ['$out() === "Mars 2"', 'Still print `Mars 2`']],
    },
    {
      t: 'Values Have Types',
      b: `Every value has a type, and so the variable holding it does too (at that moment). The **primitive** types you will use the most:

- \`number\` — \`42\`, \`-3.5\`
- \`string\` — \`"hello"\`
- \`boolean\` — \`true\` or \`false\`
- \`undefined\` and \`null\`

JavaScript is **dynamically typed**: the same variable can hold a number now and a string later.`,
      code: `// Create:\n//   count   — a number\n//   title   — a string\n//   isReady — a boolean\n`,
      sol: `let count = 5;\nlet title = "Chapter 1";\nlet isReady = true;`,
      tests: [['typeof count === "number"', '`count` is a number'], ['typeof title === "string"', '`title` is a string'], ['typeof isReady === "boolean"', '`isReady` is a boolean']],
    },
    {
      t: 'Strings to Numbers',
      b: `Text typed by users arrives as strings, even when it looks like a number. \`"7" + "5"\` is \`"75"\`!

Convert with \`Number()\`:

\`\`\`js
let input = "42";
let n = Number(input);  // 42 (a real number)
console.log(n + 1);     // 43
\`\`\``,
      code: `let a = "7";\nlet b = "5";\n// Create sum: the NUMBER 12, calculated from a and b\n`,
      sol: `let a = "7";\nlet b = "5";\nlet sum = Number(a) + Number(b);`,
      tests: [['sum === 12', '`sum` should be the number `12`'], ['$src.includes("Number(")', 'Convert with `Number()`']],
    },
    {
      t: 'Numbers to Strings',
      b: `Going the other way, \`String(value)\` turns anything into text. Numbers also have a \`.toString()\` method:

\`\`\`js
let n = 99;
String(n);       // "99"
n.toString();    // "99"
\`\`\`

This is useful for things like counting digits.`,
      code: `let bigNumber = 123456;\n// Create label: bigNumber converted to a string\n// Then create digits: how many characters label has (use label.length)\n`,
      sol: `let bigNumber = 123456;\nlet label = String(bigNumber);\nlet digits = label.length;`,
      tests: [['label === "123456"', '`label` should be the string `"123456"`'], ['digits === 6', '`digits` should be `6`']],
    },
    {
      t: 'Not a Number',
      b: `If \`Number()\` cannot make sense of a string, it gives \`NaN\` — "Not a Number". Strangely, \`NaN\` is of type \`"number"\`, and it is not even equal to itself!

To test for it, use \`Number.isNaN()\`:

\`\`\`js
let x = Number("banana");   // NaN
Number.isNaN(x);            // true
\`\`\``,
      code: `// Create result = Number("hello")\n// Create isBad = whether result is NaN (use Number.isNaN)\n`,
      sol: `let result = Number("hello");\nlet isBad = Number.isNaN(result);`,
      tests: [['Number.isNaN(result)', '`result` should be `NaN`'], ['isBad === true', '`isBad` should be `true`'], ['$src.includes("Number.isNaN")', 'Use `Number.isNaN`']],
    },
    {
      t: 'Naming Constants',
      b: `Values that are fixed settings for your whole program are often named in **UPPER_SNAKE_CASE** so they stand out:

\`\`\`js
const MAX_PLAYERS = 4;
const GRAVITY = 9.81;
\`\`\`

This is only a convention — JavaScript does not care — but other programmers will thank you.`,
      code: `// Create a constant for the maximum speed, 120, named in UPPER_SNAKE_CASE (words: max speed)\n// Then create a constant halfSpeed that is half of it\n`,
      sol: `const MAX_SPEED = 120;\nconst halfSpeed = MAX_SPEED / 2;`,
      tests: [['MAX_SPEED === 120', 'Create `MAX_SPEED` = `120`'], ['/const\\s+MAX_SPEED/.test($src)', 'Use `const`'], ['halfSpeed === 60 && $src.includes("MAX_SPEED /")', '`halfSpeed` should be calculated from `MAX_SPEED`']],
    },
    {
      t: 'Blocks and Visibility',
      b: `Curly braces \`{ }\` make a **block**. A \`let\` or \`const\` declared inside a block only exists inside it:

\`\`\`js
{
  let secret = "hidden";
}
console.log(secret); // ReferenceError: secret is not defined
\`\`\`

To use a value after the block, declare it **before** the block.`,
      code: `{\n  let message = "Made it out!";\n}\nconsole.log(message);\n`,
      sol: `let message;\n{\n  message = "Made it out!";\n}\nconsole.log(message);`,
      tests: [['$out() === "Made it out!"', 'Print `Made it out!`'], ['$src.includes("{")', 'Keep the block']],
      hint: 'Declare `let message;` above the block, and inside it just assign `message = …`.',
    },
    {
      t: 'Project: Shopping Receipt',
      b: `Put it all together. You bought **3 notebooks at 2.5 each** and **2 pens at 1.25 each**.

Create variables \`notebooks\` and \`pens\` for the two costs, \`subtotal\` for their sum, \`tax\` as 10% of the subtotal, and \`total\`. Then print:

\`\`\`
Total: 11
\`\`\`

(using a template literal).`,
      code: `// Your receipt code\n`,
      sol: 'const notebooks = 3 * 2.5;\nconst pens = 2 * 1.25;\nconst subtotal = notebooks + pens;\nconst tax = subtotal * 0.1;\nconst total = subtotal + tax;\nconsole.log(`Total: ${total}`);',
      tests: [['notebooks === 7.5 && pens === 2.5', '`notebooks` is `7.5`, `pens` is `2.5`'], ['subtotal === 10', '`subtotal` is `10`'], ['Math.abs(tax - 1) < 1e-9', '`tax` is `1`'], ['Math.abs(total - 11) < 1e-9', '`total` is `11`'], ['$out() === "Total: 11"', 'Print `Total: 11`']],
    },
  ],
});
