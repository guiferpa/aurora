# Aurora

An untyped, expression-only language written in Go. Every value is a **tape**: a fixed run
of bytes, `tape_size` wide. It runs on its own evaluator and assembles to EVM bytecode.

## What matters now

The language proves itself in the **evaluator** and in the **REPL**, and the point of the EVM
backend is that **the same program answers the same thing on a chain and off it** — Aurora
exists to let an on-chain call be simulated off-chain.

**The backend is on stand-by, and stays there.** The architecture it was waiting for has
landed; what surfaced underneath is bigger than a slice. Binding a name inside a scope —
`ident x = feed(0);` in a `defer` body — reverts on chain: the lowering hands a value to
arithmetic and to a return and to nothing else, so the one an `ident` meant to store is
dropped and its `MSTORE` finds an empty stack. The builder still counts that instruction as
covered, so the binary comes out silent. Under it sits a second ceiling: memory offsets, jump
targets and the runtime size are all written with `PUSH1`, and each truncates past 255 without
a word. Neither is a bug with a patch behind it — both are designs — and opening them now
would stop the language moving. **Do not start there.** When the turn comes, it is deliberate
and it is discussed first.

What the backend gained before stopping is the differential harness
(`hosting/cli/evm_harness_test.go`) — the same source compiled, deployed to an EVM in memory,
called, and compared against the evaluator — so whatever is written next is provable rather
than believed. What it proves today is arithmetic over arguments, across a dispatcher, at any
tape width.

A feature still does **not** need bytecode to be finished. `struct` and text-as-a-tape shipped
without it.

**The print builtins are logs, not contract instructions**, and do not compile to bytecode.
That is a decision, not a gap.

## Two layers, five kinds of package

The code is **hosting** — the logic of one kind of interaction — and **vital** — the pipeline
that turns source into something that runs.

```
(lexer) → {tokens} → (parser) → {tree} → (emitter) → {IR} → (builder) │ (evaluator)
```

Between parentheses are the phases; between braces, what crosses between them, which are
packages of their own.

| kind | who |
|---|---|
| **vital** | `lexer`, `parser`, `emitter`, `builder/evm`, `evaluator` |
| **wire** | `wire/token`, `wire/ast`, `wire/ir`, `wire/diag`, `wire/eval` |
| **util** | `byteutil`, `logger` |
| **hosting** | `hosting/cli`, `hosting/repl`, `hosting/lsp` |
| **shared** | `shared/fileutil`, `shared/manifest` |

**No package knows another** — except wire and util, which know nothing of the project and may
be known by all. **A vital package is pure**: no I/O, nothing done that is not returned.
**Errors are resolved in hosting**, which is also the only layer that touches the world, and
**wiring happens only in `main`**. A sub-package has the kind of its parent.

Why each of those, what belongs where, and what the tree does not obey yet:
**[docs/contributing/architecture.md](docs/contributing/architecture.md)**, which is the
source of truth — this table is a pointer, not a copy.

## Working here

- **Every function says what it does and why it exists.** Briefly. A loose fragment gets a
  comment only when the algorithm is hard to follow.
- **A success case and an error case** for every implementation; an integration test for
  anything the user perceives. Never verify by running and throw it away — if you had to
  execute it to believe it, it becomes a test.
- **One concern per commit**, and it compiles and passes on its own. A pull request carries
  several of them and is sized so a human reviews the whole of it in **under forty minutes**;
  when it cannot be cut that small, discuss the scope first.
- **Every new feature goes through a pull request** saying what it gives the user. A fix
  straight to `main` is called by the maintainer, not assumed.
- **Every action goes through the Makefile, without exception**: `make check` before
  committing (build, wasm, test, lint), `make complexity` to measure. When a target does not
  exist, add it in a pull request instead of working around it.
- **Never run the binary by hand against a scrap of source.** If a behaviour needs checking,
  write the test and run `make test` — that way the check survives. Cognitive complexity is
  the gate; cyclomatic is information.

## Where things are written

- **`.claude/agents/aurora-standards.md`** — the standards in full, and how to review against
  them. Read it before writing code here.
- **`.claude/agents/aurora-internals.md`** — how the compiler actually works, verified
  against the code rather than the docs.
- **`docs/`** for the end user, **`docs/contributing/`** for contributors (English),
  **`rfcs/`** for proposals and debt (Portuguese).
- **`docs/roadmap.md`** is the direction: anything the user can perceive gets an entry there.

Conversation is usually pt-BR; code, comments and `docs/` stay in English.
