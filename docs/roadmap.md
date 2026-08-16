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

A run of tapes exists only as a literal: a struct or a text. There is no concatenation, so
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
struct field cannot be called: `o.run()` does not even parse. A field can hold a scope's
index — that works — and there is no way to apply values to it. Anything higher-order needs
this.

---

## Modules

One file is one program. `use` was rolled back in v0.2.0-alpha because it never resolved a
symbol; the shape a resolver and loader should have is written down in
[module_system_design.md](module_system_design.md), including that every import carries a
mandatory alias.

Note that `.` is taken now: it reads a struct field, so a module accessor has to share it or
pick something else.

---

## The EVM backend

The gap here is wider than it looks, and it is silent, which is the dangerous part: a
contract using any of the following **compiles successfully and does nothing on chain**.

- `if`, `jump`, `call`, `printb`/`printc`/`printd`, `assert`, the tape operations, and the
  struct instructions (`OpJoin`, `OpField`) produce no bytecode at all. `WriteCode` covers
  arithmetic, `OpSave`, `OpIdent`, `OpLoad`, `OpGetFeed` and `OpReturn`.
- **Events are missing entirely.** `LOG0`–`LOG4` (`0xA0`–`0xA4`) are not in the opcode table.
  On chain they are the only way a contract says anything, and they are the honest target for
  whatever Aurora ends up calling logging.
- Jump targets and memory offsets are written with `PUSH1`, which caps a runtime at 256
  bytes and identifiers at about seven memory slots. `PUSH2` lifts it.

A first step that costs little: **warn at compile time** when a program uses something the
backend drops, instead of writing a binary that lies.

---

## Smaller, decided things

- **Comparison chains group to the right.** `3 bigger 2 bigger 1` is `3 bigger (2 bigger 1)`.
  `ParseRelExpr` has the shape `ParseMultExpr` and `ParseBoolExpr` had before they were
  fixed; `^` is then the only level meant to recurse right.
- **No modulo.** A remainder is `n - (n / d) * d`, which is also the expression that made the
  associativity bug matter.
- **`aurora test` discards what a program prints** (`io.Discard`), so a test cannot see the
  output of the source it runs.
- **Mutable state** has a proposal and no implementation
  ([state_management.md](state_management.md)): no keyword, no node, no opcode.

---

## Tooling

- The playground workflow runs on push to `main`, which at a version bump happens seconds
  before the tag exists — the deployed page then reports the previous version. Every release
  needs `gh workflow run playground.yml` by hand. Triggering on `push: tags` would fix it.
