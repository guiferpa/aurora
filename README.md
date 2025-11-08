# aurora

[![Go Reference](https://pkg.go.dev/badge/github.com/guiferpa/aurora.svg)](https://pkg.go.dev/github.com/guiferpa/aurora)
[![Last commit](https://img.shields.io/github/last-commit/guiferpa/aurora)](https://img.shields.io/github/last-commit/guiferpa/aurora)
[![Go Report Card](https://goreportcard.com/badge/github.com/guiferpa/aurora)](https://goreportcard.com/report/github.com/guiferpa/aurora)
![Pipeline workflow](https://github.com/guiferpa/aurora/actions/workflows/pipeline.yml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/guiferpa/aurora/badge.svg?branch=main)](https://coveralls.io/github/guiferpa/aurora?branch=main)

🌌 Aurora's just for studying programming language concepts

> ⚠ Don't use it to develop something that'll go to production environment

## Summary

- [Get started](#get-started)
  - [Install CLI](#install-cli)
  - [Using REPL mode](#using-repl-mode)
  - [Execute from file](#execute-from-file)
  - [Writing some code](#writing-some-code)
- [Language Design](#language-design)
  - [Untyped Philosophy](#untyped-philosophy)
  - [Tapes (Arrays)](#tapes-arrays)
  - [Reels (Strings)](#reels-strings)
  - [Arithmetic Operations](#arithmetic-operations)
- [Try it out](#try-it-out)
  - [Playground](#playground)
- [Extra options](#extra-options)
  - [Debug flag](#debug-flag)

## Get started

### Install CLI
```sh
go install -v github.com/guiferpa/aurora/cmd/aurora@HEAD
```
> 🎈 So far there's no an easier way to download aurora binary. Use Go to install, it's the better way for while.

### Using REPL mode

```sh
aurora repl
```

```java
>> ident a = 1_000;
>> a + 1;
= 1001
```

### Execute from file

#### Create aurora source code file

```java
ident result = 10 * 20;
print result + 1;
```

#### Execute file

```sh
aurora run ./<file>.ar
```

#### That's the output from evaluator
```java
[0 0 0 0 0 0 0 201]
```

### Writing some code
> 🎈 Unfortunately, this project there's are not contributors enough to make this doc better but be my guest to discovery how to write some code looking at [examples folder](/examples).

## Language Design

### Untyped Philosophy

Aurora is an **untyped language** where everything is fundamentally an array of bytes. There are no type distinctions at the language level - numbers, booleans, tapes (arrays), and functions are all represented as byte arrays.

#### Core Concept

In Aurora, all values are byte arrays:
- `ident a = 3;` → 8 bytes representing the number 3
- `ident b = [1, 2];` → 8 bytes: `[0, 0, 0, 0, 0, 0, 1, 2]` (tape values stored as direct bytes, right-aligned)
- `ident c = [];` → 8 bytes: `[0, 0, 0, 0, 0, 0, 0, 0]` (empty tape, all zeros)
- `ident d = true;` → 1 byte representing boolean true

This means that `3` (8 bytes) and `[1, 2]` (8 bytes) are just different representations of bytes. The language doesn't enforce type safety - it's up to the operations to interpret the bytes correctly.

#### Example

```javascript
ident a = 3;        // 8 bytes: [0, 0, 0, 0, 0, 0, 0, 3]
ident b = [1, 2];   // 8 bytes: [0, 0, 0, 0, 0, 0, 1, 2] (tape values as direct bytes, right-aligned)
ident c = [];       // 8 bytes: [0, 0, 0, 0, 0, 0, 0, 0] (empty tape)
ident d = true;     // 1 byte: [1]
```

Note: Tapes (arrays) store values directly as bytes, not as unsigned 64-bit integers. So `[1, 2, 3]` is represented as `[0, 0, 0, 0, 0, 1, 2, 3]` (8 bytes, right-aligned), not as 24 bytes (3 × 8 bytes). All tapes are exactly 8 bytes. If an operation would result in more than 8 bytes, an error is raised.

### Tapes (Arrays)

Tapes are just a more declarative way to create 8-byte arrays in Aurora. They provide a convenient syntax for specifying byte values directly, but fundamentally they're the same as any other 8-byte value in the language.

#### What are Tapes?

Tapes use the bracket syntax `[value1, value2, ...]` to create an 8-byte array where values are stored as direct bytes (right-aligned). This is simply syntactic sugar - under the hood, tapes are just 8-byte arrays like any other value in Aurora.

#### Creating Tapes

```javascript
// Empty tape - creates 8 bytes all zeros
ident a = [];  // [0, 0, 0, 0, 0, 0, 0, 0]

// Tape with values - values stored as direct bytes (right-aligned)
ident b = [1, 2, 3];        // [0, 0, 0, 0, 0, 1, 2, 3]
ident c = [0, 0, 0, 0, 0, 244, 254];  // [0, 0, 0, 0, 0, 244, 254]

// Equivalent ways to create 8 bytes of zeros
ident d = 0;   // [0, 0, 0, 0, 0, 0, 0, 0] (number 0)
ident e = [];  // [0, 0, 0, 0, 0, 0, 0, 0] (empty tape)
```

#### Key Points

- **Tapes are 8-byte arrays**: All tapes are exactly 8 bytes, regardless of how many values you specify
- **Values are stored as direct bytes**: Each value in `[1, 2, 3]` is stored as a single byte, not as an 8-byte unsigned 64-bit integer
- **Right-aligned storage**: Values are padded with zeros on the left (right-aligned)
- **Just syntactic sugar**: `[1, 2, 3]` is equivalent to creating an 8-byte array and using operations like `pull` to add values
- **Same as numbers**: `0` and `[]` both create the same 8-byte array of zeros

#### Tape Operations

Aurora provides several operations to work with tapes:

- **`pull tape value`**: Removes bytes from the beginning and adds the value at the end
- **`push tape value`**: Adds the value at the beginning and removes bytes from the end
- **`head tape n`**: Gets the first `n` bytes from the tape
- **`tail tape n`**: Gets all bytes after skipping the first `n` bytes

##### Index Behavior for `head` and `tail`

Since all tapes are exactly 8 bytes, the index `n` in `head` and `tail` operations is automatically applied modulo 8 to prevent boundary errors. This means:

- **Any index value works**: You can use any integer value, and it will be wrapped to the range 0-7
- **No boundary errors**: Operations never fail due to index out of bounds
- **Predictable behavior**: `head tape 10` is equivalent to `head tape 2` (since `10 % 8 = 2`)
- **Negative indices**: Negative indices are also handled correctly (e.g., `-1 % 8 = 7`)

**Examples:**
- `head [1, 2, 3, 4, 5, 6, 7, 8] 2` → Gets first 2 bytes: `[0, 0, 0, 0, 0, 0, 1, 2]`
- `head [1, 2, 3, 4, 5, 6, 7, 8] 10` → `10 % 8 = 2`, gets first 2 bytes: `[0, 0, 0, 0, 0, 0, 1, 2]`
- `head [1, 2, 3, 4, 5, 6, 7, 8] 8` → `8 % 8 = 0`, gets 0 bytes: `[0, 0, 0, 0, 0, 0, 0, 0]`
- `tail [1, 2, 3, 4, 5, 6, 7, 8] 2` → Skips first 2 bytes: `[0, 0, 3, 4, 5, 6, 7, 8]`
- `tail [1, 2, 3, 4, 5, 6, 7, 8] 18` → `18 % 8 = 2`, skips first 2 bytes: `[0, 0, 3, 4, 5, 6, 7, 8]`

#### Examples

```javascript
// Create a tape and manipulate it
ident a = [1, 2, 3];        // [0, 0, 0, 0, 0, 1, 2, 3]
ident b = pull a 4;         // Remove 1 byte, add 4: [0, 0, 0, 0, 1, 2, 3, 4]
ident c = push b 5;         // Add 5 at start, remove 1 byte: [5, 0, 0, 0, 1, 2, 3]

// Extract parts of a tape
ident d = head [1, 2, 3, 4, 5] 2;      // First 2 bytes: [0, 0, 0, 0, 0, 0, 1, 2]
ident e = tail [1, 2, 3, 4, 5] 2;      // Skip first 2 bytes: [0, 0, 0, 0, 0, 3, 4, 5]
ident f = head [1, 2, 3, 4, 5, 6, 7, 8] 10;  // 10 % 8 = 2, first 2 bytes: [0, 0, 0, 0, 0, 0, 1, 2]
ident g = tail [1, 2, 3, 4, 5, 6, 7, 8] 18;  // 18 % 8 = 2, skip first 2 bytes: [0, 0, 3, 4, 5, 6, 7, 8]

// Combine tapes (using pull)
ident h = pull [1, 2] [3, 4];  // Concatenate: [0, 0, 0, 0, 1, 2, 3, 4] (significant bytes concatenated)
```

Remember: Tapes are just a convenient way to create and work with 8-byte arrays. They're not a separate type - they're the same 8-byte arrays that Aurora uses for everything!

### Reels

Reels are arrays of tapes, where each character in a string is represented as a tape (8 bytes). In Aurora, strings are reels - a more structured way to work with sequences of characters.

#### What are Reels?

A reel is an array of tapes, where each tape represents a single character. This means:
- **Every string is a reel**: When you write `"hello"`, it becomes a reel with 5 tapes (one for each character)
- **Each character is a tape**: Each character is stored as an 8-byte tape
- **Reels can contain multiple tapes**: A string like `"hello"` contains 5 tapes concatenated together

#### Creating Strings (Reels)

```javascript
// Single character string (reel with one tape)
ident a = "a";  // One tape: [0, 0, 0, 0, 0, 0, 0, 97]

// Multi-character string (reel with multiple tapes)
ident greeting = "hello";  // Five tapes concatenated:
                           // [0,0,0,0,0,0,0,104] [0,0,0,0,0,0,0,101] [0,0,0,0,0,0,0,108] [0,0,0,0,0,0,0,108] [0,0,0,0,0,0,0,111]

// Empty string (reel with one empty tape)
ident empty = "";  // One empty tape: [0, 0, 0, 0, 0, 0, 0, 0]
```

#### Key Points

- **Strings are reels**: Every string literal `"text"` creates a reel (array of tapes)
- **Each character is a tape**: Each character in the string becomes an 8-byte tape
- **Reels store complete strings**: Unlike single tapes, reels preserve all characters in a string
- **Arithmetic works with reels**: When you do arithmetic with a string, it uses the last tape (last character) as the value

#### The `echo` Function

The `echo` function is designed to work with reels, printing all characters in sequence:

```javascript
ident greeting = "hello";
echo greeting;  // Prints: hello

ident single = "a";
echo single;    // Prints: a

ident empty = "";
echo empty;     // Prints: (empty line)
```

The `echo` function:
- **Detects reels**: If the value is a multiple of 8 bytes and greater than 8 bytes, it's treated as a reel
- **Prints each tape**: Iterates through each 8-byte tape and prints the character
- **Works with single tapes**: If the value is 8 bytes or less, it prints a single character

#### Examples

```javascript
// Basic string operations
ident greeting = "hello";
print greeting;  // Shows raw bytes: [0 0 0 0 0 0 0 104 0 0 0 0 0 0 0 101 ...]
echo greeting;   // Shows text: hello

// Arithmetic with strings (uses last character)
ident a = "a";   // Last tape: [0, 0, 0, 0, 0, 0, 0, 97] (ASCII 'a' = 97)
ident result = 1 + a;  // 1 + 97 = 98
echo result;     // Prints: b (ASCII 98)

// String with numbers
ident num_str = "123";
echo num_str;    // Prints: 123

// Empty string
ident empty = "";
echo empty;      // Prints: (empty line)
```

#### Relationship Between Reels and Tapes

- **Every reel is an array of tapes**: A reel is fundamentally a concatenation of multiple 8-byte tapes
- **Not every tape is a reel**: A single tape (8 bytes) is not a reel - it's just a tape
- **Reels preserve all characters**: Unlike single tapes which are limited to 8 bytes, reels can store multiple characters
- **Arithmetic uses the last tape**: When doing arithmetic with a reel, only the last tape (last character) is used

Remember: Strings in Aurora are reels - arrays of tapes where each character is a tape. This allows Aurora to work with text while maintaining its untyped, byte-array philosophy!

### Arithmetic Operations

Arithmetic operations in Aurora work by interpreting the first 8 bytes of a value as an unsigned 64-bit integer.

#### How It Works

1. **Values < 8 bytes**: Padded with zeros on the left (right-aligned)
   ```javascript
   ident a = 3;  // Becomes [0, 0, 0, 0, 0, 0, 0, 3] → interpreted as unsigned 64-bit integer (3)
   ```

2. **Values = 8 bytes**: Used directly
   ```javascript
   ident a = 1_000;  // [0, 0, 0, 0, 0, 0, 3, 232] → interpreted as unsigned 64-bit integer (1000)
   ```

3. **Tapes (arrays)**: Values are stored directly as bytes, padded to 8 bytes for arithmetic
   ```javascript
   ident a = [1, 2, 3];  // 8 bytes: [0, 0, 0, 0, 0, 1, 2, 3]
   // Interpreted as unsigned 64-bit integer: bytes are right-aligned, so this becomes 0x0000000000010203
   // For arithmetic, it's treated as a single 8-byte value
   ```

#### Examples

```javascript
// Simple arithmetic
ident a = 10;
ident b = 20;
print a + b;  // = 30

// All values are treated as unsigned 64-bit integers for arithmetic
ident x = 3;           // 8 bytes: [0, 0, 0, 0, 0, 0, 0, 3]
ident y = [1, 1];      // 8 bytes: [0, 0, 0, 0, 0, 1, 1] (tape as direct bytes)
print x + y;           // y interpreted as unsigned 64-bit integer from bytes [0, 0, 0, 0, 0, 1, 1]
                       // Result depends on how bytes are interpreted as unsigned 64-bit integer

// Booleans in arithmetic
ident t = true;        // 8 bytes: [0, 0, 0, 0, 0, 0, 0, 1]
ident f = false;       // 8 bytes: [0, 0, 0, 0, 0, 0, 0, 0]
print true + 1;        // [0, 0, 0, 0, 0, 0, 0, 1] + [0, 0, 0, 0, 0, 0, 0, 1] = 2
print false + 1;       // [0, 0, 0, 0, 0, 0, 0, 0] + [0, 0, 0, 0, 0, 0, 0, 1] = 1
```

#### Important Notes

- **Arithmetic operations always work on unsigned 64-bit integers**: Operations like `+`, `-`, `*`, `/`, `^` interpret values as unsigned 64-bit integers
- **Tapes store values as direct bytes**: When you write `[1, 2, 3]`, the values are stored directly as bytes in an 8-byte array: `[0, 0, 0, 0, 0, 1, 2, 3]`
- **For arithmetic, tapes are treated as 8-byte values**: The entire 8-byte array is interpreted as a single unsigned 64-bit integer for arithmetic operations
- **Booleans are padded to 8 bytes for arithmetic**: 
  - `true` → `[0, 0, 0, 0, 0, 0, 0, 1]` (8 bytes, value = 1)
  - `false` → `[0, 0, 0, 0, 0, 0, 0, 0]` (8 bytes, value = 0)
  - This means `true + 1 = 2` and `false + 1 = 1`
- **This is a design decision**: Aurora prioritizes simplicity and the untyped philosophy over strict type safety

## Try it out

### Playground
> 🚀 Feel free to try Aurora with [playground](https://guiferpa.github.io/aurora) built with WebAssembly + Go (Aurora source code)
<img width="942" alt="Screenshot 2025-01-12 at 12 27 41 AM" src="https://github.com/user-attachments/assets/51f073de-1fde-4a68-9cf8-b608b5b83032" />

## Extra options

```sh
aurora help

Usage:
  aurora [command]

Available Commands:
  build       Build binary from source code
  completion  Generate the autocompletion script for the specified shell
  eval        Evaluate aurora binary file built by build command
  help        Help about any command
  repl        Enter in Read-Eval-Print Loop mode
  run         Run program directly from source code
  version     Show toolbox version

Flags:
  -h, --help   help for aurora

Use "aurora [command] --help" for more information about a command.
```

### Debug flag

All commands it'll show deep dive in instructions and evaluating

`$ aurora repl --debug`
`$ aurora run --debug ...`
