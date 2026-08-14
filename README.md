<img src="https://raw.githubusercontent.com/guiferpa/aurora/refs/heads/main/docs/images/mascot.png" width="112px" height="140px">

# aurora

[![Go Reference](https://pkg.go.dev/badge/github.com/guiferpa/aurora.svg)](https://pkg.go.dev/github.com/guiferpa/aurora)
[![Last commit](https://img.shields.io/github/last-commit/guiferpa/aurora)](https://img.shields.io/github/last-commit/guiferpa/aurora)
[![Go Report Card](https://goreportcard.com/badge/github.com/guiferpa/aurora)](https://goreportcard.com/report/github.com/guiferpa/aurora)
![Pipeline workflow](https://github.com/guiferpa/aurora/actions/workflows/pipeline.yml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/guiferpa/aurora/badge.svg?branch=main)](https://coveralls.io/github/guiferpa/aurora?branch=main)

> ⚠ **Alpha** — don't use in production. Stuff can change. See [CHANGELOG.md](CHANGELOG.md) for known limitations and what's in/out.

## What's Aurora?

Aurora is a **study-focused** language that compiles to the Ethereum Virtual Machine (EVM). It already does basic compilation and runs code via an evaluator, but it's still in the oven: syntax and behavior may shift, and some EVM goodies aren't fully there yet. Perfect for tinkering and learning.

> **Where the name comes from.** Aurora is the author's daughter, and the language is a tribute to her. That is also why a new project says `Abidu abide` — it is what she was saying at one year old, and it seemed like the right first thing for the language to say.

## Summary

- [Get started](#get-started)
  - [Install CLI](#install-cli) → full guide: [docs/install.md](docs/install.md)
  - [Try in 30 seconds](#try-in-30-seconds-no-project-needed) (REPL, no project)
  - [Project manifest](#project-manifest) → [Manifest reference (aurora.toml)](docs/manifest.md)
  - [Run from file](#run-from-file)
  - [Compile to EVM bytecode](#compile-to-evm-bytecode)
  - [Writing some code](#writing-some-code)
- [Language Design](docs/language-design.md) (Expressions only, Untyped, Tapes, Reels, Arithmetic)
- [Testing](docs/testing.md) — `assert` and `aurora test`
- [Editor support](docs/lsp.md) — `aurorals` language server: coloring, diagnostics, hover, completion
- [Try it out](#try-it-out) — [Playground](#playground)
- [Extra options](#extra-options) — [Debug flag](#debug-flag)
- [Development](docs/development.md) (build, tests, coverage, lint, CI) — for contributors
- [Publishing releases](docs/releasing.md) (maintainers)

## Get started

### Install CLI

Full install options (including macOS workaround): **[docs/install.md](docs/install.md)**

| Platform | Command |
|----------|---------|
| **macOS (Homebrew)** | `brew tap guiferpa/tap && brew install guiferpa/tap/aurora` |
| **Linux (.deb)** | Download from [Releases](https://github.com/guiferpa/aurora/releases) → `sudo dpkg -i aurora_*_linux_*.deb` |
| **Other** | [Releases](https://github.com/guiferpa/aurora/releases) — unpack archive for your OS/arch |
| **From source** | `go install -v github.com/guiferpa/aurora/cmd/aurora@HEAD` (requires [Go](https://go.dev/)) |

<details>
<summary><strong>macOS: “Apple could not verify …” (unverified developer)</strong></summary>

If Gatekeeper blocks the binary, use one of these:

**Terminal (recommended)** — in the folder where the `aurora` binary is:

```sh
xattr -cr aurora
./aurora
```

**Finder:** Right-click the binary → **Open** → **Open** (one-time approval).

The Homebrew cask applies the workaround automatically. More detail: [docs/install.md#macos-apple-could-not-verify--unverified-developer](docs/install.md#macos-apple-could-not-verify--unverified-developer).
</details>

### Try in 30 seconds (no project needed)

Jump straight into the REPL, no project setup:

```sh
aurora repl
```

```java
>> ident a = 1;
>> a + 1;
= 2
>> ident b = true;
>> print b;
[0 0 0 0 0 0 0 1]
```

`Ctrl+D` exits; `Ctrl+C` clears the line you're typing. No `aurora.toml` needed for `repl`, `version`, or `help`.

**Line editing and history:** `↑`/`↓` browse previous commands, `←`/`→` move the cursor inside the line (also `Home`/`End` and `Ctrl+A`/`Ctrl+E`), and `Backspace`/`Delete` edit in place.

History is shared by every project in **`~/.aurora/history`** (last 1000 commands, written as you type them, `0600`). Point `AURORA_HISTORY` somewhere else to change the file. When stdin is not a terminal — e.g. `printf '1 + 1;\n' | aurora repl` — the REPL reads plain lines, with no history and no key handling.

### Project manifest

A manifest lets you name profiles instead of repeating paths — that's the `aurora.toml` file at your project root (or in a parent folder). **build** and **run** use it when you give them a profile name or nothing at all; **deploy** and **call** always need one, since they read `rpc` and `privkey` from a profile.

If it is missing, the CLI will gently remind you:

```
aurora.toml not found in current directory or any parent (run 'aurora init' to create a project manifest)
```

**Create a new project:**

```sh
mkdir my-project && cd my-project
aurora init
```

That writes the manifest and the layout it describes:

```
aurora.toml
src/main.ar        a program to run
src/main.test.ar   its tests
```

```sh
aurora run    # Abidu abide
aurora test   # 1 passed, 0 failed in 1 file
```

The manifest holds `[project]` and `[profiles.main]` (defaults: `source` = `src/main.ar`, `binary` = `bin/main`). From there `aurora run` and `aurora build` work with no arguments, and a name picks another profile: `aurora run dev`.

A manifest is only needed when you name a profile — running a file by path never needs one.

Full manifest reference (including optional on-chain bits): [docs/manifest.md](docs/manifest.md).

### Run from file

1. **With a project:** Put your code in `src/main.ar` (or whatever path you set in `aurora.toml`), then from the project root:

   ```sh
   aurora run
   ```

2. **Run any file:** From anywhere, with no project at all:

   ```sh
   aurora run path/to/your/file.ar
   ```

Example — save as `src/main.ar`:

```java
ident result = 10 * 20;
print result + 1;
```

Run `aurora run`. The evaluator prints values as raw bytes (here 201 = 8 bytes): `[0 0 0 0 0 0 0 201]`.

### Compile to EVM bytecode

Want bytecode instead of running in the evaluator?

```sh
aurora build                            # the "main" profile: source -> binary
aurora build dev                        # another profile
aurora build src/main.ar -o bin/main    # any file, with an explicit output
```

You get a raw bytecode file — deploy it or feed it to your favorite EVM client. For deploy/call (rpc, privkey, etc.) check the [Manifest reference](docs/manifest.md).

### Writing some code

A few snippets to paste in the REPL or in a file:

```java
ident x = 10;
ident y = 20;
print x + y;           // 30 (as bytes)

ident flag = true;
if flag bigger 0 then 1 else 0;   // if is an expression, returns a value

ident b = true;        // a tape holding 1, same bytes as the number 1
print b + 1;           // 2

ident t = [1, 2, 3];   // a tape holding three bytes
print pull t 4;        // shifts left, 4 enters at the right
print head t 2;        // the first two significant bytes
```

For more — tapes, reels, branches, EVM-style callables — dig into the [examples folder](https://github.com/guiferpa/aurora/tree/main/examples) (e.g. `examples/evm/ident.ar`, `examples/simple_math.ar`). What's in and what's not yet: [CHANGELOG.md](CHANGELOG.md).

## Language Design

Aurora is **untyped** — everything is bytes; numbers, booleans, tapes (arrays), and strings (reels) are all byte arrays. Full reference:

**[→ Language Design (Untyped, Tapes, Reels, Arithmetic)](docs/language-design.md)**

<details>
<summary>Quick reference</summary>

- **Values** = tapes: a fixed run of bytes, 8 by default (`ident a = 3` → `[0,0,0,0,0,0,0,3]`). Booleans are tapes too: `true` is the same bytes as `1`.
- **Tape size** is a compiler parameter: `tape_size` in `aurora.toml` or `--tape-size` (1 to 32, flag wins). A literal that does not fit is a compile-time error.
- **Tapes** are shift registers: `pull` shifts left (value in at the right), `push` shifts right (value in at the left), `head`/`tail` slice the significant bytes; index `n` is modulo the tape size.
- **Reels**: strings are runs of tapes (one per character); use `echo` to print.
- **Arithmetic**: a tape read as an unsigned big-endian integer, wrapping at the tape width.
</details>

## Try it out

### Playground
> 🚀 Try Aurora in the browser: [playground](https://guiferpa.github.io/aurora) — WebAssembly + Go, runs Aurora source right there.
<img width="942" alt="Playground demo" src="https://raw.githubusercontent.com/guiferpa/aurora/refs/heads/main/docs/images/playground_demo.gif" />

## Commands

Quick reference: `run` and `build` take one argument — a profile name, or a path ending in `.ar`. A path needs no manifest; a profile name needs an `aurora.toml` in the current dir or a parent, which `aurora init` creates.

```sh
aurora help

Usage:
  aurora [command]

Available Commands:
  build       Build binary from source code
  call        Call program on a blockchain
  completion  Generate the autocompletion script for the specified shell
  deploy      Deploy program to a blockchain
  help        Help about any command
  init        Create an aurora.toml manifest in the current directory
  repl        Enter in Read-Eval-Print Loop mode
  run         Run program directly from source code
  version     Show toolbox version

Flags:
  -h, --help   help for aurora

Use "aurora [command] --help" for more information about a command.
```

## Contributing

Building the binary, running the suite, coverage expectations, lint and CI are all in **[docs/development.md](docs/development.md)**.

Quick start:

```sh
go run ./cmd/aurora repl     # fast loop, no build needed
go test ./... -race          # tests
make lint                    # golangci-lint, same version and args CI uses
```

<details>
<summary><strong>Publishing releases (maintainers)</strong></summary>

Releases are built with [GoReleaser](https://goreleaser.com/) on tag push. Full steps (Homebrew tap, secrets, apt repo options):

**[→ docs/releasing.md](docs/releasing.md)**
</details>
