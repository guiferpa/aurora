# The IR, and how a consumer runs it

What the emitter hands over, what each piece of it means, and what a program that reads it has
to do. If you are writing a second backend — another chain, another machine, an interpreter of
your own — this is the whole contract. You should not have to read the evaluator to write one.

The types are in `wire/ir`. The evaluator (`evaluator/`) and the EVM backend (`builder/evm/`)
are two readers of the same thing, and everything below is true of both.

## A program is blocks

```go
type Program struct {
    Blocks      []Block
    Expressions []Expression
    Warnings    []diag.Warning
}
```

`Blocks[0]` is where the program begins. Everything else is reached from there — by control
arriving at it, or by a name holding it.

```go
type Block struct {
    ID     BlockID
    Params []Label
    Insts  []Instruction
    Term   Terminator
    Origin Origin
}
```

**A block has one way in and one way out.** Control arrives at its first instruction and leaves
at its terminator, and nowhere else. That is not a convention — there is no instruction that
can send control anywhere, so it cannot be broken.

Two things follow, and both are what a consumer is buying:

- **Every instruction in a block is movable.** A consumer may hold one back and emit it next to
  whoever takes its value, or drop it when nobody does. The EVM backend does exactly that,
  because a stack machine wants a value produced immediately before it is consumed.
- **Nothing has to be counted.** A jump names a block. A scope is a block. There is no length
  to read and no offset to add.

## An instruction computes one value

```go
type Instruction interface {
    GetLabel() []byte      // the name the value it computes is known by
    GetOpCode() byte
    GetOperands() []Operand
    GetOrigin() Origin     // where in the source it came from; may be unknown
}
```

Three-address code, with the operand count free: a construction of n values — the fields of a
shape, the items of a tape — is **one instruction with n operands**, not a chain of n. A
consumer never has to recognise that a run of instructions was one thing.

Every opcode `wire/ir` declares computes a value. There is no opcode for control.

### An effect says what else it does

An instruction computes a value. Some of them also **do** something, and that is a different
fact about the same instruction:

```go
ir.EffectOf(op)   // Pure, Reads, Writes or Escapes
ir.MayCross(a, b) // whether two instructions may be put in the other order
```

| effect | means | what has it |
|---|---|---|
| `Pure` | the value it leaves is all it does | arithmetic, comparison, logic, the tape operations, a literal, a call |
| `Reads` | depends on state something else can change | `OpLoad`, which reads the frame |
| `Writes` | changes that state, or says something whose order matters | `OpIdent`, which writes the frame; the three prints |
| `Escapes` | leaves the program: another contract runs, and may come back in | nothing yet |

**An effect is not a value.** It is not a tape, not a run, it reaches no bytecode, and a
program cannot name it — the language has no types and this does not add one under the floor.
It exists while compiling, to answer one question, and then it is gone. It is metadata about
the instruction, of the same kind as the `Origin` that says which line it came from.

#### Why it exists

Because a consumer is allowed to move instructions, and until it was written down, nothing
said which ones. `builder/evm` holds an instruction back and emits it next to whoever takes
its value, because the EVM wants operands on the stack in an order the program was not written
in. That is sound for an instruction whose value is the whole of what it does. For one that
writes, it is a different program:

```
printd a;
printd b;
```

What kept that from happening was three lists of opcodes inside `builder/evm`, and **a list of
opcodes is how this project has been wrong before**: the lowering once decided which operands
named values by a list of the four arithmetic instructions, so a name bound inside a scope was
not on it, and `ident x = feed(0); x + feed(1);` answered 4 on a chain where the program
answers 7. Whoever writes the next opcode has to remember every list, and nothing reminds
them. An effect is the same fact said once, next to the opcode, where forgetting it is
visible.

#### What it buys somebody writing Aurora

Nothing they can see today, and that is worth being honest about: the only instruction whose
order matters right now is the print, and a print does not reach a chain by decision. Nobody
has been bitten.

It buys the next thing. The moment a program can touch state that outlives one expression —

```
s.set("balance", 100);
s.get("balance");
```

— the order those two run **is** the program, and there is no way to notice afterwards that a
backend swapped them: the contract answers a number either way. This is exactly the class of
failure Aurora exists to remove, since the whole point of the backend is that the same program
answers the same thing on a chain and off it. So the rule lands before storage does, rather
than after the first program gets it wrong.

#### The rule

Two instructions may swap unless one of them changes something the other would notice. The
sixteen pairs are derived from that rather than listed:

```go
func MayCross(a, b Effect) bool { return !disturbs(a, b) && !disturbs(b, a) }
func disturbs(a, b Effect) bool { return changes(a) && notices(b) }
```

It started stricter — swap only if one of them is `Pure` — and a measurement over real
programs said stricter was wrong. `a - b` puts the two loads on the stack in the order the
subtraction wants, which swaps two `Reads`, which is safe. Two reads of the frame commute; the
strict rule refused them, in the most ordinary program there is.

The effect is a function of the opcode, not a field on the instruction. A field is filled by
whoever builds the instruction, so it can be filled wrong and nothing notices; an opcode has
one effect wherever it appears. The day an instruction's effect depends on more than its
opcode, the field arrives then and has a reason to.

`require` is not an effect and will not become one. It does not compute and it does not
annotate — it **branches**, to a block that reverts, and control has lived outside the
instructions since a block gained a terminator.

### Operands say what they are

This is the part that most repays reading. The same bytes mean different things, and the kind
says which:

| kind | what it is |
|---|---|
| `KindRef` | the label of a value another instruction in this block left |
| `KindImm` | the value itself, a tape — the program wrote it down |
| `KindConst` | a number the operation takes about itself: an index, a width, a count |
| `KindName` | a name that outlives a block — a binding, a scope being called |
| `KindBlock` | a block of this program: what a scope is worth |
| `KindText` | bytes written for a person, never a value — an assertion's message |

The difference between `Imm` and `Const` is the one to get right. Both are numbers written into
the instruction. An `Imm` is a **value of the program** and obeys the tape width; a `Const` is
something the operation says about itself and does not. `OpField ref, Const(1), Const(2)` reads
field 1 of a run of 2 — neither number is a value, and neither is narrowed to a tape.

The difference between `Ref` and `Imm` is what tells a stack machine who puts a value on the
stack: a `Ref` was put there by whoever produced it, an `Imm` reaches the stack at the
instruction that takes it.

## A terminator says where control goes

```go
type Terminator struct {
    Kind    TermKind   // Ret, Br, BrIf
    Cond    Operand    // BrIf
    Value   Operand    // Ret
    Targets []Target   // Br: one. BrIf: two.
}

type Target struct {
    Block BlockID
    Args  []Operand
}
```

There are three kinds and there is no fourth. A block either **answers**, **goes** somewhere,
or **chooses** between two somewheres.

`BrIf` goes to `Targets[0]` when `Cond` is not the neutral tape, and to `Targets[1]` when it
is.

### Values are handed over, not left somewhere

A `Target` carries `Args`, and the block it names takes them under the names in its `Params`.
That is how the arms of a branch meet:

```
b0:              v0 = bigger feed(0), 0
                 brif v0 -> b1, b2
b1:              v1 = 10
                 br b3(v1)
b2:              v2 = 20
                 br b3(v2)
b3(v0):          ret v0
```

Both arms hand their value to `b3`, which takes it under the name `v0` — the name the whole
branch answers by. **This is what makes an `if` an expression**, and it is written down rather
than agreed: whoever reads the value afterwards reads it under that name, without knowing which
arm ran.

It used to be an agreement kept in two places and stated in neither — off a chain a name in a
map, on one a place on the stack — and an agreement nothing can check is one a consumer can
break quietly.

A block written inside an expression is the same shape: the run up to it, the block itself, and
where the run carries on, with the value handed to the third.

## What a scope is

A scope is a block, and **what a scope is worth is the number of that block**:

```
b0()
    0x30 OpSave  block 1              <- the scope is worth block 1
    0x31 OpIdent name area  ref 0x30  <- the binding is an ordinary binding
    0x32 OpCall  name area  imm 7
b1(_, _)                              <- takes two values, unnamed
    OpGetFeed const 0
    OpGetFeed const 1
    OpMultiply ref ref
    ret ref
```

Three things to read off that:

**A scope's parameters are unnamed.** They arrive as the vector applied to it, and `OpGetFeed`
reads a position of that vector rather than a name. `len(block.Params)` is how many positions
the body addresses — the highest it feeds, plus one — and a call applying fewer is legal: a
position nobody filled answers the neutral tape.

**A scope is reached by being named, never by control arriving at it.** No terminator points at
`b1`. That is what makes reachability from `Blocks[0]` the right way to ask "which blocks does
this run pass through", and `ir.Reaches` does exactly that.

**The binding is an ordinary instruction.** A scope nobody binds still has a value, which is why
`defer {};` written on its own is worth something.

## Running one

The whole of it:

```
arrive at a block
  hand it the values the target carried, under the names in Params
  run its instructions in order
  read its terminator
    Ret   -> the run answers with that value
    Br    -> arrive at the one target
    BrIf  -> arrive at one of the two, on the condition
```

`evaluator/evaluator.go`'s `RunBlocks` is that loop and little else.

### Where names live

A consumer that keeps values under names — an interpreter — needs one more rule, and it can be
read off the arrival rather than carried:

- **arriving with values closes a place for names.** A branch handing the value of the arm that
  ran, a block handing its value on: in both, whatever was bound inside is done with. Read the
  values first, then close, then write the names down in the place that will read them.
- **arriving with none opens one.** Going into an arm, or into a block written inside an
  expression. What it binds is its own, so the `x` of an inner block is not the `x` of the one
  around it.

The values applied to the running scope come in with an opened place: a block is not applied
anything of its own — nothing calls it, control walks into it — so `feed` still means the
enclosing vector.

A consumer that keeps values somewhere else — a stack machine — ignores all of this. The EVM
backend has no places for names in this sense; it lays out one run of memory per activation.

### A call

`OpCall name, args...` — the first operand names the scope, the rest are the values applied.

Resolve the name to a value; that value is the number of a block. Build somewhere for the
applied values, run from that block, and what its `Ret` answers with is what the call answers
with.

A value that names no block is a value with no scope behind it. Block zero is the top of the
program, which nothing is bound to, so it is not a scope either.

A number equal to a block number **is** that scope, and that is deliberate: an untyped language
cannot distinguish values that are the same bytes, and does not pretend to. It is the same
bargain the language already makes for `true` and `1`.

## Where each top-level expression begins

```go
type Expression struct {
    At    Point   // Block, and how far into it
    Label []byte
}
```

A runner that wants to answer where each expression happens — rather than all of them at the
end — runs from one `At` to the next. The REPL and the playground do this, so that a line
holding several expressions interleaves what it printed with what each one was worth.

It is two numbers rather than one because **an expression is not a stretch of anything**: one
containing a branch spans four blocks, and the expression after it begins in the block where
that branch's arms meet.

## What a program of several files is

The loader compiles each file on its own — each numbers its blocks from zero — and joins them:
every block after the first file moves (`ir.Shifted`), and each file's endings become goings
into the next (`ir.GoesOnTo`). `Range.Top` is where each file begins.

A scope is untouched by that joining, and the reason is the whole point of a block: **a binding
names a block, and naming is not going.** So a scope is never reached from the top of a file,
and still ends by answering, which is what a call needs it to do.

## Writing a consumer

The shortest honest list:

1. Walk `ir.Reaches(blocks, 0)` for the code the program runs through. Anything not in it is a
   scope, reached by being named.
2. For each block: hand over the params, run the instructions, read the terminator.
3. Read operands by `Kind()`. Never guess from the opcode which operand is which — that is how
   this backend produced a contract that answered 4 where the program answered 7.
4. For a call, resolve the name to a block number and run it.
5. If you keep values under names, follow the open/close rule above. If you keep them on a
   stack, you do not need it.
6. If you move an instruction — and you will, if you emit for a stack — ask `ir.MayCross`
   first. Never keep your own list of which opcodes are safe to move.

And one warning worth its own line: **anything you derive rather than read is a second
description of the same program, and two descriptions drift.** Every silent bug this compiler
has had came from one consumer counting something a different way from another.

## Why it is not a list

It was, until it was not. That is [why-blocks.md](why-blocks.md).
