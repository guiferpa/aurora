# Aurora

An untyped, expression-only language written in Go. Every value is a **tape**: a fixed run
of bytes, `tape_size` wide. It runs on its own evaluator and assembles to EVM bytecode.

## What matters now

The language proves itself in the **evaluator** and in the **REPL**, and the point of the EVM
backend is that **the same program answers the same thing on a chain and off it** — Aurora
exists to let an on-chain call be simulated off-chain.

The backend was parked while the behaviour settled. It is being un-parked in slices, each one
proved by the differential harness (`internal/cli/evm_harness_test.go`): the same source
compiled, deployed to an EVM in memory, called, and compared against the evaluator.

A feature still does **not** need bytecode to be finished. `struct` and text-as-a-tape shipped
without it.

**The print builtins are logs, not contract instructions**, and do not compile to bytecode.
That is a decision, not a gap.

## The four premises

1. **Every function says what it does and why it exists.** Briefly. A loose fragment gets a
   comment only when the algorithm is hard to follow.
2. **A vital or auxiliary package knows nothing about the rest of the project** — enough
   that someone could take a part of it and build a different compiler.
3. **Errors are resolved in the host.** A phase returns an error; it does not write.
4. **A vital package is pure**: values in, values out.

| kind | who | may depend on |
|---|---|---|
| phase | `lexer`, `parser`, `emitter`, `evaluator`, `builder/evm` | earlier phases, auxiliaries |
| auxiliary | `byteutil`, `fileutil`, `logger`, `manifest` | nothing of the project |
| host | `cmd/*`, `repl`, `internal/cli`, `lsp` | anything; nothing depends on it |

## Working here

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
