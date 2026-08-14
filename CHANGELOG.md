# Changelog

All notable changes and release notes for Aurora are documented here.

---

## Unreleased

### Added

- **`aurora test`**, a command for the test files of a project. It reports every assertion, not only the ones that failed, and exits non-zero when something breaks. A test belongs to the source file of the same name — `greeting.test.ar` tests `greeting.ar`, which is evaluated first so the test sees what it declared, which is how a test reaches code before there is a module system. With a profile it searches from the directory of the profile's source down to the leaves; with a path it runs that file alone. See [docs/testing.md](docs/testing.md).
- **`aurora init` starts a project, not just a manifest.** It writes the layout the manifest describes — `src/main.ar` and `src/main.test.ar` — so a new project runs and passes its tests straight away. The program greets you with `Abidu abide`: Aurora is the author's daughter, and that is what she was saying at one year old.
- **`assert` now belongs to `aurora test`.** Under `aurora run` each one is reported as a warning carrying its file, line and column, and is not checked — so a program holding an assertion no longer fails because of it. `run` used to exit 3 in that case.

### Fixed

- **A deferred scope is a tape.** Its value was the hex key of its own storage — 16 bytes of ASCII text that ignored `tape_size` — so `ident b = defer {};` showed a row of zeros where `ident a = {};` showed one, `b + 1` produced ASCII digits, and a defer stayed 16 bytes wide even with `--tape-size 1`. It is the scope's index as an ordinary tape now. A consequence the language owns: a value equal to an index *is* that reference, the same way `true` is `1`.
- **The compiler warns when a scope holds more deferred scopes than its tape can name** (256 at one byte), since the index wraps and a call would reach a different scope. It is a warning rather than an error: the count is static, and a running program is never stopped by it.
- **`feed(n)` always answers with a tape.** It read the vector directly, so an index past the end gave nothing at all — `print feed(0)` with no values applied printed `[]`, and the REPL said `unknown byte sequence to encode`. The index now wraps around the length of the vector, as the design always described, and an empty vector gives a tape of zeros.

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
