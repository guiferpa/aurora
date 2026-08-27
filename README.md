<p align="center">
  <img src="https://raw.githubusercontent.com/guiferpa/aurora/refs/heads/main/docs/images/mascot_02.png" width="200px" alt="Aurora mascot" />
</p>

<h1 align="center">aurora</h1>

<p align="center">
  <em>A small language for the EVM whose programs answer the same thing on a chain and off it.</em>
</p>

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

<p align="center">
  <a href="docs/language-design.md">Language</a> ·
  <a href="examples/">Examples</a> ·
  <a href="https://guiferpa.github.io/aurora">Playground</a> ·
  <a href="docs/roadmap.md">Roadmap</a> ·
  <a href="CHANGELOG.md">Changelog</a>
</p>

> ⚠ **Alpha.** The language moves, and what moved is in the [changelog](CHANGELOG.md).

## What Aurora is

```aurora
use std/evm/storage as s;

ident deposit = defer { s.set(1, s.get(1) + feed(0)); };
ident balance = defer { s.get(1); };

deposit(10);
printd balance();
```

On a chain, that is two ways in, and what follows the name is the calldata a scope reads with
`feed`:

```sh
aurora tx deposit 10     # a change: feed(0) is the 10, sent, and what it kept stays
aurora call balance      # a question: answered for nothing, and 10 comes back
```

Off one, it is the last two lines, and the answer is the same:

```sh
$ aurora run
10
```

An **untyped, expression-only** language. Every value is a **tape**: a fixed run of bytes,
eight wide by default. Numbers, booleans, text and shapes are all the same kind of thing —
there is nothing else.

It runs on its own evaluator, and it assembles to EVM bytecode. The point of having both is
one sentence:

> **The same program answers the same thing on a chain and off it.**

That is what Aurora is for — an on-chain call you can simulate off-chain, from the same source.
It is checked rather than claimed: a differential harness compiles each program, deploys it to
an EVM in memory, calls it, and compares the answer against the evaluator — and every time the
two have disagreed, the disagreement became a test.

## Why "Aurora"

Aurora is the author's daughter, and the language is a tribute to her.

It is also why a new project says `Abidu abide` — it is what she was saying at one year old,
and it seemed like the right first thing for the language to say.

## Get started

### Install

| Platform | Command |
|---|---|
| **macOS** | `brew tap guiferpa/tap && brew install guiferpa/tap/aurora` |
| **Linux** | Download from [Releases](https://github.com/guiferpa/aurora/releases) → `sudo dpkg -i aurora_*_linux_*.deb` |
| **Other** | [Releases](https://github.com/guiferpa/aurora/releases) — unpack the archive for your OS and arch |
| **From source** | `go install -v github.com/guiferpa/aurora/cmd/aurora@HEAD` |

Full options, including the macOS Gatekeeper workaround: **[docs/install.md](docs/install.md)**.

Nothing to install, if you would rather not: the **[playground](https://guiferpa.github.io/aurora)**
runs the language in the browser.

### First lines

No project and no files:

```sh
aurora repl
```

```java
>> ident a = 1;
= [0 0 0 0 0 0 0 0]
>> a + 1;
= [0 0 0 0 0 0 0 2]
>> printd a + 1;
2
= [0 0 0 0 0 0 0 2]
```

A binding answers the neutral tape, because everything is an expression and a binding has
nothing of its own to give. A value answers as its bytes; `printd` reads those bytes as a
number, `printc` reads them as text, and `printb` shows them as they are.

`Ctrl+D` exits, `Ctrl+C` clears the line. `↑`/`↓` walk the history, which is shared by every
project in `~/.aurora/history`.

**The language itself**, in order of how much you want: the commented
**[examples/](examples/)**, which the test suite runs and compares against the output each one
declares · the reference, **[docs/language-design.md](docs/language-design.md)** · the grammar,
**[docs/grammar.md](docs/grammar.md)**.

## A project

```sh
mkdir my-project && cd my-project
aurora init
```

```
✨ dawn has broken on your project.
   aurora.toml
   src/main.ar
   src/main.test.ar

Run it with 'aurora run', test it with 'aurora test'.
```

`src/main.ar` is a program with one scope in it, and `src/main.test.ar` names that file and
checks what the scope says. Both are commented, and reading them is the shortest tour of the
language there is.

```sh
$ aurora run
Abidu abide

$ aurora test
src/main.test.ar
  ok    greet says its piece

1 passed, 0 failed in 1 file
```

### Profiles

`aurora.toml` names profiles, so paths are written once:

```toml
[project]
name = "my-project"
version = "0.1.0"
tape_size = 16

[profiles.main]
source = "src/main.ar"
binary = "bin/main"
```

`run` and `build` take a profile name, a path ending in `.ar`, or nothing at all — nothing at
all is the `main` profile, and a path never needs a manifest:

```sh
aurora run              # the main profile
aurora run dev          # another profile
aurora run one.ar       # that file, no manifest involved
```

A second profile is how one project reaches two chains, or two tape widths. `deploy`, `call`
and `tx` always need one, since they read `rpc` and `privkey` from it — and a key belongs in
the environment rather than in the file:

```toml
[profiles.sepolia]
source  = "src/main.ar"
binary  = "bin/main"
rpc     = "${{ AURORA_RPC }}"
privkey = "${{ AURORA_PRIVKEY }}"
```

Read from the project's `.env` first, then from the environment the command runs in. A name
nothing sets is refused rather than read as empty.

Manifest reference: **[docs/manifest.md](docs/manifest.md)** · editor support:
**[docs/lsp.md](docs/lsp.md)**.

## On a chain

```sh
aurora build            # src/main.ar → bin/main
aurora deploy -p sepolia
```

`build` says what it could not carry, at the line that used it:

```
src/main.ar:14:1: warning: printc is ignored in compiled code, by decision: a chain has
nowhere to put a log, and the value it was given carries on
```

That is the shape of the whole promise: a binary that does less than the source said is
announced rather than silent. Everything a program can write reaches the bytecode today — the
prints and `assert` are the two exceptions, and both are decisions rather than gaps.
**[docs/roadmap.md](docs/roadmap.md)** lists what a contract still cannot do.

### Asking and changing

A chain has two ways in, and Aurora has one command for each:

```sh
aurora call balance     # a question: answered against the state as it is, and costs nothing
aurora tx deposit       # a change: sent, paid for, mined, and what it changed stays
```

Nothing guesses between them. What the compiler knows is used to refuse the wrong one, because
a scope that keeps something, asked as a question, answers and leaves the chain exactly as it
was:

```
$ aurora call deposit
deposit changes what the chain keeps, and a call is a question: it would answer, cost
nothing, and leave the chain exactly as it was — send it with 'aurora tx deposit' instead
```

A question answers the bytes that came back, on their own, so it pipes:

```sh
aurora call balance | xxd
aurora call balance --hex     # 0x000000000000002a, for reading and for an explorer
```

### Passing arguments

Both take arguments after the name, and both encode them the same way — a scope has no
signature, so `feed(0)` reads the first, `feed(1)` the second, and a position nothing was
applied to reads as zeros:

```sh
aurora call sum 30 20
aurora tx deposit 5
```

Add `--pretend` to either and it shows what would be sent without sending it, which is the
honest way to see the calldata:

```
$ aurora call sum 30 20 --pretend
Contract:   0x7e769d0a39ae98fb7b363c41466951e481323a7d (20 bytes)
Function:   0x0fdbd160736e9b9b51ea9a79a8ed86f427a62e0e377d60335d2ec895c27025bb (32 bytes)
Arguments:  0x00…001e00…0014 (64 bytes)
```

The selector is the whole hash of the name rather than the four bytes an ABI uses, and each
value takes a word after it. Neither of those is a choice a program makes — they are what a
scope with no signature looks like from outside.

Separately from the arguments, a transaction can carry **value** — ether — which a program
reads with `callvalue`:

```sh
aurora tx deposit --value 1000000000000000
aurora run --value 1000000000000000     # what it would have carried, off a chain
```

In wei and a whole number, because ether is a decimal and a decimal is a rounding waiting to
happen. **Nothing in Aurora sends ether back** — there is no external call — so what arrives
at a contract can be counted and kept, and cannot be moved. The command says so before it
sends.

## Testing

```sh
aurora test
```

A test file is a file like any other. It imports what it checks and says what should hold:

```aurora
#- src/calc.ar
ident double = defer { feed(0) * 2; };
```

```aurora
#- src/calc.test.ar
use calc as c;

assert(c.double(21) equals 42, "double doubles");
```

There is no node to start, no second language, and no fixture: it is the same language, run by
the same evaluator, in the time a process takes to open. That is where it differs from the
usual way of testing for the EVM, where a test is written in another language and run against
a simulated chain you have to keep.

And it is worth something because of the sentence at the top. A test that passes off a chain is
evidence about a chain, since the compiler's own suite compiles each program, deploys it to an
EVM in memory, calls it, and compares the answer against the evaluator. Where the two disagree,
that is a bug in the compiler rather than a surprise in your contract.

Tests and `assert`: **[docs/testing.md](docs/testing.md)**.

## Contributing

Build, tests, coverage, lint and CI: **[docs/development.md](docs/development.md)**. How the
compiler is put together: **[docs/contributing/architecture.md](docs/contributing/architecture.md)**.
What the IR is and how to run it, for anyone writing a second backend:
**[docs/contributing/ir.md](docs/contributing/ir.md)** — and why it is blocks rather than a
list: **[docs/contributing/why-blocks.md](docs/contributing/why-blocks.md)**.

```sh
go run ./cmd/aurora repl   # fast loop, no build needed
make aurora                # the binary, into target/bin
make check                 # build, wasm, test, lint — what CI runs
make install-std           # the standard library, into $AURORA_ROOT/lib
```

Releases are built with [GoReleaser](https://goreleaser.com/) on tag push:
**[docs/releasing.md](docs/releasing.md)**.
