registerModule({
  id: 'func',
  title: 'Functions',
  icon: '🧩',
  lessons: [
    {
      t: 'Your First Function',
      b: `A **function** is a named, reusable block of code. You **declare** it once, then **call** it as often as you like:

\`\`\`js
function cheer() {
  console.log("Hip hip hooray!");
}

cheer();   // Hip hip hooray!
cheer();   // Hip hip hooray!
\`\`\`

Declaring does not run the code — calling with \`()\` does.`,
      code: `// Declare a function called sayHi that prints "Hi!"\n// Then call it twice\n`,
      sol: `function sayHi() {\n  console.log("Hi!");\n}\nsayHi();\nsayHi();`,
      tests: [['typeof sayHi === "function"', 'Declare `sayHi`'], ['$out() === "Hi!\\nHi!"', 'Call it twice']],
    },
    {
      t: 'Parameters',
      b: `**Parameters** let a function accept input. They act like variables inside the function:

\`\`\`js
function greet(name) {
  console.log("Hello, " + name + "!");
}
greet("Lin");   // Hello, Lin!
greet("Omar");  // Hello, Omar!
\`\`\`

The values you pass in (\`"Lin"\`) are called **arguments**.`,
      code: `// Write a function welcome(city) that prints "Welcome to <city>!"\n// Call it with "Rome"\n`,
      sol: `function welcome(city) {\n  console.log("Welcome to " + city + "!");\n}\nwelcome("Rome");`,
      tests: [['$out() === "Welcome to Rome!"', 'Print `Welcome to Rome!`'], ['typeof welcome === "function" && welcome.length === 1', '`welcome` takes one parameter']],
    },
    {
      t: 'Returning a Value',
      b: `\`return\` sends a value **back** to whoever called the function:

\`\`\`js
function double(n) {
  return n * 2;
}
const result = double(21);  // 42
\`\`\`

A function call becomes the value it returns, so you can use it anywhere a value fits: \`double(3) + 1\` is 7.`,
      code: `// Write square(n) that returns n times n\n`,
      sol: `function square(n) {\n  return n * n;\n}`,
      tests: [['square(4) === 16', '`square(4)` returns `16`'], ['square(-3) === 9', '`square(-3)` returns `9`'], ['square(0) === 0', '`square(0)` returns `0`']],
    },
    {
      t: 'Several Parameters',
      b: `Separate parameters with commas. The order of the arguments matches the order of the parameters:

\`\`\`js
function rectangleArea(width, height) {
  return width * height;
}
rectangleArea(3, 5);  // 15
\`\`\``,
      code: `// Write average(a, b, c) returning the mean of three numbers\n`,
      sol: `function average(a, b, c) {\n  return (a + b + c) / 3;\n}`,
      tests: [['average(1, 2, 3) === 2', '`average(1, 2, 3)` is `2`'], ['average(10, 20, 60) === 30', '`average(10, 20, 60)` is `30`']],
    },
    {
      t: 'return vs console.log',
      b: `A common beginner mix-up: **printing** a value is not the same as **returning** it.

\`\`\`js
function badAdd(a, b) { console.log(a + b); }  // prints, returns undefined
function goodAdd(a, b) { return a + b; }      // gives the value back

const x = badAdd(1, 2);   // prints 3, but x is undefined!
const y = goodAdd(1, 2);  // y is 3
\`\`\`

Return values can be stored, combined and passed on. Printed text is just gone.`,
      code: `function fullName(first, last) {\n  console.log(first + " " + last);\n}\n// Fix fullName so it RETURNS the name instead of printing it\n`,
      sol: `function fullName(first, last) {\n  return first + " " + last;\n}`,
      tests: [['fullName("Ada", "Lovelace") === "Ada Lovelace"', '`fullName("Ada", "Lovelace")` returns `"Ada Lovelace"`'], ['(fullName("a","b"), $logs.length === 0)', 'It should not print anything']],
    },
    {
      t: 'Default Parameters',
      b: `Give a parameter a **default value** to use when the argument is missing:

\`\`\`js
function greet(name = "friend") {
  return "Hi, " + name;
}
greet();        // "Hi, friend"
greet("Zoe");   // "Hi, Zoe"
\`\`\``,
      code: `// Write power(base, exponent) where exponent defaults to 2\n`,
      sol: `function power(base, exponent = 2) {\n  return base ** exponent;\n}`,
      tests: [['power(5) === 25', '`power(5)` is `25`'], ['power(2, 10) === 1024', '`power(2, 10)` is `1024`'], ['$src.includes("= 2") || $src.includes("=2")', 'Use a default parameter']],
    },
    {
      t: 'Function Expressions',
      b: `Functions are values, so you can store one in a variable. This is a **function expression**:

\`\`\`js
const triple = function (n) {
  return n * 3;
};
triple(4);  // 12
\`\`\``,
      code: `// Create a constant half that holds a function expression returning n / 2\n`,
      sol: `const half = function (n) {\n  return n / 2;\n};`,
      tests: [['half(10) === 5', '`half(10)` is `5`'], ['/const\\s+half\\s*=\\s*function/.test($src)', 'Use `const half = function …`']],
    },
    {
      t: 'Arrow Functions',
      b: `**Arrow functions** are a shorter way to write function expressions:

\`\`\`js
const add = (a, b) => {
  return a + b;
};
\`\`\`

With exactly one parameter, the parentheses are optional: \`n => { … }\`.`,
      code: `// Write isAdult as an arrow function: takes age, returns true if age >= 18\n`,
      sol: `const isAdult = (age) => {\n  return age >= 18;\n};`,
      tests: [['isAdult(20) === true && isAdult(15) === false', '`isAdult` works'], ['$src.includes("=>")', 'Use an arrow function']],
    },
    {
      t: 'Implicit Returns',
      b: `If an arrow function's body is a single expression, drop the braces and \`return\` — the value is returned automatically:

\`\`\`js
const add = (a, b) => a + b;
const shout = s => s.toUpperCase() + "!";
\`\`\``,
      code: `// Write cube as a ONE-LINE arrow function (no braces, no return)\n`,
      sol: `const cube = (n) => n ** 3;`,
      tests: [['cube(3) === 27', '`cube(3)` is `27`'], ['$src.includes("=>") && !$src.includes("return")', 'Use an implicit return']],
    },
    {
      t: 'Returning Early',
      b: `\`return\` ends the function immediately. That makes **guard clauses** easy:

\`\`\`js
function divide(a, b) {
  if (b === 0) return "Cannot divide by zero";
  return a / b;
}
\`\`\``,
      code: `// Write safeSqrt(n): return null for negative numbers, otherwise Math.sqrt(n)\n`,
      sol: `function safeSqrt(n) {\n  if (n < 0) return null;\n  return Math.sqrt(n);\n}`,
      tests: [['safeSqrt(-4) === null', '`safeSqrt(-4)` is `null`'], ['safeSqrt(81) === 9', '`safeSqrt(81)` is `9`']],
    },
    {
      t: 'Functions That Answer Questions',
      b: `Functions that return \`true\`/\`false\` are often named \`isSomething\` or \`hasSomething\`. You can return the comparison itself — no \`if\` needed:

\`\`\`js
// Long way
function isPositive(n) {
  if (n > 0) { return true; } else { return false; }
}
// Short way
const isPositive = n => n > 0;
\`\`\``,
      code: `// Write isEven(n) returning true for even numbers\n// Write isVowel(ch) returning true if ch is one of a, e, i, o, u\n`,
      sol: `const isEven = (n) => n % 2 === 0;\nconst isVowel = (ch) => "aeiou".includes(ch);`,
      tests: [['isEven(4) === true && isEven(7) === false', '`isEven` works'], ['isVowel("e") === true && isVowel("z") === false', '`isVowel` works']],
    },
    {
      t: 'Functions Calling Functions',
      b: `Build big functions from small ones. Each small function does one job well:

\`\`\`js
const square = n => n * n;
const sumOfSquares = (a, b) => square(a) + square(b);
\`\`\``,
      code: `const square = (n) => n * n;\n// Write hypotenuse(a, b): the square root of (a² + b²), using square\n`,
      sol: `const square = (n) => n * n;\nconst hypotenuse = (a, b) => Math.sqrt(square(a) + square(b));`,
      tests: [['hypotenuse(3, 4) === 5', '`hypotenuse(3, 4)` is `5`'], ['/square\\(/.test($src.split("hypotenuse")[1] || "")', 'Use `square` inside it']],
    },
    {
      t: 'Local Variables',
      b: `Variables declared inside a function are **local**: they exist only while the function runs, and cannot be seen outside it.

\`\`\`js
function makeTea() {
  const water = "hot";
  return "tea";
}
console.log(water); // ReferenceError
\`\`\`

This is a good thing — functions do not trip over each other's variables.`,
      code: `function countDown() {\n  let count = 3;\n  count--;\n}\ncountDown();\nconsole.log(count);\n// Fix: make countDown RETURN its count, and print what it returns\n`,
      sol: `function countDown() {\n  let count = 3;\n  count--;\n  return count;\n}\nconsole.log(countDown());`,
      tests: [['countDown() === 2', '`countDown()` returns `2`'], ['$out() === "2"', 'Print `2`']],
    },
    {
      t: 'Passing Functions Around',
      b: `Since functions are values, you can pass one **into** another function. A function passed in like this is called a **callback**:

\`\`\`js
function applyTwice(fn, x) {
  return fn(fn(x));
}
applyTwice(n => n + 10, 1);  // 21
\`\`\``,
      code: `// Write applyToFive(fn): calls fn with 5 and returns the result\n`,
      sol: `function applyToFive(fn) {\n  return fn(5);\n}`,
      tests: [['applyToFive((n) => n * 2) === 10', '`applyToFive(n => n * 2)` is `10`'], ['applyToFive(String) === "5"', '`applyToFive(String)` is `"5"`']],
    },
    {
      t: 'Repeating an Action',
      b: `Callbacks let a function decide **when** and **how often** to run someone else's code:

\`\`\`js
function repeat(times, action) {
  for (let i = 0; i < times; i++) action(i);
}
repeat(3, i => console.log("Round", i));
\`\`\``,
      code: `// Write repeat(times, action) that calls action(i) for i = 0 .. times-1\n// Then use it to print "Hello" 3 times\n`,
      sol: `function repeat(times, action) {\n  for (let i = 0; i < times; i++) {\n    action(i);\n  }\n}\nrepeat(3, () => console.log("Hello"));`,
      tests: [['$out() === "Hello\\nHello\\nHello"', 'Print `Hello` three times'], ['(() => { const seen = []; repeat(4, (i) => seen.push(i)); return seen.join() === "0,1,2,3"; })()', '`repeat` passes 0, 1, 2 … to the action']],
    },
    {
      t: 'Pure Functions',
      b: `A **pure** function:

1. only uses its parameters (not outside variables), and
2. does not change anything outside itself.

Pure functions are easy to test and reason about — the same input always gives the same output. The function below secretly depends on a global variable. Make it pure by passing the rate in.`,
      code: `let taxRate = 0.2;\nfunction addTax(price) {\n  return price + price * taxRate;\n}\n`,
      sol: `function addTax(price, taxRate) {\n  return price + price * taxRate;\n}`,
      tests: [['addTax(100, 0.2) === 120', '`addTax(100, 0.2)` is `120`'], ['addTax(50, 0.1) === 55', '`addTax(50, 0.1)` is `55`']],
    },
    {
      t: 'Any Number of Arguments',
      b: `A **rest parameter** \`...name\` collects all remaining arguments into an array:

\`\`\`js
function logAll(...things) {
  for (const t of things) console.log(t);
}
logAll("a", "b", "c");
\`\`\``,
      code: `// Write sum(...numbers) returning the total of all arguments (0 if none)\n`,
      sol: `function sum(...numbers) {\n  let total = 0;\n  for (const n of numbers) total += n;\n  return total;\n}`,
      tests: [['sum(1, 2, 3) === 6', '`sum(1, 2, 3)` is `6`'], ['sum() === 0', '`sum()` is `0`'], ['sum(5) === 5', '`sum(5)` is `5`'], ['$src.includes("...")', 'Use a rest parameter']],
    },
    {
      t: 'Hoisting',
      b: `Function **declarations** are "hoisted": you can call them *above* where they are written.

\`\`\`js
hello();                 // works!
function hello() { console.log("hi"); }
\`\`\`

Function **expressions** and arrow functions stored in \`const\` are not — calling them early is an error. This code crashes; fix it by turning the arrow function into a declaration.`,
      code: `console.log(shout("hey"));\n\nconst shout = (s) => s.toUpperCase() + "!";\n`,
      sol: `console.log(shout("hey"));\n\nfunction shout(s) {\n  return s.toUpperCase() + "!";\n}`,
      tests: [['$out() === "HEY!"', 'Print `HEY!`'], ['$src.indexOf("console.log") < $src.indexOf("function shout")', 'Keep the call above the function']],
    },
    {
      t: 'Returning Two Things',
      b: `A function can only return **one** value — but that value can be an array holding several things:

\`\`\`js
function minMax(a, b) {
  return a < b ? [a, b] : [b, a];
}
const [low, high] = minMax(9, 4);   // low = 4, high = 9
\`\`\``,
      code: `// Write divide(a, b) returning [quotient, remainder], using Math.floor and %\n`,
      sol: `function divide(a, b) {\n  return [Math.floor(a / b), a % b];\n}`,
      tests: [['JSON.stringify(divide(17, 5)) === "[3,2]"', '`divide(17, 5)` is `[3, 2]`'], ['JSON.stringify(divide(20, 4)) === "[5,0]"', '`divide(20, 4)` is `[5, 0]`']],
    },
    {
      t: 'Project: Password Checker',
      b: `Write \`passwordStrength(pw)\` that returns:

- \`"weak"\` if it is shorter than 8 characters
- \`"strong"\` if it is at least 12 characters **and** contains a digit
- \`"medium"\` otherwise

Write a helper \`hasDigit(s)\` that loops over the characters and returns \`true\` if any is \`"0"\`–\`"9"\`. (Tip: \`"0123456789".includes(ch)\`.)`,
      code: `function hasDigit(s) {\n}\n\nfunction passwordStrength(pw) {\n}\n`,
      sol: `function hasDigit(s) {\n  for (const ch of s) {\n    if ("0123456789".includes(ch)) return true;\n  }\n  return false;\n}\n\nfunction passwordStrength(pw) {\n  if (pw.length < 8) return "weak";\n  if (pw.length >= 12 && hasDigit(pw)) return "strong";\n  return "medium";\n}`,
      tests: [['hasDigit("abc1") === true && hasDigit("abc") === false', '`hasDigit` works'], ['passwordStrength("cat") === "weak"', '`"cat"` is weak'], ['passwordStrength("sunflowers") === "medium"', '`"sunflowers"` is medium'], ['passwordStrength("longpassword") === "medium"', '12+ characters without a digit is medium'], ['passwordStrength("correcthorse7") === "strong"', '`"correcthorse7"` is strong']],
    },
  ],
});
