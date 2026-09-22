registerModule({
  id: 'strings',
  title: 'Working with Strings',
  icon: '🔤',
  lessons: [
    {
      t: 'String Length',
      b: `Every string knows how many characters it has, through its \`.length\` **property**:

\`\`\`js
"hello".length        // 5
"hi there".length     // 8 — the space counts
"".length             // 0 — the empty string
\`\`\``,
      code: `const password = "correct horse";\n// Create size: the number of characters in password\n`,
      sol: `const password = "correct horse";\nconst size = password.length;`,
      tests: [['size === 13', '`size` should be `13`'], ['$src.includes(".length")', 'Use `.length`']],
    },
    {
      t: 'Picking Characters',
      b: `Each character has a position called an **index**, counting from **0**:

\`\`\`js
const word = "cat";
word[0]      // "c"
word[2]      // "t"
word.at(-1)  // "t" — .at() accepts negatives, counting from the end
\`\`\`

The last index is always \`length - 1\`.`,
      code: `const language = "JavaScript";\n// Create first (the first character) and last (the last character)\n`,
      sol: `const language = "JavaScript";\nconst first = language[0];\nconst last = language.at(-1);`,
      tests: [['first === "J"', '`first` is `"J"`'], ['last === "t"', '`last` is `"t"`'], ['!/["\']t["\']/.test($src)', 'Take the characters from `language`']],
    },
    {
      t: 'Changing Case',
      b: `\`toUpperCase()\` and \`toLowerCase()\` return a **new** string with the case changed:

\`\`\`js
"Hello".toUpperCase()  // "HELLO"
"Hello".toLowerCase()  // "hello"
\`\`\`

The parentheses \`()\` mean you are **calling** a method — asking the string to do something.`,
      code: `const shout = "stop";\nconst whisper = "PSST";\n// Create loud (shout in capitals) and quiet (whisper in lowercase)\n`,
      sol: `const shout = "stop";\nconst whisper = "PSST";\nconst loud = shout.toUpperCase();\nconst quiet = whisper.toLowerCase();`,
      tests: [['loud === "STOP"', '`loud` is `"STOP"`'], ['quiet === "psst"', '`quiet` is `"psst"`']],
    },
    {
      t: 'Trimming Spaces',
      b: `Text typed by people often has stray spaces. \`trim()\` removes whitespace from both ends (not the middle):

\`\`\`js
"   hi there  ".trim()     // "hi there"
"  x ".trimStart()         // "x "
"  x ".trimEnd()           // "  x"
\`\`\``,
      code: `const typed = "    ada@example.com   ";\n// Create email: typed without the surrounding spaces\n`,
      sol: `const typed = "    ada@example.com   ";\nconst email = typed.trim();`,
      tests: [['email === "ada@example.com"', '`email` should have no surrounding spaces']],
    },
    {
      t: 'Searching with includes',
      b: `\`includes\` answers "does this string contain that text?" with \`true\` or \`false\`:

\`\`\`js
"I love pizza".includes("pizza")   // true
"I love pizza".includes("Pizza")   // false — case-sensitive!
\`\`\``,
      code: `const message = "Meet me at the Secret Base at noon";\n// Create hasSecret: whether message includes "secret", ignoring case\n`,
      sol: `const message = "Meet me at the Secret Base at noon";\nconst hasSecret = message.toLowerCase().includes("secret");`,
      tests: [['hasSecret === true', '`hasSecret` should be `true`'], ['$src.includes("includes")', 'Use `includes`']],
      hint: 'Lowercase the message first, then check: `message.toLowerCase().includes("secret")`.',
    },
    {
      t: 'Starts and Ends',
      b: `\`startsWith\` and \`endsWith\` check the beginning and end of a string:

\`\`\`js
const file = "photo.png";
file.endsWith(".png")      // true
file.startsWith("photo")   // true
\`\`\``,
      code: `const url = "https://example.com/report.pdf";\n// Create secure (url starts with "https://") and isPdf (url ends with ".pdf")\n`,
      sol: `const url = "https://example.com/report.pdf";\nconst secure = url.startsWith("https://");\nconst isPdf = url.endsWith(".pdf");`,
      tests: [['secure === true', '`secure` is `true`'], ['isPdf === true', '`isPdf` is `true`'], ['$src.includes("startsWith") && $src.includes("endsWith")', 'Use `startsWith` and `endsWith`']],
    },
    {
      t: 'Finding Positions',
      b: `\`indexOf\` returns the index where some text first appears, or \`-1\` if it is not there:

\`\`\`js
"banana".indexOf("n")      // 2
"banana".lastIndexOf("n")  // 4
"banana".indexOf("z")      // -1
\`\`\``,
      code: `const email = "grace@navy.mil";\n// Create at: the position of "@" in email\n// Create missing: the result of searching email for "#"\n`,
      sol: `const email = "grace@navy.mil";\nconst at = email.indexOf("@");\nconst missing = email.indexOf("#");`,
      tests: [['at === 5', '`at` is `5`'], ['missing === -1', '`missing` is `-1`']],
    },
    {
      t: 'Slicing Strings',
      b: `\`slice(start, end)\` copies part of a string. It includes \`start\` but **stops before** \`end\`:

\`\`\`js
const s = "JavaScript";
s.slice(0, 4)   // "Java"
s.slice(4)      // "Script" — to the end
s.slice(-3)     // "ipt"    — negatives count from the end
\`\`\``,
      code: `const email = "grace@navy.mil";\nconst at = email.indexOf("@");\n// Create user (everything before the @) and domain (everything after it)\n`,
      sol: `const email = "grace@navy.mil";\nconst at = email.indexOf("@");\nconst user = email.slice(0, at);\nconst domain = email.slice(at + 1);`,
      tests: [['user === "grace"', '`user` is `"grace"`'], ['domain === "navy.mil"', '`domain` is `"navy.mil"`'], ['$src.includes("slice")', 'Use `slice`']],
    },
    {
      t: 'Slicing from the End',
      b: `Negative numbers in \`slice\` count backwards from the end — perfect for "the last few characters":

\`\`\`js
const card = "4111 1111 1111 1234";
card.slice(-4)       // "1234"
card.slice(0, -5)    // everything except the last 5
\`\`\``,
      code: `const cardNumber = "4000 1234 5678 9010";\n// Create masked: "**** " followed by the last 4 characters\n`,
      sol: `const cardNumber = "4000 1234 5678 9010";\nconst masked = "**** " + cardNumber.slice(-4);`,
      tests: [['masked === "**** 9010"', '`masked` is `"**** 9010"`'], ['$src.includes("slice(-4)")', 'Use `slice(-4)`']],
    },
    {
      t: 'Replacing Text',
      b: `\`replace\` swaps the **first** match; \`replaceAll\` swaps **every** match:

\`\`\`js
"a-b-c".replace("-", "+")      // "a+b-c"
"a-b-c".replaceAll("-", "+")   // "a+b+c"
\`\`\`

Like every string method, they return a new string and leave the original alone.`,
      code: `const date = "2024-01-15";\n// Create slashed: date with every "-" changed to "/"\n`,
      sol: `const date = "2024-01-15";\nconst slashed = date.replaceAll("-", "/");`,
      tests: [['slashed === "2024/01/15"', '`slashed` is `"2024/01/15"`']],
    },
    {
      t: 'Splitting into Pieces',
      b: `\`split\` cuts a string into an **array** of pieces wherever a separator appears:

\`\`\`js
"red,green,blue".split(",")   // ["red", "green", "blue"]
"one two".split(" ")          // ["one", "two"]
"abc".split("")               // ["a", "b", "c"]
\`\`\`

We will learn all about arrays soon — for now, know that \`.length\` works on them too.`,
      code: `const sentence = "the quick brown fox jumps";\n// Create words: sentence split into words\n// Create wordCount: how many words there are\n`,
      sol: `const sentence = "the quick brown fox jumps";\nconst words = sentence.split(" ");\nconst wordCount = words.length;`,
      tests: [['Array.isArray(words) && words[1] === "quick"', '`words` is an array of the words'], ['wordCount === 5', '`wordCount` is `5`']],
    },
    {
      t: 'Repeating Strings',
      b: `\`repeat(n)\` glues \`n\` copies of a string together:

\`\`\`js
"ha".repeat(3)    // "hahaha"
"=".repeat(10)    // "=========="
\`\`\`

Great for drawing lines and simple bar charts in the console.`,
      code: `const score = 7;\n// Create bar: the "#" character repeated score times\n`,
      sol: `const score = 7;\nconst bar = "#".repeat(score);`,
      tests: [['bar === "#######"', '`bar` is seven `#`'], ['$src.includes("repeat(score)")', 'Use `repeat(score)`']],
    },
    {
      t: 'Padding',
      b: `\`padStart\` and \`padEnd\` add characters until the string reaches a target length:

\`\`\`js
"7".padStart(3, "0")    // "007"
"Hi".padEnd(6, ".")     // "Hi...."
\`\`\`

They are handy for clocks, IDs and lining up columns.`,
      code: `const minutes = 5;\nconst seconds = 9;\n// Create clock: "05:09" — each part padded to 2 digits with "0"\n`,
      sol: 'const minutes = 5;\nconst seconds = 9;\nconst clock = `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;',
      tests: [['clock === "05:09"', '`clock` is `"05:09"`'], ['$src.includes("padStart")', 'Use `padStart`']],
      hint: '`padStart` is a string method, so turn the numbers into strings first with `String(minutes)`.',
    },
    {
      t: 'Multi-line Strings',
      b: `Template literals (backticks) can span several lines, keeping the line breaks:

\`\`\`js
const poem = \`Roses are red,
Violets are blue\`;
\`\`\``,
      code: `// Create haiku: these three lines in ONE template literal\n//   An old silent pond\n//   A frog jumps into the pond\n//   Splash! Silence again\n`,
      sol: 'const haiku = `An old silent pond\nA frog jumps into the pond\nSplash! Silence again`;',
      tests: [['haiku === "An old silent pond\\nA frog jumps into the pond\\nSplash! Silence again"', '`haiku` holds the three lines'], ['$src.includes("`")', 'Use a template literal']],
    },
    {
      t: 'Expressions in Templates',
      b: `Anything that produces a value can go inside \`\${ }\` — maths, method calls, even choices:

\`\`\`js
const items = 3;
\`You have \${items} item\${items === 1 ? "" : "s"}\`  // "You have 3 items"
\`Total: \${(19.99 * 2).toFixed(2)}\`               // "Total: 39.98"
\`\`\``,
      code: `const name = "maya";\nconst age = 11;\n// Print "MAYA will be 12 next year" — compute both parts inside \${}\n`,
      sol: 'const name = "maya";\nconst age = 11;\nconsole.log(`${name.toUpperCase()} will be ${age + 1} next year`);',
      tests: [['$out() === "MAYA will be 12 next year"', 'Print `MAYA will be 12 next year`'], ['$src.includes("toUpperCase") && $src.includes("age + 1")', 'Compute with `toUpperCase()` and `age + 1`']],
    },
    {
      t: 'Strings Cannot Be Changed',
      b: `Strings are **immutable**: you cannot change a character inside one.

\`\`\`js
let word = "cat";
word[0] = "b";          // silently does nothing
console.log(word);      // "cat"
\`\`\`

Instead, build a **new** string and assign it:

\`\`\`js
word = "b" + word.slice(1);   // "bat"
\`\`\``,
      code: `let animal = "mouse";\nanimal[0] = "h";\n// That line does nothing. Fix it so animal becomes "house"\nconsole.log(animal);\n`,
      sol: `let animal = "mouse";\nanimal = "h" + animal.slice(1);\nconsole.log(animal);`,
      tests: [['animal === "house"', '`animal` should be `"house"`'], ['$src.includes("slice")', 'Build the new string with `slice`']],
    },
    {
      t: 'Comparing Strings',
      b: `\`===\` compares strings exactly, including case. To compare ignoring case, convert both sides first:

\`\`\`js
"Yes" === "yes"                                  // false
"Yes".toLowerCase() === "yes".toLowerCase()      // true
\`\`\`

\`<\` and \`>\` compare alphabetically (by character code): \`"apple" < "banana"\` is \`true\`.`,
      code: `const answer = "  PaRiS ";\nconst correct = "paris";\n// Create isRight: whether answer matches correct, ignoring case and outer spaces\n`,
      sol: `const answer = "  PaRiS ";\nconst correct = "paris";\nconst isRight = answer.trim().toLowerCase() === correct;`,
      tests: [['isRight === true', '`isRight` should be `true`'], ['$src.includes("trim") && $src.includes("toLowerCase")', 'Use `trim` and `toLowerCase`']],
    },
    {
      t: 'Capitalising a Word',
      b: `Combine what you know: take the first character, make it upper case, and add the rest:

\`\`\`js
const w = "hello";
w[0].toUpperCase() + w.slice(1)   // "Hello"
\`\`\``,
      code: `const city = "tokyo";\n// Create nice: city with a capital first letter\n`,
      sol: `const city = "tokyo";\nconst nice = city[0].toUpperCase() + city.slice(1);`,
      tests: [['nice === "Tokyo"', '`nice` is `"Tokyo"`']],
    },
    {
      t: 'Character Codes',
      b: `Computers store characters as numbers. \`charCodeAt\` gives the number, and \`String.fromCharCode\` goes back:

\`\`\`js
"A".charCodeAt(0)          // 65
String.fromCharCode(66)    // "B"
\`\`\`

Letters are in order, so adding 1 to a code gives the next letter — the heart of simple secret codes.`,
      code: `const letter = "H";\n// Create code: the character code of letter\n// Create next: the letter that comes after it\n`,
      sol: `const letter = "H";\nconst code = letter.charCodeAt(0);\nconst next = String.fromCharCode(code + 1);`,
      tests: [['code === 72', '`code` is `72`'], ['next === "I"', '`next` is `"I"`'], ['$src.includes("fromCharCode")', 'Use `String.fromCharCode`']],
    },
    {
      t: 'Project: Username Maker',
      b: `A website makes usernames from a full name and a birth year like this:

- first name's first letter, **lower case**
- the whole last name, **lower case**
- the last **two** digits of the year

So \`"Katherine Johnson"\` born \`1918\` becomes \`kjohnson18\`.

Create \`username\` from \`fullName\` and \`year\`, and \`initials\` (like \`"KJ"\`). It should work for any two-word name.`,
      code: `const fullName = "Katherine Johnson";\nconst year = 1918;\n`,
      sol: 'const fullName = "Katherine Johnson";\nconst year = 1918;\nconst [first, last] = fullName.split(" ");\nconst username = (first[0] + last).toLowerCase() + String(year).slice(-2);\nconst initials = (first[0] + last[0]).toUpperCase();',
      tests: [['username === "kjohnson18"', '`username` is `"kjohnson18"`'], ['initials === "KJ"', '`initials` is `"KJ"`'], ['!$src.includes("kjohnson")', 'Build it from the variables']],
      hint: '`fullName.split(" ")` gives you both names. `String(year).slice(-2)` gives the last two digits.',
    },
  ],
});
