# Aurora

An untyped, expression-only language written in Go. Every value is a **tape**: a fixed run
of bytes, `tape_size` wide. It runs on its own evaluator and assembles to EVM bytecode.

## What matters now

The language proves itself in the **evaluator** and in the **REPL**, and the point of the EVM
backend is that **the same program answers the same thing on a chain and off it** — Aurora
exists to let an on-chain call be simulated off-chain.

**The backend is open again, deliberately, and it is being walked in order.** The first of the
two things the stand-by named is done. The lowering decided which operands named values from
the opcode, by a list of the four arithmetic instructions, so a name bound inside a scope was
not on it: the value it meant to store was never put on the stack and its `MSTORE` found an
empty one. `ident x = feed(0); x + feed(1);` answered 4 on the chain where the program answers
7 — the quieter failure of the two, since it answered rather than reverted. The IR says what
each operand is now, so the lowering reads instead of guessing, and the harness covers the
case.

The second is done too. Memory offsets, jump targets and the runtime size were written with
`PUSH1` and truncated past 255, each in its own way: a contract was published cut short, a
scope past byte 255 was jumped to at the wrong address, and the ninth name in a contract was
given the address of the first. They go in two bytes now, and there is no third size after it —
a deployed contract cannot pass 24,576 bytes, so two bytes cover every legal one. Past that the
builder refuses rather than writing.

A branch is written now. The bytes of a scope are measured before they are written, so a jump
forward has an address; nothing is moved across a branch, so what an arm leaves is what the
arm computed; and both arms leave one value, which is what makes an `if` an expression on a
chain the way it is off one.

What is left of that line is `call`, which needs a frame — the convention is in
[rfcs/if_and_call.md](rfcs/if_and_call.md). **Anything else in `builder/evm` still waits its
turn**, and the turn is discussed first.

What the backend gained before stopping is the differential harness
(`hosting/cli/evm_harness_test.go`) — the same source compiled, deployed to an EVM in memory,
called, and compared against the evaluator — so whatever is written next is provable rather
than believed. What it proves today is arithmetic over arguments, across a dispatcher, at any
tape width.

A feature still does **not** need bytecode to be finished. `shape` and text-as-a-tape shipped
without it.

**The print builtins are logs, not contract instructions**, and do not compile to bytecode.
That is a decision, not a gap.

## Two layers, five kinds of package

The code is **hosting** — the logic of one kind of interaction — and **vital** — the pipeline
that turns source into something that runs.

```
(resolver) → {modules} → (loader) → {stream} → (evaluator) │ (builder)
     ↑                        ↑
  (lexer) → {tokens}      (emitter) → {IR}
     → (parser) → {tree}
```

Between parentheses are the phases; between braces, what crosses between them, which are
packages of their own.

| kind | who |
|---|---|
| **vital** | `lexer`, `parser`, `emitter`, `builder/evm`, `evaluator`, `resolver`, `loader` |
| **wire** | `wire/token`, `wire/ast`, `wire/ir`, `wire/diag`, `wire/eval`, `wire/module` |
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
