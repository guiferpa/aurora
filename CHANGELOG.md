# Changelog

All notable changes and release notes for Aurora are documented here.

---

## Unreleased

### Language

- **Text is a tape, and reels are gone.** `"hi"` is the tape holding its bytes, right aligned like every other value — one more way of writing a tape, next to `1`, `0x2a`, `[1, 2]` and `true`.

  ```aurora
  printb "hi";          # [0 0 0 0 0 0 104 105]
  printd "hi";          # 26729
  printd "a" equals 97; # 1 — the same tape, not a conversion
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
  printd area(Point{10, 20});   # 200
  ```

  Braces build the value, as in Go — which is also what tells a construction from applying values to a scope: `Point(1, 2)` and `greet(1, 2)` are the same shape, `Point{1, 2}` is not.

  A struct value is not a new kind of value: `Point{10, 20}` is two tapes laid end to end, which is exactly what a reel of two characters is — no header, no length, no tag. `Pair{97, 98} equals "ab"` is true, and two structs of the same width are the same value.

  `struct` and `as` are **directives for whoever writes the source**: they exist so the compiler can turn a name into an index, report a mistake where it was written, and tell the language server what is there. None of it reaches the IR or the binary, because the flow is static, the fields are positional and each one is exactly a tape wide. So the errors are the substance: a field the struct does not have, a value whose shape nothing declared, a construction that miscounts the fields, a struct name used as a value, and a reel of several characters in a field all stop the compilation — a field is one tape, and quietly keeping the `e` of `"Guilherme"` is exactly the wrong answer the directive is there to prevent. Reading a field past the end of a value does not — it gives the neutral value, as `head` and `feed` do.

  `as` names the shape where the compiler cannot see it, which is above all when a value crosses into a scope. It claims rather than checks: there is nothing in a run of bytes to check against.

  Not supported on chain: like `if`, `call` and the prints, the two new instructions produce no EVM bytecode. At `--tape-size 32` a struct of N fields is exactly the ABI encoding of a tuple of N words, which is where that support would start.

### Tooling

- **The language server completes and expands.** A declared struct comes back as a way of building one, with its own field names as the places to fill in (`Point{${1:x}, ${2:y}}`), and the keywords expand into the forms that have a shape to get wrong — where the semicolon goes, that a `branch` ends in a fallback with no test, that the index of `head` is a literal number. Field completion after a `.` already worked; the dot is declared as a trigger character now, so the client asks for it on its own instead of waiting to be prodded. Snippets go only to a client that said it expands them — to anyone else the placeholders are literal text.
- **The REPL writes the tape, not the decimal.** A value is a run of bytes, and the decimal was one of the three readings a program can ask for — showing it hid the value behind a choice the line never made. `byteutil.Encode` existed for that one caller and is gone with it.
- **[docs/roadmap.md](docs/roadmap.md)**, a working list of what the language does not do yet: values that outlive a tape (text of any length, a computed index, building a value while the program runs — one piece of work, and the largest), closures and calling a scope held in a value, the module resolver and loader, and the silent gaps in the EVM backend. Every entry was checked against the compiler.
- `aurora init` writes `tape_size = 16` in the manifest it generates, since the greeting it also writes is eleven bytes. Its test compares the greeting to the text itself now, rather than to the number of its last character.

- The language server knows about structs: completing after a dot offers that struct's fields and nothing else, hover lists a struct's fields and says which tape a field reads, and `struct`, `as`, the struct's name and the field names are coloured for what they are. It reads the tokens rather than the tree, because a document being edited hardly ever parses — and typing `p.` is exactly when completion is wanted.

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
