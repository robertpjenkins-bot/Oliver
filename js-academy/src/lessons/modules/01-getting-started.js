registerModule({
  id: 'start',
  title: 'Getting Started',
  icon: '🚀',
  lessons: [
    {
      t: 'Hello, World!',
      b: `Welcome to **JavaScript** — the language that runs every website, and plenty of desktop apps too (including this one!).

Every programmer's first program prints a greeting. In JavaScript you print with \`console.log\`:

\`\`\`js
console.log("Hello there!");
\`\`\`

The text inside the quotes is called a **string**. Whatever you put between the parentheses shows up in the **console** panel below the editor.

Type your code in the editor on the right, then press **▶ Run & check** (or **Ctrl+Enter**).`,
      code: `// Write your code below this line\n`,
      sol: `console.log("Hello, World!");`,
      tests: [['$out() === "Hello, World!"', 'Print exactly `Hello, World!`']],
      hint: 'Copy the example, and change the text between the quotes to `Hello, World!` — capital letters and the comma matter.',
    },
    {
      t: 'Printing Numbers',
      b: `\`console.log\` can print numbers too. Numbers do **not** need quotes:

\`\`\`js
console.log(7);
console.log(3.14);
\`\`\`

If you put quotes around a number, JavaScript treats it as text instead of a number. We will see why that matters soon.`,
      code: `// Print the number 42\n`,
      sol: `console.log(42);`,
      tests: [['$logs[0] === "42"', 'Print the number `42`'], ['!/["\'`]42/.test($src)', 'Print it as a number — no quotes']],
    },
    {
      t: 'Many Lines',
      b: `A program is a list of instructions that run **from top to bottom**, one after another. Each \`console.log\` prints on its own line:

\`\`\`js
console.log("First");
console.log("Second");
\`\`\`

The semicolon \`;\` marks the end of an instruction (called a **statement**).`,
      code: `// Print Ready, Set and Go! on three lines\n`,
      sol: `console.log("Ready");\nconsole.log("Set");\nconsole.log("Go!");`,
      tests: [['$out() === "Ready\\nSet\\nGo!"', 'Print `Ready`, then `Set`, then `Go!` on three separate lines']],
    },
    {
      t: 'Single-Line Comments',
      b: `A **comment** is a note for humans. JavaScript ignores it completely. Start one with two slashes:

\`\`\`js
// This line is a comment
console.log("This runs"); // a comment can go at the end of a line too
\`\`\`

Programmers also "comment out" code to switch it off without deleting it.`,
      code: `console.log("Keep me");\nconsole.log("Oops");\n`,
      sol: `console.log("Keep me");\n// console.log("Oops");\n`,
      tests: [['$out() === "Keep me"', 'Only `Keep me` should be printed'], ['$src.includes("//")', 'Turn the second line into a comment with `//`']],
      hint: 'Put `//` at the very start of the second line.',
    },
    {
      t: 'Block Comments',
      b: `For longer notes, use a **block comment**. Everything between \`/*\` and \`*/\` is ignored, even across many lines:

\`\`\`js
/*
  This whole paragraph
  is a comment.
*/
\`\`\``,
      code: `console.log("Start");\nconsole.log("Skip 1");\nconsole.log("Skip 2");\nconsole.log("Skip 3");\nconsole.log("End");\n`,
      sol: `console.log("Start");\n/*\nconsole.log("Skip 1");\nconsole.log("Skip 2");\nconsole.log("Skip 3");\n*/\nconsole.log("End");\n`,
      tests: [['$out() === "Start\\nEnd"', 'Only `Start` and `End` should print'], ['$src.includes("/*") && $src.includes("*/")', 'Use one block comment `/* … */`']],
    },
    {
      t: 'Printing Several Values',
      b: `\`console.log\` accepts several values separated by commas. It prints them all on one line with a space between each:

\`\`\`js
console.log("Score:", 100);   // Score: 100
console.log(1, 2, 3);         // 1 2 3
\`\`\``,
      code: `// Print the text "Age:" and the number 30 using ONE console.log\n`,
      sol: `console.log("Age:", 30);`,
      tests: [['$out() === "Age: 30"', 'Print `Age: 30`'], ['$logs.length === 1 && /,\\s*30/.test($src)', 'Use one `console.log` with two values separated by a comma']],
    },
    {
      t: 'Quotes Inside Strings',
      b: `Strings can use **double quotes** \`"…"\`, **single quotes** \`'…'\` or **backticks** \`\\\`…\\\`\`. They all make strings.

Choosing the right one helps when your text contains a quote:

\`\`\`js
console.log("It's sunny");       // double quotes around an apostrophe
console.log('She said "wow"');   // single quotes around double quotes
\`\`\``,
      code: `// Print: It's fun\n`,
      sol: `console.log("It's fun");`,
      tests: [['$out() === "It\'s fun"', 'Print exactly `It\'s fun`']],
      hint: 'Wrap the text in double quotes so the apostrophe does not end the string.',
    },
    {
      t: 'Doing Math',
      b: `JavaScript is a great calculator. The main arithmetic operators are:

- \`+\` add
- \`-\` subtract
- \`*\` multiply
- \`/\` divide

\`\`\`js
console.log(10 + 5);  // 15
console.log(10 / 4);  // 2.5
\`\`\`

The computer works out the answer before printing it.`,
      code: `// Print the result of 7 times 6, then 100 minus 1\n`,
      sol: `console.log(7 * 6);\nconsole.log(100 - 1);`,
      tests: [['$logs[0] === "42" && $logs[1] === "99"', 'Print `42` then `99`'], ['$src.includes("*") && $src.includes("-")', 'Let JavaScript do the math with `*` and `-`']],
    },
    {
      t: 'Order of Operations',
      b: `Just like in maths class, \`*\` and \`/\` happen before \`+\` and \`-\`:

\`\`\`js
console.log(2 + 3 * 4);   // 14, not 20
\`\`\`

Use **parentheses** to decide the order yourself:

\`\`\`js
console.log((2 + 3) * 4); // 20
\`\`\``,
      code: `// This prints 14. Add parentheses so it prints 20.\nconsole.log(2 + 3 * 4);\n`,
      sol: `console.log((2 + 3) * 4);`,
      tests: [['$out() === "20"', 'Print `20`'], ['$src.includes("(2 + 3)") || $src.includes("(2+3)")', 'Group `2 + 3` with parentheses']],
    },
    {
      t: 'Fixing a Syntax Error',
      b: `A **syntax error** means JavaScript could not understand your code — like a sentence with missing punctuation. The program does not run at all.

The most common causes:

- a missing closing quote \`"\`
- a missing closing parenthesis \`)\`
- a typo in a keyword

Read the error in the console: it usually points at the problem.`,
      code: `console.log("Fix me);\n`,
      sol: `console.log("Fix me");`,
      tests: [['$out() === "Fix me"', 'Make the program print `Fix me`']],
      hint: 'The string is opened with `"` but never closed.',
    },
    {
      t: 'Reading Error Messages',
      b: `Some mistakes only show up **while** the program runs. These are **runtime errors**.

\`\`\`
ReferenceError: consol is not defined
\`\`\`

A \`ReferenceError\` means you used a name JavaScript does not know. Often it is just a typo! Error messages are your friends: they tell you *what* went wrong and usually *where*.`,
      code: `consol.log("Errors are helpful");\n`,
      sol: `console.log("Errors are helpful");`,
      tests: [['$out() === "Errors are helpful"', 'Fix the typo so the message prints']],
    },
    {
      t: 'Case Matters',
      b: `JavaScript is **case-sensitive**: \`console\`, \`Console\` and \`CONSOLE\` are three different names. Only the all-lowercase \`console.log\` exists.

The same rule applies to everything you name yourself later on: \`score\` and \`Score\` are different.`,
      code: `Console.Log("Case matters");\n`,
      sol: `console.log("Case matters");`,
      tests: [['$out() === "Case matters"', 'Print `Case matters`']],
    },
    {
      t: 'Warnings and Errors',
      b: `\`console\` has more than \`log\`. Programs use different methods to show how important a message is:

\`\`\`js
console.info("Just so you know");
console.warn("Careful!");        // shown in orange
console.error("Something broke"); // shown in red
\`\`\``,
      code: `// Warn "Low battery", then report the error "Out of power"\n`,
      sol: `console.warn("Low battery");\nconsole.error("Out of power");`,
      tests: [['$logs[0] === "Low battery" && $logs[1] === "Out of power"', 'Print `Low battery` then `Out of power`'], ['$src.includes("console.warn") && $src.includes("console.error")', 'Use `console.warn` and `console.error`']],
    },
    {
      t: 'New Lines and Tabs',
      b: `Inside a string, the backslash \`\\\` starts an **escape sequence** — a way to type characters you cannot easily type:

- \`\\n\` a new line
- \`\\t\` a tab
- \`\\\\\` a real backslash

\`\`\`js
console.log("Roses\\nViolets"); // prints on two lines
\`\`\``,
      code: `// Print "Line 1" and "Line 2" on two lines using ONE console.log\n`,
      sol: `console.log("Line 1\\nLine 2");`,
      tests: [['$out() === "Line 1\\nLine 2"', 'Output should be `Line 1` then `Line 2`'], ['$logs.length === 1', 'Use a single `console.log` with `\\n`']],
    },
    {
      t: 'Escaping Quotes',
      b: `What if a string needs the same quote that surrounds it? Put a backslash before it: \`\\"\` or \`\\'\`.

\`\`\`js
console.log("The sign said \\"STOP\\"");
// The sign said "STOP"
\`\`\``,
      code: `// Print: He said "hi"   (use double quotes around the whole string)\n`,
      sol: `console.log("He said \\"hi\\"");`,
      tests: [['$out() === "He said \\"hi\\""', 'Print `He said "hi"`'], ['$src.includes(\'\\\\"\')', 'Escape the inner quotes with `\\"`']],
    },
    {
      t: 'What Type Is It?',
      b: `Every value in JavaScript has a **type**. The \`typeof\` operator tells you which:

\`\`\`js
console.log(typeof 42);       // "number"
console.log(typeof "hello");  // "string"
console.log(typeof true);     // "boolean"
\`\`\`

\`true\` and \`false\` are **booleans** — yes/no values. You will use them a lot.`,
      code: `// Use typeof to print the types of: "hi", false and 3.5 (in that order)\n`,
      sol: `console.log(typeof "hi");\nconsole.log(typeof false);\nconsole.log(typeof 3.5);`,
      tests: [['$out() === "string\\nboolean\\nnumber"', 'Print `string`, `boolean`, `number`'], ['($src.match(/typeof/g) || []).length >= 3', 'Use `typeof` each time']],
    },
    {
      t: 'Joining Strings',
      b: `The \`+\` operator also **joins** strings together. This is called **concatenation**:

\`\`\`js
console.log("Sun" + "flower");   // Sunflower
console.log("Hello, " + "Sam");  // Hello, Sam
\`\`\`

Notice that \`+\` does not add a space for you — put it inside a string if you need one.`,
      code: `// Print "JavaScript" by joining "Java" and "Script" with +\n`,
      sol: `console.log("Java" + "Script");`,
      tests: [['$out() === "JavaScript"', 'Print `JavaScript`'], ['/["\']Java["\']\\s*\\+\\s*["\']Script["\']/.test($src)', 'Join `"Java"` and `"Script"` with `+`']],
    },
    {
      t: 'Numbers vs Strings',
      b: `\`+\` does two different jobs. With two numbers it **adds**. If either side is a string, it **joins**:

\`\`\`js
console.log(5 + 3);     // 8
console.log("5" + 3);   // "53"  ← joined, not added!
\`\`\`

This surprise catches everyone at some point. Watch out for numbers hiding in quotes.`,
      code: `// 1. Print "5" + 3   (a string plus a number)\n// 2. Print 5 + 3     (two numbers)\n`,
      sol: `console.log("5" + 3);\nconsole.log(5 + 3);`,
      tests: [['$logs[0] === "53"', 'First line should be `53`'], ['$logs[1] === "8"', 'Second line should be `8`'], ['$src.includes("+")', 'Use `+` both times']],
    },
    {
      t: 'A First Look at Variables',
      b: `A **variable** is a named box that stores a value so you can use it later:

\`\`\`js
let city = "Paris";
console.log("I love " + city);  // I love Paris
\`\`\`

\`let\` creates the variable, \`city\` is its name, and \`=\` puts a value in it. The next module is all about variables.`,
      code: `// Create a variable called name holding your name,\n// then print "Hello, " followed by name\n`,
      sol: `let name = "Ada";\nconsole.log("Hello, " + name);`,
      tests: [['typeof name === "string" && name.length > 0', 'Create a variable `name` holding a string'], ['$out() === "Hello, " + name', 'Print `Hello, ` followed by the name, using the variable']],
    },
    {
      t: 'Project: Name Badge',
      b: `Time to combine what you know! Print this badge exactly, border and all:

\`\`\`
+--------+
| Hi, JS |
+--------+
\`\`\`

You can use three \`console.log\` calls, or one call with \`\\n\` — your choice.`,
      code: `// Print the badge\n`,
      sol: `console.log("+--------+");\nconsole.log("| Hi, JS |");\nconsole.log("+--------+");`,
      tests: [['$out() === "+--------+\\n| Hi, JS |\\n+--------+"', 'Print the three-line badge exactly']],
      hint: 'Count the dashes: there are 8 between the `+` signs.',
    },
  ],
});
