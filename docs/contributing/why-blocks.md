# Why the IR is blocks and not a list

Aurora's IR was a flat list of instructions for most of its life, and is now blocks with
terminators. This is what each form costs, what the change bought, and what it did not.

It is written down because the choice looks arbitrary from the outside — both forms hold the
same instructions, and one of them is obviously simpler to build. The reason for the other is
made of specific bugs, and they are named here.

## What the list was

Structure was written as instructions, and the instructions counted:

```
 0  OpDefer         ref 0x3038      target 10        <- my body is the next 10
 1  OpBeginScope
 2  OpGetFeed       const 0
 3  OpBigger        ref 0x3032      imm 0
 4  OpIf            ref 0x3033      target 3         <- skip 3 when false
 5  OpSave          imm 1
 6  OpReturn        ref 0x3034      ref 0x3031       <- the value of this arm
 7  OpJump          target 2                         <- skip 2
 8  OpSave          imm 0
 9  OpReturn        ref 0x3034      ref 0x3030
10  OpReturn        ref 0x3038      ref 0x3034       <- the value of this scope
11  OpIdent         name sign       ref 0x303130
```

Read that and the shape of the program is there — but it is in the **order of the list** and in
**arithmetic over indices**, and nowhere else. `target 10`, `target 3`, `target 2` are counts.
The two `OpReturn` at 6 and 9 mean "the value of this arm"; the one at 10 means "the value of
this scope"; they are the same opcode, told apart by looking at what their first operand names.

## What it cost

Not in the abstract. Each of these happened.

**Every consumer worked the structure out again, in its own way.** The evaluator walked a
cursor and let each instruction move it. The builder sliced the list to find scopes, added
counts to find where a jump landed, and measured bytes to turn those into addresses. Two ways
of counting one thing is two chances to count it differently.

**A scope written inside another was written as straight-line code.** The builder found scopes
by scanning the top of the list, so a nested one escaped it and its body ran on the way past —
its return firing in the middle of code that had not asked for it.

```aurora
ident outer = defer { ident inner = defer { 1; }; 2; };
```

answered 1 on chain where the program answers 2. It compiled, it deployed, and it said nothing.
The evaluator never had that bug, because it counted differently.

**Nothing could be reordered.** The order *was* the structure, so any pass that moved or
inserted an instruction made every count a lie. The EVM backend needs to move instructions —
a stack machine wants a value produced immediately before it is consumed — so it kept a list of
opcodes it refused to move across, and the correctness of that list was an assumption.

**Two things that should have been checkable were agreements.** That both arms of a branch
leave exactly one value, and that whoever reads a branch's value finds it: off a chain that was
a name in a map, on a chain a place on the stack. Neither was in the IR, so neither could be
checked, and a consumer that got it wrong got it wrong quietly.

**A block written inside an expression could not be told from a scope's body.** Both open with
the same instruction and close with the same return. The derivation read the inner return as
the outer scope's and ended the scope there, dropping everything written after it — again,
compiling and deploying.

## What blocks cost

They are not free, and the costs are real:

**More types.** `Block`, `Terminator`, `Target`, `Point`, `BlockID`, plus operations over them.
A list needed none of that.

**Building one is more work than appending to a slice.** Blocks have to be reserved before they
are filled, because a branch names blocks that have not been walked yet.

**A place is two numbers.** Anything that used to point into the program with one integer now
needs a block and an offset. That touched the REPL, the playground, and how a module's range is
recorded.

**Reading one is less immediate.** A list is the program in order, top to bottom. Blocks are a
graph, and following it means following terminators.

That last one is the honest complaint, and the answer to it is that the list only *looked* like
the program in order. It stopped being the program in order the moment it held a jump.

## What blocks bought

**The counting is gone, and with it the code that did it.** From `builder/evm` alone:
`PickDeferAtCursor`, `withoutNestedScopes`, `landingsOf`, `armsOf`, `targetOf`, `PositionsOf`,
`WriteCode`, `IdentManager` — 800 lines out, 200 in. From the evaluator: the cursor, the
offsets, the four entry points that took a pair of them, and the machinery a call needed to
find a scope by index and fetch its answer from an agreed key — 516 lines out, 37 in.

**Every instruction is movable, by construction.** A block has one way in and one way out, so
nothing inside it moves control. The list of opcodes the lowering refused to cross is down to
one entry, and that one is there for a different reason: what a call takes has to be put in
front of it.

**The two agreements are written down.** A target carries the values it hands over, and the
block it names says what it takes them as. `if` being an expression is structure now.

**A name for a place instead of a distance.** `brif v0 -> b1, b2` cannot be off by one. A count
can.

**Structure left the IR.** `OpDefer`, `OpBeginScope`, `OpIf`, `OpJump` and `OpReturn` are the
emitter's own, used on its way to the blocks. Every opcode `wire/ir` declares computes a value,
which means a consumer has one kind of thing to handle.

## What it did not buy

**It did not make the compiler correct.** Six bugs were found while crossing over, and two of
them were introduced by the crossing itself — a nested scope losing its structure, and an
inline block ending its enclosing scope. Both were caught by the differential harness, which is
the thing actually keeping this honest.

**It did not remove the derivation, only moved it.** The emitter still builds a list and turns
it into blocks. That derivation is one place, tested against the list it came from, and private
to the emitter — which is different from every consumer deriving its own, and it is not the
same as being gone.

**It did not change what a program answers.** Not once, deliberately: every step was proved by
the same tests passing, and the differential harness compares the chain against the evaluator
on every feature the language has.

## Would you choose this for a new language?

For Aurora, yes, and the reason is specific rather than general.

Aurora has two consumers of its IR that must agree — an evaluator and a backend that compiles
to EVM bytecode — and the whole promise is that *the same program answers the same thing on a
chain and off it*. A form that lets each consumer derive the structure its own way is a form
that lets that promise break silently, and it did, more than once.

A language with one consumer would feel the cost and not the benefit. A list is fine when
nobody else has to agree with you about what it means.

## Where to read next

[ir.md](ir.md) is what the IR is now and how to run it, written for whoever wants to build a
consumer of their own.
