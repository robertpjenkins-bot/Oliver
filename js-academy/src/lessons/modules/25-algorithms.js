registerModule({
  id: 'algorithms',
  title: 'Algorithms & Challenges',
  icon: '🏆',
  lessons: [
    {
      t: 'Linear Search',
      b: `An **algorithm** is a step-by-step recipe for solving a problem. The simplest search algorithm checks each item in turn until it finds a match.

Its running time grows **linearly** with the size of the list: twice the items, twice the work. We say it is **O(n)**.`,
      code: `// Write linearSearch(arr, target) returning the index of target, or -1 (no indexOf/includes/findIndex)\n`,
      sol: `function linearSearch(arr, target) {\n  for (let i = 0; i < arr.length; i++) {\n    if (arr[i] === target) return i;\n  }\n  return -1;\n}`,
      tests: [['linearSearch([5, 3, 8, 1], 8) === 2', 'Finds 8 at index 2'], ['linearSearch([5, 3], 9) === -1', 'Returns -1 when missing'], ['!/indexOf|includes|findIndex/.test($src)', 'Write the loop yourself']],
    },
    {
      t: 'Bubble Sort',
      b: `**Bubble sort** repeatedly walks through the list, swapping neighbours that are in the wrong order. After each pass, the biggest remaining item has "bubbled" to the end.

It is slow — **O(n²)** — but it is the classic first sorting algorithm.`,
      code: `// Write bubbleSort(arr) returning a NEW sorted array (ascending), without using .sort\n`,
      sol: `function bubbleSort(arr) {\n  const a = [...arr];\n  for (let end = a.length - 1; end > 0; end--) {\n    for (let i = 0; i < end; i++) {\n      if (a[i] > a[i + 1]) [a[i], a[i + 1]] = [a[i + 1], a[i]];\n    }\n  }\n  return a;\n}`,
      tests: [['bubbleSort([5, 2, 9, 1, 5, 6]).join() === "1,2,5,5,6,9"', 'Sorts numbers'], ['(() => { const x = [3, 1]; bubbleSort(x); return x.join() === "3,1"; })()', 'Leaves the input alone'], ['!$src.includes(".sort(")', 'Do not use `.sort`']],
    },
    {
      t: 'Selection Sort',
      b: `**Selection sort**: find the smallest item and swap it into position 0; then the smallest of the rest into position 1; and so on.

Also O(n²), but it makes at most n swaps — useful when swapping is expensive.`,
      code: `// Write selectionSort(arr) returning a new sorted array, without .sort\n`,
      sol: `function selectionSort(arr) {\n  const a = [...arr];\n  for (let i = 0; i < a.length; i++) {\n    let min = i;\n    for (let j = i + 1; j < a.length; j++) if (a[j] < a[min]) min = j;\n    [a[i], a[min]] = [a[min], a[i]];\n  }\n  return a;\n}`,
      tests: [['selectionSort([64, 25, 12, 22, 11]).join() === "11,12,22,25,64"', 'Sorts numbers'], ['selectionSort([]).length === 0', 'Handles empty arrays'], ['!$src.includes(".sort(")', 'Do not use `.sort`']],
    },
    {
      t: 'Insertion Sort',
      b: `**Insertion sort** works like sorting playing cards in your hand: take each new item and slide it left until it sits in the right place among the items already sorted.

It is very fast on lists that are *almost* sorted already.`,
      code: `// Write insertionSort(arr) returning a new sorted array, without .sort\n`,
      sol: `function insertionSort(arr) {\n  const a = [...arr];\n  for (let i = 1; i < a.length; i++) {\n    const item = a[i];\n    let j = i - 1;\n    while (j >= 0 && a[j] > item) {\n      a[j + 1] = a[j];\n      j--;\n    }\n    a[j + 1] = item;\n  }\n  return a;\n}`,
      tests: [['insertionSort([3, 7, 4, 9, 5, 2, 6, 1]).join() === "1,2,3,4,5,6,7,9"', 'Sorts numbers'], ['insertionSort(["pear", "apple", "fig"]).join() === "apple,fig,pear"', 'Sorts strings too'], ['!$src.includes(".sort(")', 'Do not use `.sort`']],
    },
    {
      t: 'Merge Sort',
      b: `**Merge sort** is a "divide and conquer" algorithm:

1. Split the list in half
2. Sort each half (recursively!)
3. **Merge** the two sorted halves by repeatedly taking the smaller front item

It runs in **O(n log n)** — enormously faster than O(n²) for big lists.`,
      code: `// Write merge(a, b) combining two sorted arrays into one sorted array,\n// then mergeSort(arr) using it\n`,
      sol: `function merge(a, b) {\n  const out = [];\n  let i = 0, j = 0;\n  while (i < a.length && j < b.length) out.push(a[i] <= b[j] ? a[i++] : b[j++]);\n  return [...out, ...a.slice(i), ...b.slice(j)];\n}\nfunction mergeSort(arr) {\n  if (arr.length <= 1) return arr;\n  const mid = Math.floor(arr.length / 2);\n  return merge(mergeSort(arr.slice(0, mid)), mergeSort(arr.slice(mid)));\n}`,
      tests: [['merge([1, 4, 9], [2, 3, 10]).join() === "1,2,3,4,9,10"', '`merge` works'], ['mergeSort([38, 27, 43, 3, 9, 82, 10]).join() === "3,9,10,27,38,43,82"', '`mergeSort` works'], ['(() => { const big = Array.from({ length: 2000 }, (_, i) => (i * 7919) % 2003); const s = mergeSort(big); return s.every((v, i) => i === 0 || s[i - 1] <= v); })()', 'Sorts 2000 items'], ['!$src.includes(".sort(")', 'Do not use `.sort`']],
    },
    {
      t: 'Quicksort',
      b: `**Quicksort** picks a **pivot**, splits the other items into "smaller than the pivot" and "not smaller", then sorts each side recursively:

\`\`\`js
[...quicksort(smaller), pivot, ...quicksort(bigger)]
\`\`\`

On average it is O(n log n), and in practice one of the fastest sorts.`,
      code: `// Write quicksort(arr) using the first item as the pivot, without .sort\n`,
      sol: `function quicksort(arr) {\n  if (arr.length <= 1) return arr;\n  const [pivot, ...rest] = arr;\n  const smaller = rest.filter((x) => x < pivot);\n  const bigger = rest.filter((x) => x >= pivot);\n  return [...quicksort(smaller), pivot, ...quicksort(bigger)];\n}`,
      tests: [['quicksort([10, 80, 30, 90, 40, 50, 70]).join() === "10,30,40,50,70,80,90"', 'Sorts numbers'], ['quicksort([2, 2, 1, 2]).join() === "1,2,2,2"', 'Handles duplicates'], ['!$src.includes(".sort(")', 'Do not use `.sort`']],
    },
    {
      t: 'Binary Search, Iteratively',
      b: `You wrote binary search recursively before. Here it is with a loop: keep \`lo\` and \`hi\` bounds and shrink them until the target is found or they cross.

Binary search is **O(log n)**: a billion items need only about 30 comparisons.`,
      code: `// Write binarySearch(sorted, target) with a while loop, returning the index or -1\n`,
      sol: `function binarySearch(sorted, target) {\n  let lo = 0;\n  let hi = sorted.length - 1;\n  while (lo <= hi) {\n    const mid = (lo + hi) >> 1;\n    if (sorted[mid] === target) return mid;\n    if (sorted[mid] < target) lo = mid + 1;\n    else hi = mid - 1;\n  }\n  return -1;\n}`,
      tests: [['binarySearch([2, 5, 8, 12, 16, 23, 38, 56, 72, 91], 23) === 5', 'Finds 23 at index 5'], ['binarySearch([2, 5, 8], 6) === -1', 'Returns -1 when missing'], ['/while/.test($src)', 'Use a loop']],
    },
    {
      t: 'Anagrams',
      b: `Two words are **anagrams** if they use exactly the same letters: *listen* and *silent*. Two approaches:

- sort the letters of both and compare — O(n log n)
- count the letters with an object or Map — O(n)`,
      code: `// Write isAnagram(a, b), ignoring case and spaces\n`,
      sol: `function isAnagram(a, b) {\n  const norm = (s) => [...s.toLowerCase().replace(/\\s/g, "")].sort().join("");\n  return norm(a) === norm(b);\n}`,
      tests: [['isAnagram("listen", "silent") === true', 'listen / silent'], ['isAnagram("Dormitory", "dirty room") === true', 'Ignores case and spaces'], ['isAnagram("abc", "abd") === false', 'abc / abd are not']],
    },
    {
      t: 'Caesar Cipher',
      b: `Julius Caesar hid messages by shifting each letter along the alphabet: with a shift of 3, A→D, B→E, … X→A. Wrap around with \`%\`, keep the case, and leave non-letters alone.

\`\`\`js
const code = ch.charCodeAt(0) - 65;          // A = 0
String.fromCharCode(((code + shift) % 26) + 65)
\`\`\``,
      code: `// Write caesar(text, shift). It should also work with negative shifts (to decode)\n`,
      sol: `function caesar(text, shift) {\n  const s = ((shift % 26) + 26) % 26;\n  return text.replace(/[a-z]/gi, (ch) => {\n    const base = ch <= "Z" ? 65 : 97;\n    return String.fromCharCode(((ch.charCodeAt(0) - base + s) % 26) + base);\n  });\n}`,
      tests: [['caesar("Hello, World!", 3) === "Khoor, Zruog!"', 'Encodes with shift 3'], ['caesar("xyz", 3) === "abc"', 'Wraps around'], ['caesar(caesar("Secret Plan", 7), -7) === "Secret Plan"', 'Negative shifts decode']],
    },
    {
      t: 'Run-Length Encoding',
      b: `**Run-length encoding** compresses repeated characters: \`"aaabccdddd"\` becomes \`"3a1b2c4d"\`. It is used in fax machines and some image formats.`,
      code: `// Write encode(s) and decode(s) (counts may have several digits, like "12a")\n`,
      sol: `function encode(s) {\n  let out = "";\n  for (let i = 0; i < s.length; ) {\n    let j = i;\n    while (s[j] === s[i]) j++;\n    out += (j - i) + s[i];\n    i = j;\n  }\n  return out;\n}\nfunction decode(s) {\n  return s.replace(/(\\d+)(\\D)/g, (_, n, ch) => ch.repeat(Number(n)));\n}`,
      tests: [['encode("aaabccdddd") === "3a1b2c4d"', '`encode` works'], ['decode("3a1b2c4d") === "aaabccdddd"', '`decode` works'], ['decode(encode("x".repeat(12) + "yy")) === "x".repeat(12) + "yy"', 'Round trip with 12 in a row'], ['encode("") === ""', 'Empty string']],
    },
    {
      t: 'Balanced Brackets',
      b: `Is \`"{[()()]}"\` balanced? A **stack** solves it: push each opening bracket; for each closing bracket, the top of the stack must be its partner. At the end the stack must be empty.

Code editors, compilers and calculators all do this.`,
      code: `// Write isBalanced(s) for (), [] and {} — other characters are ignored\n`,
      sol: `function isBalanced(s) {\n  const pairs = { ")": "(", "]": "[", "}": "{" };\n  const stack = [];\n  for (const ch of s) {\n    if ("([{".includes(ch)) stack.push(ch);\n    else if (ch in pairs) {\n      if (stack.pop() !== pairs[ch]) return false;\n    }\n  }\n  return stack.length === 0;\n}`,
      tests: [['isBalanced("{[()()]}") === true', 'Nested brackets'], ['isBalanced("([)]") === false', 'Crossed brackets'], ['isBalanced("((") === false', 'Unclosed brackets'], ['isBalanced("f(a[1]) { return x; }") === true', 'Ignores other characters']],
    },
    {
      t: 'Roman Numerals',
      b: `Convert numbers to Roman numerals. The trick is a table that includes the "subtractive" pairs, used greedily from largest to smallest:

\`\`\`
M 1000, CM 900, D 500, CD 400, C 100, XC 90,
L 50, XL 40, X 10, IX 9, V 5, IV 4, I 1
\`\`\``,
      code: `// Write toRoman(n) for 1 to 3999\n`,
      sol: `function toRoman(n) {\n  const table = [[1000, "M"], [900, "CM"], [500, "D"], [400, "CD"], [100, "C"], [90, "XC"], [50, "L"], [40, "XL"], [10, "X"], [9, "IX"], [5, "V"], [4, "IV"], [1, "I"]];\n  let out = "";\n  for (const [value, symbol] of table) {\n    while (n >= value) {\n      out += symbol;\n      n -= value;\n    }\n  }\n  return out;\n}`,
      tests: [['toRoman(4) === "IV" && toRoman(9) === "IX"', '4 and 9'], ['toRoman(1994) === "MCMXCIV"', '1994 is MCMXCIV'], ['toRoman(3888) === "MMMDCCCLXXXVIII"', '3888']],
    },
    {
      t: 'Rotating a Grid',
      b: `Rotating a square grid 90° clockwise is a common puzzle (think Tetris). The new row \`r\` is made from the old column \`r\`, read from bottom to top.`,
      code: `// Write rotate(grid) returning a NEW grid rotated 90° clockwise\n`,
      sol: `function rotate(grid) {\n  const n = grid.length;\n  return grid[0].map((_, c) => grid.map((row) => row[c]).reverse());\n}`,
      tests: [['JSON.stringify(rotate([[1, 2], [3, 4]])) === "[[3,1],[4,2]]"', 'Rotates a 2×2 grid'], ['JSON.stringify(rotate([[1, 2, 3], [4, 5, 6], [7, 8, 9]])) === "[[7,4,1],[8,5,2],[9,6,3]]"', 'Rotates a 3×3 grid']],
    },
    {
      t: 'Most Frequent',
      b: `Finding the most common item by comparing every pair is O(n²). Counting in one pass with a Map, then scanning the counts, is O(n) — a huge difference for big data.`,
      code: `// Write mostFrequent(arr) returning the item that appears most often (the earliest one if tied)\n`,
      sol: `function mostFrequent(arr) {\n  const counts = new Map();\n  for (const x of arr) counts.set(x, (counts.get(x) ?? 0) + 1);\n  let best;\n  let bestCount = 0;\n  for (const [x, c] of counts) {\n    if (c > bestCount) {\n      best = x;\n      bestCount = c;\n    }\n  }\n  return best;\n}`,
      tests: [['mostFrequent([3, 1, 3, 2, 1, 3]) === 3', 'Finds 3'], ['mostFrequent(["a", "b", "b", "a"]) === "a"', 'Earliest wins a tie'], ['(() => { const big = Array.from({ length: 50000 }, (_, i) => i % 97); const t = Date.now(); mostFrequent(big); return Date.now() - t < 500; })()', 'Fast on 50,000 items']],
    },
    {
      t: 'Sieve of Eratosthenes',
      b: `An ancient and elegant way to find **all** primes up to n:

1. Write out 2…n, all marked "maybe prime"
2. Take the next unmarked number p — it is prime
3. Cross out every multiple of p (starting from p²)
4. Repeat

Much faster than testing each number separately.`,
      code: `// Write sieve(n) returning all primes up to and including n\n`,
      sol: `function sieve(n) {\n  const isPrime = new Array(n + 1).fill(true);\n  isPrime[0] = isPrime[1] = false;\n  for (let p = 2; p * p <= n; p++) {\n    if (!isPrime[p]) continue;\n    for (let m = p * p; m <= n; m += p) isPrime[m] = false;\n  }\n  return isPrime.flatMap((prime, i) => (prime ? [i] : []));\n}`,
      tests: [['sieve(30).join() === "2,3,5,7,11,13,17,19,23,29"', 'Primes up to 30'], ['sieve(100000).length === 9592', 'There are 9592 primes below 100,000'], ['sieve(1).length === 0', 'No primes up to 1']],
    },
    {
      t: 'Counting Islands',
      b: `In a grid of \`1\` (land) and \`0\` (water), an island is a group of land cells connected up/down/left/right. Count them with **flood fill**: scan the grid; when you find unvisited land, count an island and "sink" all of its connected land so it is not counted again.`,
      code: `// Write countIslands(grid) — do not change the grid you are given\n`,
      sol: `function countIslands(grid) {\n  const g = grid.map((row) => [...row]);\n  const sink = (r, c) => {\n    if (r < 0 || c < 0 || r >= g.length || c >= g[0].length || g[r][c] !== 1) return;\n    g[r][c] = 0;\n    sink(r + 1, c);\n    sink(r - 1, c);\n    sink(r, c + 1);\n    sink(r, c - 1);\n  };\n  let count = 0;\n  for (let r = 0; r < g.length; r++) {\n    for (let c = 0; c < g[0].length; c++) {\n      if (g[r][c] === 1) {\n        count++;\n        sink(r, c);\n      }\n    }\n  }\n  return count;\n}`,
      tests: [['countIslands([[1, 1, 0, 0], [1, 0, 0, 1], [0, 0, 1, 1], [1, 0, 0, 0]]) === 3', 'Finds 3 islands'], ['countIslands([[0, 0], [0, 0]]) === 0', 'All water'], ['(() => { const g = [[1]]; countIslands(g); return g[0][0] === 1; })()', 'Does not change the input']],
    },
    {
      t: 'Shortest Path in a Maze',
      b: `**Breadth-first search** (BFS) finds the shortest path in a grid. Use a **queue**: start from the entrance, and repeatedly take the oldest cell, adding its unvisited open neighbours with distance + 1. The first time you reach the exit, that distance is the shortest.

\`#\` is a wall, \`.\` is open.`,
      code: `// Write shortestPath(maze, [startRow, startCol], [endRow, endCol]) returning the number of steps, or -1\n// maze is an array of strings\n`,
      sol: `function shortestPath(maze, start, end) {\n  const seen = new Set([start.join()]);\n  const queue = [[...start, 0]];\n  while (queue.length) {\n    const [r, c, d] = queue.shift();\n    if (r === end[0] && c === end[1]) return d;\n    for (const [dr, dc] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) {\n      const nr = r + dr, nc = c + dc;\n      const key = nr + "," + nc;\n      if (maze[nr]?.[nc] === "." && !seen.has(key)) {\n        seen.add(key);\n        queue.push([nr, nc, d + 1]);\n      }\n    }\n  }\n  return -1;\n}`,
      tests: [['shortestPath(["....", ".##.", "....", ".#.."], [0, 0], [3, 3]) === 6', 'Finds a 6-step path'], ['shortestPath([".#.", "##.", "..."], [0, 0], [2, 2]) === -1', 'Returns -1 when blocked'], ['shortestPath(["."], [0, 0], [0, 0]) === 0', 'Already there']],
    },
    {
      t: 'Climbing Stairs',
      b: `You can climb 1 or 2 steps at a time. How many different ways are there to climb \`n\` stairs?

The ways to reach step n = ways to reach n−1 (then 1 step) + ways to reach n−2 (then 2 steps). Building the answer from small cases upward, storing results, is called **dynamic programming**.`,
      code: `// Write climbWays(n) using a loop (not plain recursion — it must be fast for n = 80)\n`,
      sol: `function climbWays(n) {\n  let a = 1, b = 1;\n  for (let i = 2; i <= n; i++) [a, b] = [b, a + b];\n  return b;\n}`,
      tests: [['climbWays(1) === 1 && climbWays(2) === 2 && climbWays(3) === 3', 'Small cases'], ['climbWays(10) === 89', '10 stairs: 89 ways'], ['climbWays(80) === 37889062373143906', '80 stairs, instantly']],
    },
    {
      t: 'Making Change',
      b: `What is the **fewest** coins that make an amount? Greedy (always take the biggest coin) fails for some coin sets — with coins 1, 3 and 4, greedy makes 6 as 4+1+1, but 3+3 is better.

Dynamic programming gets it right: \`best[a] = 1 + min(best[a − coin])\` over every coin, built up from \`best[0] = 0\`.`,
      code: `// Write minCoins(coins, amount) returning the fewest coins, or -1 if impossible\n`,
      sol: `function minCoins(coins, amount) {\n  const best = new Array(amount + 1).fill(Infinity);\n  best[0] = 0;\n  for (let a = 1; a <= amount; a++) {\n    for (const c of coins) {\n      if (c <= a) best[a] = Math.min(best[a], best[a - c] + 1);\n    }\n  }\n  return best[amount] === Infinity ? -1 : best[amount];\n}`,
      tests: [['minCoins([1, 3, 4], 6) === 2', '6 = 3 + 3'], ['minCoins([1, 5, 10, 25], 63) === 6', '63 cents takes 6 coins'], ['minCoins([2], 3) === -1', 'Impossible amounts give -1'], ['minCoins([7], 0) === 0', 'Zero needs no coins']],
    },
    {
      t: 'Final Project: Game of Life',
      b: `Congratulations on reaching lesson 500! 🎉 Finish with a classic: **Conway's Game of Life**. A grid of cells is either alive (\`1\`) or dead (\`0\`). Each generation:

- a live cell with **2 or 3** live neighbours survives
- a dead cell with **exactly 3** live neighbours comes alive
- every other cell dies or stays dead

Neighbours are the 8 surrounding cells; cells outside the grid count as dead.

Write \`nextGeneration(grid)\` returning a **new** grid. Then open the Playground and draw it on a canvas — you are ready to build anything! 🚀`,
      code: `function nextGeneration(grid) {\n}\n`,
      sol: `function nextGeneration(grid) {\n  const alive = (r, c) => grid[r]?.[c] === 1 ? 1 : 0;\n  return grid.map((row, r) =>\n    row.map((cell, c) => {\n      let n = 0;\n      for (let dr = -1; dr <= 1; dr++) {\n        for (let dc = -1; dc <= 1; dc++) {\n          if (dr || dc) n += alive(r + dr, c + dc);\n        }\n      }\n      return n === 3 || (cell === 1 && n === 2) ? 1 : 0;\n    }),\n  );\n}`,
      tests: [
        ['JSON.stringify(nextGeneration([[0, 1, 0], [0, 1, 0], [0, 1, 0]])) === "[[0,0,0],[1,1,1],[0,0,0]]"', 'A blinker flips'],
        ['JSON.stringify(nextGeneration([[1, 1], [1, 1]])) === "[[1,1],[1,1]]"', 'A block stays still'],
        ['JSON.stringify(nextGeneration(nextGeneration([[0, 0, 0, 0], [0, 1, 1, 1], [1, 1, 1, 0], [0, 0, 0, 0]]))) === "[[0,0,0,0],[0,1,1,1],[1,1,1,0],[0,0,0,0]]"', 'A toad returns after two generations'],
        ['(() => { const g = [[0, 1, 0], [0, 1, 0], [0, 1, 0]]; nextGeneration(g); return g[0][1] === 1; })()', 'The input is not changed'],
      ],
    },
  ],
});
