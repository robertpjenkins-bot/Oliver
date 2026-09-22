registerModule({
  id: 'recursion',
  title: 'Recursion',
  icon: '🌀',
  lessons: [
    {
      t: 'A Function That Calls Itself',
      b: `A **recursive** function calls itself to solve a smaller version of the same problem.

\`\`\`js
function countdown(n) {
  if (n === 0) {
    console.log("Go!");
    return;
  }
  console.log(n);
  countdown(n - 1);   // the same job, one smaller
}
countdown(3);   // 3, 2, 1, Go!
\`\`\``,
      code: `// Write countUp(n) that RECURSIVELY prints 1 up to n (hint: call countUp(n - 1) first, then print n)\n// then call countUp(4)\n`,
      sol: `function countUp(n) {\n  if (n === 0) return;\n  countUp(n - 1);\n  console.log(n);\n}\ncountUp(4);`,
      tests: [['$out() === "1\\n2\\n3\\n4"', 'Print 1, 2, 3, 4'], ['!/for|while/.test($src)', 'No loops — use recursion']],
    },
    {
      t: 'The Base Case',
      b: `Every recursive function needs a **base case**: a situation simple enough to answer without recursing. Without one, the function calls itself forever until JavaScript gives up:

\`\`\`
RangeError: Maximum call stack size exceeded
\`\`\`

This function is missing its base case. Add one so that \`sumTo(0)\` is \`0\`.`,
      code: `function sumTo(n) {\n  return n + sumTo(n - 1);\n}\n`,
      sol: `function sumTo(n) {\n  if (n === 0) return 0;\n  return n + sumTo(n - 1);\n}`,
      tests: [['sumTo(0) === 0', '`sumTo(0)` is `0`'], ['sumTo(10) === 55', '`sumTo(10)` is `55`']],
    },
    {
      t: 'Recursive Factorial',
      b: `Factorial has a beautiful recursive definition:

- \`0! = 1\` (base case)
- \`n! = n × (n − 1)!\`

\`\`\`js
factorial(4) = 4 × factorial(3)
             = 4 × 3 × factorial(2)
             = 4 × 3 × 2 × factorial(1) …
\`\`\``,
      code: `// Write factorial(n) recursively\n`,
      sol: `function factorial(n) {\n  if (n <= 1) return 1;\n  return n * factorial(n - 1);\n}`,
      tests: [['factorial(0) === 1 && factorial(1) === 1', 'Base cases work'], ['factorial(6) === 720', '`factorial(6)` is `720`'], ['!/for|while/.test($src)', 'No loops']],
    },
    {
      t: 'Recursive Powers',
      b: `\`base ** exp\` can be defined recursively too:

- anything to the power 0 is 1
- \`base^exp = base × base^(exp − 1)\``,
      code: `// Write power(base, exp) recursively (no ** or Math.pow)\n`,
      sol: `function power(base, exp) {\n  if (exp === 0) return 1;\n  return base * power(base, exp - 1);\n}`,
      tests: [['power(2, 10) === 1024', '`power(2, 10)` is `1024`'], ['power(7, 0) === 1', '`power(7, 0)` is `1`'], ['!$src.includes("**") && !$src.includes("Math.pow")', 'Do not use `**` or `Math.pow`']],
    },
    {
      t: 'Fibonacci',
      b: `The Fibonacci sequence starts 0, 1, and each number is the sum of the two before: 0, 1, 1, 2, 3, 5, 8, 13…

- \`fib(0) = 0\`, \`fib(1) = 1\`
- \`fib(n) = fib(n − 1) + fib(n − 2)\`

This function calls itself **twice** — a tree of calls.`,
      code: `// Write fib(n) recursively\n`,
      sol: `function fib(n) {\n  if (n < 2) return n;\n  return fib(n - 1) + fib(n - 2);\n}`,
      tests: [['fib(0) === 0 && fib(1) === 1', 'Base cases work'], ['fib(10) === 55', '`fib(10)` is `55`'], ['fib(15) === 610', '`fib(15)` is `610`']],
    },
    {
      t: 'Summing a List',
      b: `Recursion on arrays: the sum of a list is **the first item plus the sum of the rest**. The empty list sums to 0.

\`\`\`js
const [first, ...rest] = list;
\`\`\``,
      code: `// Write sumList(list) recursively (no loops or reduce)\n`,
      sol: `function sumList(list) {\n  if (list.length === 0) return 0;\n  const [first, ...rest] = list;\n  return first + sumList(rest);\n}`,
      tests: [['sumList([]) === 0', 'Empty list is `0`'], ['sumList([4, 5, 6]) === 15', '`sumList([4, 5, 6])` is `15`'], ['!/for|while|reduce/.test($src)', 'No loops or `reduce`']],
    },
    {
      t: 'Reversing Recursively',
      b: `The reverse of a string is **the reverse of everything after the first letter**, followed by the first letter:

\`\`\`
reverse("abc") = reverse("bc") + "a"
               = (reverse("c") + "b") + "a"
               = "cba"
\`\`\``,
      code: `// Write reverse(s) recursively\n`,
      sol: `function reverse(s) {\n  if (s === "") return "";\n  return reverse(s.slice(1)) + s[0];\n}`,
      tests: [['reverse("hello") === "olleh"', '`reverse("hello")` is `"olleh"`'], ['reverse("") === ""', '`reverse("")` is `""`'], ['!/for|while|\\.reverse\\(/.test($src)', 'Use recursion, not loops or `.reverse()`']],
    },
    {
      t: 'Palindromes',
      b: `A palindrome reads the same backwards ("racecar"). Recursively:

- strings of length 0 or 1 are palindromes
- otherwise the first and last letters must match, **and** the middle must be a palindrome`,
      code: `// Write isPalindrome(s) recursively\n`,
      sol: `function isPalindrome(s) {\n  if (s.length <= 1) return true;\n  if (s[0] !== s.at(-1)) return false;\n  return isPalindrome(s.slice(1, -1));\n}`,
      tests: [['isPalindrome("racecar") === true', '`"racecar"` is a palindrome'], ['isPalindrome("abca") === false', '`"abca"` is not'], ['isPalindrome("") === true', '`""` is']],
    },
    {
      t: 'Building a Range',
      b: `Recursive functions can **build** things too, not just compute numbers. A range from 1 to n is the range from 1 to n − 1, with n added on the end.`,
      code: `// Write range(n) recursively, returning [1, 2, …, n] ([] for 0)\n`,
      sol: `function range(n) {\n  if (n === 0) return [];\n  return [...range(n - 1), n];\n}`,
      tests: [['range(0).length === 0', '`range(0)` is `[]`'], ['range(5).join() === "1,2,3,4,5"', '`range(5)` is [1, 2, 3, 4, 5]']],
    },
    {
      t: 'Adding Up Digits',
      b: `\`n % 10\` gives the last digit, and \`Math.floor(n / 10)\` removes it. So the digit sum of \`n\` is its last digit plus the digit sum of the rest.`,
      code: `// Write digitSum(n) recursively: digitSum(4096) is 4 + 0 + 9 + 6 = 19\n`,
      sol: `function digitSum(n) {\n  if (n < 10) return n;\n  return (n % 10) + digitSum(Math.floor(n / 10));\n}`,
      tests: [['digitSum(4096) === 19', '`digitSum(4096)` is `19`'], ['digitSum(7) === 7', '`digitSum(7)` is `7`']],
    },
    {
      t: 'Nested Arrays',
      b: `Recursion shines with **nested** data, where you do not know how deep things go:

\`\`\`js
[1, [2, [3, [4]]], 5]
\`\`\`

Loop over the items: numbers get added, arrays get handled by… calling yourself. \`Array.isArray(x)\` tells them apart.`,
      code: `// Write deepSum(arr) adding up all numbers at any depth\n`,
      sol: `function deepSum(arr) {\n  let total = 0;\n  for (const item of arr) {\n    total += Array.isArray(item) ? deepSum(item) : item;\n  }\n  return total;\n}`,
      tests: [['deepSum([1, [2, [3, [4]]], 5]) === 15', '`deepSum([1, [2, [3, [4]]], 5])` is `15`'], ['deepSum([]) === 0', 'Empty is `0`']],
    },
    {
      t: 'Flattening Any Depth',
      b: `The same idea can **flatten** nested arrays into one flat list (like \`.flat(Infinity)\` does).`,
      code: `// Write flatten(arr) returning all non-array items in order, without using .flat\n`,
      sol: `function flatten(arr) {\n  const out = [];\n  for (const item of arr) {\n    if (Array.isArray(item)) out.push(...flatten(item));\n    else out.push(item);\n  }\n  return out;\n}`,
      tests: [['flatten([1, [2, ["a", [true]]], 3]).join() === "1,2,a,true,3"', 'Flattens every level'], ['!$src.includes(".flat(")', 'Do not use `.flat`']],
    },
    {
      t: 'How Deep Is It?',
      b: `Find the **maximum depth** of nesting. A flat array has depth 1; each extra level of brackets adds one:

\`\`\`js
depth([1, 2])        // 1
depth([1, [2, [3]]]) // 3
\`\`\``,
      code: `// Write depth(arr)\n`,
      sol: `function depth(arr) {\n  let deepest = 0;\n  for (const item of arr) {\n    if (Array.isArray(item)) deepest = Math.max(deepest, depth(item));\n  }\n  return deepest + 1;\n}`,
      tests: [['depth([1, 2]) === 1', '`depth([1, 2])` is `1`'], ['depth([1, [2, [3]], [4]]) === 3', '`depth([1, [2, [3]], [4]])` is `3`'], ['depth([]) === 1', '`depth([])` is `1`']],
    },
    {
      t: 'Walking a Tree',
      b: `A **tree** is data where each node can have children, which have children… A family tree, a folder structure, a web page's elements.

\`\`\`js
const tree = { name: "Gran", children: [
  { name: "Mum", children: [{ name: "Me", children: [] }] },
  { name: "Uncle", children: [] },
]};
\`\`\`

To visit every node: handle this node, then recurse into each child.`,
      code: `const family = {\n  name: "Rosa",\n  children: [\n    { name: "Ali", children: [{ name: "Bea", children: [] }, { name: "Cy", children: [] }] },\n    { name: "Dee", children: [{ name: "Eli", children: [] }] },\n  ],\n};\n// Write allNames(node) returning every name in the tree (the node first, then its children in order)\n`,
      sol: `const family = {\n  name: "Rosa",\n  children: [\n    { name: "Ali", children: [{ name: "Bea", children: [] }, { name: "Cy", children: [] }] },\n    { name: "Dee", children: [{ name: "Eli", children: [] }] },\n  ],\n};\nfunction allNames(node) {\n  return [node.name, ...node.children.flatMap(allNames)];\n}`,
      tests: [['allNames(family).join() === "Rosa,Ali,Bea,Cy,Dee,Eli"', 'Lists every name in order']],
    },
    {
      t: 'Greatest Common Divisor',
      b: `Euclid's algorithm (from 300 BC!) finds the largest number dividing both \`a\` and \`b\`:

- if \`b\` is 0, the answer is \`a\`
- otherwise, the answer is \`gcd(b, a % b)\`

It is one of the oldest algorithms still in use.`,
      code: `// Write gcd(a, b) recursively\n`,
      sol: `function gcd(a, b) {\n  if (b === 0) return a;\n  return gcd(b, a % b);\n}`,
      tests: [['gcd(48, 18) === 6', '`gcd(48, 18)` is `6`'], ['gcd(17, 5) === 1', '`gcd(17, 5)` is `1`'], ['gcd(100, 75) === 25', '`gcd(100, 75)` is `25`']],
    },
    {
      t: 'Binary Search',
      b: `Searching a **sorted** array is fast if you keep halving it: look at the middle item. If it is your target, done. If the target is smaller, search the left half; otherwise the right half.

That is 20 steps to search a million items!`,
      code: `// Write search(arr, target, lo = 0, hi = arr.length - 1) recursively,\n// returning the index of target or -1\n`,
      sol: `function search(arr, target, lo = 0, hi = arr.length - 1) {\n  if (lo > hi) return -1;\n  const mid = Math.floor((lo + hi) / 2);\n  if (arr[mid] === target) return mid;\n  if (target < arr[mid]) return search(arr, target, lo, mid - 1);\n  return search(arr, target, mid + 1, hi);\n}`,
      tests: [['search([1, 3, 5, 7, 9, 11], 7) === 3', 'Finds 7 at index 3'], ['search([1, 3, 5, 7, 9, 11], 4) === -1', 'Returns -1 when missing'], ['search([2], 2) === 0', 'Works with one item'], ['!/indexOf|includes|findIndex/.test($src)', 'Do the search yourself']],
    },
    {
      t: 'Permutations',
      b: `All the orderings of "abc" are: abc, acb, bac, bca, cab, cba.

Recursive idea: for each letter, put it first, and append every permutation of the **remaining** letters.`,
      code: `// Write permutations(s) returning an array of all orderings of the characters\n`,
      sol: `function permutations(s) {\n  if (s.length <= 1) return [s];\n  const out = [];\n  for (let i = 0; i < s.length; i++) {\n    const rest = s.slice(0, i) + s.slice(i + 1);\n    for (const p of permutations(rest)) out.push(s[i] + p);\n  }\n  return out;\n}`,
      tests: [['permutations("abc").sort().join() === "abc,acb,bac,bca,cab,cba"', 'All 6 orderings of "abc"'], ['permutations("abcd").length === 24', '"abcd" has 24']],
    },
    {
      t: 'Faster Fibonacci',
      b: `The simple \`fib\` recomputes the same values again and again — \`fib(40)\` makes over 300 million calls! Store answers you have already worked out (memoization) and it becomes instant.`,
      code: `// Write fastFib(n, memo = {}) that caches results in memo\n`,
      sol: `function fastFib(n, memo = {}) {\n  if (n < 2) return n;\n  if (memo[n] !== undefined) return memo[n];\n  memo[n] = fastFib(n - 1, memo) + fastFib(n - 2, memo);\n  return memo[n];\n}`,
      tests: [['fastFib(10) === 55', '`fastFib(10)` is `55`'], ['fastFib(80) === 23416728348467685', '`fastFib(80)` finishes instantly']],
    },
    {
      t: 'Recursion or Loop?',
      b: `Anything recursive can be written with a loop, and vice versa. Loops are usually faster and cannot overflow the stack; recursion is often clearer for trees and nested data.

Write **both** versions of "count the characters \`ch\` in string \`s\`".`,
      code: `// countLoop(s, ch) — with a loop\n// countRec(s, ch) — recursively\n`,
      sol: `function countLoop(s, ch) {\n  let n = 0;\n  for (const c of s) if (c === ch) n++;\n  return n;\n}\nfunction countRec(s, ch) {\n  if (s === "") return 0;\n  return (s[0] === ch ? 1 : 0) + countRec(s.slice(1), ch);\n}`,
      tests: [['countLoop("banana", "a") === 3', '`countLoop` works'], ['countRec("banana", "a") === 3 && countRec("", "a") === 0', '`countRec` works']],
    },
    {
      t: 'Project: Folder Sizes',
      b: `A file system is a tree. Files look like \`{ name, size }\` and folders like \`{ name, children: [...] }\`.

Write:

- \`totalSize(node)\` — the size of a file, or the total of everything inside a folder
- \`findLargeFiles(node, min)\` — names of all files with size ≥ \`min\`, in the order they appear`,
      code: `const disk = {\n  name: "/",\n  children: [\n    { name: "notes.txt", size: 12 },\n    { name: "photos", children: [\n      { name: "cat.jpg", size: 400 },\n      { name: "old", children: [{ name: "dog.jpg", size: 350 }] },\n    ] },\n    { name: "game.exe", size: 900 },\n  ],\n};\n\nfunction totalSize(node) {\n}\n\nfunction findLargeFiles(node, min) {\n}\n`,
      sol: `const disk = {\n  name: "/",\n  children: [\n    { name: "notes.txt", size: 12 },\n    { name: "photos", children: [\n      { name: "cat.jpg", size: 400 },\n      { name: "old", children: [{ name: "dog.jpg", size: 350 }] },\n    ] },\n    { name: "game.exe", size: 900 },\n  ],\n};\n\nfunction totalSize(node) {\n  if (!node.children) return node.size;\n  return node.children.reduce((sum, child) => sum + totalSize(child), 0);\n}\n\nfunction findLargeFiles(node, min) {\n  if (!node.children) return node.size >= min ? [node.name] : [];\n  return node.children.flatMap((child) => findLargeFiles(child, min));\n}`,
      tests: [['totalSize(disk) === 1662', '`totalSize(disk)` is `1662`'], ['totalSize(disk.children[1]) === 750', 'The photos folder is `750`'], ['findLargeFiles(disk, 350).join() === "cat.jpg,dog.jpg,game.exe"', '`findLargeFiles(disk, 350)` is correct']],
    },
  ],
});
