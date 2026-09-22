registerModule({
  id: 'cond',
  title: 'Making Decisions',
  icon: '🔀',
  lessons: [
    {
      t: 'The if Statement',
      b: `\`if\` runs a block of code **only when** a condition is true:

\`\`\`js
const temperature = 30;
if (temperature > 25) {
  console.log("It's hot!");
}
\`\`\`

The condition goes in parentheses, and the code to run goes in curly braces.`,
      code: `const battery = 15;\n// If battery is below 20, print "Low battery"\n`,
      sol: `const battery = 15;\nif (battery < 20) {\n  console.log("Low battery");\n}`,
      tests: [['$out() === "Low battery"', 'Print `Low battery`'], ['/if\\s*\\(/.test($src)', 'Use an `if` statement']],
    },
    {
      t: 'if … else',
      b: `\`else\` gives the other path — what to do when the condition is false:

\`\`\`js
if (score >= 50) {
  console.log("Pass");
} else {
  console.log("Try again");
}
\`\`\`

Exactly one of the two blocks runs.`,
      code: `const age = 12;\n// Print "Adult ticket" if age is 18 or more, otherwise "Child ticket"\n`,
      sol: `const age = 12;\nif (age >= 18) {\n  console.log("Adult ticket");\n} else {\n  console.log("Child ticket");\n}`,
      tests: [['$out() === "Child ticket"', 'Print `Child ticket` for age 12'], ['$src.includes("else")', 'Use `else`']],
    },
    {
      t: 'else if Chains',
      b: `Chain several conditions with \`else if\`. They are checked top to bottom, and only the **first** true one runs:

\`\`\`js
if (hour < 12) {
  console.log("Good morning");
} else if (hour < 18) {
  console.log("Good afternoon");
} else {
  console.log("Good evening");
}
\`\`\``,
      code: `const score = 83;\nlet grade;\n// Set grade: "A" for 90+, "B" for 80+, "C" for 70+, otherwise "F"\n`,
      sol: `const score = 83;\nlet grade;\nif (score >= 90) {\n  grade = "A";\n} else if (score >= 80) {\n  grade = "B";\n} else if (score >= 70) {\n  grade = "C";\n} else {\n  grade = "F";\n}`,
      tests: [['grade === "B"', '`grade` is `"B"` for 83'], ['$src.includes("else if")', 'Use `else if`']],
    },
    {
      t: 'Order Matters',
      b: `Because only the first matching branch runs, put the **most specific** checks first. This code is buggy: a score of 95 prints "Good", because \`>= 50\` is checked first and is already true.`,
      code: `const score = 95;\nif (score >= 50) {\n  console.log("Good");\n} else if (score >= 90) {\n  console.log("Excellent");\n}\n`,
      sol: `const score = 95;\nif (score >= 90) {\n  console.log("Excellent");\n} else if (score >= 50) {\n  console.log("Good");\n}`,
      tests: [['$out() === "Excellent"', 'A score of 95 should print `Excellent`']],
    },
    {
      t: 'Conditions with Logic',
      b: `Conditions can use \`&&\`, \`||\` and \`!\` — anything that produces a boolean:

\`\`\`js
if (hasKey && !alarmOn) {
  console.log("Door opens");
}
\`\`\``,
      code: `const username = "admin";\nconst password = "letmein";\n// Print "Welcome" if username is "admin" AND password is "letmein", otherwise print "Denied"\n`,
      sol: `const username = "admin";\nconst password = "letmein";\nif (username === "admin" && password === "letmein") {\n  console.log("Welcome");\n} else {\n  console.log("Denied");\n}`,
      tests: [['$out() === "Welcome"', 'Print `Welcome`'], ['$src.includes("&&")', 'Combine both checks with `&&`']],
    },
    {
      t: 'Nested if',
      b: `You can put an \`if\` inside another \`if\`:

\`\`\`js
if (isLoggedIn) {
  if (isAdmin) {
    console.log("Admin panel");
  } else {
    console.log("Dashboard");
  }
}
\`\`\`

If nesting gets deep, try combining conditions instead — flat code is easier to read.`,
      code: `const hasTicket = true;\nconst seat = "VIP";\n// If they have a ticket: print "VIP lounge" for a "VIP" seat, else "Main hall".\n// Without a ticket, print "Box office".\n`,
      sol: `const hasTicket = true;\nconst seat = "VIP";\nif (hasTicket) {\n  if (seat === "VIP") {\n    console.log("VIP lounge");\n  } else {\n    console.log("Main hall");\n  }\n} else {\n  console.log("Box office");\n}`,
      tests: [['$out() === "VIP lounge"', 'Print `VIP lounge`']],
    },
    {
      t: 'Truthy Conditions',
      b: `An \`if\` condition does not have to be a real boolean. Any truthy value runs the block:

\`\`\`js
const name = "";
if (name) {
  console.log("Hi " + name);
} else {
  console.log("Who are you?");  // runs: "" is falsy
}
\`\`\``,
      code: `const cartItems = 0;\n// If cartItems is truthy print "Checkout", otherwise print "Cart is empty"\n// Use cartItems directly as the condition\n`,
      sol: `const cartItems = 0;\nif (cartItems) {\n  console.log("Checkout");\n} else {\n  console.log("Cart is empty");\n}`,
      tests: [['$out() === "Cart is empty"', 'Print `Cart is empty`'], ['/if\\s*\\(\\s*cartItems\\s*\\)/.test($src)', 'Use `if (cartItems)`']],
    },
    {
      t: 'The Ternary Operator',
      b: `For a quick either/or **value**, use the ternary operator \`condition ? a : b\`:

\`\`\`js
const label = age >= 18 ? "adult" : "minor";
\`\`\`

It reads: *is age ≥ 18? If so "adult", otherwise "minor".*`,
      code: `const lives = 0;\n// Create status: "Game over" if lives is 0, otherwise "Playing" — use ? :\n`,
      sol: `const lives = 0;\nconst status = lives === 0 ? "Game over" : "Playing";`,
      tests: [['status === "Game over"', '`status` is `"Game over"`'], ['$src.includes("?") && $src.includes(":")', 'Use the ternary operator']],
    },
    {
      t: 'Ternaries in Templates',
      b: `Ternaries shine inside template literals, for example to get plurals right:

\`\`\`js
console.log(\`You have \${n} \${n === 1 ? "apple" : "apples"}\`);
\`\`\``,
      code: `const count = 1;\n// Print "1 new message" (or "N new messages" when count is not 1)\n`,
      sol: 'const count = 1;\nconsole.log(`${count} new ${count === 1 ? "message" : "messages"}`);',
      tests: [['$out() === "1 new message"', 'Print `1 new message`'], ['$src.includes("?")', 'Use a ternary to choose the word']],
    },
    {
      t: 'The switch Statement',
      b: `When comparing one value against many options, \`switch\` can be tidier:

\`\`\`js
switch (fruit) {
  case "apple":
    console.log("Red");
    break;
  case "banana":
    console.log("Yellow");
    break;
  default:
    console.log("Unknown");
}
\`\`\`

\`break\` stops the switch; \`default\` runs if nothing matched.`,
      code: `const day = 3;\nlet name;\n// Using switch, set name to "Mon" for 1, "Tue" for 2, "Wed" for 3, otherwise "?"\n`,
      sol: `const day = 3;\nlet name;\nswitch (day) {\n  case 1:\n    name = "Mon";\n    break;\n  case 2:\n    name = "Tue";\n    break;\n  case 3:\n    name = "Wed";\n    break;\n  default:\n    name = "?";\n}`,
      tests: [['name === "Wed"', '`name` is `"Wed"`'], ['$src.includes("switch") && $src.includes("default")', 'Use `switch` with a `default`']],
    },
    {
      t: 'Forgetting break',
      b: `Without \`break\`, a switch **falls through** into the next case and keeps running! Sometimes that is useful — grouping cases:

\`\`\`js
switch (day) {
  case "Sat":
  case "Sun":
    console.log("Weekend");
    break;
  default:
    console.log("Weekday");
}
\`\`\`

The code below prints too much. Fix it.`,
      code: `const light = "red";\nswitch (light) {\n  case "red":\n    console.log("Stop");\n  case "yellow":\n    console.log("Slow down");\n  case "green":\n    console.log("Go");\n}\n`,
      sol: `const light = "red";\nswitch (light) {\n  case "red":\n    console.log("Stop");\n    break;\n  case "yellow":\n    console.log("Slow down");\n    break;\n  case "green":\n    console.log("Go");\n    break;\n}`,
      tests: [['$out() === "Stop"', 'Only print `Stop` for red']],
    },
    {
      t: 'Grouping Cases',
      b: `Stack several \`case\` labels to share one block:

\`\`\`js
case "a":
case "e":
case "i":
  kind = "vowel";
  break;
\`\`\``,
      code: `const month = "Feb";\nlet season;\n// Using switch with grouped cases:\n// "Dec", "Jan", "Feb" → "winter"; "Jun", "Jul", "Aug" → "summer"; anything else → "other"\n`,
      sol: `const month = "Feb";\nlet season;\nswitch (month) {\n  case "Dec":\n  case "Jan":\n  case "Feb":\n    season = "winter";\n    break;\n  case "Jun":\n  case "Jul":\n  case "Aug":\n    season = "summer";\n    break;\n  default:\n    season = "other";\n}`,
      tests: [['season === "winter"', '`season` is `"winter"`'], ['$src.includes("switch")', 'Use `switch`']],
    },
    {
      t: 'Early Exits with Guards',
      b: `A **guard** checks for a problem first and stops early, so the main code does not need to be nested:

\`\`\`js
if (!input) {
  console.log("Nothing to do");
} else {
  // the real work
}
\`\`\`

You will use this pattern with \`return\` once we learn functions.`,
      code: `const quantity = -2;\n// If quantity is less than 1, print "Invalid quantity".\n// Otherwise print "Ordering N items" (with the number)\n`,
      sol: 'const quantity = -2;\nif (quantity < 1) {\n  console.log("Invalid quantity");\n} else {\n  console.log(`Ordering ${quantity} items`);\n}',
      tests: [['$out() === "Invalid quantity"', 'Print `Invalid quantity` for -2']],
    },
    {
      t: 'Even or Odd',
      b: `Combine \`%\` with \`if\` to check divisibility. A number is **even** when dividing by 2 leaves nothing:

\`\`\`js
if (n % 2 === 0) { /* even */ }
\`\`\``,
      code: `const n = 17;\nlet parity;\n// Set parity to "even" or "odd"\n`,
      sol: `const n = 17;\nlet parity;\nif (n % 2 === 0) {\n  parity = "even";\n} else {\n  parity = "odd";\n}`,
      tests: [['parity === "odd"', '`parity` is `"odd"` for 17'], ['$src.includes("% 2")', 'Use `% 2`']],
    },
    {
      t: 'FizzBuzz (One Number)',
      b: `FizzBuzz is a famous programming puzzle. For a number \`n\`:

- divisible by **both 3 and 5** → \`"FizzBuzz"\`
- divisible by **3** → \`"Fizz"\`
- divisible by **5** → \`"Buzz"\`
- otherwise → the number itself

Careful with the order!`,
      code: `const n = 30;\nlet result;\n// Set result using the FizzBuzz rules\n`,
      sol: `const n = 30;\nlet result;\nif (n % 15 === 0) {\n  result = "FizzBuzz";\n} else if (n % 3 === 0) {\n  result = "Fizz";\n} else if (n % 5 === 0) {\n  result = "Buzz";\n} else {\n  result = n;\n}`,
      tests: [['result === "FizzBuzz"', '`result` is `"FizzBuzz"` for 30']],
      hint: 'Check "both" first — otherwise the `% 3` branch wins.',
    },
    {
      t: 'Clamping a Value',
      b: `Keeping a value inside limits is called **clamping**. Games do this all the time — health cannot go above the max or below zero.`,
      code: `let health = 130;\nconst MAX = 100;\n// Clamp health so it is never above MAX and never below 0\n`,
      sol: `let health = 130;\nconst MAX = 100;\nif (health > MAX) {\n  health = MAX;\n} else if (health < 0) {\n  health = 0;\n}`,
      tests: [['health === 100', '`health` becomes `100`']],
    },
    {
      t: 'Comparing Three Values',
      b: `Finding the largest of several values with \`if\` is a good logic workout (even though \`Math.max\` exists). Assume the first is the biggest, then check the others one by one.`,
      code: `const a = 12, b = 47, c = 33;\nlet biggest = a;\n// Using if statements (not Math.max), update biggest\n`,
      sol: `const a = 12, b = 47, c = 33;\nlet biggest = a;\nif (b > biggest) {\n  biggest = b;\n}\nif (c > biggest) {\n  biggest = c;\n}`,
      tests: [['biggest === 47', '`biggest` is `47`'], ['!$src.includes("Math.max")', 'Do not use `Math.max`']],
    },
    {
      t: 'Rock, Paper, Scissors',
      b: `Decide the winner of a round. Rock beats scissors, scissors beats paper, paper beats rock. Same choice is a draw.

Set \`result\` to \`"draw"\`, \`"player"\` or \`"computer"\`.`,
      code: `const player = "paper";\nconst computer = "rock";\nlet result;\n`,
      sol: `const player = "paper";\nconst computer = "rock";\nlet result;\nif (player === computer) {\n  result = "draw";\n} else if (\n  (player === "rock" && computer === "scissors") ||\n  (player === "scissors" && computer === "paper") ||\n  (player === "paper" && computer === "rock")\n) {\n  result = "player";\n} else {\n  result = "computer";\n}`,
      tests: [['result === "player"', 'Paper beats rock: `result` is `"player"`']],
    },
    {
      t: 'Choosing with an Object',
      b: `A neat alternative to long \`if\`/\`switch\` chains is a **lookup object** (you will learn objects properly later):

\`\`\`js
const sounds = { dog: "woof", cat: "meow" };
const sound = sounds[animal] ?? "...";
\`\`\``,
      code: `const colors = { red: "#f00", green: "#0f0", blue: "#00f" };\nconst choice = "green";\n// Create hex: the color for choice, or "#000" if it is not in the object\n`,
      sol: `const colors = { red: "#f00", green: "#0f0", blue: "#00f" };\nconst choice = "green";\nconst hex = colors[choice] ?? "#000";`,
      tests: [['hex === "#0f0"', '`hex` is `"#0f0"`'], ['$src.includes("colors[choice]")', 'Look it up with `colors[choice]`']],
    },
    {
      t: 'Project: Shipping Cost',
      b: `An online shop charges shipping like this:

- orders of **$50 or more** ship **free**
- otherwise: **$5** for up to 2 kg, **$9** for up to 10 kg, **$15** above that
- **express** delivery adds **$10** — even on free orders

Set \`shipping\` from \`orderTotal\`, \`weightKg\` and \`express\`, then print \`Shipping: $N\`.`,
      code: `const orderTotal = 42;\nconst weightKg = 3.5;\nconst express = true;\n`,
      sol: 'const orderTotal = 42;\nconst weightKg = 3.5;\nconst express = true;\nlet shipping;\nif (orderTotal >= 50) {\n  shipping = 0;\n} else if (weightKg <= 2) {\n  shipping = 5;\n} else if (weightKg <= 10) {\n  shipping = 9;\n} else {\n  shipping = 15;\n}\nif (express) {\n  shipping += 10;\n}\nconsole.log(`Shipping: $${shipping}`);',
      tests: [['shipping === 19', '`shipping` is `19`'], ['$out() === "Shipping: $19"', 'Print `Shipping: $19`']],
    },
  ],
});
