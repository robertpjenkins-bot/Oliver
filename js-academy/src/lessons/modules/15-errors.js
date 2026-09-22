registerModule({
  id: 'errors',
  title: 'Errors & Debugging',
  icon: '🐞',
  lessons: [
    {
      t: 'When Things Go Wrong',
      b: `The most common runtime error in JavaScript:

\`\`\`
TypeError: Cannot read properties of undefined (reading 'name')
\`\`\`

It means you wrote \`something.name\` but \`something\` was \`undefined\`. Ask: *why* is it undefined? Often an index is off, or a key is misspelled.`,
      code: `const users = [{ name: "Ana" }, { name: "Ben" }];\nconst last = users[users.length];\nconsole.log(last.name);\n`,
      sol: `const users = [{ name: "Ana" }, { name: "Ben" }];\nconst last = users[users.length - 1];\nconsole.log(last.name);`,
      tests: [['$out() === "Ben"', 'Print `Ben`']],
    },
    {
      t: 'try and catch',
      b: `Wrap risky code in \`try\`. If it throws, the program jumps to \`catch\` instead of crashing:

\`\`\`js
try {
  riskyThing();
  console.log("worked");
} catch (err) {
  console.log("Oops:", err.message);
}
console.log("still running!");
\`\`\``,
      code: `// JSON.parse("{bad json") throws. Wrap it in try/catch and print "Could not read data" when it fails\nconst data = JSON.parse("{bad json");\n`,
      sol: `try {\n  const data = JSON.parse("{bad json");\n} catch (err) {\n  console.log("Could not read data");\n}`,
      tests: [['$out() === "Could not read data"', 'Print `Could not read data`'], ['$src.includes("try") && $src.includes("catch")', 'Use `try` / `catch`']],
    },
    {
      t: 'The Error Object',
      b: `\`catch\` receives an **error object**. Its most useful properties:

- \`err.name\` — the kind of error, like \`"TypeError"\`
- \`err.message\` — a human description
- \`err.stack\` — where it happened (great for debugging)`,
      code: `let kind;\nlet text;\ntry {\n  null.length;\n} catch (err) {\n  // store the error's name in kind and its message in text\n}\n`,
      sol: `let kind;\nlet text;\ntry {\n  null.length;\n} catch (err) {\n  kind = err.name;\n  text = err.message;\n}`,
      tests: [['kind === "TypeError"', '`kind` is `"TypeError"`'], ['typeof text === "string" && text.includes("null")', '`text` holds the message']],
    },
    {
      t: 'finally',
      b: `A \`finally\` block runs **no matter what** — success or error. Use it for clean-up, like hiding a loading spinner:

\`\`\`js
try {
  load();
} catch (e) {
  showError(e);
} finally {
  hideSpinner();
}
\`\`\``,
      code: `function attempt(fn) {\n  const log = [];\n  try {\n    fn();\n    log.push("ok");\n  } catch (e) {\n    log.push("error");\n  }\n  // always push "done" at the end using finally\n  return log;\n}\n`,
      sol: `function attempt(fn) {\n  const log = [];\n  try {\n    fn();\n    log.push("ok");\n  } catch (e) {\n    log.push("error");\n  } finally {\n    log.push("done");\n  }\n  return log;\n}`,
      tests: [['attempt(() => {}).join() === "ok,done"', 'Success → ok, done'], ['attempt(() => { throw new Error("x"); }).join() === "error,done"', 'Failure → error, done'], ['$src.includes("finally")', 'Use `finally`']],
    },
    {
      t: 'Throwing Your Own Errors',
      b: `Use \`throw\` to signal a problem yourself. Always throw an \`Error\` object (not a plain string), so you get a message and a stack trace:

\`\`\`js
function withdraw(amount) {
  if (amount <= 0) {
    throw new Error("Amount must be positive");
  }
  …
}
\`\`\``,
      code: `// Write setAge(age) that throws an Error with message "Age cannot be negative" for negative ages,\n// and otherwise returns age\n`,
      sol: `function setAge(age) {\n  if (age < 0) {\n    throw new Error("Age cannot be negative");\n  }\n  return age;\n}`,
      tests: [['setAge(12) === 12', '`setAge(12)` returns `12`'], ['(() => { try { setAge(-1); } catch (e) { return e instanceof Error && e.message === "Age cannot be negative"; } return false; })()', '`setAge(-1)` throws the right Error']],
    },
    {
      t: 'Validating Input',
      b: `Functions should check their input and fail **loudly and early** with a clear message, instead of producing nonsense later.

\`\`\`js
if (typeof name !== "string") throw new TypeError("name must be a string");
\`\`\``,
      code: `// Write area(width, height): throw a TypeError "Sizes must be numbers" if either is not a number,\n// throw a RangeError "Sizes must be positive" if either is 0 or less, otherwise return the area\n`,
      sol: `function area(width, height) {\n  if (typeof width !== "number" || typeof height !== "number") {\n    throw new TypeError("Sizes must be numbers");\n  }\n  if (width <= 0 || height <= 0) {\n    throw new RangeError("Sizes must be positive");\n  }\n  return width * height;\n}`,
      tests: [['area(3, 4) === 12', '`area(3, 4)` is `12`'], ['(() => { try { area("3", 4); } catch (e) { return e instanceof TypeError; } return false; })()', 'Non-numbers throw a `TypeError`'], ['(() => { try { area(3, 0); } catch (e) { return e instanceof RangeError && e.message === "Sizes must be positive"; } return false; })()', 'Zero or negative throws a `RangeError`']],
    },
    {
      t: 'Kinds of Errors',
      b: `JavaScript's built-in error types:

- \`SyntaxError\` — code that cannot be parsed
- \`ReferenceError\` — using a name that does not exist
- \`TypeError\` — using a value the wrong way (calling a non-function…)
- \`RangeError\` — a number out of allowed range

Predict which error each snippet throws.`,
      code: `// Fill in the error name each would throw: "ReferenceError", "TypeError" or "RangeError"\nconst a = ""; // notDefinedAnywhere + 1\nconst b = ""; // (42)()\nconst c = ""; // new Array(-1)\n`,
      sol: `const a = "ReferenceError";\nconst b = "TypeError";\nconst c = "RangeError";`,
      tests: [['a === "ReferenceError"', 'Using an unknown name'], ['b === "TypeError"', 'Calling a number'], ['c === "RangeError"', 'An array of negative length']],
    },
    {
      t: 'Custom Error Types',
      b: `You can make your own error types by **extending** \`Error\` (you will learn classes properly soon):

\`\`\`js
class NotFoundError extends Error {
  constructor(message) {
    super(message);
    this.name = "NotFoundError";
  }
}
throw new NotFoundError("No such user");
\`\`\``,
      code: `// Create a class ValidationError extending Error, with name "ValidationError"\n`,
      sol: `class ValidationError extends Error {\n  constructor(message) {\n    super(message);\n    this.name = "ValidationError";\n  }\n}`,
      tests: [['new ValidationError("bad") instanceof Error', 'It extends `Error`'], ['new ValidationError("bad").name === "ValidationError"', 'Its name is `"ValidationError"`'], ['new ValidationError("bad").message === "bad"', 'It keeps the message']],
    },
    {
      t: 'Catching the Right Error',
      b: `One \`catch\` block catches **everything** — even bugs you did not expect. Use \`instanceof\` to handle only the errors you planned for, and **rethrow** the rest:

\`\`\`js
try {
  save(data);
} catch (err) {
  if (err instanceof ValidationError) {
    showMessage(err.message);
  } else {
    throw err;   // not ours — let it crash loudly
  }
}
\`\`\``,
      code: `class InputError extends Error {}\n\n// Write safeRun(fn): calls fn and returns its result.\n// If fn throws an InputError, return "input problem: " + its message.\n// Any other error must be rethrown.\n`,
      sol: `class InputError extends Error {}\n\nfunction safeRun(fn) {\n  try {\n    return fn();\n  } catch (err) {\n    if (err instanceof InputError) return "input problem: " + err.message;\n    throw err;\n  }\n}`,
      tests: [['safeRun(() => 5) === 5', 'Returns normal results'], ['safeRun(() => { throw new InputError("empty"); }) === "input problem: empty"', 'Handles `InputError`'], ['(() => { try { safeRun(() => { throw new TypeError("bug"); }); } catch (e) { return e instanceof TypeError; } return false; })()', 'Rethrows other errors']],
    },
    {
      t: 'Errors as Values',
      b: `Throwing is not the only option. Some code returns a **result object** instead, which forces the caller to check:

\`\`\`js
function parseAge(text) {
  const n = Number(text);
  if (Number.isNaN(n)) return { ok: false, error: "Not a number" };
  return { ok: true, value: n };
}
\`\`\``,
      code: `// Write safeDivide(a, b) returning { ok: true, value } or { ok: false, error: "Division by zero" }\n`,
      sol: `function safeDivide(a, b) {\n  if (b === 0) return { ok: false, error: "Division by zero" };\n  return { ok: true, value: a / b };\n}`,
      tests: [['safeDivide(10, 4).ok === true && safeDivide(10, 4).value === 2.5', 'Normal division works'], ['safeDivide(1, 0).ok === false && safeDivide(1, 0).error === "Division by zero"', 'Division by zero is reported']],
    },
    {
      t: 'Parsing Safely',
      b: `Data from files, users and the internet is often broken. Wrap parsing in a helper that never crashes:

\`\`\`js
function tryParse(text, fallback) {
  try { return JSON.parse(text); } catch { return fallback; }
}
\`\`\`

(You can leave out \`(err)\` if you do not need it.)`,
      code: `// Write tryParse(text, fallback)\n`,
      sol: `function tryParse(text, fallback) {\n  try {\n    return JSON.parse(text);\n  } catch {\n    return fallback;\n  }\n}`,
      tests: [['tryParse(\'{"a":1}\', null).a === 1', 'Parses good JSON'], ['tryParse("oops", "default") === "default"', 'Returns the fallback for bad JSON']],
    },
    {
      t: 'Error Causes',
      b: `When you catch a low-level error and throw a friendlier one, keep the original as its **cause**:

\`\`\`js
try {
  JSON.parse(text);
} catch (err) {
  throw new Error("Settings file is damaged", { cause: err });
}
\`\`\`

Debugging tools show both.`,
      code: `// Write loadSettings(text): JSON.parse it; if that fails, throw new Error("Bad settings")\n// with the original error as its cause\n`,
      sol: `function loadSettings(text) {\n  try {\n    return JSON.parse(text);\n  } catch (err) {\n    throw new Error("Bad settings", { cause: err });\n  }\n}`,
      tests: [['loadSettings(\'{"x":2}\').x === 2', 'Good text parses'], ['(() => { try { loadSettings("??"); } catch (e) { return e.message === "Bad settings" && e.cause instanceof SyntaxError; } return false; })()', 'Bad text throws with a `cause`']],
    },
    {
      t: 'Assertions',
      b: `An **assertion** checks something that should always be true, and fails loudly if it is not. Writing your own tiny \`assert\` is a great habit:

\`\`\`js
function assert(condition, message) {
  if (!condition) throw new Error("Assertion failed: " + message);
}
assert(total >= 0, "total should never be negative");
\`\`\``,
      code: `// Write assert(condition, message) as above\n`,
      sol: `function assert(condition, message) {\n  if (!condition) throw new Error("Assertion failed: " + message);\n}`,
      tests: [['(assert(true, "fine"), true)', 'Passing assertions do nothing'], ['(() => { try { assert(1 > 2, "maths"); } catch (e) { return e.message === "Assertion failed: maths"; } return false; })()', 'Failing assertions throw']],
    },
    {
      t: 'Debugging with console.log',
      b: `When code gives the wrong answer, print what is happening inside it. Log values at each step and compare with what you *expected*.

This function should average an array, but gives the wrong answer. Add logging if it helps, find the bug, and fix it.`,
      code: `function average(nums) {\n  let total = 0;\n  for (let i = 1; i < nums.length; i++) {\n    total += nums[i];\n  }\n  return total / nums.length;\n}\n`,
      sol: `function average(nums) {\n  let total = 0;\n  for (let i = 0; i < nums.length; i++) {\n    total += nums[i];\n  }\n  return total / nums.length;\n}`,
      tests: [['average([2, 4, 6]) === 4', '`average([2, 4, 6])` is `4`'], ['average([10]) === 10', '`average([10])` is `10`']],
    },
    {
      t: 'Off-by-One Errors',
      b: `An **off-by-one** error is being one too many or one too few: \`<\` versus \`<=\`, starting at 0 versus 1. They are so common they have their own name!

This should return the numbers from \`start\` to \`end\` **inclusive**.`,
      code: `function between(start, end) {\n  const out = [];\n  for (let i = start; i < end; i++) {\n    out.push(i);\n  }\n  return out;\n}\n`,
      sol: `function between(start, end) {\n  const out = [];\n  for (let i = start; i <= end; i++) {\n    out.push(i);\n  }\n  return out;\n}`,
      tests: [['between(3, 6).join() === "3,4,5,6"', '`between(3, 6)` is [3, 4, 5, 6]'], ['between(5, 5).join() === "5"', '`between(5, 5)` is [5]']],
    },
    {
      t: 'Assignment Instead of Comparison',
      b: `Another classic: writing \`=\` (assign) when you meant \`===\` (compare). \`if (x = 5)\` **sets** x to 5, and 5 is truthy, so the block always runs!

This function says everyone is the admin. Fix it.`,
      code: `function isAdmin(user) {\n  if (user.role = "admin") {\n    return true;\n  }\n  return false;\n}\n`,
      sol: `function isAdmin(user) {\n  if (user.role === "admin") {\n    return true;\n  }\n  return false;\n}`,
      tests: [['isAdmin({ role: "admin" }) === true', 'Admins are admins'], ['isAdmin({ role: "guest" }) === false', 'Guests are not'], ['(() => { const u = { role: "guest" }; isAdmin(u); return u.role === "guest"; })()', 'Does not change the user']],
    },
    {
      t: 'Testing Your Code',
      b: `Professional programmers write **tests**: small programs that check other code. Here is a mini test runner:

\`\`\`js
function test(name, fn) {
  try {
    fn();
    console.log("✓ " + name);
  } catch (e) {
    console.log("✗ " + name + ": " + e.message);
  }
}
\`\`\`

(This app's "checks" work the same way!)`,
      code: `function capitalize(s) {\n  return s[0].toUpperCase() + s.slice(1);\n}\n// 1. Write test(name, fn) as above\n// 2. Use it to test capitalize with two tests named "capitalizes a word" and "handles one letter"\n//    (throw an Error inside a test if the result is wrong)\n`,
      sol: `function capitalize(s) {\n  return s[0].toUpperCase() + s.slice(1);\n}\nfunction test(name, fn) {\n  try {\n    fn();\n    console.log("✓ " + name);\n  } catch (e) {\n    console.log("✗ " + name + ": " + e.message);\n  }\n}\ntest("capitalizes a word", () => {\n  if (capitalize("hello") !== "Hello") throw new Error("expected Hello");\n});\ntest("handles one letter", () => {\n  if (capitalize("a") !== "A") throw new Error("expected A");\n});`,
      tests: [['$out() === "✓ capitalizes a word\\n✓ handles one letter"', 'Both tests pass and print ✓'], ['(() => { const before = $logs.length; test("x", () => { throw new Error("no"); }); return $logs[before] === "✗ x: no"; })()', 'Failing tests print ✗ with the message']],
    },
    {
      t: 'Silent Failures',
      b: `Some of the worst bugs produce **no error at all** — just wrong results. \`NaN\` spreading through calculations is a classic: one bad value and everything becomes \`NaN\`.

This function totals prices from text like \`"$4.50"\`, but returns \`NaN\`. Fix it (hint: strip the \`$\` before converting).`,
      code: `function totalPrices(prices) {\n  let total = 0;\n  for (const p of prices) {\n    total += Number(p);\n  }\n  return total;\n}\n`,
      sol: `function totalPrices(prices) {\n  let total = 0;\n  for (const p of prices) {\n    total += Number(p.replace("$", ""));\n  }\n  return total;\n}`,
      tests: [['totalPrices(["$4.50", "$1.25", "$10"]) === 15.75', 'Totals the prices']],
    },
    {
      t: 'Defensive Defaults',
      b: `Code that receives data from elsewhere should expect the unexpected: missing fields, wrong types, empty lists. Combine \`?.\`, \`??\` and \`Array.isArray\` to stay safe.`,
      code: `// Write countComments(post) returning the number of comments in post.comments,\n// or 0 if post is missing, comments is missing, or comments is not an array\n`,
      sol: `function countComments(post) {\n  const comments = post?.comments;\n  return Array.isArray(comments) ? comments.length : 0;\n}`,
      tests: [['countComments({ comments: ["hi", "wow"] }) === 2', 'Counts comments'], ['countComments({}) === 0 && countComments(null) === 0', 'Missing data gives 0'], ['countComments({ comments: "oops" }) === 0', 'Wrong types give 0']],
    },
    {
      t: 'Project: Safe Calculator',
      b: `Write \`calculate(expression)\` for strings like \`"12 * 4"\`: a number, a space, an operator (\`+ - * /\`), a space, a number.

- return the numeric result
- throw a \`SyntaxError\` with message \`"Expected: number operator number"\` if there are not exactly 3 parts
- throw a \`TypeError\` \`"Not a number: X"\` if either side is not a number (X is the bad part)
- throw a \`RangeError\` \`"Cannot divide by zero"\` for division by zero
- throw an \`Error\` \`"Unknown operator: X"\` for other operators`,
      code: `function calculate(expression) {\n}\n`,
      sol: 'function calculate(expression) {\n  const parts = expression.trim().split(" ");\n  if (parts.length !== 3) throw new SyntaxError("Expected: number operator number");\n  const [left, op, right] = parts;\n  const a = Number(left);\n  const b = Number(right);\n  if (Number.isNaN(a)) throw new TypeError(`Not a number: ${left}`);\n  if (Number.isNaN(b)) throw new TypeError(`Not a number: ${right}`);\n  switch (op) {\n    case "+": return a + b;\n    case "-": return a - b;\n    case "*": return a * b;\n    case "/":\n      if (b === 0) throw new RangeError("Cannot divide by zero");\n      return a / b;\n    default:\n      throw new Error(`Unknown operator: ${op}`);\n  }\n}',
      tests: [
        ['calculate("12 * 4") === 48 && calculate("7 - 10") === -3 && calculate("9 / 2") === 4.5', 'Calculates correctly'],
        ['(() => { try { calculate("1 +"); } catch (e) { return e instanceof SyntaxError && e.message === "Expected: number operator number"; } return false; })()', 'Wrong shape → SyntaxError'],
        ['(() => { try { calculate("x + 1"); } catch (e) { return e instanceof TypeError && e.message === "Not a number: x"; } return false; })()', 'Bad number → TypeError'],
        ['(() => { try { calculate("5 / 0"); } catch (e) { return e instanceof RangeError; } return false; })()', 'Divide by zero → RangeError'],
        ['(() => { try { calculate("2 ^ 3"); } catch (e) { return e.message === "Unknown operator: ^"; } return false; })()', 'Unknown operator → Error'],
      ],
    },
  ],
});
