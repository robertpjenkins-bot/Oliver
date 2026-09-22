registerModule({
  id: 'classes',
  title: 'Classes & Objects',
  icon: '🏗️',
  lessons: [
    {
      t: 'Your First Class',
      b: `A **class** is a blueprint for making objects of the same shape. The \`constructor\` runs when you create one with \`new\`:

\`\`\`js
class Dog {
  constructor(name) {
    this.name = name;
  }
}
const rex = new Dog("Rex");
rex.name;  // "Rex"
\`\`\`

Each object made from a class is called an **instance**.`,
      code: `// Create a class Book whose constructor takes title and author and stores them\n// Then create myBook = new Book("Matilda", "Roald Dahl")\n`,
      sol: `class Book {\n  constructor(title, author) {\n    this.title = title;\n    this.author = author;\n  }\n}\nconst myBook = new Book("Matilda", "Roald Dahl");`,
      tests: [['myBook instanceof Book', '`myBook` is a `Book`'], ['myBook.title === "Matilda" && myBook.author === "Roald Dahl"', 'It stores title and author']],
    },
    {
      t: 'Class Methods',
      b: `Methods go inside the class body (no \`function\` keyword, no commas). All instances share them:

\`\`\`js
class Dog {
  constructor(name) { this.name = name; }
  bark() { return this.name + " says woof"; }
}
new Dog("Rex").bark();   // "Rex says woof"
\`\`\``,
      code: `class Rectangle {\n  constructor(width, height) {\n    this.width = width;\n    this.height = height;\n  }\n  // add area() and perimeter()\n}\n`,
      sol: `class Rectangle {\n  constructor(width, height) {\n    this.width = width;\n    this.height = height;\n  }\n  area() {\n    return this.width * this.height;\n  }\n  perimeter() {\n    return 2 * (this.width + this.height);\n  }\n}`,
      tests: [['new Rectangle(3, 4).area() === 12', '`area()` works'], ['new Rectangle(3, 4).perimeter() === 14', '`perimeter()` works']],
    },
    {
      t: 'Many Instances',
      b: `Every instance has its **own** data, even though they share methods:

\`\`\`js
const a = new Counter();
const b = new Counter();
a.up(); a.up();
b.up();
a.count; // 2
b.count; // 1
\`\`\``,
      code: `// Write class Counter with count starting at 0, up() adding 1 and down() subtracting 1\n`,
      sol: `class Counter {\n  constructor() {\n    this.count = 0;\n  }\n  up() {\n    this.count++;\n  }\n  down() {\n    this.count--;\n  }\n}`,
      tests: [['(() => { const a = new Counter(), b = new Counter(); a.up(); a.up(); b.down(); return a.count === 2 && b.count === -1; })()', 'Counters are independent']],
    },
    {
      t: 'Constructor Defaults',
      b: `Constructors can use default parameters like any function:

\`\`\`js
class Player {
  constructor(name, lives = 3) {
    this.name = name;
    this.lives = lives;
    this.score = 0;   // always starts at 0
  }
}
\`\`\``,
      code: `// Write class Pizza(size = "medium", toppings = []) that also sets this.baked = false,\n// and a method bake() that sets baked to true\n`,
      sol: `class Pizza {\n  constructor(size = "medium", toppings = []) {\n    this.size = size;\n    this.toppings = toppings;\n    this.baked = false;\n  }\n  bake() {\n    this.baked = true;\n  }\n}`,
      tests: [['new Pizza().size === "medium" && new Pizza().toppings.length === 0', 'Defaults work'], ['(() => { const p = new Pizza("large", ["ham"]); p.bake(); return p.size === "large" && p.toppings[0] === "ham" && p.baked; })()', 'Custom values and `bake()` work']],
    },
    {
      t: 'Getters in Classes',
      b: `A \`get\` method is read like a property — perfect for values calculated from other data:

\`\`\`js
class Circle {
  constructor(r) { this.r = r; }
  get area() { return Math.PI * this.r ** 2; }
}
new Circle(2).area;   // no ()!
\`\`\``,
      code: `class Person {\n  constructor(first, last, birthYear) {\n    this.first = first;\n    this.last = last;\n    this.birthYear = birthYear;\n  }\n  // add a getter fullName, and a getter initials (like "A.L.")\n}\n`,
      sol: 'class Person {\n  constructor(first, last, birthYear) {\n    this.first = first;\n    this.last = last;\n    this.birthYear = birthYear;\n  }\n  get fullName() {\n    return `${this.first} ${this.last}`;\n  }\n  get initials() {\n    return `${this.first[0]}.${this.last[0]}.`;\n  }\n}',
      tests: [['new Person("Ada", "Lovelace", 1815).fullName === "Ada Lovelace"', '`fullName` works'], ['new Person("Ada", "Lovelace", 1815).initials === "A.L."', '`initials` works']],
    },
    {
      t: 'Setters that Validate',
      b: `A \`set\` method runs when a property is assigned — a good place to check values. Store the real value under a different name (often with an underscore):

\`\`\`js
class Thermostat {
  set target(t) {
    if (t < 10 || t > 30) throw new RangeError("10–30 only");
    this._target = t;
  }
  get target() { return this._target; }
}
\`\`\``,
      code: `// Write class Volume with a getter/setter level (stored in this._level, starting at 5)\n// The setter clamps values into 0..10 instead of throwing\n`,
      sol: `class Volume {\n  constructor() {\n    this._level = 5;\n  }\n  get level() {\n    return this._level;\n  }\n  set level(v) {\n    this._level = Math.max(0, Math.min(10, v));\n  }\n}`,
      tests: [['new Volume().level === 5', 'Starts at 5'], ['(() => { const v = new Volume(); v.level = 8; return v.level === 8; })()', 'Normal values are stored'], ['(() => { const v = new Volume(); v.level = 99; const hi = v.level; v.level = -3; return hi === 10 && v.level === 0; })()', 'Values are clamped to 0..10']],
    },
    {
      t: 'Static Methods',
      b: `A \`static\` method belongs to the **class itself**, not to instances. Use it for helpers and alternative ways to create objects:

\`\`\`js
class Point {
  constructor(x, y) { this.x = x; this.y = y; }
  static origin() { return new Point(0, 0); }
}
Point.origin();   // called on the class
\`\`\``,
      code: `class Temperature {\n  constructor(celsius) {\n    this.celsius = celsius;\n  }\n  // add static fromFahrenheit(f) returning a new Temperature\n}\n`,
      sol: `class Temperature {\n  constructor(celsius) {\n    this.celsius = celsius;\n  }\n  static fromFahrenheit(f) {\n    return new Temperature((f - 32) * 5 / 9);\n  }\n}`,
      tests: [['Temperature.fromFahrenheit(212) instanceof Temperature', 'Returns a `Temperature`'], ['Temperature.fromFahrenheit(212).celsius === 100', '212°F is 100°C'], ['$src.includes("static")', 'Use `static`']],
    },
    {
      t: 'Static Counters',
      b: `Static **properties** are shared by the whole class. They can count how many instances exist or hand out IDs:

\`\`\`js
class Ticket {
  static nextId = 1;
  constructor() {
    this.id = Ticket.nextId++;
  }
}
\`\`\``,
      code: `// Write class Order where each new order gets id 1, 2, 3, … using a static nextId\n`,
      sol: `class Order {\n  static nextId = 1;\n  constructor() {\n    this.id = Order.nextId++;\n  }\n}`,
      tests: [['(() => { Order.nextId = 1; const a = new Order(), b = new Order(), c = new Order(); return a.id === 1 && b.id === 2 && c.id === 3; })()', 'Orders get ids 1, 2, 3']],
    },
    {
      t: 'Private Fields',
      b: `Fields starting with \`#\` are **truly private**: code outside the class cannot read or change them at all.

\`\`\`js
class Safe {
  #code;
  constructor(code) { this.#code = code; }
  open(guess) { return guess === this.#code; }
}
new Safe(1234).#code;   // SyntaxError!
\`\`\``,
      code: `// Write class Account with a private #balance starting at 0,\n// deposit(n), and a getter balance that returns it\n`,
      sol: `class Account {\n  #balance = 0;\n  deposit(n) {\n    this.#balance += n;\n  }\n  get balance() {\n    return this.#balance;\n  }\n}`,
      tests: [['(() => { const a = new Account(); a.deposit(50); a.deposit(25); return a.balance === 75; })()', 'Deposits add up'], ['(() => { const a = new Account(); a.balance = 1000000; return a.balance === 0; })()', 'Nobody can set the balance directly'], ['$src.includes("#balance")', 'Use a private `#balance`']],
    },
    {
      t: 'Private Methods',
      b: `Methods can be private too. They are helpers only the class itself may call:

\`\`\`js
class Form {
  submit() {
    if (!this.#isValid()) return "Fix the errors";
    return "Sent!";
  }
  #isValid() { … }
}
\`\`\``,
      code: `// Write class Password(text) with a public strength() returning "strong" or "weak".\n// It is strong if the private method #long() (length >= 10) AND #mixed() (has a digit and a letter) are true\n`,
      sol: `class Password {\n  constructor(text) {\n    this.text = text;\n  }\n  #long() {\n    return this.text.length >= 10;\n  }\n  #mixed() {\n    return /\\d/.test(this.text) && /[a-z]/i.test(this.text);\n  }\n  strength() {\n    return this.#long() && this.#mixed() ? "strong" : "weak";\n  }\n}`,
      tests: [['new Password("abc123").strength() === "weak"', 'Short → weak'], ['new Password("abcdefghijk").strength() === "weak"', 'No digit → weak'], ['new Password("abcdefg12345").strength() === "strong"', 'Long and mixed → strong'], ['$src.includes("#long") && $src.includes("#mixed")', 'Use private methods']],
    },
    {
      t: 'Inheritance',
      b: `A class can **extend** another, inheriting all its methods, then add more:

\`\`\`js
class Animal {
  constructor(name) { this.name = name; }
  eat() { return this.name + " is eating"; }
}
class Bird extends Animal {
  fly() { return this.name + " is flying"; }
}
const b = new Bird("Tweety");
b.eat();  // inherited!
b.fly();
\`\`\``,
      code: `class Vehicle {\n  constructor(wheels) {\n    this.wheels = wheels;\n  }\n  describe() {\n    return "A vehicle with " + this.wheels + " wheels";\n  }\n}\n// Write class Bike extending Vehicle, with ringBell() returning "Ring ring!"\n`,
      sol: `class Vehicle {\n  constructor(wheels) {\n    this.wheels = wheels;\n  }\n  describe() {\n    return "A vehicle with " + this.wheels + " wheels";\n  }\n}\nclass Bike extends Vehicle {\n  ringBell() {\n    return "Ring ring!";\n  }\n}`,
      tests: [['new Bike(2).ringBell() === "Ring ring!"', '`ringBell()` works'], ['new Bike(2).describe() === "A vehicle with 2 wheels"', '`describe()` is inherited'], ['new Bike(2) instanceof Vehicle', 'A `Bike` is a `Vehicle`']],
    },
    {
      t: 'super in Constructors',
      b: `If a subclass has its own constructor, it **must** call \`super(...)\` first — that runs the parent's constructor:

\`\`\`js
class Cat extends Animal {
  constructor(name, indoor) {
    super(name);          // sets this.name
    this.indoor = indoor;
  }
}
\`\`\``,
      code: `class Shape {\n  constructor(name) {\n    this.name = name;\n  }\n}\n// Write class Square extends Shape: constructor(side) calls super("square") and stores side,\n// with an area() method\n`,
      sol: `class Shape {\n  constructor(name) {\n    this.name = name;\n  }\n}\nclass Square extends Shape {\n  constructor(side) {\n    super("square");\n    this.side = side;\n  }\n  area() {\n    return this.side ** 2;\n  }\n}`,
      tests: [['new Square(5).name === "square"', 'Its name is `"square"`'], ['new Square(5).area() === 25', '`area()` is `25`'], ['$src.includes("super(")', 'Call `super(...)`']],
    },
    {
      t: 'Overriding Methods',
      b: `A subclass can **override** a parent method by defining one with the same name. Call the parent's version with \`super.method()\`:

\`\`\`js
class Loud extends Speaker {
  speak() {
    return super.speak().toUpperCase() + "!!!";
  }
}
\`\`\``,
      code: `class Employee {\n  constructor(name, salary) {\n    this.name = name;\n    this.salary = salary;\n  }\n  pay() {\n    return this.salary;\n  }\n}\n// Write class Manager extends Employee that overrides pay() to add a 1000 bonus, using super.pay()\n`,
      sol: `class Employee {\n  constructor(name, salary) {\n    this.name = name;\n    this.salary = salary;\n  }\n  pay() {\n    return this.salary;\n  }\n}\nclass Manager extends Employee {\n  pay() {\n    return super.pay() + 1000;\n  }\n}`,
      tests: [['new Manager("Kim", 3000).pay() === 4000', 'Managers get the bonus'], ['new Employee("Lu", 3000).pay() === 3000', 'Employees do not'], ['$src.includes("super.pay()")', 'Use `super.pay()`']],
    },
    {
      t: 'What Kind of Object?',
      b: `\`instanceof\` checks whether an object was made from a class — **or any class extending it**:

\`\`\`js
b instanceof Bird     // true
b instanceof Animal   // true — Bird extends Animal
b instanceof Array    // false
\`\`\``,
      code: `class Animal {}\nclass Fish extends Animal {}\nclass Mammal extends Animal {}\nclass Whale extends Mammal {}\n// Write classify(x) returning "mammal" for any Mammal, "fish" for a Fish, "animal" for other Animals,\n// and "not an animal" otherwise\n`,
      sol: `class Animal {}\nclass Fish extends Animal {}\nclass Mammal extends Animal {}\nclass Whale extends Mammal {}\nfunction classify(x) {\n  if (x instanceof Mammal) return "mammal";\n  if (x instanceof Fish) return "fish";\n  if (x instanceof Animal) return "animal";\n  return "not an animal";\n}`,
      tests: [['classify(new Whale()) === "mammal"', 'A whale is a mammal'], ['classify(new Fish()) === "fish"', 'Fish'], ['classify(new Animal()) === "animal"', 'Plain animal'], ['classify({}) === "not an animal"', 'Anything else']],
    },
    {
      t: 'Custom toString',
      b: `When an object is turned into a string (in a template literal or with \`+\`), JavaScript calls its \`toString()\` method. Override it for nicer output:

\`\`\`js
class Money {
  constructor(cents) { this.cents = cents; }
  toString() { return "$" + (this.cents / 100).toFixed(2); }
}
\`Total: \${new Money(1250)}\`   // "Total: $12.50"
\`\`\``,
      code: `// Write class Fraction(top, bottom) with toString() returning "top/bottom"\n`,
      sol: 'class Fraction {\n  constructor(top, bottom) {\n    this.top = top;\n    this.bottom = bottom;\n  }\n  toString() {\n    return `${this.top}/${this.bottom}`;\n  }\n}',
      tests: [['`${new Fraction(3, 4)}` === "3/4"', 'Template literals show "3/4"'], ['"half = " + new Fraction(1, 2) === "half = 1/2"', 'Joining with + works too']],
    },
    {
      t: 'Class Fields',
      b: `You can declare properties directly in the class body, outside the constructor. They are set on every new instance:

\`\`\`js
class Game {
  score = 0;
  lives = 3;
  players = [];
}
\`\`\`

Handy for values that do not depend on constructor arguments.`,
      code: `// Write class Playlist with fields name = "New playlist" and songs = [],\n// plus add(song) and a getter length\n`,
      sol: `class Playlist {\n  name = "New playlist";\n  songs = [];\n  add(song) {\n    this.songs.push(song);\n  }\n  get length() {\n    return this.songs.length;\n  }\n}`,
      tests: [['new Playlist().name === "New playlist"', 'Default name'], ['(() => { const a = new Playlist(), b = new Playlist(); a.add("x"); a.add("y"); return a.length === 2 && b.length === 0; })()', 'Each playlist has its own songs']],
    },
    {
      t: 'Polymorphism',
      b: `**Polymorphism** means different classes offer the same method, so code can use them without caring which is which:

\`\`\`js
for (const shape of [new Circle(1), new Square(2)]) {
  console.log(shape.area());   // each knows its own formula
}
\`\`\``,
      code: `// Write classes Circle(r), Rect(w, h) and Triangle(b, h), each with area()\n// Then write totalArea(shapes) adding up all their areas\n`,
      sol: `class Circle {\n  constructor(r) { this.r = r; }\n  area() { return Math.PI * this.r ** 2; }\n}\nclass Rect {\n  constructor(w, h) { this.w = w; this.h = h; }\n  area() { return this.w * this.h; }\n}\nclass Triangle {\n  constructor(b, h) { this.b = b; this.h = h; }\n  area() { return (this.b * this.h) / 2; }\n}\nfunction totalArea(shapes) {\n  return shapes.reduce((sum, s) => sum + s.area(), 0);\n}`,
      tests: [['new Rect(2, 3).area() === 6 && new Triangle(4, 5).area() === 10', '`Rect` and `Triangle` work'], ['Math.abs(new Circle(1).area() - Math.PI) < 1e-9', '`Circle` works'], ['totalArea([new Rect(2, 3), new Triangle(4, 5)]) === 16', '`totalArea` works']],
    },
    {
      t: 'Method Chaining',
      b: `If each method returns \`this\`, calls can be **chained**:

\`\`\`js
new Builder().add("a").add("b").build();
\`\`\``,
      code: `// Write class Sentence with add(word) and exclaim() both returning this, and toString()\n// new Sentence().add("hello").add("world").exclaim().toString() → "hello world!"\n`,
      sol: `class Sentence {\n  words = [];\n  ending = "";\n  add(word) {\n    this.words.push(word);\n    return this;\n  }\n  exclaim() {\n    this.ending = "!";\n    return this;\n  }\n  toString() {\n    return this.words.join(" ") + this.ending;\n  }\n}`,
      tests: [['new Sentence().add("hello").add("world").exclaim().toString() === "hello world!"', 'Chaining works'], ['new Sentence().add("hi").toString() === "hi"', 'Without exclaim there is no "!"']],
    },
    {
      t: 'Composition',
      b: `Inheritance is not always the best tool. Often it is simpler to build an object **from** other objects — "has a" instead of "is a":

\`\`\`js
class Car {
  constructor() {
    this.engine = new Engine();   // a car HAS an engine
  }
  start() { return this.engine.start(); }
}
\`\`\``,
      code: `class Engine {\n  constructor() {\n    this.running = false;\n  }\n  start() {\n    this.running = true;\n    return "Vroom";\n  }\n}\n// Write class Car that HAS an engine (this.engine = new Engine()) and a start() method using it\n`,
      sol: `class Engine {\n  constructor() {\n    this.running = false;\n  }\n  start() {\n    this.running = true;\n    return "Vroom";\n  }\n}\nclass Car {\n  constructor() {\n    this.engine = new Engine();\n  }\n  start() {\n    return this.engine.start();\n  }\n}`,
      tests: [['new Car().engine instanceof Engine', 'A car has an `Engine`'], ['(() => { const c = new Car(); return c.start() === "Vroom" && c.engine.running; })()', '`start()` starts the engine'], ['!$src.includes("extends")', 'Use composition, not `extends`']],
    },
    {
      t: 'Project: Monster Battle',
      b: `Build a tiny battle system:

- \`class Character(name, hp, attack)\` with \`isAlive()\` (hp > 0), and \`hit(target)\` which lowers the target's hp by this character's attack (never below 0) and returns a message like \`"Hero hits Slime for 4"\`
- \`class Hero extends Character\` with a \`potions\` count starting at 2, and \`heal()\`: if a potion is left, use it to restore 10 hp (up to the starting hp) and return \`true\`; otherwise return \`false\`
- a function \`battle(a, b)\`: they take turns hitting (a first) until one dies; return the winner's name`,
      code: `class Character {\n}\n\nclass Hero extends Character {\n}\n\nfunction battle(a, b) {\n}\n`,
      sol: 'class Character {\n  constructor(name, hp, attack) {\n    this.name = name;\n    this.hp = hp;\n    this.maxHp = hp;\n    this.attack = attack;\n  }\n  isAlive() {\n    return this.hp > 0;\n  }\n  hit(target) {\n    target.hp = Math.max(0, target.hp - this.attack);\n    return `${this.name} hits ${target.name} for ${this.attack}`;\n  }\n}\n\nclass Hero extends Character {\n  potions = 2;\n  heal() {\n    if (this.potions === 0) return false;\n    this.potions--;\n    this.hp = Math.min(this.maxHp, this.hp + 10);\n    return true;\n  }\n}\n\nfunction battle(a, b) {\n  let attacker = a;\n  let defender = b;\n  while (a.isAlive() && b.isAlive()) {\n    attacker.hit(defender);\n    [attacker, defender] = [defender, attacker];\n  }\n  return a.isAlive() ? a.name : b.name;\n}',
      tests: [
        ['(() => { const h = new Hero("Hero", 30, 4), s = new Character("Slime", 10, 2); return h.hit(s) === "Hero hits Slime for 4" && s.hp === 6; })()', '`hit` lowers hp and returns a message'],
        ['(() => { const h = new Hero("Hero", 30, 4), s = new Character("Blob", 3, 1); h.hit(s); return s.hp === 0 && !s.isAlive(); })()', 'hp never goes below 0'],
        ['(() => { const h = new Hero("Hero", 30, 4); h.hp = 25; const a = h.heal(); const hp1 = h.hp; h.hp = 5; h.heal(); return a && hp1 === 30 && h.hp === 15 && h.heal() === false; })()', '`heal` uses potions and caps at max hp'],
        ['battle(new Hero("Hero", 30, 4), new Character("Ogre", 20, 3)) === "Hero"', 'The hero beats the ogre'],
        ['battle(new Hero("Hero", 5, 1), new Character("Dragon", 50, 10)) === "Dragon"', 'The dragon beats a weak hero'],
      ],
    },
  ],
});
