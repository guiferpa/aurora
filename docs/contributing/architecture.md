# Architecture

Where each package lives, what it may know, and why. This is the source of truth: everything
else — `CLAUDE.md`, the agents under `.claude/` — points here rather than repeating it, because
a rule written in two places drifts in one of them.

## Two layers

The code is **hosting** and **vital**.

**Vital** is the pipeline that turns source into something that runs. **Hosting** is the logic
of one kind of interaction with a person or a machine: a command line, a REPL, a language
server. Everything else exists to serve one of the two.

The pipeline, and what travels along it:

```
(lexer) → {tokens} → (parser) → {tree} → (emitter) → {IR} → (builder) │ (evaluator)
```

Between parentheses are the phases. Between braces is what crosses between them, and **those
are packages too** — which is the least obvious decision here and the one the rest follows
from.

## Five kinds of package

| kind | what it is | may import | is imported by |
|---|---|---|---|
| **vital** | a step of the pipeline: `lexer`, `parser`, `emitter`, `builder/evm`, `evaluator` | wire, util | nothing directly — it arrives injected |
| **wire** | what crosses a boundary: `wire/token`, `wire/ast`, `wire/ir`, `wire/diag` | wire, util, and nothing else of the project | everyone |
| **util** | behaviour worth reusing that never touches the world: `byteutil`, `logger` | nothing of the project | everyone |
| **hosting** | one kind of interaction: `hosting/cli`, `hosting/repl`, `hosting/lsp` | wire, util, shared | `main` |
| **shared** | serves the hosting *layer* rather than one interaction: `shared/fileutil`, `shared/manifest`, `shared/trace` | wire, util | hosting, `main` |

Plus `cmd/*`, which is `main`: it may import anything, and it is the only place that wires
things together.

**A sub-package has the kind of its parent.** `evaluator/environ` and `evaluator/builtin` are
vital; `hosting/lsp/textdoc` is hosting. One that needs I/O takes the port and calls it — it
never decides to write.

## The four rules

**1. No package knows another.** What would tie two together is injected instead. The
exceptions are wire and util, which know nothing of the project and so cannot tie anything to
anything — that is what makes them safe to import from anywhere.

**2. A vital package is pure**: no I/O, and nothing done that is not returned. What a program
printed comes back as a value, and the host decides what to do with it. Returning it *even
when the program fails* is part of the rule: a run that prints three lines and then breaks
must not lose the three.

**3. Errors are resolved in hosting.** A package returns an error. It never writes one, and it
never ends the process.

**4. I/O happens in hosting, and wiring happens only in `main`.** Not in a host, not in a
phase.

## Why wire is not util

This is the decision everything else rests on, so it is worth being explicit.

A **util** is behaviour you may reuse, and nobody is obliged to. Two packages using `byteutil`
have nothing to do with each other.

A **wire** package is *data crossing a boundary*, and two phases sharing exactly that type is
the whole point — a phase declaring its own version of it is the mistake the kind exists to
prevent.

It also answers a question that came up while this was being decided: **injection alone cannot
remove the coupling.** If the lexer produces a chain of tokens and the parser consumes it, both
have to name that type. Injecting the *behaviour* does not remove the type of the *data*. The
choice is between giving the artefact a home of its own, or having every phase declare its own
interface and `main` convert — which, for the vocabulary (tags, opcodes), means a translation
table in `main` where a missing entry is wrong output in silence instead of a compile error.

A wire package may lean on another — the tree holds tokens, the IR holds warnings — and on a
util. Neither can tie it to a phase, which is what the rule is protecting.

**What wire holds:** the types, their constructors, the vocabulary (tags, opcodes), and how
they are written down — spelling something out is part of the vocabulary, the way a token
knows how to spell itself. It also holds the comparison of two of them, which is a question
about the shape rather than about whoever built it.

**What wire does not hold:** deciding to show something, and choosing where it goes. That is a
host's, and `shared/trace` is where it lives.

## Why shared is not hosting

A **hosting** package serves one interaction and is a leaf: only `main` depends on it.

A **shared** package serves the layer. It may be imported by any host, it may touch the world —
`shared/manifest` reads `aurora.toml`, `shared/fileutil` reads directories — and it **must not
know which interaction is using it**. That second half is what keeps it from becoming the
drawer where cohesion goes to die.

A util cannot do those things, which is why these three are not utils: a util never touches the
world.

## Why the folders are named after the kinds

The tree says the taxonomy out loud: `wire/`, `shared/`, `hosting/`, and the phases at the root
where the pipeline is.

**There is no `internal/`.** It answers a different question — who may import this from outside
the module — and answering it contradicts the premise that someone could take a part of this
and build a different compiler. `hosting/cli` lived there for a while for no reason anyone
could name.

**The phases and the utils stay at the root** rather than under `vital/` and `util/`. The two
kinds that get a folder are the two that were being confused with something else; `lexer` and
`byteutil` never were.

## Why `hosting/repl` is not `hosting/cli/repl`

`aurora repl` is a command of the CLI, so nesting looks right — until the graph is read:
**`hosting/cli` does not import `hosting/repl`.** `cmd/aurora` imports both. Nesting would
claim an ownership that does not exist, and it would be the only sub-package in the project
that its parent never uses.

The second signal: `build`, `run` and `test` are **functions inside `hosting/cli`**, not
packages. The same argument that makes `cli/repl` would make `cli/build`.

## Where a test goes

**With the thing it tests**, which for an artefact means the artefact's own package. Before
`wire/` existed, the IR was tested by the emitter testing what it made and the builder testing
what it read — the thing they agreed on was covered sideways, and it moved packages without
anyone noticing it was uncovered.

A test that reads the repository — `docs/`, `examples/` — finds the root by walking up to the
`go.mod`, never by counting directories: the second is a fact about where the test sits rather
than about what it is reading, and it breaks on the next move.

## What is not done yet

**Wiring still happens outside `main`.** `hosting/cli`, `hosting/repl` and `hosting/lsp` build
phases themselves instead of being handed them. Rule 4 is the one rule the tree does not obey
yet, and [rfcs/phase_coupling.md](../../rfcs/phase_coupling.md) carries the plan.

**The evaluator still writes.** The print builtins take a writer and use it while the program
runs, which is I/O inside a vital package — injected, so rule 4 is satisfied, but not returned,
so rule 2 is not. The same RFC carries it, along with the cost: output would stop appearing as
it happens, which changes what the REPL feels like.

Both are written down here rather than left to be discovered, because a document describing a
repository that does not exist teaches people to ignore documents.
