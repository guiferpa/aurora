# Changelog

All notable changes and release notes for Aurora are documented here.

---

## Unreleased

### The EVM backend carries the whole language

The point of the backend is that **the same program answers the same thing on a chain and off
it**. Until now it answered for arithmetic across a dispatcher, and everything else compiled,
deployed, and did something else. Now nothing a program can write is missing from it.

- **A branch is an expression on a chain.** Both arms leave one value, so whoever is under an
  `if` finds a value without knowing which way the program went.

- **Comparisons, `and`/`or` and `^`.** A comparison answers with a tape like any other value.
  `and` and `or` are the logical ones and not the bitwise ones — `2 and 1` is true, which a
  bitwise AND would answer zero.

- **A scope calls another, and can call itself.**

  ```
  ident down = defer { if feed(0) equals 0 { 0; } else { down(feed(0) - 1) + 1; }; };
  ```

  It is a jump inside one contract, deliberately not a message call: a message call to your own
  contract is a transaction against yourself, and paying for that would make a call inside a
  program cost what a call between contracts costs. Each activation gets its own run of memory,
  so a scope can be entered while it is already running — recursion and two scopes calling each
  other both work, and neither needed anything written for it.

- **`shape` reaches the chain.** A shape is not a new kind of value: `Point{10, 20}` is two
  tapes laid end to end, so on a chain it is one word with the first tape at the far end. That
  is also its ceiling — a run past thirty-two bytes does not fit, which is four fields at the
  default tape.

- **`pull`, `push`, `head` and `tail`.** All four are defined by how much of a value means
  something, which off a chain is the length of a slice and on one is worked out from the word
  itself, without branching. When the values were written down none of that is needed: a tape
  literal is one shift and one or on a chain, however many values were written between the
  brackets.

### What a contract can no longer do quietly

Three things are refused at compile time rather than written as bytes that deploy and are
wrong. Each of them used to compile and produce a contract that did something else.

- A **shape whose run passes a word** — five fields at the default tape of eight.
- A **call to anything but a scope bound at the top of a program**. A scope written inside
  another is not written at all: the name it was bound to holds the neutral value.
- A **tape literal built out of values a program works out** — `[a, b]`. `pull t a` one value
  at a time is written.

### Fixes

- **A name bound inside a scope reached the chain with the wrong value.** The lowering decided
  which operands named values by a list of four arithmetic opcodes, so a name bound in a scope
  was not on it: `ident x = feed(0); x + feed(1);` answered 4 on a chain where the program
  answers 7. It answered rather than reverted, which is the quieter of the two ways to be
  wrong.

- **A contract was published cut short.** Memory offsets, jump targets and the runtime size
  were written in one byte and truncated past 255 — a program with three deferred scopes
  reached that, and the chain kept a contract that ended in the middle of an instruction. A
  scope past byte 255 was jumped to at the wrong address, and the ninth name in a contract was
  given the address of the first, so two names shared one place and each wrote over the other.

- **A scope written inside another ran on the way past.**
  `ident outer = defer { ident inner = defer { 1; }; 2; };` answered 1 where the program
  answers 2.

- **A call short of what a scope reads read the last call's values.** Reading past what was
  applied answers zeros — free on the way in from a transaction, since the calldata gives zeros
  past its end, and not free in memory an earlier call already used.

### The compiler says more

- **A print says where it was written.** The three print builtins were the only nodes without
  the token they came from, so `printb writes a log, and a chain has nowhere to put one`
  pointed at line 0.

- **A gap nobody wrote down is no longer silent.** The backend passed over any instruction it
  had not been told about, which was harmless while the list of missing features was long and
  stops being harmless now that it is empty.

### Inside

- **The IR says how long a run is.** `OpField` carried only an index, which is enough for a
  reader that holds a run as a slice and counts from the left. It is not enough for one that
  keeps a run as a value of fixed width, and it is not something that reader can work out — the
  run arrives under a name, as a value applied to a scope, or as a field of another run.

- The differential harness (`hosting/cli/evm_harness_test.go`) compiles the same source,
  deploys it to an EVM in memory, calls it, and compares against the evaluator. Everything
  above is pinned there.

---

## v0.7.0-alpha — 2026-08-22

### The language

- **Reading past what was applied answers with zeros.** A scope still has no signature — applying is handing a vector of values to a block, and `feed(n)` reads a position of it — but a position nothing was applied to now answers with a tape of zeros instead of wrapping around to one that was.

  ```
  ident sum = defer { feed(0) + feed(1); };
  printd sum(5);     # 5, not 10
  ```

  Wrapping turned a value that was never sent into a repeat of one that was, so a forgotten argument came back as a plausible number. Zeros is the answer the language already gave for the same shape of question — reading a field past the end of a shape gives the neutral value — so `feed` stops being the exception.

  It also made the answer depend on how many values the caller sent, which no backend can know without carrying the count, and the EVM one never did: the same program answered 10 off the chain and 5 on it. There is a differential case pinning it now.

  **Breaking:** a scope reading further than the call applies answers differently.

- **A block can promise the shape it answers with.** `as` is a claim the compiler believes and cannot check; `returns` is the other end, and a block that does not keep the promise does not compile.

  ```
  shape Result { failed, value };

  ident divide = defer {
    if feed(1) equals 0 { Result{1, 0}; } else { Result{0, feed(0) / feed(1)}; };
  } returns Result;
  ```

  What it buys is the other end: a call to a scope that promised has a shape, so `as` disappears from the place it was repeated once per call. The promise is never required, and never will be.

- **A shape's name starts with a capital letter.** `shape person { name };` is refused where it is written.

  A shape's name is the one name in a file that is not a value, and the two forms sit one character apart: `Point{1, 2}` builds a run of tapes, `point(1, 2)` feeds a scope. The braces already said which; the capital says it first, at the declaration. Fields keep whatever case their author likes.

  **Breaking:** a shape declared in lower case no longer compiles.

### The compiler says more

- **A call that applies fewer values than the scope reads is told about it**, at the line the call was written on.

  ```
  main.ar:2:8: warning: sum reads 2 positions and 1 were applied: feed(1) answers with a tape of zeros
  ```

  A body says how many positions it can address — the highest index it feeds, plus one — and that is known where it is written. Applying fewer is not an error and will not become one; it is said out loud because the answer is silent. It stays quiet when it is not sure: a name bound twice, an alias, a name answered by an `if`.

- **Every warning names the line it came from.** `aurora build` said which feature it could not carry and left the person to find where. Instructions carry an origin now, so the warning points at the first line that used it, in the form an editor follows.

- **A name bound inside a scope answers the same on a chain and off it.** It compiled to an `MSTORE` with nothing under it, so the contract answered a different number than the program did — `ident x = feed(0); x + feed(1);` applied to 3 and 4 answered 7 off the chain and 4 on it. The lowering decided which operands named values from the opcode, and a binding was not on that list.

### The editor

- **Go to definition**, including across a module boundary, and **rename** a name everywhere it is written in a file.
- **Completion** offers a module's shapes, and the fields of an imported shape after the dot.
- **The editor hears what the compiler says**, not only whether the document parses: a short call, a scope holding more deferred scopes than a tape can name, an `assert` outside `aurora test`.

### Inside

The IR was reshaped so that it describes the program rather than one way of running it, which is what lets a backend read it as commands instead of guessing. An operand says what it is; a construction of n values is one instruction rather than a chain; a call carries the values applied to it; a value written down goes into the instruction that uses it. The reasoning is in [rfcs/ir.md](rfcs/ir.md).

Nothing there changes what a program answers, except where it fixed something that was answering wrong.

---

## v0.6.0-alpha — 2026-08-20

### Modules

- **A program can be more than one file.** A module is a file, `use` brings one in under a name of your choosing, and that name is the only way to reach what is inside it.

  ```
  # src/geometry.ar
  ident area = defer { feed(0) * feed(1); };

  # src/main.ar
  use geometry as g;
  printd g.area(30, 20);
  ```

  The path is written from the source root without the extension, and it is the module's name everywhere: `use a/b/c as x;` is the module `a/b/c` whoever writes the line. There is no relative form, and a directory is not a module.

  Everything is answered while compiling. A module that is not there, a circle between modules, and a name a module does not have are all refused before the program runs, where they were written — which is what the previous attempt at modules never did, and why it was rolled back in v0.2.0-alpha.

  A module runs once, before whoever needs it, however many files import it. What it offers is what it binds with `ident` at the top of its file — a tape or a deferred scope. A `struct` does not cross a module, and an import is not passed on: if `main` uses `a` and `a` uses `b`, `main` writes its own line for `b`, so a name used in a file has its origin declared in that file.

  Full reference: [docs/modules.md](docs/modules.md). Why it has this shape, including what was refused: [docs/module_system_design.md](docs/module_system_design.md).

  **Breaking:** `use` is a keyword again, so it is no longer a valid identifier. It was one until namespaces were rolled back, and it comes back with them.

- **`source_root` says where module names resolve from**, under `[project]`, and `src` when a project says nothing.

  ```toml
  [project]
    source_root = "lib"
  ```

  It is relative to the directory the command was run in, with a manifest or without — one rule with no exception, and the price of it is that a project with modules is run from its own root. When a module is not found there, the error says where it looked. Like `tape_size` it belongs to the project rather than to a profile, and a profile carrying it is refused.

- **`run`, `build` and `test` compile every module.** A test and the file it tests are one module written in two files, so a test reaches what its source declared — including a module the source brought in — and may name modules of its own.

- **The REPL brings a module in.** `use geometry as g;` reads the file and the lines after it can use what is inside. A module is a program, so it runs once per session: importing it again is a use of what is already there.

### The editor and the playground

- **A module is answered for as a module.** Hover on an alias says which module it names instead of calling it an identifier, and hover on a name reached through one says which module it comes from and what it is.

- **The editor follows an import.** A module that is not there, a circle, and a name a module does not have are underlined where they were written, and what a module declared is offered after the dot — which used to offer every keyword in the language, the one thing that certainly cannot go there.

  It reads the editor's buffers before the disk, so a module you are editing answers with what you are seeing rather than with what was last saved.

  It does not go to a definition in another file, and it only notices a change in a file you have open.

### The EVM backend

- **A build says what the chain will not do.** A contract using something the builder does not write compiled, deployed and did nothing, and the only way to find out was on chain. `aurora build` now names each of those, once, in the order the program uses them — saying `by decision` for a print or an assert, which have nowhere to go on a chain, and `yet` for everything else, which is being written in slices. A program the builder writes whole is built without a word.

- **A value on chain is as wide as the tape says.** A result leaving a call was cut back to the width the project declared, the way the evaluator does it, so the same program answers the same thing on a chain and off it.

### Fixes

- **A test can use a struct its source declared.** The two files are one scope, and they were being parsed as if they were two.

### Removed

- **`-l` is gone.** It printed what each phase produced, and what it showed is what a test checks: every phase has tests that read what it answers with, and those do not depend on someone running a command and looking. If it comes back, it comes back as a port, the way printing did.

  **Breaking:** `aurora run -l lexer,parser,emitter` is no longer a flag.

- **`-r` is gone.** It turned on a reader nobody read: the evaluator held it, nothing ever asked it for a line, and the flag had never done anything. When a program needs to be given something, the way in will be a port.

  **Breaking:** `aurora run -r` is no longer a flag.

---

## v0.5.0-alpha — 2026-08-16

### Manifest

- **`tape_size` moved from a profile to the project.** It is not a path or a setting: it is the dialect the source is written in. At one byte `255 + 1` is `0` and `"Gui"` does not compile; at eight both answer otherwise.

  ```toml
  [project]
    tape_size = 16
  ```

  As a profile field it made one file mean two things inside the same project, and left everything reading the project as a whole with no width to answer for.

  **Breaking:** a manifest still carrying `tape_size` inside a profile is refused, naming where it sits and where it goes. Reading it and doing nothing would leave a project pinned to one byte compiling at eight without a word.

### Tooling

- **The editor reads a file in its project's width.** The language server parsed every document at the default eight, so in a project written at sixteen it underlined `"Guilherme"` as too long for a tape while `aurora run` compiled it without complaint. The same for a number that does not fit and a tape literal with too many items.

  The width is found by walking up from the file, and re-read when the manifest changes — widening a project reaches the next keystroke, not the next restart.

- **A file compiles the same named by path or by profile.** `aurora run src/main.ar` read the file at eight bytes while `aurora run` read the same file at the project's width. A loose file now takes the width of the project it sits in; a file outside any project still needs no manifest at all.

- **`aurora init`** writes `tape_size` under `[project]`.

- **The playground answers what a word is and what can follow it.** Hover shows what the compiler knows — a keyword's description, an identifier's value, which tape of a run a field reads — and completion offers the keywords as snippets, the structs declared in the document, and, right after a dot, the fields of that struct and nothing else. The same answers the language server gives an editor, in the page.

- **The playground colours the code and marks what does not compile**, using the same analyses the language server answers editors with — the colours are the lexer's semantic tokens and the marks are the parser's diagnostics. Nothing about the language is written a second time in JavaScript, so a keyword added to Aurora shows up in the page without anyone remembering to add it twice. Changing the tape size re-marks the document, since the width decides what fits.

  The page also registers the language properly now: `monaco.languages.register` sat outside the editor loader's callback, where Monaco does not exist yet, so the language was never registered at all.

- **The playground picks its tape size**, and starts at 32 bytes rather than the language's 8. The page opens on `printc "Hello, World!";` — thirteen bytes, which does not fit an eight-byte tape, so the first thing it used to do was refuse its own example. Changing the width re-runs the program at once: the output on screen is never the output of a different width than the one shown.

---

## v0.4.0-alpha — 2026-08-15

### Language

- **Text is a tape, and reels are gone.** `"hi"` is the tape holding its bytes, right aligned like every other value — one more way of writing a tape, next to `1`, `0x2a`, `[1, 2]` and `true`.

  ```aurora
  printb "hi";   #- [0 0 0 0 0 0 104 105]
  printd "hi";   #- 26729
  printd "a" equals 97;   #- 1 — the same tape, not a conversion
  ```

  Text used to be a **reel**: one tape per character, so `"hi"` was two tapes and `"Gui"` was 96 bytes at `--tape-size 32` — a cost that grew exactly when the tape got wider. It was also the one value in the language that was not a tape. Now `"a"` and `97` are the same value rather than a coincidence of one-character strings, arithmetic on text needs no rule of its own, and text fits in a struct field, which a reel never could.

  A tape holds `tape_size` bytes, so that is how much text fits: nine bytes at the default eight is a compile error where the text was written, the same rule a number literal that does not fit follows. Text longer than a tape has no answer yet — that needs building a value at runtime and reading a position out of it, neither of which exists.

  **Breaking:** `printc` of a number is its bytes as UTF-8 rather than the character that number names. `printc 44` is still `,`; a character above ASCII is written as text now, not as its code point. `printd` of text is one number per tape rather than one per character.

- **An assert message is a literal, not a value.** It is written for whoever reads the result of a test, like a struct's field names are written for whoever reads the source, so it rides in the instruction as its bytes. That is also what keeps it longer than a tape, which a message usually is. **Breaking:** the message must be written as text, not computed.

- **`struct`**, a way of grouping values by naming the tapes of a run.

  ```aurora
  struct Point { x, y };

  ident area = defer {
    ident p = feed(0) as Point;
    p.x * p.y;
  };
  printd area(Point{10, 20});   #- 200
  ```

  Braces build the value, as in Go — which is also what tells a construction from applying values to a scope: `Point(1, 2)` and `greet(1, 2)` are the same shape, `Point{1, 2}` is not.

  A struct value is not a new kind of value: `Point{10, 20}` is two tapes laid end to end, which is exactly what a reel of two characters is — no header, no length, no tag. `Pair{97, 98} equals "ab"` is true, and two structs of the same width are the same value.

  `struct` and `as` are **directives for whoever writes the source**: they exist so the compiler can turn a name into an index, report a mistake where it was written, and tell the language server what is there. None of it reaches the IR or the binary, because the flow is static, the fields are positional and each one is exactly a tape wide. So the errors are the substance: a field the struct does not have, a value whose shape nothing declared, a construction that miscounts the fields, a struct name used as a value, and a reel of several characters in a field all stop the compilation — a field is one tape, and quietly keeping the `e` of `"Guilherme"` is exactly the wrong answer the directive is there to prevent. Reading a field past the end of a value does not — it gives the neutral value, as `head` and `feed` do.

  `as` names the shape where the compiler cannot see it, which is above all when a value crosses into a scope. It claims rather than checks: there is nothing in a run of bytes to check against.

  Not supported on chain: like `if`, `call` and the prints, the two new instructions produce no EVM bytecode. At `--tape-size 32` a struct of N fields is exactly the ABI encoding of a tuple of N words, which is where that support would start.

### Tooling

- **The language server learned structs, and learned to type for you.** Completing after a `.` offers that struct's fields and nothing else; hover lists a struct's fields and says which tape a field reads; and `struct`, `as`, a struct's name and its field names are coloured for what they are. All of it reads the tokens rather than the tree, because a document being edited hardly ever parses — and typing `p.` is exactly when completion is wanted, which is also why the dot is now declared as a trigger character rather than waiting to be prodded.

  Keywords come back as the forms that have a shape to get wrong — where the semicolon goes, that a `branch` ends in a fallback with no test, that the index of `head` is a literal number — and a declared struct comes back as a way of building one, with its own field names as the places to fill in (`Point{${1:x}, ${2:y}}`). Snippets go only to a client that said it expands them; to anyone else the placeholders would be literal text in the buffer.
- **The REPL writes the tape, not the decimal.** A value is a run of bytes, and the decimal was one of the three readings a program can ask for — showing it hid the value behind a choice the line never made. `byteutil.Encode` existed for that one caller and is gone with it.
- **[docs/roadmap.md](docs/roadmap.md)**, a working list of what the language does not do yet: values that outlive a tape (text of any length, a computed index, building a value while the program runs — one piece of work, and the largest), closures and calling a scope held in a value, the module resolver and loader, and the silent gaps in the EVM backend. Every entry was checked against the compiler.
- `aurora init` writes `tape_size = 16` in the manifest it generates, since the greeting it also writes is eleven bytes. Its test compares the greeting to the text itself now, rather than to the number of its last character.

### Documentation

- **Every Aurora snippet in the docs runs.** They are checked by being executed rather than by being read, and every number in them was taken off a run. That turned up a `defer` example that never demonstrated what it claimed, arithmetic examples whose comments said "the result depends on how bytes are interpreted" rather than the result, and a negation example printing the same line twice with one commented as if it had been run at another tape width.
- **The language has no signs**, said plainly for the first time: a byte runs from 0 to 255, nothing marks a value negative, and `-x` is x taken away from zero. `-5 + 5` is 0 and `-5 bigger 5` is true, which is the part worth knowing. Two claims that contradicted it are gone — `head` and `tail` were documented as taking negative indices, which never parsed.
- **[docs/state_management.md](docs/state_management.md) says at the top that it is a proposal**, as [docs/module_system_design.md](docs/module_system_design.md) already did. It documents a `state` keyword and `name!` functions that have no token, no node and no opcode: the compiler rejects every example in it.

### Fixed

- **`feed` hands a run of tapes over whole.** It narrowed every read to a single tape, which cut a struct down to its last field on the way into a scope. The narrowing that command-line arguments need already happens where they enter, in `NewEnviron`.

---

## v0.3.2-alpha — 2026-08-14

### Fixed

- **`and` binds tighter than `or`.** The two shared one precedence level and recursed to the right, so `a and b or c` read as `a and (b or c)` and `false and true or true` answered false — every language that gives `and` the tighter binding answers true there, reading it as `(false and true) or true`. Each level now loops, so a chain groups to the left as well; that one is invisible in the answer, since both operations are associative. Comparison still binds tighter than both, which keeps `n bigger 18 and n smaller 65` reading as one range.

---

## v0.3.1-alpha — 2026-08-14

### Fixed

- **Multiplication and division group to the left**, like addition and subtraction. They recursed to the right, so a chain answered for a different expression than the one written: `20 / 5 / 2` gave 10 instead of 2, and `4 / 2 * 6` gave 0 instead of 12. Multiplication on its own hid it — it is associative, so only a chain mixing it with division tells the two groupings apart, and `n - (n / d) * d`, the way to take a remainder without a `%` operator, is exactly that. Exponentiation still groups to the right, which is the convention: `2 ^ 3 ^ 2` is 512.
- **Negating a value negates it.** The emitter took the unary expression and emitted only its operand, dropping the operator, so `-5` was 5 and `10 + -5` was 15. A tape is unsigned, so `-x` is the value taken away from zero, wrapping at the tape width — the same tape `0 - x` gives, which is what the compiler now emits.

---

## v0.3.0-alpha — 2026-08-14

### Language

- **Printing is three builtins instead of two.** `print` and `echo` became `printb`, `printd` and `printc`, and the suffix names the reading: bytes, decimal, character. A value is a tape and never a number or a string on its own, so reading one is a choice the program makes — the old pair named the act of printing rather than the reading, and left no room for the third. `printd` is new: reading a tape as a number meant counting bytes by hand.

  ```
  printb 44;   [0 0 0 0 0 0 0 44]
  printd 44;   44
  printc 44;   ,
  ```

  **Breaking:** `print` and `echo` no longer parse. `print x` becomes `printb x`, `echo x` becomes `printc x`.
- **Text outside ASCII survives.** `echo` read one byte per byte and kept it only when it landed in printable ASCII, so every accented character was dropped — `echo "café"` wrote `caf`. `printc` reads the number the whole tape holds and writes it as UTF-8, which is what makes 233 an é and 514 an Ȃ. A reel is read tape by tape, so a character is never cut in half.

### Added

- **`aurora test`**, a command for the test files of a project. It reports every assertion, not only the ones that failed, and exits non-zero when something breaks. A test belongs to the source file of the same name — `greeting.test.ar` tests `greeting.ar`, which is evaluated first so the test sees what it declared, which is how a test reaches code before there is a module system. With a profile it searches from the directory of the profile's source down to the leaves; with a path it runs that file alone. See [docs/testing.md](docs/testing.md).
- **`aurora init` starts a project, not just a manifest.** It writes the layout the manifest describes — `src/main.ar` and `src/main.test.ar` — so a new project runs and passes its tests straight away. The program greets you with `Abidu abide`: Aurora is the author's daughter, and that is what she was saying at one year old.
- **`aurora build` says what it produced**: where the binary went, how many instructions it holds, how large it is and the tape width it was compiled at. A build that succeeded used to print nothing at all, which left no way to tell it apart from one that found nothing to do — and the output path often comes from a profile rather than from the command line.

  ```
  ✨ src/main.ar → bin/main
     17 instructions, 52 bytes, 8-byte tapes
  ```
- **`assert` now belongs to `aurora test`.** Under `aurora run` each one is reported as a warning carrying its file, line and column, and is not checked — so a program holding an assertion no longer fails because of it. `run` used to exit 3 in that case.

### Fixed

- **A deferred scope is a tape.** Its value was the hex key of its own storage — 16 bytes of ASCII text that ignored `tape_size` — so `ident b = defer {};` showed a row of zeros where `ident a = {};` showed one, `b + 1` produced ASCII digits, and a defer stayed 16 bytes wide even with `--tape-size 1`. It is the scope's index as an ordinary tape now. A consequence the language owns: a value equal to an index *is* that reference, the same way `true` is `1`.
- **The compiler warns when a scope holds more deferred scopes than its tape can name** (256 at one byte), since the index wraps and a call would reach a different scope. It is a warning rather than an error: the count is static, and a running program is never stopped by it.
- **`feed(n)` always answers with a tape.** It read the vector directly, so an index past the end gave nothing at all — `printb feed(0)` with no values applied printed `[]`, and the REPL said `unknown byte sequence to encode`. The index now wraps around the length of the vector, as the design always described, and an empty vector gives a tape of zeros.

---

## v0.2.0-alpha — 2026-08-14

### Language

- **Every value is a tape of the same width.** Booleans are no longer a one-byte exception: `true` is a tape holding 1, indistinguishable from the number 1.
- **The `nothing` keyword was removed.** Once every value became a tape, it was byte-for-byte the same as `false`, and one value does not need two names. Everywhere it served as the neutral default — an empty block, an `if` without `else`, the value of a binding, a scope returning no value — the language now produces `false`. **Breaking:** source that uses `nothing` now reads it as an ordinary identifier, which will not resolve.
- **Tape width is a compiler parameter**: `tape_size` in `aurora.toml` or `--tape-size` on `build`, `run` and `repl` (1 to 32 bytes, default 8, flag wins over the manifest). A literal that does not fit is a compile-time error instead of being truncated silently.
- Arithmetic now wraps at the tape width (`255 + 1` is `0` with one-byte tapes) and runs on 256-bit integers, which also fixes `^` losing precision above 2^53.
- The EVM backend emits `PUSH<n>` for the configured width, from `PUSH1` to `PUSH32`.
- The REPL prints values as numbers: with no boolean type of its own, there is no `true`/`false` to show. It also prints the value of the line that was typed, instead of every intermediate value of the expression in map order.
- **Tape operations run.** `pull`, `push`, `head` and `tail` were parsed and emitted but never evaluated, which also left tape literals like `[1, 2, 3]` broken. A tape is a shift register: `pull` shifts left with the value entering at the right, `push` shifts right with the value entering at the left, and what reaches the far end is discarded.

- **Namespaces were removed.** `use ... as ...`, the `::` operator and the directory-as-namespace rule are gone. The rule meant that compiling one file compiled every `.ar` file next to it, so independent programs in one folder collided; and the half that was implemented never resolved a symbol anyway — the emitter ignored `use` entirely. The unit of compilation is now the file. A program spanning several files is not supported until the module system is designed: see [docs/module_system_design.md](docs/module_system_design.md). **Breaking:** `use` and `as` are ordinary identifiers now, and `::` is two colons.
- **`arguments(n)` is now `feed(n)`.** Aurora has no functions, so it has no parameters — and "argument" only means something against "parameter". `feed` says what happens: values are fed to a scope, and `feed(n)` reads the nth one. The reasoning is recorded in [docs/grammar.md](docs/grammar.md#feed-formerly-arguments). **Breaking:** `arguments` no longer parses as a builtin.
- **`echo` prints text again in the CLI.** `aurora run` gave `print` and `echo` the same writer, so `echo` showed raw bytes; the evaluator always took two writers for exactly this reason, and the REPL was already correct.

### CLI

- **`run` and `build` take one positional argument**: nothing means the `main` profile, a name means that profile, and a path ending in `.ar` means that file. Running a loose file no longer requires a project to exist anywhere above it — which is what learning the language mostly looks like. **Breaking:** `-s/--source` is gone from `run`, `build` and `deploy`, and `-p/--profile` from `run` and `build`.
- **A failed command prints one thing.** Cobra printed the error and the entire usage block, and the CLI then printed the error again in colour; usage now belongs to `aurora help` and `--help`. Errors also go to stderr rather than stdout.

### Fixed

- **Output now follows source order** ([#11](https://github.com/guiferpa/aurora/issues/11)). The playground ran the whole program and then walked the temp map, so every `print` appeared before every value, and the values among themselves came out in no order at all. Both the playground and the REPL evaluate one top-level expression at a time and report its value before moving on.
- A scope ending in a binding (`{ ident a = 1; }`) returned the emitter's fallback for an unrecognised node instead of the binding's value.

### Tooling

- `aurorals`, the language server, returned with semantic tokens, diagnostics, hover and completion. See [docs/lsp.md](docs/lsp.md).
- The REPL gained command history (`~/.aurora/history`) and line editing with the arrow keys.

---

## Alpha (v0.13.1)

First alpha release: Aurora compiles source code to EVM bytecode. Use for study and experimentation only; **do not use in production**.

### Highlights

- **Pipeline:** Lexer → Parser → Emitter → Lowering (builder/evm) → Builder EVM → bytecode. See [docs/compiler_pipeline_and_lowering.md](docs/compiler_pipeline_and_lowering.md).
- **Lowering:** Reordering for left-associative Sub/Div and RPN stack order; covered by tests.
- **CLI:** `aurora init`, `aurora run`, `aurora build`, `aurora version`, `aurora help`, `aurora repl`, `aurora deploy`, `aurora call`.
- **Manifest:** `aurora.toml` with `[project]` and `[profiles]`; optional `rpc` and `privkey` for deploy/call. See [docs/manifest.md](docs/manifest.md).
- **Language:** Untyped; tapes (arrays), reels (strings), arithmetic; `if`/block expressions; deferred callables; `print`, `echo`, `assert`, `arguments`.

### Known limitations (alpha)

- **Emitter** has no direct unit tests (covered indirectly via evaluator and builder/evm).
- **Parser** and **builder/evm** coverage is partial; some grammar paths (e.g. tape operations, list built-ins) are not fully exercised.
- **Built-ins for list:** Not yet defined (see [docs/grammar.md](docs/grammar.md) demand list).
- **Deploy/call:** Documented and implemented; test on Sepolia or local chain for your use case. Do not commit `privkey` in `aurora.toml`; use env/secrets.
- **Bytecode compatibility:** Bytecode is generated for the documented pipeline; validate on your target chain (e.g. Sepolia) before relying on it.

### Reporting

- Bugs and feature requests: [GitHub Issues](https://github.com/guiferpa/aurora/issues).
- Project and docs: [README](README.md), [manifest](docs/manifest.md), [grammar](docs/grammar.md).
