# JS Academy

A desktop app for learning JavaScript from scratch, packaged as an AppImage.

- **500 lessons** in 25 modules, from `console.log("Hello, World!")` up to
  closures, classes, generators, async/await, the DOM and classic algorithms.
  Each lesson explains one idea, gives you a task, and checks your code the
  moment you press **Run & check** (Ctrl+Enter). Hints and example solutions
  are there when you are stuck, and your progress and code are saved.
- **A playground** where you make your own things with HTML, CSS and
  JavaScript, with a live preview and console. Start from a blank page or from
  one of the templates: click counter, to-do list, canvas game, drawing pad or
  quiz.
- **Pack as AppImage.** Any playground project can be turned into its own
  single-file Linux app with one click. Double-click the file to run it, or
  send it to a friend.

## Running it

Download `JS-Academy-<version>-x86_64.AppImage` (see *Building* below, or the
`js-academy-appimage` artifact from the repository's CI), then:

```console
$ chmod +x JS-Academy-1.0.0-x86_64.AppImage
$ ./JS-Academy-1.0.0-x86_64.AppImage
```

Progress and projects are kept in `~/.config/JS Academy/`.

## The modules

| # | Module | # | Module |
|---|---|---|---|
| 1 | Getting Started | 14 | Recursion |
| 2 | Variables & Types | 15 | Errors & Debugging |
| 3 | Numbers & Math | 16 | Classes & Objects |
| 4 | Working with Strings | 17 | Maps, Sets & Symbols |
| 5 | Booleans & Logic | 18 | Iterators & Generators |
| 6 | Making Decisions | 19 | Regular Expressions |
| 7 | Loops | 20 | Dates & JSON |
| 8 | Functions | 21 | Asynchronous JavaScript |
| 9 | Arrays | 22 | Functional Programming |
| 10 | Array Superpowers | 23 | The DOM: Changing Web Pages |
| 11 | Objects | 24 | Events & Interactive Pages |
| 12 | Scope & Closures | 25 | Algorithms & Challenges |
| 13 | Modern Syntax | | |

Each module has 20 lessons and ends with a small project.

## How packing works

A packed project needs a browser engine to show its page. The one JS Academy
already runs on is the obvious choice, so the packer copies the Electron
runtime it was started from, swaps JS Academy's own code for a tiny app that
opens your project's page, and builds the AppImage with the bundled
`appimagetool`. Nothing is downloaded, so packing works offline. A packed app
is about 110 MB, because it carries its own copy of Chromium.

Packed apps store `localStorage` per app, so something like the to-do
template remembers its data between runs.

## Building

Needs Node.js 20 or newer, on x86-64 Linux.

```console
$ cd js-academy
$ npm install
$ npm start          # run from source
$ npm test           # check all 500 lessons, then pack and launch a test AppImage
$ npm run dist       # build dist/JS-Academy-<version>-x86_64.AppImage
```

`npm run dist` downloads `appimagetool` and the AppImage runtime into
`build-tools/` the first time. They are bundled into the app, and that is what
the **Pack as AppImage** button uses. To use packing when running from source
(`npm start`), run `npm run fetch-tools` once.

## Writing lessons

Lessons live in `src/lessons/modules/`, one file per module. A lesson looks
like this:

```js
{
  t: 'Returning a Value',                 // title
  b: `Markdown explanation…`,             // body
  code: `// starting code\n`,             // what the editor starts with
  sol: `function square(n) { return n * n; }`,
  tests: [['square(4) === 16', '`square(4)` returns `16`']],
  hint: 'Optional hint',
  dom: '<p id="x"></p>',                  // optional: run in a real page instead of a worker
  wait: 300,                              // optional: ms to wait for timers before checking
}
```

Each test is a JavaScript expression that runs in the same scope as the
learner's code, next to its message. Test expressions can use `$out()` (all
printed output), `$logs` (the printed lines) and `$src` (the learner's code).
`npm run validate` confirms that every solution passes its tests and that no
starting code already passes them all.
