# Roadmap

What Aurora does not do yet, and what each thing would take. It is a working list rather
than a promise: order and scope are decided as we go.

Everything here was checked against the compiler, not assumed. Where a limit has a cause in
the code, the cause is named — that is usually the actual size of the work.

---

## Values that outlive a tape

The premise is that every value is a tape: a fixed run of bytes, `tape_size` wide. Three
wanted things all run into the same wall.

### Text longer than a tape

`"Guilherme"` is nine bytes and needs `--tape-size 16`. There is no way to hold more text
than a tape holds, and the two candidate answers were weighed when reels were removed: a
field of declared width (`name[16]`), and a handle into a pool of literals. Neither was
taken, because both are less useful than what unblocks them below.

### Reading a position that is computed while the program runs

`head t 2` compiles and `head t i` does not. The index is written into the instruction as an
immediate (`emitter.go`, `OpHead`/`OpTail`/`OpField`), so it has to be known at compile
time. Nothing can walk a run of tapes, which is why there is no indexing, no slicing and no
parsing of anything.

### Building a value while the program runs

A run of tapes exists only as a literal: a shape or a text. There is no concatenation, so
nothing can be assembled — no joining text, no growing a list, no formatting a message.

**Together these three are one piece of work**, and it is the largest one on this list. An
index as a label rather than an immediate, an instruction that joins two values, and a way
of asking a value how long it is. It touches the emitter, the evaluator and the EVM writer,
and it is what turns Aurora from a language of fixed values into one that can process them.

---

## Scopes

### Closures

A `defer` body sees the **caller's** chain, not the scope it was written in
([defer_scope_visibility.md](defer_scope_visibility.md)). That is deliberate and documented,
and it means a scope cannot carry values with it.

Two consequences behave badly enough to be bugs rather than design:

- a `defer` as the last expression of a block loses its value —
  `ident r = { ident x = 10; defer { x; }; }; printd r;` answers `0`;
- a `defer` returned by another `defer` is not callable — the call runs nothing, with no
  error.

### Calling a scope held in a value

`OpCall` resolves a **name** in the environ, not a value, so a deferred scope kept in a
shape field cannot be called: `o.run()` does not even parse. A field can hold a scope's
index — that works — and there is no way to apply values to it. Anything higher-order needs
this.

---

## Shapes

Which shape a value is, is worked out while compiling. A call has a shape because the
compiler reads what the scope's body ends with, looking through an `if` when both arms agree,
so a field is reached off a call without anybody writing a word — across a module too, since
what a scope returns is what crosses. `returns` is a declaration the compiler keeps, not the
way to be understood, and `as` is for a value the program never says the shape of.

Three places it stops, and each of them falls back on `as` or `returns`, which is what those
are for:

- **A scope written further down the file is not known yet.** `ident a = defer { b(); };`
  before `b` is bound reads a table that has no `b` in it, so `a` returns nothing as far as
  the compiler is concerned. Reading a file twice would answer it, and nothing reads a file
  twice today.
- **A scope that calls itself is not known either**, for the same reason and by the same
  mechanism: the name is not in the table while its own body is being read. This is what
  keeps the walk from being a recursion at all, so it is a limit bought on purpose.
- **The editor knows what the compiler knows, and reads tokens where it did not get there.**
  The language server hands the parser its table and reads it back, so what the compiler
  worked out is what the editor offers — and because the table is filled as the file is read,
  a parse that broke on the line being typed still leaves everything above it understood. The
  walk of the tokens stays underneath, matching patterns, for the document that breaks before
  the parser reads anything at all. It recognises less, and less is all it has to be.

## Modules

A program is as many files as it needs. `use a/b/c as x;` reads `a/b/c.ar` from the source
root, every import carries an alias, and a name inside a module is written with the module in
front of it — so which `add` a call means is answered while compiling.
[modules.md](modules.md) is the reference; [module_system_design.md](module_system_design.md)
is why it has that shape.

What it does not do yet:

- **There is no `private` for a shape either.** Every shape a module declares is reachable
  as `m.Point`, the same way everything it binds is. Marking part of it later breaks nothing.
- **There is no `private`.** A module offers everything it binds with `ident` at the top. The
  boundary exists now, so marking part of it later breaks nothing.
- **The language server follows an import, and stops short of one thing.** It reads what a
  document imports on every pass, from the editor's buffers before the disk, so a module that
  is not there and a name a module does not have are underlined where they were written, and
  what a module declared is offered after the dot — its shapes included, and the fields of a
  name whose shape came from another file, because what the imports offer is written down
  before the document is read. A name jumps to where it was declared, here or in the module
  it came from. What it does not do is watch a file nobody has open, so editing a module
  outside the editor updates what depends on it only when that file is touched. It does not
  need the resolver; it needs a capability the server does not have.
- **A rename stops at the file it is asked in.** A name bound inside a scope and an alias are
  renamed everywhere they are written; a name bound at the top of a file is refused, with the
  reason, because another file may be importing it. Completing those needs the walk that is
  missing: which files of a project import this module, and under which alias. Nothing walks
  a project today — the resolver reads what a file names, never who names it.
- **Nothing lists where a name is used.** There is no `textDocument/references`. Renaming
  works the list out and shows it to nobody, which is the whole of what is missing.
- **Nothing measures what following an import costs.** Every pass reads and parses every
  module the document imports, with no cache, which is the honest place to start: a cache has
  invalidation, and invalidation is a bug that reads as the editor lying. The shape to copy
  when it is measured and found wanting is next door — the width of a project is cached per
  directory and invalidated by the manifest's mtime.
- **Nothing lists a dependency but the file system.** There are no third-party packages, so
  there is nothing for the manifest to name yet.
- **The playground has no standard library.** What comes with the language is read from a
  directory the toolchain installed, and the playground is wasm in a browser, where there is
  none. So `use std/evm/storage` works everywhere `aurora` runs and nowhere the playground
  does. What would close it is embedding the same files in the binary and reading the directory
  over them when it is there — one source, two ways to reach it — and it waits until somebody
  wants the standard library in the playground.
- **Nothing installs the standard library but `make install-std`.** A release ships binaries
  and not the files beside them, so somebody who downloads one has the language and not its
  library, until they clone this repository and run that.

  **Carrying it inside the binary was tried and refused.** It works, it costs 33 KB, and it
  makes a second copy that can disagree with the first: install the library with one version,
  upgrade the binary, forget to write it out again, and the toolchain is one version while its
  library is another — silently, which is what one directory was chosen to prevent. Go does not
  do it either; its library is files in `$GOROOT/src`, shipped in the same archive as the
  binary, and upgrading replaces the whole tree.

  So the shape of the answer is that one: the release archive carries `std/` beside the
  binaries, and the toolchain finds it. What that costs is a place to look that is not one
  place — a `.deb` puts the binary in `/usr/bin`, Homebrew reaches it through a symlink, and a
  tarball lands wherever it was unpacked — and that is the search path this design set out to
  avoid. It waits until somebody is stopped by it.

Two things are decided against rather than missing. **An import is not passed on**: if `main`
uses `a` and `a` uses `b`, `main` writes its own line for `b`, so a name used in a file has
its origin declared in that file. And **an import is not a value** — `ident m = use ...;` was
considered and refused, with the reasoning in the design document.

---

## The EVM backend

The point of the backend is that **the same program answers the same thing on a chain and off
it**. Since the differential harness exists (`hosting/cli/evm_harness_test.go`) that is no
longer a hope: an EVM is built in memory, the contract is deployed and called, and the answer
is compared against the evaluator. What it proves is arithmetic across a dispatcher at any
tape width — a result that leaves the width is cut back to it, the way the evaluator does —
names bound inside a scope, branches, the comparisons and `and`/`or`, a scope calling another
as deep as it nests, shapes, and the tape operations.

**A contract keeps things now.** `sload` and `sstore` are the two EVM opcodes, and
`std/evm/storage` is two scopes over them, so `s.set(1, s.get(1) + feed(0))` is a deposit that
outlives the transaction. The harness proves it: kept, kept over, a key nothing was written
under, and a running total — through the standard library, which is a call reaching a module
reaching the machine.

Nothing a program can write is missing from it any more, and a shape is no longer the
exception it used to be: **a run of any width reaches a chain**. It lives in memory rather than
on the stack, laid tape after tape in the order it was written, so what a contract hands back
is the same bytes the evaluator does rather than the same number. A stack item is exactly a
word, which is four tapes of eight, and a shape wider than that used to be refused rather than
written — a program that ran off a chain and could not reach one. Nothing goes on the stack
even when it would fit, because a run kept two ways is two things every consumer has to tell
apart.

What a contract does not get is what was decided not to reach a chain, and two limits that are
written down:

- **A scope only reaches what it bound itself**, so everything a scope works on arrives as a
  value applied to it. What would let a scope read a name from around it is a static link —
  the frame carrying where the scope was written — and it is not built.
- **A contract keeps things, and still says nothing and does not know who called it.** Storage
  is written — `sload` and `sstore`, with `std/evm/storage` over them — and it answers the same
  off a chain, where a map stands in for what one transaction sees. What is left of that list is
  an event, a caller, and a way to refuse a transaction. Each is its own decision, and each has
  the same question under it: what does it mean off a chain, where there is no log and nobody
  calling. `rfcs/` is where those are argued.
- **Only a scope bound at the top of a program can be called.** A scope written inside another
  is not written at all: on a chain the name it was bound to holds the neutral value, and
  calling it is refused rather than written as a jump to an address no scope has.
- **A tape literal built out of values a program works out** is refused rather than written
  wrong — `pull t x` one value at a time is written.

- **`printb`, `printc` and `printd` are logs and do not compile**, by decision. What a program
  says on the way is for whoever is watching it run, not for the chain.
- **Events are missing entirely.** `LOG0`–`LOG4` (`0xA0`–`0xA4`) are not in the opcode table.
  On chain they are the only way a contract says anything, and they are the honest target for
  whatever Aurora ends up calling logging.

`aurora build` now **says what it could not carry**, once per feature, in the order the
program uses it, and names the line the program first used it on — so a binary that does less
than the source said is announced, in the form an editor follows.

## Asking and changing

A call is a question and a transaction is a change, and they are two commands because they are
two things:

```
aurora call read      answered against the state as it is, costing nothing, keeping nothing
aurora tx inc         sent, paid for, mined, and what it changed stays
```

Nothing decides between them from what a program looks like. What the compiler knows is used to
refuse the wrong one: a scope that keeps something, asked as a question, is a write thrown away
— it answers, looking right, and the chain is exactly as it was. That happened on a real chain
before this existed, and a counter answered 1 every time.

Which scopes keep something is read from the instructions they reach, following the scopes they
call — because with a standard library the scope somebody writes holds no `sstore` at all:
`s.set(...)` is a call, and the write is one file away.

**What a chain keeps is narrower than what an instruction writes**, and the two were read as
one for a while. A binding writes the frame, because a frame is memory and a load after a store
must stay after it — and a frame is gone when the scope is, and nobody outside the program ever
saw it. A print is sharper: its effect is a write, and it reaches no bytecode at all. So a scope
that only computed was refused for changing the chain. What counts is storage, and events and
an external call when they are written.

What is missing:

- **A transaction has one gas limit for every scope.** It is 500,000, which is enough for
  anything a program can write today and nothing decides it per call. What would replace it is
  asking the network to estimate, which is a round trip somebody has to want.
- **Nothing reads what a transaction answered.** A scope returns a value, and a mined
  transaction keeps it nowhere a caller can see — only logs and storage cross that line, and
  logs do not reach a chain by decision. So `aurora tx` says what it spent and where it landed,
  and what the scope returned is read with a `call` afterwards.

## Simulating a call off the chain

`aurora call` speaks to a network and nothing else, so there is no way to ask for one scope by
name with these arguments and have the evaluator answer — the call has to be written into the
source. The differential harness works around it by appending the call to the program, which
is exactly the shape of what is missing: a local call that takes a function and its arguments
and answers from the evaluator, the same way the chain would.

---

## Name resolution is done twice

The emitter knows where every name lives and does not say, so `builder/evm` works it out again
in `Scope.Names`. It is the third of the six costs `rfcs/ir.md` named and the only one still
open, and it is open because it is the only one whose price is higher than what it returns.

The other five each paid in a bug found. This one pays in gas and bytes, and buys nothing
somebody writing Aurora would notice.

There are two ways out and both cost more than the line in the RFC suggests:

- **A local becomes a numbered slot**, decided by the emitter, and the backend does arithmetic
  rather than keeping a table. It closes the cost — the name never reaches the backend — but
  only for a scope's own names: the top of a program keeps names, because a module offers what
  it binds by name and the REPL keeps one evaluator for a session, so what a line binds the next
  line sees. Two mechanisms for one thing, which is what this compiler spends its time removing.
- **A local becomes a value and stops being memory.** It is what an immutable binding already
  is, and `OpIdent` and `OpLoad` would go entirely. On the EVM a value read deep in the stack
  needs a DUP, and past sixteen it needs memory anyway — so the backend would need a stack
  allocator with spilling, which is the thing putting every bound name in memory was avoiding.

So it waits for somebody to measure gas and be bothered by it. That is the trigger, and it is
written here so the next person does not rediscover the argument.

## What the IR asserts

`ir.Verify` asserts all six things the IR was reshaped to make assertable, and `builder/evm`
refuses a program that fails any of them rather than writing bytes for it.

Two things stay deliberately unasserted: what a call hands over, because a scope has no arity —
running one is applying a vector of values to it; and anything about a stack, a frame or an
address, because those mean something on one target and nothing on another.

What is left is not about the IR: **`Scope.Names` is name resolution done again.** The emitter
knows where every name lives and does not say, so `builder/evm` works it out a second time —
the third of the six costs `rfcs/ir.md` named, and the only one still open.

## Seeing what a phase produced

`-l lexer,parser,emitter` used to print what each phase made, and it is gone. It went in two
steps, and both are worth remembering before anyone brings it back.

First `-l evaluator` and `-l builder` went, because they printed from inside a phase, which a
phase may not do. Then the other three went with the flag itself: what they showed was a
developer reading a shape, and a shape is what a test checks — every phase has tests that read
what it answers with, and those do not depend on someone running a command and looking.

If it comes back, it comes back as a port, the way printing did: the phase hands each thing to
whoever asked, and nothing decides to write. The one that costs something to design is the
evaluator's, since a recursive program executes hundreds of thousands of instructions, and
neither keeping them nor handing them over one at a time is free.

## Smaller, decided things

- **Comparison chains group to the right.** `3 bigger 2 bigger 1` is `3 bigger (2 bigger 1)`.
  `ParseRelExpr` has the shape `ParseMultExpr` and `ParseBoolExpr` had before they were
  fixed; `^` is then the only level meant to recurse right.
- **No modulo.** A remainder is `n - (n / d) * d`, which is also the expression that made the
  associativity bug matter.
- **A node the emitter does not know emits nothing**, silently: the dispatch in
  `EmitInstruction` ends by answering with the neutral value, because some nodes are handled by
  their parent and it cannot tell those from a node nobody implemented. So a node type added to
  the tree and wired into the parser, but never given a case, compiles a program that answers
  zero. Saying so instead means an error where there is no way to raise one today —
  `EmitInstruction` and the twenty-five `emit*` functions answer with a label and nothing else.
- **`aurora test` discards what a program prints** (`io.Discard`), so a test cannot see the
  output of the source it runs.
- **Mutable state** has a proposal and no implementation
  ([state_management.md](state_management.md)): no keyword, no node, no opcode.

---

## Tooling

- **The playground has nothing testing it.** `cmd/playground` builds only for `js/wasm`, so
  `go test ./...` never compiles it, and there is no runner for the page itself. What can be
  tested has been pushed down — reading a tape size out of text lives in `byteutil`, with its
  own cases — and the rest, the control and the re-run it triggers, is checked by looking at
  the page. A `go_js_wasm_exec` harness, or a headless page driving Run, would close it.

- The playground workflow runs on push to `main`, which at a version bump happens seconds
  before the tag exists — the deployed page then reports the previous version. Every release
  needs `gh workflow run playground.yml` by hand. Triggering on `push: tags` would fix it.
