registerModule({
  id: 'datesjson',
  title: 'Dates & JSON',
  icon: '📅',
  lessons: [
    {
      t: 'Creating Dates',
      b: `\`new Date()\` is right now. You can also build a specific date from its parts:

\`\`\`js
const moonLanding = new Date(1969, 6, 20);   // 20 July 1969
\`\`\`

⚠️ **Months count from 0**: January is 0, December is 11. Days count from 1 as normal. This trips up everyone!`,
      code: `// Create newYearsEve: 31 December 2025\n// Create birthday: 14 March 2012\n`,
      sol: `const newYearsEve = new Date(2025, 11, 31);\nconst birthday = new Date(2012, 2, 14);`,
      tests: [['newYearsEve.getFullYear() === 2025 && newYearsEve.getMonth() === 11 && newYearsEve.getDate() === 31', '`newYearsEve` is 31 Dec 2025'], ['birthday.getMonth() === 2 && birthday.getDate() === 14', '`birthday` is 14 March 2012']],
    },
    {
      t: 'Reading Date Parts',
      b: `Getters pull a date apart:

\`\`\`js
const d = new Date(2024, 1, 29);
d.getFullYear()  // 2024
d.getMonth()     // 1 (February!)
d.getDate()      // 29  — day of the month
d.getDay()       // 4   — day of the WEEK, 0 = Sunday
\`\`\``,
      code: `const launch = new Date(2030, 8, 5);\n// Create year, monthNumber (1-12, the human way!) and weekday (0-6)\n`,
      sol: `const launch = new Date(2030, 8, 5);\nconst year = launch.getFullYear();\nconst monthNumber = launch.getMonth() + 1;\nconst weekday = launch.getDay();`,
      tests: [['year === 2030', '`year` is `2030`'], ['monthNumber === 9', '`monthNumber` is `9`'], ['weekday === 4', '`weekday` is `4` (a Thursday)']],
    },
    {
      t: 'Day Names',
      b: `\`getDay()\` gives a number. Use an array to turn it into a name:

\`\`\`js
const days = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
days[date.getDay()]
\`\`\``,
      code: `// Write dayName(date) returning the full English day name, like "Saturday"\n`,
      sol: `const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];\nconst dayName = (date) => DAYS[date.getDay()];`,
      tests: [['dayName(new Date(2024, 0, 1)) === "Monday"', '1 Jan 2024 was a Monday'], ['dayName(new Date(2000, 0, 1)) === "Saturday"', '1 Jan 2000 was a Saturday']],
    },
    {
      t: 'Timestamps',
      b: `Inside, a date is just a number: **milliseconds since 1 January 1970** (UTC). \`getTime()\` gives it, and \`Date.now()\` gives the current one:

\`\`\`js
const start = Date.now();
// … do some work …
const took = Date.now() - start;   // milliseconds
\`\`\`

A day is \`24 * 60 * 60 * 1000\` = 86,400,000 ms.`,
      code: `const MS_PER_DAY = 24 * 60 * 60 * 1000;\n// Write msToDays(ms) returning how many WHOLE days are in ms\n`,
      sol: `const MS_PER_DAY = 24 * 60 * 60 * 1000;\nconst msToDays = (ms) => Math.floor(ms / MS_PER_DAY);`,
      tests: [['msToDays(MS_PER_DAY * 3) === 3', 'Three days'], ['msToDays(MS_PER_DAY * 2.9) === 2', 'Partial days are dropped']],
    },
    {
      t: 'Days Between Dates',
      b: `Subtracting two dates gives the difference in milliseconds. Divide to get days. Use \`Math.round\`, because daylight-saving changes can make a "day" 23 or 25 hours long!

\`\`\`js
const diff = later - earlier;   // ms
const days = Math.round(diff / 86400000);
\`\`\``,
      code: `// Write daysBetween(a, b) returning the whole number of days from a to b\n`,
      sol: `const daysBetween = (a, b) => Math.round((b - a) / 86400000);`,
      tests: [['daysBetween(new Date(2024, 0, 1), new Date(2024, 11, 25)) === 359', 'New Year to Christmas 2024 is 359 days'], ['daysBetween(new Date(2024, 2, 1), new Date(2024, 1, 28)) === -2', 'Negative when b is earlier']],
    },
    {
      t: 'Adding Days',
      b: `\`setDate\` changes the day of the month — and it **rolls over** months and years automatically:

\`\`\`js
const d = new Date(2024, 0, 30);
d.setDate(d.getDate() + 5);   // 4 February 2024
\`\`\`

Copy the date first (\`new Date(d)\`) if you do not want to change the original.`,
      code: `// Write addDays(date, n) returning a NEW date n days later (without changing date)\n`,
      sol: `function addDays(date, n) {\n  const copy = new Date(date);\n  copy.setDate(copy.getDate() + n);\n  return copy;\n}`,
      tests: [['(() => { const d = addDays(new Date(2024, 0, 30), 5); return d.getMonth() === 1 && d.getDate() === 4; })()', 'Rolls over into February'], ['(() => { const d = addDays(new Date(2023, 11, 31), 1); return d.getFullYear() === 2024 && d.getDate() === 1; })()', 'Rolls over the year'], ['(() => { const orig = new Date(2024, 5, 1); addDays(orig, 10); return orig.getDate() === 1; })()', 'The original is unchanged']],
    },
    {
      t: 'Formatting Dates',
      b: `A common format is ISO-style \`YYYY-MM-DD\`. Build it with \`padStart\` so single digits get a leading zero:

\`\`\`js
String(7).padStart(2, "0")   // "07"
\`\`\``,
      code: `// Write formatDate(date) returning "YYYY-MM-DD" using the date's local parts\n`,
      sol: 'function formatDate(date) {\n  const y = date.getFullYear();\n  const m = String(date.getMonth() + 1).padStart(2, "0");\n  const d = String(date.getDate()).padStart(2, "0");\n  return `${y}-${m}-${d}`;\n}',
      tests: [['formatDate(new Date(2024, 2, 7)) === "2024-03-07"', '7 March 2024 → "2024-03-07"'], ['formatDate(new Date(1999, 11, 31)) === "1999-12-31"', '"1999-12-31"']],
    },
    {
      t: 'Friendly Date Formats',
      b: `\`Intl.DateTimeFormat\` formats dates for any language and style:

\`\`\`js
const fmt = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "long", year: "numeric" });
fmt.format(new Date(2024, 6, 4));   // "4 July 2024"
\`\`\`

Try \`"de-DE"\`, \`"ja-JP"\` or \`"en-US"\` to see the differences.`,
      code: `// Write longDate(date) using Intl.DateTimeFormat("en-US", { month: "long", day: "numeric", year: "numeric" })\n`,
      sol: `const longDate = (date) => new Intl.DateTimeFormat("en-US", { month: "long", day: "numeric", year: "numeric" }).format(date);`,
      tests: [['longDate(new Date(2024, 6, 4)) === "July 4, 2024"', '"July 4, 2024"'], ['$src.includes("Intl.DateTimeFormat")', 'Use `Intl.DateTimeFormat`']],
    },
    {
      t: 'How Old?',
      b: `Age is the difference in years — minus one if this year's birthday has not happened yet. Compare months first, then days.`,
      code: `// Write ageOn(birth, today) returning the age in whole years\n`,
      sol: `function ageOn(birth, today) {\n  let age = today.getFullYear() - birth.getFullYear();\n  const beforeBirthday =\n    today.getMonth() < birth.getMonth() ||\n    (today.getMonth() === birth.getMonth() && today.getDate() < birth.getDate());\n  if (beforeBirthday) age--;\n  return age;\n}`,
      tests: [['ageOn(new Date(2010, 5, 15), new Date(2024, 5, 14)) === 13', 'The day before the birthday'], ['ageOn(new Date(2010, 5, 15), new Date(2024, 5, 15)) === 14', 'On the birthday'], ['ageOn(new Date(2010, 11, 1), new Date(2024, 0, 1)) === 13', 'Early in the year']],
    },
    {
      t: 'Comparing Dates',
      b: `\`<\` and \`>\` work on dates (they compare timestamps). But \`===\` compares **objects**, so two equal dates are not \`===\`! Compare \`getTime()\` instead:

\`\`\`js
new Date(2024, 0, 1) === new Date(2024, 0, 1)                    // false
new Date(2024, 0, 1).getTime() === new Date(2024, 0, 1).getTime() // true
\`\`\``,
      code: `// Write sameDay(a, b): true if both dates fall on the same calendar day (any time)\n// Write isPast(date, now): true if date is before now\n`,
      sol: `const sameDay = (a, b) =>\n  a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();\nconst isPast = (date, now) => date < now;`,
      tests: [['sameDay(new Date(2024, 4, 1, 9), new Date(2024, 4, 1, 22)) === true', 'Same day, different times'], ['sameDay(new Date(2024, 4, 1), new Date(2024, 4, 2)) === false', 'Different days'], ['isPast(new Date(2020, 0, 1), new Date(2024, 0, 1)) && !isPast(new Date(2030, 0, 1), new Date(2024, 0, 1))', '`isPast` works']],
    },
    {
      t: 'Sorting Dates',
      b: `Because subtracting dates gives a number, sorting them is easy:

\`\`\`js
dates.sort((a, b) => a - b);   // earliest first
\`\`\``,
      code: `const events = [\n  { name: "Concert", when: new Date(2025, 4, 20) },\n  { name: "Dentist", when: new Date(2025, 1, 3) },\n  { name: "Holiday", when: new Date(2025, 7, 10) },\n];\n// Create order: the event names from soonest to latest\n`,
      sol: `const events = [\n  { name: "Concert", when: new Date(2025, 4, 20) },\n  { name: "Dentist", when: new Date(2025, 1, 3) },\n  { name: "Holiday", when: new Date(2025, 7, 10) },\n];\nconst order = events.toSorted((a, b) => a.when - b.when).map((e) => e.name);`,
      tests: [['order.join() === "Dentist,Concert,Holiday"', '`order` is Dentist, Concert, Holiday']],
    },
    {
      t: 'JSON: Saving Data as Text',
      b: `**JSON** (JavaScript Object Notation) is the world's favourite format for storing and sending data. \`JSON.stringify\` turns values into JSON text:

\`\`\`js
JSON.stringify({ name: "Ada", langs: ["js", "py"] })
// '{"name":"Ada","langs":["js","py"]}'
\`\`\`

Note that JSON keys always use double quotes.`,
      code: `const save = { level: 4, coins: 120, items: ["sword", "map"] };\n// Create text: save as a JSON string\n`,
      sol: `const save = { level: 4, coins: 120, items: ["sword", "map"] };\nconst text = JSON.stringify(save);`,
      tests: [['text === \'{"level":4,"coins":120,"items":["sword","map"]}\'', '`text` is the JSON string'], ['typeof text === "string"', 'It is a string']],
    },
    {
      t: 'JSON: Reading It Back',
      b: `\`JSON.parse\` turns JSON text back into real values:

\`\`\`js
const data = JSON.parse('{"x": 1, "ok": true}');
data.x    // 1
\`\`\`

Together, stringify and parse are how apps save data to files or \`localStorage\`, and talk to servers.`,
      code: `const fromServer = '{"user":{"name":"Lin","friends":["Mo","Jo"]},"online":true}';\n// Create data by parsing fromServer, then firstFriend from it\n`,
      sol: `const fromServer = '{"user":{"name":"Lin","friends":["Mo","Jo"]},"online":true}';\nconst data = JSON.parse(fromServer);\nconst firstFriend = data.user.friends[0];`,
      tests: [['data.online === true && data.user.name === "Lin"', '`data` is parsed'], ['firstFriend === "Mo"', '`firstFriend` is `"Mo"`']],
    },
    {
      t: 'Pretty JSON',
      b: `The third argument of \`JSON.stringify\` adds indentation, making the output readable:

\`\`\`js
JSON.stringify({ a: 1, b: [2] }, null, 2)
// {
//   "a": 1,
//   "b": [
//     2
//   ]
// }
\`\`\``,
      code: `const config = { theme: "dark", size: 14 };\n// Print config as JSON indented with 2 spaces\n`,
      sol: `const config = { theme: "dark", size: 14 };\nconsole.log(JSON.stringify(config, null, 2));`,
      tests: [['$out() === \'{\\n  "theme": "dark",\\n  "size": 14\\n}\'', 'Print the indented JSON']],
    },
    {
      t: 'What JSON Cannot Hold',
      b: `JSON only knows strings, numbers, booleans, \`null\`, arrays and plain objects. Other things change or vanish:

\`\`\`js
JSON.stringify({ f() {}, u: undefined, d: new Date(0), n: NaN })
// '{"d":"1970-01-01T00:00:00.000Z","n":null}'
\`\`\`

Functions and \`undefined\` disappear, dates become strings, and \`NaN\` becomes \`null\`. Predict what survives a round trip.`,
      code: `const original = { name: "Box", size: undefined, open() {}, made: new Date(0), tags: ["a"] };\nconst copy = JSON.parse(JSON.stringify(original));\n// Set each to true or false:\nconst keptName = null;   // is copy.name still "Box"?\nconst keptOpen = null;   // is copy.open still a function?\nconst madeIsDate = null; // is copy.made still a Date?\n`,
      sol: `const original = { name: "Box", size: undefined, open() {}, made: new Date(0), tags: ["a"] };\nconst copy = JSON.parse(JSON.stringify(original));\nconst keptName = true;\nconst keptOpen = false;\nconst madeIsDate = false;`,
      tests: [['keptName === (copy.name === "Box")', '`keptName` is right'], ['keptOpen === (typeof copy.open === "function")', '`keptOpen` is right'], ['madeIsDate === (copy.made instanceof Date)', '`madeIsDate` is right']],
    },
    {
      t: 'Reviving Dates',
      b: `\`JSON.parse\` takes an optional **reviver** function that can transform each value as it is read. Use it to turn date strings back into \`Date\` objects:

\`\`\`js
JSON.parse(text, (key, value) =>
  key === "created" ? new Date(value) : value);
\`\`\``,
      code: `// Write loadTask(text) parsing JSON, turning any key ending in "At" (like dueAt) into a Date\n`,
      sol: `function loadTask(text) {\n  return JSON.parse(text, (key, value) => (key.endsWith("At") ? new Date(value) : value));\n}`,
      tests: [['(() => { const t = loadTask(JSON.stringify({ title: "Pay rent", dueAt: new Date(2025, 0, 1) })); return t.dueAt instanceof Date && t.dueAt.getFullYear() === 2025 && t.title === "Pay rent"; })()', '`dueAt` becomes a Date']],
    },
    {
      t: 'Choosing What to Save',
      b: `\`JSON.stringify\`'s second argument (the **replacer**) can be an array of keys to keep, or a function:

\`\`\`js
JSON.stringify(user, ["name", "email"])   // only these keys
JSON.stringify(data, (k, v) => typeof v === "number" ? Math.round(v) : v)
\`\`\``,
      code: `const player = { name: "Kai", level: 12, password: "x", sessionToken: "y" };\n// Create safe: player as JSON with only name and level\n`,
      sol: `const player = { name: "Kai", level: 12, password: "x", sessionToken: "y" };\nconst safe = JSON.stringify(player, ["name", "level"]);`,
      tests: [['safe === \'{"name":"Kai","level":12}\'', '`safe` only has name and level']],
    },
    {
      t: 'Formatting Money',
      b: `\`Intl.NumberFormat\` formats numbers for any country and currency:

\`\`\`js
new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(1234.5)
// "$1,234.50"
new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" }).format(1234.5)
// "1.234,50 €"
\`\`\``,
      code: `// Write usd(amount) formatting as US dollars, and big(n) adding thousands separators ("en-US", no currency)\n`,
      sol: `const usd = (amount) => new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(amount);\nconst big = (n) => new Intl.NumberFormat("en-US").format(n);`,
      tests: [['usd(1234.5) === "$1,234.50"', '`usd(1234.5)` is "$1,234.50"'], ['big(9876543) === "9,876,543"', '`big(9876543)` is "9,876,543"']],
    },
    {
      t: 'Time Ago',
      b: `Apps show "5 minutes ago" instead of exact times. Work out the difference in seconds and pick the biggest sensible unit.`,
      code: `// Write timeAgo(then, now) returning:\n//   "just now" (under 60 s), "N minutes ago" (under 60 min), "N hours ago" (under 24 h), else "N days ago"\n// Use Math.floor, and "1 minute ago" / "1 hour ago" / "1 day ago" for exactly one\n`,
      sol: 'function timeAgo(then, now) {\n  const s = Math.floor((now - then) / 1000);\n  const plural = (n, unit) => `${n} ${unit}${n === 1 ? "" : "s"} ago`;\n  if (s < 60) return "just now";\n  const m = Math.floor(s / 60);\n  if (m < 60) return plural(m, "minute");\n  const h = Math.floor(m / 60);\n  if (h < 24) return plural(h, "hour");\n  return plural(Math.floor(h / 24), "day");\n}',
      tests: [['timeAgo(new Date(0), new Date(30 * 1000)) === "just now"', '30 seconds → "just now"'], ['timeAgo(new Date(0), new Date(61 * 1000)) === "1 minute ago"', '61 seconds → "1 minute ago"'], ['timeAgo(new Date(0), new Date(5 * 3600 * 1000)) === "5 hours ago"', '5 hours'], ['timeAgo(new Date(0), new Date(3 * 86400 * 1000)) === "3 days ago"', '3 days']],
    },
    {
      t: 'Project: Countdown Planner',
      b: `Write \`planner(events, today)\`. Each event is \`{ name, date }\` where \`date\` is a \`"YYYY-MM-DD"\` string.

Return an array of strings for the **upcoming** events (today or later), soonest first:

- \`"Party is today!"\`
- \`"Exam is tomorrow"\`
- \`"Holiday in 12 days"\`

Past events are left out. Parse the strings into **local** dates with \`new Date(y, m - 1, d)\` (not \`new Date("2025-01-01")\`, which uses UTC).`,
      code: `function planner(events, today) {\n}\n`,
      sol: 'function planner(events, today) {\n  const toDate = (s) => {\n    const [y, m, d] = s.split("-").map(Number);\n    return new Date(y, m - 1, d);\n  };\n  const start = new Date(today.getFullYear(), today.getMonth(), today.getDate());\n  return events\n    .map((e) => ({ name: e.name, days: Math.round((toDate(e.date) - start) / 86400000) }))\n    .filter((e) => e.days >= 0)\n    .sort((a, b) => a.days - b.days)\n    .map(({ name, days }) =>\n      days === 0 ? `${name} is today!` : days === 1 ? `${name} is tomorrow` : `${name} in ${days} days`,\n    );\n}',
      tests: [['(() => { const r = planner([{ name: "Holiday", date: "2025-07-22" }, { name: "Party", date: "2025-07-10" }, { name: "Exam", date: "2025-07-11" }, { name: "Old", date: "2025-06-01" }], new Date(2025, 6, 10, 15, 30)); return r.join("|") === "Party is today!|Exam is tomorrow|Holiday in 12 days"; })()', 'Lists upcoming events, soonest first'], ['planner([], new Date()).length === 0', 'No events → empty list']],
    },
  ],
});
