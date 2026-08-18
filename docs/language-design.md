# Language Design

How Aurora thinks about values and the cool stuff you can do with them.

## Tape size (how many bytes a value has)

Every value in Aurora is a **tape**: a fixed run of bytes, big-endian, right-aligned. How many bytes is a property of the program being compiled, not of the language — the default is **8**, the floor is **1** and the ceiling is **32** (the EVM word, and the widest operand `PUSH32` can carry).

```toml
# aurora.toml
[project]
  tape_size = 1
```

It belongs to the project because it decides what the source *means*, not how it is built:
the same file cannot be one dialect for one profile and another for the next.

```sh
aurora build --tape-size 2    # the flag wins over the manifest
aurora run -t 1
aurora repl -t 1
```

The size applies to the whole compilation unit, and the compiled binary carries the choice implicitly.

**A literal that does not fit is a compile-time error**, not a silent truncation:

```
value 300 does not fit in a 1-byte tape (max 255)
```

What changes with a narrow tape is worth saying out loud:

- **Arithmetic wraps at the tape width.** With `tape_size = 1`, `255 + 1` is `0` and `0 - 1` is `255`. A tape of N bytes holds values modulo 2^(8N).
- **`head` and `tail` take the index modulo the width**, so with one byte the index is always 0.
- **Text is a tape**, so the width is how much text fits: one byte per tape holds one character, thirty-two hold thirty-two of ASCII.

## Everything is a tape

Numbers, conditions, text, the neutral value and **deferred scopes** are all tapes of `tape_size` bytes. A `defer` produces the index of its scope, so `defer { }` is a tape holding 0 — the same bytes as `false` and as the number `0`.

That is the bargain of an untyped language: values that are the same bytes are the same value, and calling a number equal to a scope's index calls that scope.

A byte runs from 0 to 255, and nothing in a tape is set aside to mean "negative". So the
language has no signs and no negative numbers at all: `-5` is a tape of bytes like any
other, the one you reach by taking 5 away from zero. See
[No signs, and no negative numbers](#no-signs-and-no-negative-numbers).

**How many scopes a tape can name.** Since the value is an index, a tape of N bytes names 2^(8N) scopes — 256 of them at one byte. Declaring more in the same scope makes the index wrap, so a call reaches a different scope. The compiler warns about it:

```
warning: 257 deferred scopes in one scope, but a 1-byte tape can only name 256:
calls past that reach the wrong scope
```

It is a **warning, not an error**, and it exists only at compile time. The count is static, so it cannot know how often a scope really runs; and a program that is already running is never stopped by it — a limit implied by the shape of the source should not become a failure mid-execution. The tally is per scope, since each running scope keeps its own. See [examples/defer_capacity.ar](../examples/defer_capacity.ar).

## Expressions only (no Statements)

Aurora is an **expression-only** language: there are no statements. Everything at the top level and inside blocks is an expression that produces a value.

- **Top level**: A "program" is a sequence of expressions separated by semicolons. Each one is evaluated in order; the last value is the result of the program.
- **Blocks** (`{ ... }`): The body of a block is a sequence of expressions. Blocks are expressions that evaluate their body and produce the value of the last expression.
- **Control flow**: `if`/`else` and `branch` are expressions — they have a value (the branch that was taken). There is no "statement" form of conditionals.

So when you write `ident a = 3;` or `printb x;` or `false;`, you are always writing expressions. The parser and the AST reflect this: the module holds a list of expression nodes (often still named "statements" in code for historical reasons), and blocks hold lists of expressions. There is no separate "statement" node type.

## The neutral value

Aurora has no keyword for "no value". An expression that has nothing to say produces a **tape of zeros** — the very same value as `false` and as the number `0`, because in an untyped language a tape of zeros is a tape of zeros and there is nothing else it could be.

- **Where it appears**: an empty block `{ }`, an `if` without `else` whose test fails, the value of a binding (`ident a = 1;` itself evaluates to zeros), and a scope that returns no value.
- **How to write it**: use `false` when the neutral value is the point, or `0` when it reads better as a number. They are the same bytes.

```aurora
ident x = false;   #- the neutral value
if false { 1; };   #- no else: the whole expression is zeros
{ };   #- an empty block is zeros
```

> A `nothing` keyword existed until it became indistinguishable from `false`. It was removed rather than kept as a second name for one value; source using it now reads `nothing` as an ordinary identifier, which will not resolve.

## Untyped Philosophy

Aurora is **untyped** — everything is just bytes under the hood. There are no type distinctions at the language level - numbers, booleans, tapes (arrays), and functions are all represented as byte arrays.

### Core Concept

In Aurora, all values are byte arrays:
- `ident a = 3;` → `[0, 0, 0, 0, 0, 0, 0, 3]`
- `ident b = [1, 2];` → `[0, 0, 0, 0, 0, 0, 1, 2]` (tape values stored as direct bytes, right-aligned)
- `ident c = [];` → `[0, 0, 0, 0, 0, 0, 0, 0]` (empty tape, all zeros)
- `ident d = true;` → `[0, 0, 0, 0, 0, 0, 0, 1]` — **a boolean is a tape like any other value**, indistinguishable from the number 1

This means that `3` (8 bytes) and `[1, 2]` (8 bytes) are just different representations of bytes. The language doesn't enforce type safety - it's up to the operations to interpret the bytes correctly.

### Example

```aurora
ident a = 3;   #- 8 bytes: [0, 0, 0, 0, 0, 0, 0, 3]
ident b = [1, 2];   #- 8 bytes: [0, 0, 0, 0, 0, 0, 1, 2] (tape values as direct bytes, right-aligned)
ident c = [];   #- 8 bytes: [0, 0, 0, 0, 0, 0, 0, 0] (empty tape)
ident d = true;   #- [0, 0, 0, 0, 0, 0, 0, 1] (same bytes as the number 1)
```

Note: Tapes (arrays) store values directly as bytes, not as unsigned 64-bit integers. So `[1, 2, 3]` is represented as `[0, 0, 0, 0, 0, 1, 2, 3]` (8 bytes, right-aligned), not as 24 bytes (3 × 8 bytes). All tapes have the same width (`tape_size`, 8 by default). The width never changes: an operation that would need more room discards what reaches the far end (see Tape Operations).

## Tapes (Arrays)

Tapes are just a more declarative way to create 8-byte arrays in Aurora. They provide a convenient syntax for specifying byte values directly, but fundamentally they're the same as any other 8-byte value in the language.

### What are Tapes?

Tapes use the bracket syntax `[value1, value2, ...]` to create an 8-byte array where values are stored as direct bytes (right-aligned). This is simply syntactic sugar - under the hood, tapes are just 8-byte arrays like any other value in Aurora.

### Creating Tapes

```aurora
#- Empty tape - creates 8 bytes all zeros
ident a = [];   #- [0, 0, 0, 0, 0, 0, 0, 0]

#- Tape with values - values stored as direct bytes (right-aligned)
ident b = [1, 2, 3];   #- [0, 0, 0, 0, 0, 1, 2, 3]
ident c = [0, 0, 0, 0, 0, 244, 254];   #- [0, 0, 0, 0, 0, 244, 254]

#- Equivalent ways to create 8 bytes of zeros
ident d = 0;   #- [0, 0, 0, 0, 0, 0, 0, 0] (number 0)
ident e = [];   #- [0, 0, 0, 0, 0, 0, 0, 0] (empty tape)
```

### Key Points

- **Tapes all have the same width**: `tape_size` bytes (8 by default), regardless of how many values you specify
- **Values are stored as direct bytes**: Each value in `[1, 2, 3]` is stored as a single byte, not as an 8-byte unsigned 64-bit integer
- **Right-aligned storage**: Values are padded with zeros on the left (right-aligned)
- **Just syntactic sugar**: `[1, 2, 3]` is equivalent to creating an 8-byte array and using operations like `pull` to add values
- **Same as numbers**: `0` and `[]` both create the same 8-byte array of zeros

### Tape Operations

A tape behaves as a **shift register** of fixed width. `pull` moves it left and lets the value in at the right end; `push` moves it right and lets the value in at the left end. Whatever reaches the far end is discarded — the width never changes and nothing grows.

The value contributes only its **significant bytes** (from its first non-zero byte on), so pulling `4` moves the tape by one byte, not by a whole width.

- **`pull tape value`**: shifts left, the value enters at the right; bytes leaving on the left are discarded
- **`push tape value`**: shifts right, the value enters at the left; bytes leaving on the right are discarded
- **`head tape n`**: keeps the first `n` significant bytes
- **`tail tape n`**: drops the first `n` significant bytes and keeps the rest

Because everything is a tape, these work on any value — `ident a = 1; push a 2;` is as valid as operating on a `[…]` literal.

#### Index Behavior for `head` and `tail`

Since all tapes have the same width, the index `n` in `head` and `tail` operations is applied modulo `tape_size` to prevent boundary errors. This means:

- **The index is a literal number**: the grammar takes a number there, not an expression and not a name, so `head t 2` compiles and `head t n` does not. An index computed while the program runs is not expressible today.
- **Any index works**: it is taken modulo the tape width, so it lands in `0 .. tape_size - 1`
- **No boundary errors**: the operation never fails for being out of bounds
- **Predictable behavior**: `head tape 10` is `head tape 2` with the default 8-byte tape, since `10 % 8 = 2`
- **No negative index**: there is no negative literal to write, because [the language has no signs](#no-signs-and-no-negative-numbers)

**Examples:**
- `head [1, 2, 3, 4, 5, 6, 7, 8] 2` → Gets first 2 bytes: `[0, 0, 0, 0, 0, 0, 1, 2]`
- `head [1, 2, 3, 4, 5, 6, 7, 8] 10` → `10 % 8 = 2`, gets first 2 bytes: `[0, 0, 0, 0, 0, 0, 1, 2]`
- `head [1, 2, 3, 4, 5, 6, 7, 8] 8` → `8 % 8 = 0`, gets 0 bytes: `[0, 0, 0, 0, 0, 0, 0, 0]`
- `tail [1, 2, 3, 4, 5, 6, 7, 8] 2` → Skips first 2 bytes: `[0, 0, 3, 4, 5, 6, 7, 8]`
- `tail [1, 2, 3, 4, 5, 6, 7, 8] 18` → `18 % 8 = 2`, skips first 2 bytes: `[0, 0, 3, 4, 5, 6, 7, 8]`

### Examples

```aurora
#- Create a tape and manipulate it
ident a = [1, 2, 3];   #- [0, 0, 0, 0, 0, 1, 2, 3]
ident b = pull a 4;   #- 4 enters at the right: [0, 0, 0, 0, 1, 2, 3, 4]
ident c = push b 5;   #- 5 enters at the left, the 4 falls off the right: [5, 0, 0, 0, 0, 1, 2, 3]

#- Extract parts of a tape
ident d = head [1, 2, 3, 4, 5] 2;   #- First 2 bytes: [0, 0, 0, 0, 0, 0, 1, 2]
ident e = tail [1, 2, 3, 4, 5] 2;   #- Skip first 2 bytes: [0, 0, 0, 0, 0, 3, 4, 5]
ident f = head [1, 2, 3, 4, 5, 6, 7, 8] 10;   #- 10 % 8 = 2, first 2 bytes: [0, 0, 0, 0, 0, 0, 1, 2]
ident g = tail [1, 2, 3, 4, 5, 6, 7, 8] 18;   #- 18 % 8 = 2, skip first 2 bytes: [0, 0, 3, 4, 5, 6, 7, 8]

#- Combine tapes (using pull)
ident h = pull [1, 2] [3, 4];   #- Concatenate: [0, 0, 0, 0, 1, 2, 3, 4] (significant bytes concatenated)
```

Remember: Tapes are just a convenient way to create and work with 8-byte arrays. They're not a separate type - they're the same 8-byte arrays that Aurora uses for everything!

## Text

Text is one more way of writing a tape, next to `1`, `0x2a`, `[1, 2]` and `true`. `"hi"` is
the tape holding the bytes of `h` and `i` — not a kind of value of its own.

```aurora
ident greeting = "hi";
printb greeting;   #- [0 0 0 0 0 0 104 105]
printd greeting;   #- 26729 — the number those bytes spell
printc greeting;   #- hi
```

### What follows from that

- **`"a"` is 97.** A text of one character is the tape its number is, so `1 + "a"` is 98 and
  needs no rule of its own.
- **Comparing text is comparing bytes.** `"hi" equals 26729` is true, because they are the
  same tape. Nothing in a value says it was written as text.
- **The bytes are UTF-8.** `"café"` is five bytes, and `printc` gives it back whole.
- **`""` is the neutral value**, a tape of zeros, like `false` and `0`.
- **A tape holds `tape_size` bytes, so that is how much text fits.** Nine bytes at the
  default eight is a compile error, reported where the text was written — the same rule a
  number literal that does not fit follows. `--tape-size 16` makes room for sixteen.

### Text longer than a tape

There is none yet. Text that does not fit is rejected rather than split, and the way to hold
more is a wider tape. Holding text of any length needs something the language does not have:
building a value at runtime, and reading a position out of it with an index computed while
it runs — `head` and `tail` take a literal.

### Reels, and why they are gone

Text used to be a **reel**: one tape per character, so `"hi"` was two tapes and 16 bytes at
the default width, and `"Gui"` was 96 bytes at `--tape-size 32`. It was a way of holding more
than a tape could, and it cost `tape_size - 1` bytes of zero for every character — a cost
that grew exactly when the tape got wider.

It also made text the one value that was not a tape, in a language whose premise is that
everything is. Removing it made `"a"` and `97` the same value rather than a coincidence of
one-character strings, and made text fit in a struct field, which a reel never could.

**Breaking:** `printc` of a number is now its bytes as UTF-8, not the character that number
names — `printc 44` is still `,`, but a character above ASCII is written as text rather than
as its code point.

### Three Readings of a Tape

A value is a tape. It is not a number, a character or a string on its own — those are ways of
reading the same bytes, and there is one builtin per reading:

| Builtin | Reads the tape as | `44` |
|---|---|---|
| `printb` | the bytes it is | `[0 0 0 0 0 0 0 44]` |
| `printd` | the number they spell, big-endian | `44` |
| `printc` | the character that number names, as UTF-8 | `,` |

Nothing is converted between them, and none of them changes the value: they are three ways of
looking at it.

Which is also what a print is worth. Aurora is expression-only, so a print produces a value
like anything else, and the value it produces is the one it showed — reading a tape does not
change it. A block whose last expression is a print is worth what the print showed, rather
than nothing at all.

A run of tapes — a struct, say — is read the same way: every byte for `printb`, one number
per tape for `printd`, and for `printc` the bytes of the whole run, with the zeros that pad
each tape dropped.

```aurora
ident greeting = "hello";
printb greeting;   #- [0 0 0 104 101 108 108 111]
printd greeting;   #- 448378203247
printc greeting;   #- hello
```

`printc` reads the bytes as UTF-8, which is what keeps text outside ASCII intact — `é` is
the two bytes it is:

```aurora
printc "café";   #- café
printb "café";   #- [0 0 0 99 97 102 195 169]
printc 44;       #- , — the byte 44
printc 26729;    #- hi — the bytes 104 and 105
```

A tape of zeros is the neutral value, so it spells `0` and `printc` writes nothing for it.
Bytes that are not UTF-8 have nothing to write either; the value is still whatever it is.

### Examples

```aurora
#- Text of one character is the tape its number is, so arithmetic needs no rule
#- of its own.
ident a = "a";         #- the tape holding 97
ident result = 1 + a;  #- 98
printc result;         #- b

#- Digits are their bytes, not the number they read as.
ident num_str = "123";
printc num_str;   #- 123
printd num_str;   #- 3224115 — the bytes 49, 50 and 51, not one hundred and twenty-three

#- Empty text is no bytes, which is the neutral value.
ident empty = "";
printb empty;     #- [0 0 0 0 0 0 0 0]
```

## Structs

A struct names the tapes of a run:

```aurora
struct Point { x, y };

ident p = Point{10, 20};
printd p.x;   #- 10
printd p;   #- 10 20 — the whole run
```

`Point{10, 20}` is two tapes laid end to end, with nothing in it saying a struct built it:
no header, no length, no tag. A struct is not a new kind of value, and a field is exactly one
tape wide, so the field at index *i* sits at `i × tape_size`.

### The directives die at compile time

`struct` and `as` are **directives for whoever writes the source**, and they exist for three
things, all of them before the program runs:

1. reporting a mistake — a field that does not exist, a construction that miscounts;
2. saying how to reach the data — which index a name is;
3. telling the language server what is there, so it can complete a field and describe one.

None of it needs to exist in the compiled program: the flow is static, the fields are
positional and every one is the same width. The compiler reads the directive, resolves an
index and drops the rest. Nothing about a struct — not its name, not its fields — reaches
the IR or the binary.

Which is why two structs of the same width are the same value, and why nothing is checked
at runtime.

### Naming a shape with `as`

Behaviour lives in `defer`, and a `defer` receives values through `feed`, which hands over
bytes and nothing else. The shape does not survive that crossing, so it is named again:

```aurora
struct Point { x, y };

ident area = defer {
  ident q = feed(0) as Point;
  q.x * q.y;
};
printd area(Point{10, 20});   #- 200
```

`as` is a claim, not a cast: there is nothing in a run of bytes to check it against. A wrong
claim reads the wrong tapes, and reading past the end gives the neutral value rather than
stopping the program — the same rule `head` and `feed` follow.

### What is an error

Because the directive exists to catch mistakes, these stop the compilation, with the line
and column where they were written:

- reading a field the struct does not have;
- reading a field of a value whose shape nothing declared;
- building with the wrong number of values;
- using a struct name as a value: it is a directive, not something to load.

Padding a short construction with the neutral value would match how `feed` and `head` never
fail, and would give up the only thing the directive does.

See [examples/structs.ar](../examples/structs.ar).

## Arithmetic Operations

Arithmetic reads a tape as an unsigned big-endian integer and writes the result back as a tape, wrapping at the tape width.

### How It Works

1. **Values < 8 bytes**: Padded with zeros on the left (right-aligned)
   ```aurora
   ident a = 3;   #- Becomes [0, 0, 0, 0, 0, 0, 0, 3] → interpreted as unsigned 64-bit integer (3)
   
```

2. **Values = 8 bytes**: Used directly
   ```aurora
   ident a = 1_000;   #- [0, 0, 0, 0, 0, 0, 3, 232] → interpreted as unsigned 64-bit integer (1000)
   
```

3. **Tapes (arrays)**: Values are stored directly as bytes, padded to 8 bytes for arithmetic
   ```aurora
   ident a = [1, 2, 3];   #- 8 bytes: [0, 0, 0, 0, 0, 1, 2, 3]
   #- Interpreted as unsigned 64-bit integer: bytes are right-aligned, so this becomes 0x0000000000010203
   #- For arithmetic, it's treated as a single 8-byte value
   
```

### Examples

```aurora
#- Simple arithmetic
ident a = 10;
ident b = 20;
printb a + b;   #- [0 0 0 0 0 0 0 30]

#- A tape is read as one unsigned number, whatever it was written as: [1, 1] is
#- the bytes 1 and 1, which spell 257, so this is 260.
ident x = 3;
ident y = [1, 1];
printb x + y;   #- [0 0 0 0 0 0 1 4]
printd x + y;   #- 260

#- Booleans in arithmetic
ident t = true;   #- 8 bytes: [0, 0, 0, 0, 0, 0, 0, 1]
ident f = false;   #- 8 bytes: [0, 0, 0, 0, 0, 0, 0, 0]
printb true + 1;   #- [0, 0, 0, 0, 0, 0, 0, 1] + [0, 0, 0, 0, 0, 0, 0, 1] = 2
printb false + 1;   #- [0, 0, 0, 0, 0, 0, 0, 0] + [0, 0, 0, 0, 0, 0, 0, 1] = 1
```

### No signs, and no negative numbers

A value is a run of bytes, and a byte runs from 0 to 255. Nothing is set aside to carry a
sign — there is no sign bit, no signed type, no negative literal. The language cannot write
a negative number and no operation can produce one.

So `-x` is not "negative x". It is x taken away from zero, wrapping at the tape width:

```aurora
printd -5;   #- 18446744073709551611, which is 2^64 - 5 at the default width
printb -1;   #- [255 255 255 255 255 255 255 255]
```

The width decides how far it wraps: the same `printd -5;` writes `251` under
`--tape-size 1`, which is 2^8 - 5.

That value stands in for -5 wherever wrapping is the whole story:

```aurora
printd -5 + 5;   #- 0 — it comes back
printd 1 - 2 + 1;   #- 0 — the wrap in the middle cancels
```

and stops standing in for it the moment an operation has to know the sign:

```aurora
printd -5 bigger 5;   #- 1, true: 2^64 - 5 is an enormous number, not a small one
printd -5 smaller 5;   #- 0
printd -5 / 2;   #- 9223372036854775805 — half of an enormous number, not -2
```

Comparison and division read the bytes as unsigned, like everything else does. A program that
needs negative quantities has to carry the sign itself: a second value holding it, or an
offset agreed across the program (store `n + 1000`, read it back as `value - 1000`).

This is the same reasoning as the one behind [the neutral value](#the-neutral-value) and
booleans-as-tapes: there is one kind of value, and reading is the program's business.

### Important Notes

- **Arithmetic reads a tape as an unsigned big-endian integer**: `+`, `-`, `*`, `/` and `^` interpret the bytes as one unsigned number of the tape's width — 64 bits at the default 8 bytes, 8 bits at `--tape-size 1`, up to 256 bits at 32. There is no signed reading, at any width
- **Tapes store values as direct bytes**: When you write `[1, 2, 3]`, the values are stored directly as bytes in an 8-byte array: `[0, 0, 0, 0, 0, 1, 2, 3]`
- **For arithmetic, a tape is one value**: the whole run of bytes is interpreted as a single unsigned integer, whatever the tape width
- **Booleans need no special rule**: `true` is a tape holding 1 and `false` a tape of zeros, so `true + 1 = 2` and `false + 1 = 1` fall out of ordinary arithmetic
- **Arithmetic wraps at the tape width**: with `tape_size = 1`, `255 + 1` is `0`
- **This is a design decision**: Aurora prioritizes simplicity and the untyped philosophy over strict type safety
