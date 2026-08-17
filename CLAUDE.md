# Aurora

An untyped, expression-only language written in Go. Every value is a **tape**: a fixed run
of bytes, `tape_size` wide. It runs on its own evaluator and assembles to EVM bytecode.

## What matters now

The language proves itself in the **evaluator** and in the **REPL**, and the point of the EVM
backend is that **the same program answers the same thing on a chain and off it** — Aurora
exists to let an on-chain call be simulated off-chain.

**The backend is on stand-by** while the architecture lands. What it gained before stopping is
the differential harness (`internal/cli/evm_harness_test.go`) — the same source compiled,
deployed to an EVM in memory, called, and compared against the evaluator — so whatever is
written next is provable rather than believed.

A feature still does **not** need bytecode to be finished. `struct` and text-as-a-tape shipped
without it.

**The print builtins are logs, not contract instructions**, and do not compile to bytecode.
That is a decision, not a gap.

## Two layers, five kinds of package

The code is **hosting** — the logic of one kind of interaction — and **vital** — the pipeline
that turns source into something that runs. Everything else exists to serve one of the two.

```
(lexer) → {tokens} → (parser) → {tree} → (emitter) → {IR} → (builder) │ (evaluator)
```

Between parentheses are the phases; between braces, what crosses between them.

| kind | who | imports | is imported by |
|---|---|---|---|
| **vital** | `lexer`, `parser`, `emitter`, `builder/evm`, `evaluator` | wire, util | nothing directly — it arrives injected |
| **wire** | what crosses: tokens, tree, IR, warnings | **nothing of the project** | everyone |
| **util** | reusable behaviour that never touches the world: `byteutil`, `logger` | **nothing of the project** | everyone |
| **hosting** | one interaction: `internal/cli`, `repl`, `lsp` | wire, util, shared | `main` |
| **shared** | serves the hosting layer, not one interaction: `fileutil`, `manifest`, `internal/trace` | wire, util | hosting, `main` |

**The four rules that follow:**

1. **No package knows another** — except wire and util, which know nothing of the project and
   may be known by all. What would tie two together is injected instead.
2. **A vital package is pure**: no I/O, and nothing done that is not returned. What a program
   printed comes back as a value, so the host decides what to do with it.
3. **Errors are resolved in hosting.** A package returns an error; it never writes one, and it
   never ends the process.
4. **I/O happens in hosting, and injection happens only in `main`.** Nothing is wired
   together anywhere else.

**A sub-package has the kind of its parent** — `evaluator/builtin` is vital, `lsp/textdoc` is
hosting. One that needs I/O takes the port and calls it; it never decides to write.

The tree does not obey this yet. What it costs, in which order, and what it finds today is in
[rfcs/phase_coupling.md](rfcs/phase_coupling.md).

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
