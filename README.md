<img src="https://raw.githubusercontent.com/guiferpa/aurora/refs/heads/main/docs/images/mascot.png" width="224px" height="280px">

# aurora

[![Go Reference](https://pkg.go.dev/badge/github.com/guiferpa/aurora.svg)](https://pkg.go.dev/github.com/guiferpa/aurora)
[![Last commit](https://img.shields.io/github/last-commit/guiferpa/aurora)](https://img.shields.io/github/last-commit/guiferpa/aurora)
[![Go Report Card](https://goreportcard.com/badge/github.com/guiferpa/aurora)](https://goreportcard.com/report/github.com/guiferpa/aurora)
![Pipeline workflow](https://github.com/guiferpa/aurora/actions/workflows/pipeline.yml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/guiferpa/aurora/badge.svg?branch=main)](https://coveralls.io/github/guiferpa/aurora?branch=main)

> ⚠ **Alpha** — don't use in production. Stuff can change. See [CHANGELOG.md](CHANGELOG.md) for known limitations and what's in/out.

## What's Aurora?

Aurora is a **study-focused** language that compiles to the Ethereum Virtual Machine (EVM). It already does basic compilation and runs code via an evaluator, but it's still in the oven: syntax and behavior may shift, and some EVM goodies aren't fully there yet. Perfect for tinkering and learning.

## Summary

- [Get started](#get-started)
  - [Install CLI](#install-cli) → full guide: [docs/install.md](docs/install.md)
  - [Try in 30 seconds](#try-in-30-seconds-no-project-needed) (REPL, no project)
  - [Project manifest](#project-manifest) → [Manifest reference (aurora.toml)](docs/manifest.md)
  - [Run from file](#run-from-file)
  - [Compile to EVM bytecode](#compile-to-evm-bytecode)
  - [Writing some code](#writing-some-code)
- [Language Design](docs/language-design.md) (Untyped, Tapes, Reels, Arithmetic)
- [Try it out](#try-it-out) — [Playground](#playground)
- [Extra options](#extra-options) — [Debug flag](#debug-flag)
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

`Ctrl+D` or `Ctrl+C` to exit. No `aurora.toml` needed for `repl`, `version`, or `help`.

### Project manifest

Most commands (**build**, **run**, **deploy**, **call**) want a project manifest — that's the `aurora.toml` file at your project root (or in a parent folder). It holds stuff like default source and output paths.

If you forget, the CLI will gently remind you:

```
aurora.toml not found in current directory or any parent (run 'aurora init' to create a project manifest)
```

**Create a new project:**

```sh
mkdir my-project && cd my-project
aurora init
```

That drops an `aurora.toml` with `[project]` and `[profiles.main]` (defaults: `source` = `src/main.ar`, `binary` = `bin/main`). From there you can `aurora run`, `aurora build`, etc. The only commands that *don't* need a manifest: `aurora init`, `aurora version`, `aurora help`, `aurora repl`.

Full manifest reference (including optional on-chain bits): [docs/manifest.md](docs/manifest.md).

### Run from file

1. **With a project:** Put your code in `src/main.ar` (or whatever path you set in `aurora.toml`), then from the project root:

   ```sh
   aurora run
   ```

2. **Run any file:** From a dir that has (or inherits) `aurora.toml`:

   ```sh
   aurora run -s path/to/your/file.ar
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
aurora build                              # uses manifest: source -> binary
aurora build -s src/main.ar -o bin/main   # or point to any file and output
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

ident t = [1, 2, 3];   // tape = 8-byte array
print head t 2;
```

For more — tapes, reels, branches, EVM-style callables — dig into the [examples folder](https://github.com/guiferpa/aurora/tree/main/examples) (e.g. `examples/evm/ident.ar`, `examples/simple_math.ar`). What's in and what's not yet: [CHANGELOG.md](CHANGELOG.md).

## Language Design

Aurora is **untyped** — everything is bytes; numbers, booleans, tapes (arrays), and strings (reels) are all byte arrays. Full reference:

**[→ Language Design (Untyped, Tapes, Reels, Arithmetic)](docs/language-design.md)**

<details>
<summary>Quick reference</summary>

- **Values** = byte arrays (e.g. `ident a = 3` → 8 bytes; `ident b = [1,2,3]` → 8-byte tape).
- **Tapes**: `pull`, `push`, `head tape n`, `tail tape n`; index `n` is modulo 8.
- **Reels**: strings are arrays of 8-byte tapes (one per character); use `echo` to print.
- **Arithmetic**: first 8 bytes interpreted as unsigned 64-bit integer; booleans padded to 8 bytes (`true`=1, `false`=0).
</details>

## Try it out

### Playground
> 🚀 Try Aurora in the browser: [playground](https://guiferpa.github.io/aurora) — WebAssembly + Go, runs Aurora source right there.
<img width="942" alt="Playground demo" src="https://raw.githubusercontent.com/guiferpa/aurora/refs/heads/main/docs/images/playground_demo.gif" />

## Commands

Quick reference: everything except **init**, **version**, **help**, and **repl** wants an `aurora.toml` in the current dir (or a parent). No manifest? Run `aurora init` and you're good.

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

<details>
<summary><strong>Publishing releases (maintainers)</strong></summary>

Releases are built with [GoReleaser](https://goreleaser.com/) on tag push. Full steps (Homebrew tap, secrets, apt repo options):

**[→ docs/releasing.md](docs/releasing.md)**
</details>
