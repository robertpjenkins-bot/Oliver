registerModule({
  id: 'logic',
  title: 'Booleans & Logic',
  icon: '⚖️',
  lessons: [
    {
      t: 'True or False',
      b: `A **boolean** is a value that is either \`true\` or \`false\` — no quotes. Booleans are how programs make decisions:

\`\`\`js
let isRaining = true;
let hasUmbrella = false;
\`\`\`

A good boolean name reads like a yes/no question: \`isOpen\`, \`hasKey\`, \`canJump\`.`,
      code: `// Create lightsOn set to true and doorLocked set to false\n`,
      sol: `let lightsOn = true;\nlet doorLocked = false;`,
      tests: [['lightsOn === true', '`lightsOn` is `true`'], ['doorLocked === false', '`doorLocked` is `false`']],
    },
    {
      t: 'Comparing Numbers',
      b: `Comparison operators produce booleans:

\`\`\`js
5 > 3     // true   greater than
5 < 3     // false  less than
5 >= 5    // true   greater than or equal
4 <= 3    // false  less than or equal
\`\`\``,
      code: `const age = 14;\nconst score = 850;\n// Create isTeen (age is at least 13) and highScore (score is more than 1000)\n`,
      sol: `const age = 14;\nconst score = 850;\nconst isTeen = age >= 13;\nconst highScore = score > 1000;`,
      tests: [['isTeen === true', '`isTeen` is `true`'], ['highScore === false', '`highScore` is `false`'], ['$src.includes(">=") && $src.includes(">")', 'Use comparison operators']],
    },
    {
      t: 'Strict Equality',
      b: `To check whether two values are the same, use **three** equals signs: \`===\`.

\`\`\`js
3 === 3        // true
"3" === 3      // false — different types
"hi" === "hi"  // true
\`\`\`

Remember: one \`=\` **assigns**, three \`===\` **compare**.`,
      code: `const code = 4271;\nconst entered = 4271;\n// Create unlocked: whether entered equals code\n`,
      sol: `const code = 4271;\nconst entered = 4271;\nconst unlocked = entered === code;`,
      tests: [['unlocked === true', '`unlocked` is `true`'], ['$src.includes("===")', 'Use `===`']],
    },
    {
      t: 'Not Equal',
      b: `\`!==\` is the opposite of \`===\`: it is \`true\` when the values are **different**.

\`\`\`js
"cat" !== "dog"  // true
7 !== 7          // false
\`\`\``,
      code: `const oldPassword = "hunter2";\nconst newPassword = "hunter3";\n// Create changed: whether the new password is different from the old one\n`,
      sol: `const oldPassword = "hunter2";\nconst newPassword = "hunter3";\nconst changed = newPassword !== oldPassword;`,
      tests: [['changed === true', '`changed` is `true`'], ['$src.includes("!==")', 'Use `!==`']],
    },
    {
      t: 'Loose Equality',
      b: `You may see \`==\` (two equals). It converts types before comparing, which leads to strange results:

\`\`\`js
"5" == 5      // true
0 == ""       // true   (!)
null == undefined  // true
\`\`\`

Stick with \`===\` and \`!==\`. The code below has a bug caused by \`==\`: the input is the **string** \`"0"\`, but it counts as "no apples".`,
      code: `const input = "0";\nconst noApples = input == 0 && input == false;\n// Fix: noApples should only be true if input is exactly the NUMBER 0\n`,
      sol: `const input = "0";\nconst noApples = input === 0;`,
      tests: [['noApples === false', '`noApples` should be `false` for the string `"0"`'], ['!/[^=!]==[^=]/.test($src)', 'Do not use `==`']],
    },
    {
      t: 'AND: Both Must Be True',
      b: `\`&&\` (AND) is \`true\` only if **both** sides are true:

\`\`\`js
true && true    // true
true && false   // false
\`\`\`

\`\`\`js
const canDrive = age >= 17 && hasLicence;
\`\`\``,
      code: `const hasTicket = true;\nconst hasPassport = false;\n// Create canBoard: true only if they have a ticket AND a passport\n`,
      sol: `const hasTicket = true;\nconst hasPassport = false;\nconst canBoard = hasTicket && hasPassport;`,
      tests: [['canBoard === false', '`canBoard` is `false`'], ['$src.includes("&&")', 'Use `&&`']],
    },
    {
      t: 'OR: At Least One',
      b: `\`||\` (OR) is \`true\` if **either** side is true:

\`\`\`js
false || true   // true
false || false  // false
\`\`\`

\`\`\`js
const isWeekend = day === "Sat" || day === "Sun";
\`\`\``,
      code: `const day = "Sun";\n// Create isWeekend: true if day is "Sat" or "Sun"\n`,
      sol: `const day = "Sun";\nconst isWeekend = day === "Sat" || day === "Sun";`,
      tests: [['isWeekend === true', '`isWeekend` is `true`'], ['$src.includes("||")', 'Use `||`']],
    },
    {
      t: 'NOT: Flip It',
      b: `\`!\` (NOT) flips a boolean:

\`\`\`js
!true   // false
!false  // true
\`\`\`

\`\`\`js
const isAwake = !isAsleep;
\`\`\``,
      code: `const isMuted = true;\n// Create soundOn: the opposite of isMuted (use !)\n`,
      sol: `const isMuted = true;\nconst soundOn = !isMuted;`,
      tests: [['soundOn === false', '`soundOn` is `false`'], ['$src.includes("!isMuted")', 'Use `!isMuted`']],
    },
    {
      t: 'Combining Conditions',
      b: `\`&&\` is evaluated before \`||\`, just like \`*\` before \`+\`. Use parentheses to make your meaning clear:

\`\`\`js
const goOut = isSunny && (isWeekend || onHoliday);
\`\`\``,
      code: `const temperature = 24;\nconst isRaining = false;\nconst isWeekend = true;\nconst onHoliday = false;\n// Create beachDay: warm (at least 20), not raining, and either weekend or holiday\n`,
      sol: `const temperature = 24;\nconst isRaining = false;\nconst isWeekend = true;\nconst onHoliday = false;\nconst beachDay = temperature >= 20 && !isRaining && (isWeekend || onHoliday);`,
      tests: [['beachDay === true', '`beachDay` is `true`'], ['$src.includes("(") && $src.includes("||") && $src.includes("&&")', 'Combine `&&`, `||` and parentheses']],
    },
    {
      t: 'Truthy and Falsy',
      b: `In a yes/no situation, JavaScript treats **every** value as either "truthy" or "falsy". Only these are falsy:

\`\`\`js
false, 0, -0, 0n, "", null, undefined, NaN
\`\`\`

**Everything else is truthy** — including \`"0"\`, \`"false"\`, \`[]\` and \`{}\`.

\`Boolean(value)\` tells you which it is.`,
      code: `// Set each variable to true or false: is the value truthy?\nconst a = /* Boolean("") */ ;\nconst b = /* Boolean("0") */ ;\nconst c = /* Boolean(0) */ ;\nconst d = /* Boolean([]) */ ;\n`,
      sol: `const a = false;\nconst b = true;\nconst c = false;\nconst d = true;`,
      tests: [['a === false', '`""` is falsy'], ['b === true', '`"0"` is truthy'], ['c === false', '`0` is falsy'], ['d === true', '`[]` is truthy']],
    },
    {
      t: 'Converting to Boolean',
      b: `Two ways to turn any value into a real boolean:

\`\`\`js
Boolean("hello")  // true
!!"hello"         // true — "not not"
!!0               // false
\`\`\``,
      code: `const nickname = "";\nconst messages = 3;\n// Create hasNickname and hasMessages as real booleans\n`,
      sol: `const nickname = "";\nconst messages = 3;\nconst hasNickname = Boolean(nickname);\nconst hasMessages = !!messages;`,
      tests: [['hasNickname === false', '`hasNickname` is `false`'], ['hasMessages === true', '`hasMessages` is `true`']],
    },
    {
      t: 'Short-Circuit AND',
      b: `\`&&\` stops as soon as it finds a falsy value, and returns **that value**. If everything is truthy, it returns the last one:

\`\`\`js
0 && "never checked"    // 0
"a" && "b"              // "b"
\`\`\`

So \`user && user.name\` safely gets the name only if there *is* a user.`,
      code: `const user = null;\n// Create name: user && user.name  (should not crash!)\n`,
      sol: `const user = null;\nconst name = user && user.name;`,
      tests: [['name === null', '`name` should be `null`, without crashing'], ['$src.includes("&&")', 'Use `&&`']],
    },
    {
      t: 'Defaults with OR',
      b: `\`||\` returns the **first truthy** value. This makes it a classic way to set a fallback:

\`\`\`js
const title = input || "Untitled";
\`\`\`

If \`input\` is \`""\`, \`title\` becomes \`"Untitled"\`.`,
      code: `const typedName = "";\n// Create displayName: typedName, or "Guest" if it is empty\n`,
      sol: `const typedName = "";\nconst displayName = typedName || "Guest";`,
      tests: [['displayName === "Guest"', '`displayName` is `"Guest"`'], ['$src.includes("||")', 'Use `||`']],
    },
    {
      t: 'Nullish Coalescing',
      b: `\`||\` treats \`0\` and \`""\` as "missing" — sometimes wrong! A volume of \`0\` is a real setting.

\`??\` only falls back when the value is \`null\` or \`undefined\`:

\`\`\`js
0 || 50    // 50  ✗
0 ?? 50    // 0   ✓
null ?? 50 // 50
\`\`\``,
      code: `const savedVolume = 0;\nconst savedTheme = undefined;\n// Create volume (default 50) and theme (default "dark"), keeping real values like 0\n`,
      sol: `const savedVolume = 0;\nconst savedTheme = undefined;\nconst volume = savedVolume ?? 50;\nconst theme = savedTheme ?? "dark";`,
      tests: [['volume === 0', '`volume` keeps `0`'], ['theme === "dark"', '`theme` falls back to `"dark"`'], ['$src.includes("??")', 'Use `??`']],
    },
    {
      t: 'Alphabetical Order',
      b: `\`<\` and \`>\` compare strings letter by letter:

\`\`\`js
"apple" < "banana"  // true
"Zoo" < "apple"     // true — all capitals come before lowercase!
\`\`\`

For fair comparisons, lowercase both first.`,
      code: `const a = "zebra";\nconst b = "Aardvark";\n// Create aFirst: whether a comes before b alphabetically, ignoring case\n`,
      sol: `const a = "zebra";\nconst b = "Aardvark";\nconst aFirst = a.toLowerCase() < b.toLowerCase();`,
      tests: [['aFirst === false', '`aFirst` is `false`'], ['$src.includes("toLowerCase")', 'Ignore case with `toLowerCase`']],
    },
    {
      t: 'In a Range',
      b: `To check that a number is between two others, you need **two** comparisons joined by \`&&\`. Maths-style \`1 < x < 10\` does **not** work in JavaScript!

\`\`\`js
const inRange = x >= 1 && x <= 10;
\`\`\``,
      code: `const guess = 42;\n// Create valid: whether guess is from 1 to 100 inclusive\n`,
      sol: `const guess = 42;\nconst valid = guess >= 1 && guess <= 100;`,
      tests: [['valid === true', '`valid` is `true`'], ['/guess\\s*>=\\s*1/.test($src) && /guess\\s*<=\\s*100/.test($src)', 'Compare with both ends of the range']],
    },
    {
      t: 'Leap Years',
      b: `The leap year rule is a great logic puzzle. A year is a leap year if it is:

- divisible by 4, **and**
- **not** divisible by 100, **unless** it is also divisible by 400.

In code: \`(y % 4 === 0 && y % 100 !== 0) || y % 400 === 0\``,
      code: `const year = 1900;\n// Create isLeap using the rule above\n`,
      sol: `const year = 1900;\nconst isLeap = (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;`,
      tests: [['isLeap === false', '1900 is not a leap year'], ['$src.includes("% 400")', 'Include the 400 rule']],
    },
    {
      t: 'Exactly One',
      b: `Sometimes you need "one or the other, but not both" (called **exclusive or**, XOR). For booleans, that is simply \`a !== b\`:

\`\`\`js
true !== false  // true  — exactly one
true !== true   // false — both
\`\`\``,
      code: `const hasCoupon = true;\nconst isMember = true;\n// Discounts cannot be combined. Create oneDiscount: true if exactly one applies\n`,
      sol: `const hasCoupon = true;\nconst isMember = true;\nconst oneDiscount = hasCoupon !== isMember;`,
      tests: [['oneDiscount === false', '`oneDiscount` is `false` when both apply']],
    },
    {
      t: "De Morgan's Laws",
      b: `Two handy rules for flipping conditions:

\`\`\`js
!(a && b)  ===  !a || !b
!(a || b)  ===  !a && !b
\`\`\`

"Not (hungry and tired)" is the same as "not hungry, or not tired".`,
      code: `const isOpen = true;\nconst hasStock = false;\n// Create cannotBuy = !(isOpen && hasStock) but WITHOUT parentheses-with-!, using || instead\n`,
      sol: `const isOpen = true;\nconst hasStock = false;\nconst cannotBuy = !isOpen || !hasStock;`,
      tests: [['cannotBuy === true', '`cannotBuy` is `true`'], ['$src.includes("!isOpen") && $src.includes("!hasStock") && $src.includes("||")', 'Write it as `!isOpen || !hasStock`']],
    },
    {
      t: 'Project: Roller Coaster Rules',
      b: `The park's rules for the *Thunder Loop*:

- riders must be **at least 120 cm** tall, **and**
- either **12 or older**, or accompanied by an adult, **and**
- **not** have a heart condition.

Create \`canRide\` from the variables. Then set \`reason\` to \`"Enjoy the ride!"\` if they can ride, otherwise \`"Sorry, not today"\`, using \`&&\` / \`||\` (no \`if\` yet!).`,
      code: `const height = 131;\nconst age = 10;\nconst withAdult = true;\nconst heartCondition = false;\n`,
      sol: `const height = 131;\nconst age = 10;\nconst withAdult = true;\nconst heartCondition = false;\nconst canRide = height >= 120 && (age >= 12 || withAdult) && !heartCondition;\nconst reason = (canRide && "Enjoy the ride!") || "Sorry, not today";`,
      tests: [['canRide === true', '`canRide` is `true`'], ['reason === "Enjoy the ride!"', '`reason` is `"Enjoy the ride!"`'], ['!$src.includes("if")', 'No `if` statements yet']],
    },
  ],
});
