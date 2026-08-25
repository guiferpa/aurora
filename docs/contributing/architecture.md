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
(resolver) → {modules} → (loader) → {stream} → (evaluator) │ (builder)
     ↑                        ↑
  (lexer) → {tokens}      (emitter) → {IR}
     → (parser) → {tree}
```

Between parentheses are the phases. Between braces is what crosses between them, and **those
are packages too** — which is the least obvious decision here and the one the rest follows
from.

**It stopped being a straight line when a program stopped being one file.** A resolver reads
what the entry says it needs and keeps going until nothing is left to find; a loader turns what
it found into one stream of instructions with the range of each module in it. Both are phases
by the rule that matters — they take values, answer with values and touch nothing — and both
drive other phases, which arrive as ports the way anything else a phase needs does: a resolver
is handed a way to read a file and a way to parse one, a loader a way to emit.

That is what lets the same resolver serve a command line reading a disk and a page in a browser
holding its files in memory, which is not a hypothetical — the playground compiles in
`js/wasm`, where there is no file system to read.

## Five kinds of package

| kind | what it is | may import | is imported by |
|---|---|---|---|
| **vital** | a step of the pipeline: `lexer`, `parser`, `emitter`, `builder/evm`, `evaluator`, `resolver`, `loader` | wire, util | nothing directly — it arrives injected |
| **wire** | what crosses a boundary: `wire/token`, `wire/ast`, `wire/ir`, `wire/diag`, `wire/eval`, `wire/module` | wire, util, and nothing else of the project | everyone |
| **util** | behaviour worth reusing that never touches the world: `byteutil`, `logger` | nothing of the project | everyone |
| **hosting** | one kind of interaction: `hosting/cli`, `hosting/repl`, `hosting/lsp` | wire, util, shared | `main` |
| **shared** | serves the hosting *layer* rather than one interaction: `shared/fileutil`, `shared/manifest`, `shared/printer` | wire, util | hosting, `main` |

Plus `cmd/*`, which is `main`: it may import anything, and it is the only place that wires
things together.

**A sub-package has the kind of its parent.** `evaluator/environ` and `evaluator/builtin` are
vital; `hosting/lsp/textdoc` is hosting. One that needs I/O takes the port and calls it — it
never decides to write.

## The four rules

**1. No package knows another.** What would tie two together is injected instead. The
exceptions are wire and util, which know nothing of the project and so cannot tie anything to
anything — that is what makes them safe to import from anywhere.

**2. A vital package is pure**: no I/O, and nothing done that is not returned. It takes ports
and answers with values — it never opens, writes to or reads from anything itself.

A print is where this shows. The evaluator does not write what a program says; it asks a
`Printer` it was handed, and the answer is the value of the print expression. Three ports,
one per reading, because `printb`, `printc` and `printd` are three readings of the same tape
and only the host knows what a reading looks like.

The alternative was collecting everything the program printed and returning it at the end.
That would have been just as pure and worse to use: the output would stop appearing while the
program runs, which is most of what a REPL is, and a run that broke halfway would have to be
careful not to lose what it had already said. A port keeps the writing where it happens and
still leaves the evaluator with nothing to write to.

**3. Errors are resolved in hosting.** A package returns an error. It never writes one, and it
never ends the process.

**4. I/O happens in hosting, and wiring happens only in `main`.** Not in a host, not in a
phase.

A host is handed what it needs and calls it. `cmd/aurora` builds the lexer, the parser, the
emitter and the evaluator — next to the flags that decide how wide a value is, since that is
what they are built with — and hands them over:

```go
cli.NewSession(cli.NewSessionOptions{Lexer: …, Parser: …, Emitter: …, NewEvaluator: …}).
	Run(ctx, source)
```

A command is a method on that session, and there is nothing else: no `Compile` step of its
own, no function that takes a bag of options and makes phases on the way. What the pipeline
is put together from is read in one place, and that place is `main`.

The evaluator arrives as a *way of making one* rather than as one. It is per program, not per
session: it holds the names a program bound, the scopes it deferred and the assertions that
ran, and `aurora test` checks one file after another. The REPL is the other way round — one
evaluator lasts the session, because a name bound on one line has to be there on the next.

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

**The end of the pipeline is a boundary too.** What the evaluator answers with — the value of
every expression, and what became of every assertion — crosses to a host exactly the way
tokens cross to the parser, so it is wire as well: `wire/eval`. Without it, `aurora test`
had to name the evaluator to report on assertions.

**What wire does not hold:** deciding to show something, and choosing where it goes. That is a
host's — `shared/printer` is where it lives, and `shared/trace` was where the other half lived
until the loggers went.

## Why shared is not hosting

A **hosting** package serves one interaction and is a leaf: only `main` depends on it.

A **shared** package serves the layer. It may be imported by any host, it may touch the world —
`shared/manifest` reads `aurora.toml`, `shared/fileutil` reads directories, `shared/printer`
writes what a program says — and it **must not know which interaction is using it**. That
second half is what keeps it from becoming the drawer where cohesion goes to die.

`shared/printer` is why the kind earns its keep: a printed value has to look the same from the
command line, from the REPL and from the page, and none of those three should decide it on its
own. It fills the evaluator's port, and it does not know which host handed it over.

A util cannot do any of that, which is why these are not utils: a util never touches the world.

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

## The words

One thing, one word. A concept with two names drifts: the two get used in different places,
then in the same place, and then somebody adds a third because neither of the two seemed to fit.

| word | what it names |
|---|---|
| **shape** | the declaration and the kind: `shape Point { x, y }` |
| **run** | the value a shape builds: tapes laid end to end |
| **field** | one tape of a run, by the name the shape gave it |
| **returns** | what an expression gives. A scope returns a shape; the keyword declares it |
| **tape** | a value: a fixed run of bytes, `tape_size` wide |
| **scope** | a block of code that is applied to a vector of values |

**`returns` is the verb, everywhere.** An expression returns a value; a scope returns a shape;
`returns Point` declares what a scope returns and the compiler checks it. There is no second
word for this — not "answers", not "promises". "Promise" in particular came from another
tradition and never said anything this language means.

**One exception, and it is a different thing.** A server *answers* a request. That is a
protocol answering a client, not an expression giving a value, and `hosting/lsp` is full of it
for a good reason. It stays.

**Declared or worked out.** What a scope returns is known two ways: it was declared with
`returns`, or it was worked out from the body. Both are the same fact and carry the same word;
which of the two it is travels beside it, because an editor may want to show the difference and
a warning may want to name it.

## Where a test goes

**With the thing it tests**, which for an artefact means the artefact's own package. Before
`wire/` existed, the IR was tested by the emitter testing what it made and the builder testing
what it read — the thing they agreed on was covered sideways, and it moved packages without
anyone noticing it was uncovered.

A test that reads the repository — `docs/`, `examples/` — finds the root by walking up to the
`go.mod`, never by counting directories: the second is a fact about where the test sits rather
than about what it is reading, and it breaks on the next move.

## What a host may still name

Every host is handed its phases, and none of them calls `lexer.New`, `parser.New`,
`emitter.New` or `evaluator.New` any more. They do still **import** those packages, to name a
type: `hosting/cli` says `parser.ParseInput` and `parser.Declarations` because that is what it
passes.

That is not a breach of rule 1 and it is written down so nobody sets out to "fix" it. What the
rule forbids is a **vital package knowing another** — the coupling that made the builder read
the emitter's opcodes. Hosting and `main` exist to put the tribes together; naming the type
you are handing over is what putting them together looks like.
