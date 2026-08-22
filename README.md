<p align="center">
  <img src="https://raw.githubusercontent.com/guiferpa/aurora/refs/heads/main/docs/images/mascot_02.png" width="200px" alt="Aurora mascot" />
</p>

<h1 align="center">aurora</h1>

<p align="center">
  <a href="https://pkg.go.dev/github.com/guiferpa/aurora">
    <img src="https://pkg.go.dev/badge/github.com/guiferpa/aurora.svg" alt="Go Reference" />
  </a>
  <img src="https://img.shields.io/github/last-commit/guiferpa/aurora" alt="Last commit" />
  <img src="https://github.com/guiferpa/aurora/actions/workflows/pipeline.yml/badge.svg" alt="Pipeline status" />
  <a href="https://coveralls.io/github/guiferpa/aurora?branch=main">
    <img src="https://coveralls.io/repos/github/guiferpa/aurora/badge.svg?branch=main" alt="Coverage status" />
  </a>
</p>

> ⚠ **Alpha.** The language moves. What is in and what is out: [CHANGELOG.md](CHANGELOG.md).

## What Aurora is

An **untyped, expression-only** language. Every value is a **tape**: a fixed run of bytes,
eight wide by default. Numbers, booleans, text and arrays are all the same kind of thing —
there is nothing else.

It runs on its own evaluator, and it assembles to EVM bytecode. The point of having both is
one sentence:

> **The same program answers the same thing on a chain and off it.**

That is what Aurora is for — an on-chain call you can simulate off-chain, with the same
source. A differential harness compiles a program, deploys it to an EVM in memory, calls it,
and compares the answer against the evaluator, so the sentence is checked rather than claimed.

> **Where the name comes from.** Aurora is the author's daughter, and the language is a
> tribute to her. It is also why a new project says `Abidu abide` — it is what she was saying
> at one year old, and it seemed like the right first thing for the language to say.

## Install

Full options, including the macOS Gatekeeper workaround: **[docs/install.md](docs/install.md)**

| Platform | Command |
|---|---|
| **macOS** | `brew tap guiferpa/tap && brew install guiferpa/tap/aurora` |
| **Linux** | Download from [Releases](https://github.com/guiferpa/aurora/releases) → `sudo dpkg -i aurora_*_linux_*.deb` |
| **Other** | [Releases](https://github.com/guiferpa/aurora/releases) — unpack the archive for your OS and arch |
| **From source** | `go install -v github.com/guiferpa/aurora/cmd/aurora@HEAD` |

## Thirty seconds

No project, no files:

```sh
aurora repl
```

```java
>> ident a = 1;
>> a + 1;
= 2
>> ident b = true;
>> printb b;
[0 0 0 0 0 0 0 1]
```

`Ctrl+D` exits, `Ctrl+C` clears the line. `↑`/`↓` walk the history, which is shared by every
project in `~/.aurora/history`.

Or in the browser, with nothing installed: **[playground](https://guiferpa.github.io/aurora)**.

<img width="942" alt="Playground demo" src="https://raw.githubusercontent.com/guiferpa/aurora/refs/heads/main/docs/images/playground_demo.gif" />

## The language, in one page

Every block below runs. They are executed by the test suite, so they cannot say what the
language used to do.

**A value is a tape, and printing is three readings of one.**

```aurora
ident a = 10;
printb a;        #- the bytes:  [0 0 0 0 0 0 0 10]
printd a;        #- the number: 10
printc 44;       #- those bytes as text: ,
```

**Everything is an expression**, including `if`, so it answers with a value.

```aurora
ident answer = if 10 bigger 9 { 42; } else { 0; };
printd answer;
```

**A scope is delayed with `defer` and applied to a vector of values.** There is no signature
and no arity: `feed(n)` reads the nth value applied, and a position nothing was applied to
answers with zeros.

```aurora
ident double = defer { feed(0) * 2; };
printd double(21);
```

**Tapes are shift registers.** `pull` shifts left with the value entering at the right, `push`
shifts right, and `head`/`tail` slice the significant bytes.

```aurora
ident t = [1, 2, 3];
printb pull t 4;
printb head t 2;
```

**Text is one more way of writing a tape.** `"hi"` is the tape holding its bytes, so comparing
text is comparing bytes and how much fits is how wide a tape is.

```aurora
printc "hi";
```

**Shapes name the tapes of a run.** The names are a compile-time directive: nothing about the
declaration reaches the binary.

```aurora
shape Point { x, y };
ident p = Point{10, 20};
printd p.x + p.y;
```

**A file is a module**, and `use` brings one in under a name you choose.

```aurora
#- geometry.ar
ident area = defer { feed(0) * feed(1); };
```

```aurora
#- main.ar
use geometry as g;
printd g.area(30, 20);
```

**There are no negative numbers.** A byte runs from 0 to 255 and no bit marks a value
negative, so `-x` is x taken away from zero and wrapped. Signed arithmetic is a convention you
write yourself, and the language treats it as what it is: a reading of bytes.

Full reference: **[docs/language-design.md](docs/language-design.md)** ·
grammar: **[docs/grammar.md](docs/grammar.md)** · more to paste:
**[examples/](examples/)**

## A project

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
aurora run     # Abidu abide
aurora test    # 1 passed, 0 failed in 1 file
aurora build   # src/main.ar → bin/main
```

`aurora.toml` names profiles so you stop repeating paths. `run` and `build` take a profile
name, or a path ending in `.ar`, or nothing at all — a path never needs a manifest. `deploy`
and `call` always need one, since they read `rpc` and `privkey` from a profile.

Manifest reference: **[docs/manifest.md](docs/manifest.md)** · tests and `assert`:
**[docs/testing.md](docs/testing.md)** · editor support:
**[docs/lsp.md](docs/lsp.md)**

## What reaches the chain today

The evaluator runs the whole language. The EVM backend is being written in slices, and
`aurora build` **says what it could not carry**, once per feature, at the line that used it —
a binary that does less than the source said is announced rather than silent.

| | on a chain |
|---|---|
| arithmetic, and a scope called from a transaction | **yes**, at any tape width |
| a name bound inside a scope | **yes** |
| a value written down, wherever it is used | **yes** |
| a branch, and the value it answers with | **yes** |
| calling a scope from another | not yet |
| comparisons, `and`/`or`, `^` | **yes** |
| tape operations, `shape` | not yet |
| `printb` / `printd` / `printc` | **by decision** — a log has nowhere to go on a chain |
| `assert` | **by decision** — it belongs to `aurora test` |

What each of those would take: **[docs/roadmap.md](docs/roadmap.md)**. What is being decided
now: **[rfcs/](rfcs/)**.

## Commands

```
build       Build binary from source code
call        Call program on a blockchain
completion  Generate the autocompletion script for the specified shell
deploy      Deploy program to a blockchain
help        Help about any command
init        Start an Aurora project in the current directory
repl        Enter in Read-Eval-Print Loop mode
run         Run program directly from source code
test        Run the test files of a project
version     Show toolbox version
```

`aurora <command> --help` for the flags of one. `--tape-size` (1 to 32) sets how wide a value
is, and overrides `tape_size` from the manifest.

## Contributing

Build, tests, coverage, lint and CI: **[docs/development.md](docs/development.md)**. How the
compiler is put together: **[docs/contributing/architecture.md](docs/contributing/architecture.md)**.

```sh
go run ./cmd/aurora repl   # fast loop, no build needed
make check                 # build, wasm, test, lint — what CI runs
```

Releases are built with [GoReleaser](https://goreleaser.com/) on tag push:
**[docs/releasing.md](docs/releasing.md)**.
