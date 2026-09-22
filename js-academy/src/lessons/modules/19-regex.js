registerModule({
  id: 'regex',
  title: 'Regular Expressions',
  icon: '🔍',
  lessons: [
    {
      t: 'Patterns in Text',
      b: `A **regular expression** (regex) describes a *pattern* of text. Write one between slashes, and use \`test\` to check whether a string contains a match:

\`\`\`js
/cat/.test("concatenate")   // true
/dog/.test("hot diggity")   // false
\`\`\`

So far this is just like \`includes\` — but patterns can do far more.`,
      code: `// Create hasError: whether /error/ matches the log line below\nconst line = "2024-05-01 error: disk full";\n`,
      sol: `const line = "2024-05-01 error: disk full";\nconst hasError = /error/.test(line);`,
      tests: [['hasError === true', '`hasError` is `true`'], ['$src.includes("/error/")', 'Use the regex `/error/`']],
    },
    {
      t: 'Ignoring Case',
      b: `**Flags** go after the closing slash. The \`i\` flag makes a regex case-insensitive:

\`\`\`js
/hello/i.test("HeLLo there")   // true
\`\`\``,
      code: `// Write isGreeting(s) returning true if s contains "hello" in any case\n`,
      sol: `const isGreeting = (s) => /hello/i.test(s);`,
      tests: [['isGreeting("Oh HELLO") === true && isGreeting("hello") === true', 'Matches any case'], ['isGreeting("help") === false', '`"help"` does not match']],
    },
    {
      t: 'Character Classes',
      b: `Square brackets match **any one** of the characters inside:

\`\`\`js
/gr[ae]y/.test("grey")   // true
/gr[ae]y/.test("gray")   // true
/[aeiou]/.test("rhythm") // false — no vowels!
\`\`\`

A \`^\` at the start of the brackets means "anything **except**": \`[^0-9]\` is any non-digit.`,
      code: `// Write hasVowel(word) using a character class\n`,
      sol: `const hasVowel = (word) => /[aeiou]/i.test(word);`,
      tests: [['hasVowel("sky") === false && hasVowel("Tree") === true', 'Detects vowels'], ['$src.includes("[aeiou]")', 'Use `[aeiou]`']],
    },
    {
      t: 'Ranges and Shortcuts',
      b: `Ranges save typing: \`[a-z]\`, \`[A-Z]\`, \`[0-9]\`. There are shortcuts for common classes:

- \`\\d\` a digit, \`\\D\` a non-digit
- \`\\w\` a "word" character (letter, digit or \`_\`)
- \`\\s\` whitespace (space, tab, newline)
- \`.\` any character except a newline`,
      code: `// Write hasDigit(s) using \\d and hasSpace(s) using \\s\n`,
      sol: `const hasDigit = (s) => /\\d/.test(s);\nconst hasSpace = (s) => /\\s/.test(s);`,
      tests: [['hasDigit("agent007") && !hasDigit("agent")', '`hasDigit` works'], ['hasSpace("a b") && !hasSpace("ab")', '`hasSpace` works']],
    },
    {
      t: 'Quantifiers',
      b: `Quantifiers say **how many** of the previous thing:

- \`+\` one or more
- \`*\` zero or more
- \`?\` zero or one (optional)

\`\`\`js
/colou?r/     // color or colour
/go+al/       // goal, gooal, gooooal
/\\d+/         // one or more digits
\`\`\``,
      code: `// Write isGoal(s) matching "goal" with ONE OR MORE o's: goal, gooal, goooooal …\n`,
      sol: `const isGoal = (s) => /^go+al$/.test(s);`,
      tests: [['isGoal("goal") && isGoal("gooooooal")', 'Matches any number of o\'s'], ['!isGoal("gal")', '"gal" does not match']],
    },
    {
      t: 'Exact Counts',
      b: `Curly braces give exact counts:

- \`{3}\` exactly 3
- \`{2,4}\` between 2 and 4
- \`{2,}\` 2 or more

\`\`\`js
/\\d{4}/   // four digits in a row
\`\`\``,
      code: `// Write hasYear(s): true if s contains 4 digits in a row\n`,
      sol: `const hasYear = (s) => /\\d{4}/.test(s);`,
      tests: [['hasYear("Born in 1990!") === true', 'Finds a year'], ['hasYear("Room 12") === false', 'Two digits is not a year'], ['$src.includes("{4}")', 'Use `{4}`']],
    },
    {
      t: 'Anchors',
      b: `\`^\` means "start of the string" and \`$\` means "end". Use both to match the **whole** string:

\`\`\`js
/^\\d+$/.test("12345")    // true — only digits
/^\\d+$/.test("123abc")   // false
\`\`\`

Without anchors, a regex is happy to match anywhere inside.`,
      code: `// Write isPin(s): true only if s is EXACTLY 4 digits\n`,
      sol: `const isPin = (s) => /^\\d{4}$/.test(s);`,
      tests: [['isPin("1234") === true', '"1234" is a PIN'], ['isPin("12345") === false && isPin("12a4") === false && isPin("x1234") === false', 'Anything else is not']],
    },
    {
      t: 'Validating Formats',
      b: `Combine everything to check a format. A UK-style time like \`09:30\`: two digits, a colon, two digits.

\`\`\`js
/^\\d{2}:\\d{2}$/
\`\`\`

(A real validator would also check the hours are below 24 — regex is best for *shape*, and code for *meaning*.)`,
      code: `// Write isHexColor(s): true for strings like "#a1b2c3" or "#FFF" (# followed by exactly 3 or 6 hex digits 0-9 a-f)\n`,
      sol: `const isHexColor = (s) => /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.test(s);`,
      tests: [['isHexColor("#a1b2c3") && isHexColor("#FFF")', 'Valid colours pass'], ['!isHexColor("#abcd") && !isHexColor("a1b2c3") && !isHexColor("#ggg")', 'Invalid ones fail']],
    },
    {
      t: 'Finding All Matches',
      b: `\`string.match(regex)\` with the \`g\` (global) flag returns an **array of every match** (or \`null\` if none):

\`\`\`js
"a1 b22 c333".match(/\\d+/g)   // ["1", "22", "333"]
\`\`\``,
      code: `const text = "Call 555-1234 or 555-9876, not 12.";\n// Create numbers: every run of 3 or more digits in text\n`,
      sol: `const text = "Call 555-1234 or 555-9876, not 12.";\nconst numbers = text.match(/\\d{3,}/g);`,
      tests: [['numbers.join() === "555,1234,555,9876"', '`numbers` is ["555", "1234", "555", "9876"]']],
    },
    {
      t: 'Capture Groups',
      b: `Parentheses **capture** parts of a match. Without the \`g\` flag, \`match\` returns the full match followed by each group:

\`\`\`js
const m = "2024-07-15".match(/(\\d{4})-(\\d{2})-(\\d{2})/);
m[1]  // "2024"
m[2]  // "07"
m[3]  // "15"
\`\`\``,
      code: `// Write parseTime(s) for strings like "14:05", returning { hours: 14, minutes: 5 } (numbers)\n`,
      sol: `function parseTime(s) {\n  const m = s.match(/^(\\d{1,2}):(\\d{2})$/);\n  return { hours: Number(m[1]), minutes: Number(m[2]) };\n}`,
      tests: [['JSON.stringify(parseTime("14:05")) === \'{"hours":14,"minutes":5}\'', '`parseTime("14:05")` works'], ['parseTime("9:30").hours === 9', 'One-digit hours work']],
    },
    {
      t: 'Named Groups',
      b: `Name your groups with \`(?<name>…)\` and read them from \`.groups\` — much clearer than numbers:

\`\`\`js
const m = "Ada Lovelace".match(/(?<first>\\w+) (?<last>\\w+)/);
m.groups.first   // "Ada"
m.groups.last    // "Lovelace"
\`\`\``,
      code: `// Write parseEmail(s) returning { user, domain } using named groups\n`,
      sol: `function parseEmail(s) {\n  const { user, domain } = s.match(/^(?<user>[^@\\s]+)@(?<domain>[^@\\s]+)$/).groups;\n  return { user, domain };\n}`,
      tests: [['JSON.stringify(parseEmail("grace@navy.mil")) === \'{"user":"grace","domain":"navy.mil"}\'', 'Splits the email'], ['$src.includes("?<")', 'Use named groups']],
    },
    {
      t: 'Replacing with Patterns',
      b: `\`replace\` accepts a regex. Add the \`g\` flag to replace **every** match:

\`\`\`js
"a  b    c".replace(/\\s+/g, " ")   // "a b c"
\`\`\``,
      code: `// Write tidy(s): collapse every run of whitespace into a single space, and trim the ends\n`,
      sol: `const tidy = (s) => s.replace(/\\s+/g, " ").trim();`,
      tests: [['tidy("  too    many   spaces ") === "too many spaces"', 'Collapses spaces'], ['tidy("tabs\\tand\\nnewlines") === "tabs and newlines"', 'Tabs and newlines too']],
    },
    {
      t: 'Rearranging with Groups',
      b: `In the replacement text, \`$1\`, \`$2\`… insert captured groups — great for reordering:

\`\`\`js
"Lovelace, Ada".replace(/(\\w+), (\\w+)/, "$2 $1")   // "Ada Lovelace"
\`\`\``,
      code: `// Write toUSDate(s) turning "DD/MM/YYYY" into "MM/DD/YYYY" using replace with $1, $2, $3\n`,
      sol: `const toUSDate = (s) => s.replace(/(\\d{2})\\/(\\d{2})\\/(\\d{4})/, "$2/$1/$3");`,
      tests: [['toUSDate("25/12/2024") === "12/25/2024"', 'Swaps day and month'], ['$src.includes("$2")', 'Use `$2`-style references']],
    },
    {
      t: 'Replacing with a Function',
      b: `For clever replacements, pass a **function**. It receives the match (and groups) and returns the replacement:

\`\`\`js
"3 cats and 4 dogs".replace(/\\d+/g, n => n * 2)
// "6 cats and 8 dogs"
\`\`\``,
      code: `// Write titleCase(s): capitalise the first letter of every word, using replace with a function\n`,
      sol: `const titleCase = (s) => s.replace(/\\b\\w/g, (c) => c.toUpperCase());`,
      tests: [['titleCase("the lord of the rings") === "The Lord Of The Rings"', 'Capitalises every word'], ['$src.includes("replace(")', 'Use `replace`']],
    },
    {
      t: 'Splitting with Patterns',
      b: `\`split\` accepts a regex too — handy when separators vary:

\`\`\`js
"a, b;c ; d".split(/\\s*[,;]\\s*/)   // ["a", "b", "c", "d"]
\`\`\``,
      code: `// Write parseList(s) splitting on commas, semicolons or the word "and", ignoring spaces around them\n`,
      sol: `const parseList = (s) => s.split(/\\s*(?:,|;|\\band\\b)\\s*/);`,
      tests: [['parseList("eggs, milk;bread and jam").join("|") === "eggs|milk|bread|jam"', 'Splits on all separators']],
    },
    {
      t: 'Alternatives',
      b: `\`|\` means "or". Group alternatives with parentheses:

\`\`\`js
/^(cat|dog|bird)$/.test("dog")   // true
/\\.(jpe?g|png|gif)$/i            // image file names
\`\`\``,
      code: `// Write isImage(filename): true if it ends with .jpg, .jpeg, .png or .gif (any case)\n`,
      sol: `const isImage = (filename) => /\\.(jpe?g|png|gif)$/i.test(filename);`,
      tests: [['isImage("cat.JPG") && isImage("a.jpeg") && isImage("icon.png")', 'Images match'], ['!isImage("notes.txt") && !isImage("png.doc")', 'Other files do not']],
    },
    {
      t: 'Escaping Special Characters',
      b: `Characters like \`. * + ? ( ) [ ] { } ^ $ | \\ /\` have special meanings. To match them literally, put a backslash before them:

\`\`\`js
/3.14/     // matches "3x14" too — . means "any character"!
/3\\.14/    // only "3.14"
\`\`\``,
      code: `// Write isPrice(s): true for strings like "$4.99" or "$120.00" — a dollar sign, digits, a dot, exactly two digits\n`,
      sol: `const isPrice = (s) => /^\\$\\d+\\.\\d{2}$/.test(s);`,
      tests: [['isPrice("$4.99") && isPrice("$120.00")', 'Prices match'], ['!isPrice("4.99") && !isPrice("$4x99") && !isPrice("$4.9")', 'Other strings do not']],
    },
    {
      t: 'Looping Over Matches',
      b: `\`matchAll\` (with the \`g\` flag) gives every match **with its groups**:

\`\`\`js
for (const m of "a=1, b=2".matchAll(/(\\w)=(\\d)/g)) {
  console.log(m[1], m[2]);
}
\`\`\``,
      code: `// Write parseQuery(s) turning "name=ada&lang=js" into { name: "ada", lang: "js" } using matchAll\n`,
      sol: `function parseQuery(s) {\n  const out = {};\n  for (const m of s.matchAll(/(\\w+)=(\\w*)/g)) out[m[1]] = m[2];\n  return out;\n}`,
      tests: [['JSON.stringify(parseQuery("name=ada&lang=js")) === \'{"name":"ada","lang":"js"}\'', 'Parses the query'], ['$src.includes("matchAll")', 'Use `matchAll`']],
    },
    {
      t: 'Word Boundaries',
      b: `\`\\b\` matches the **edge of a word** (between a word character and a non-word character). It stops "cat" matching inside "concatenate":

\`\`\`js
/\\bcat\\b/.test("the cat sat")    // true
/\\bcat\\b/.test("concatenate")    // false
\`\`\``,
      code: `// Write countWord(text, word) counting whole-word, case-insensitive matches\n// Hint: new RegExp("\\\\b" + word + "\\\\b", "gi") builds a regex from a string\n`,
      sol: `function countWord(text, word) {\n  const matches = text.match(new RegExp("\\\\b" + word + "\\\\b", "gi"));\n  return matches ? matches.length : 0;\n}`,
      tests: [['countWord("The cat and the other cat scattered", "cat") === 2', 'Counts whole words only'], ['countWord("The THE the", "the") === 3', 'Ignores case'], ['countWord("nothing here", "cat") === 0', 'Zero when missing']],
    },
    {
      t: 'Project: Tag Extractor',
      b: `Write \`analysePost(text)\` for social media posts, returning an object:

- \`hashtags\` — every \`#tag\` (letters, digits, \`_\`), lowercased, without \`#\`, no duplicates, in order
- \`mentions\` — every \`@name\`, without the \`@\`, in order
- \`links\` — every \`http://\` or \`https://\` link (up to the next whitespace)
- \`words\` — the number of words **not** counting tags, mentions or links`,
      code: `function analysePost(text) {\n}\n`,
      sol: `function analysePost(text) {\n  const hashtags = [...new Set((text.match(/#\\w+/g) ?? []).map((t) => t.slice(1).toLowerCase()))];\n  const mentions = (text.match(/@\\w+/g) ?? []).map((m) => m.slice(1));\n  const links = text.match(/https?:\\/\\/\\S+/g) ?? [];\n  const words = text\n    .split(/\\s+/)\n    .filter((w) => w && !/^[#@]\\w/.test(w) && !/^https?:\\/\\//.test(w)).length;\n  return { hashtags, mentions, links, words };\n}`,
      tests: [
        ['(() => { const r = analysePost("Loving #JavaScript and #coding with @ada! More at https://example.com/js #javascript"); return r.hashtags.join() === "javascript,coding"; })()', 'Finds unique lowercase hashtags'],
        ['(() => { const r = analysePost("Hi @ada and @bob_1"); return r.mentions.join() === "ada,bob_1"; })()', 'Finds mentions'],
        ['(() => { const r = analysePost("see http://a.io and https://b.org/x?y=1 now"); return r.links.join() === "http://a.io,https://b.org/x?y=1"; })()', 'Finds links'],
        ['analysePost("Loving #JavaScript and #coding with @ada! More at https://example.com/js").words === 5', 'Counts plain words only'],
      ],
    },
  ],
});
