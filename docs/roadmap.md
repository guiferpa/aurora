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

The point of the backend is that **the same program answers the same thing on a chain and off
it**. Since the differential harness exists (`internal/cli/evm_harness_test.go`) that is no
longer a hope: an EVM is built in memory, the contract is deployed and called, and the answer
is compared against the evaluator. Arithmetic across a dispatcher is proved; everything below
is not.

The gap is wider than it looks, and it is silent, which is the dangerous part: a contract
using any of the following **compiles successfully and does nothing on chain**.

- `if`, `jump`, `call`, `assert`, the tape operations, and the struct instructions (`OpJoin`,
  `OpField`) produce no bytecode at all. `WriteCode` covers arithmetic, `OpSave`, `OpIdent`,
  `OpLoad`, `OpGetFeed` and `OpReturn`. They are not refusals — a tape is at most 32 bytes and
  an EVM word is exactly 32, a struct is a run of words in memory, and a call is a jump with a
  return address. They are simply not written yet.
- **`printb`, `printc` and `printd` are logs and do not compile**, by decision. What a program
  says on the way is for whoever is watching it run, not for the chain.
- **Arithmetic is not masked to the tape width.** At `--tape-size 1`, `255 + 1` is `0` in the
  evaluator and `256` on chain: the EVM wraps at 2²⁵⁶ and the tape wraps at 2⁸. The harness
  reports it; the fix is a mask after every operation.
- **Events are missing entirely.** `LOG0`–`LOG4` (`0xA0`–`0xA4`) are not in the opcode table.
  On chain they are the only way a contract says anything, and they are the honest target for
  whatever Aurora ends up calling logging.
- Jump targets and memory offsets are written with `PUSH1`, which caps a runtime at 256
  bytes and identifiers at about seven memory slots. `PUSH2` lifts it.

`aurora build` now **says what it could not carry**, once per feature, in the order the
program uses it — so a binary that does less than the source said is at least announced. What
is missing is a place to point at: the IR carries instructions, not lines, so a warning names
the feature and not where it was written.

## Simulating a call off the chain

`aurora call` speaks to a network and nothing else, so there is no way to ask for one scope by
name with these arguments and have the evaluator answer — the call has to be written into the
source. The differential harness works around it by appending the call to the program, which
is exactly the shape of what is missing: a local call that takes a function and its arguments
and answers from the evaluator, the same way the chain would.

---

## The evaluator's instruction trace

`-l evaluator` used to print each instruction as it ran. It printed from inside the
evaluator, which a phase may not do — a phase returns values and does not write — so it went
when the loggers left the phases.

Getting it back means the evaluator **returning** a trace rather than writing one: values in,
values out, like everything else it does. What has to be decided first is what it costs, since
a recursive program executes hundreds of thousands of instructions and a slice of all of them
is not free. The other four loggers survived the move because a phase's whole output is a
value already; this one is the exception, and it is the only thing lost.

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

- **The playground has nothing testing it.** `cmd/playground` builds only for `js/wasm`, so
  `go test ./...` never compiles it, and there is no runner for the page itself. What can be
  tested has been pushed down — reading a tape size out of text lives in `byteutil`, with its
  own cases — and the rest, the control and the re-run it triggers, is checked by looking at
  the page. A `go_js_wasm_exec` harness, or a headless page driving Run, would close it.

- The playground workflow runs on push to `main`, which at a version bump happens seconds
  before the tag exists — the deployed page then reports the previous version. Every release
  needs `gh workflow run playground.yml` by hand. Triggering on `push: tags` would fix it.
