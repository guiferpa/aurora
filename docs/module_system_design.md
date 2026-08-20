# Design: the module system

Why modules have the shape they have. What they *do* is in [modules.md](modules.md); this is
the reasoning under it, and what was decided against.

The short version: **a module is a file, a name carries the module it belongs to, and each
module keeps its names in an environ of its own.** Everything below follows from those three.

---

## What went wrong the first time

Aurora had a namespace layer with `use path::to::ns as alias;` and a `linker` package. It was
removed in v0.2.0-alpha because it never worked end to end. Three separate mistakes, worth
naming so they are not repeated:

**1. A directory was the unit.** The entry point's *directory* defined the namespace, and every
`.ar` file in it was compiled into one program. Ten independent examples in `examples/` collided
on `ident a`, and passing one file compiled its neighbours too.

**2. Symbol identity was never implemented.** The parser recorded a namespace on every
identifier and produced a `use` declaration, and then *nothing read them*. `m::add(1, 2)`
emitted a call to `add` with no notion of where it came from.

**3. What made multi-file work was the accident, not the feature.** Because the whole directory
was concatenated into one flat scope, names found each other by luck. The explicit mechanism
contributed nothing.

The lesson, and the reason the second job below exists: **a module system is not a file-loading
feature, it is a name-resolution feature.** Loading files is the easy half.

---

## Three jobs

| Job | Question | Where |
|---|---|---|
| **Resolver** | Which files is this program made of, and in what order? | `resolver` |
| **Loader** | How does that become a program? | `loader` |
| **Binding** | Which declaration does this name refer to? | half in `parser`, half in `loader` |

The third is the one the last attempt lacked. It is split because its two halves need different
things:

- **Translating** `x.add` into `a/b/c.add` needs only what is in the file: the `use` line above
  it. No knowledge of any other module.
- **Checking** that `a/b/c` really has an `add` needs that module's table.

Splitting them keeps the parse of a file independent of every other file — which is what the
language server needs to answer quickly, and what lets modules be parsed in any order — and
puts the check where the cross-file knowledge is. The parser visits every node anyway, so it
notes the qualified names it saw and hands the list over with the tree; the loader reads it
against the export tables. Nothing walks a tree twice.

---

## The prefix

Inside a module, every identifier is written with the module in front of it: in `a/b/c`,
`ident base` is written `a/b/c.base`. Binding and mention alike.

Uniformly, which is what makes it cheap: it is a constant renaming of one file's names, so the
parser needs no scope analysis — it does not have to know what is a top-level binding and what
is a local one, because every mention is renamed the same way and they go on finding each
other.

The separator is a character no identifier can hold. The lexer accepts letters, digits and
`_ - ? ! > <`, so `a/b/c.base` is a name nobody could have typed, and reading one back is
unambiguous.

**The file somebody asked to run has no module and keeps the names it typed.** It is unique by
construction and every imported module has a prefix of its own, so nothing can collide. That is
not a special case for its own sake: it is what keeps every program that existed before modules
compiling to the same instructions, the language server and the REPL included, since a file
compiled on its own is nobody's module.

**A `shape` is left out of all of this.** Its name never reaches an instruction and never
leaves the file, so it is looked up and reported as it was typed while everything around it is
renamed.

---

## One environ per module, and two hops

A name is looked up in two steps:

1. **the chain** — every scope open around here, which is what a deferred scope sees of
   whoever called it, and which is what a lookup has always been;
2. **the environ of the module the name says it belongs to**, which the prefix supplies.

The second hop is why the prefix exists at all. A scope written in one file and called from
another runs at the head of its caller's chain — that is what `defer` means here — and still
has to find what its own file bound:

```
a/b/c.ar   ident base = 10;
           ident add  = defer { feed(0) + feed(1) + base; };

main.ar    use a/b/c as x;
           ident base = 3;
           printd x.add(base, 1);
```

This answers `14`. Answering `7` would mean the imported scope had read the caller's `base`,
which is the accident the old namespace layer ran on.

A module's environ stands on its own with no chain behind it, so a module sees what it declared
and what it imported and nothing of whoever imported it. They are kept in one flat map from the
module's name to its environ, born on the environ every chain ends at.

### Why a flat map and not a tree

`a/b` and `a/b/c` have no relationship. `a/b` may not even be a file, and neither reaches the
other's names, so a tree by path segment would invent a node for something that is not there
and claim a parent-child link that does not exist. In DNS the tree is earned because delegation
is real; here nobody delegates. **The name is hierarchical; the storage does not have to be.**

The import graph is the other tree that is not this one. It exists — the resolver builds it and
keeps it — but it answers *in what order to load* and *is there a cycle*, and by the time the
program runs it has done its work. It could not structure the index anyway: it is a DAG rather
than a tree, so a module imported by two has two parents and one environ.

### Why not one flat environ

The first draft of this design had every module binding into the environ every chain ends at,
with the prefix keeping names apart. The argument against giving a module an environ of its own
was that a deferred scope is an index counted in the environ that created it, so `main` calling
`x.area` would find `main`'s scope `0`.

That is true only while resolution walks the caller's chain. Once a name says which module it
belongs to, the call looks for the body **where it found the name**, and an index only has to be
unique inside its own module — which it already is. Nothing has to be packed into the value, so
the width of a tape never enters into it.

---

## What a module offers

Everything it binds with `ident` at the top of its file, and nothing else. A `defer` needs no
special case: its value is its index, as a tape, so it is already what an `ident` binds.

An import is not passed on. If `main` uses `a` and `a` uses `b`, `main` writes its own line for
`b`. The alias belongs to the file that declared it, so a name used in a file has its origin
declared in that file — which is the same reason the alias is mandatory.

---

## Why an import is not an `ident`

`ident m = use a/b/c;` was proposed and refused, and the record matters because it is **more
coherent with the philosophy of the language** than what was chosen: Aurora is only
expressions, `ident` is *the* way to bind a name, and `ident area = defer { ... };` is the
precedent.

It loses to the premise. **A name bound by `ident` holds a tape**, and a module has no bytes and
no width. Worse, the form promises what it cannot give: bound that way, `m` should print, pass
to a `defer` and sit in a shape field, and each of those becomes a surprise rather than an
ordinary error. The language already separates the two — `ident x = 10;` binds, `shape Point
{ x, y };` declares, and `Point` is a name you use and cannot print. **An import is the sibling
of `shape`, not of `ident`.**

There is a variant that saves the premise: `m` holding an index into a table of modules, the way
a `defer` holds one. The storage half works. The access half is new machinery — `m.add` would
read a value and *then* look for a symbol inside it — and it costs an opcode, moves resolution
from compile time to run time, makes the dot mean two mechanisms, and would make a module a
first-class value while a scope still is not.

And the move to run time is not a fee somebody could squeeze out later. The moment `m` is a
value, choosing one is grammatical — bind two modules, pick between them with an `if`, call a
member of the result — and which `add` that is depends on something only the running program
knows. Allowing it makes every access dynamic; forbidding it leaves a value you cannot pass,
return or choose, which is the surprise being a value was supposed to remove.

---

## What three other languages taught

**Clojure** keeps namespaces and files independent: `ns` creates a namespace, `require` loads,
`:as` is the local alias, and a var is global — `m/add` resolves at read time to `#'math/add`.
Resolution early, value in a global space. That is the half Aurora takes.

**Python** makes the module a runtime object: `import a.b.c as x` executes the file and binds an
object whose `__dict__` is the namespace, and `x.add` is an attribute lookup while running. That
is the half Aurora takes for the environ, and refuses for the lookup.

**Solidity** answers the question the other two do not: what is left when the target is the EVM.
Its compiler does not read a disk — a virtual file system and an import callback, with contents
handed in, which is how Remix compiles in a browser, and the same reason reading is a port here.
Its identity is the source unit name rather than the path. And nothing resolves a name at run
time: free functions and internal library functions are inlined into the calling contract, which
is our single stream under another name, while an external library is a separate contract reached
by `DELEGATECALL` with the address stitched in at deploy — which is where a real linker exists,
if Aurora ever wants one.

The parallel worth keeping: `DELEGATECALL` runs another contract's code in the caller's storage,
`CALL` runs it in its own. Aurora's lookup is both, in that order.

**So modules cost nothing on chain.** The prefix is written while parsing and the environ index
belongs to the evaluator, so `builder/evm` goes on receiving one flat program with unique names
— only more of them, which brings the `PUSH1` ceiling the roadmap already names closer without
being a new one.

---

## Where the pieces live

`resolver` and `loader` are phases, at the root with the others. They take values, answer with
values and touch nothing: reading a file arrives as a port, and so do parsing and emitting. That
is what lets the same resolver serve a command line reading a disk and a page in a browser
holding its files in memory — see
[contributing/architecture.md](contributing/architecture.md), where the pipeline stopped being a
straight line.

`wire/module` holds what crosses between them: a module's name, which is its path from the
source root, and the tree.
