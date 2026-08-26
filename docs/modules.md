# Modules

A program can be more than one file. A **module is a file**, `use` brings one in under a name
of your choosing, and that name is the only way to reach what is inside it.

```aurora
#- src/geometry.ar
ident area = defer { feed(0) * feed(1); };
ident perimeter = defer { (feed(0) + feed(1)) * 2; };
```

```aurora
#- src/main.ar
use geometry as g;

ident width = 30;
ident height = 20;

printd g.area(width, height);
```

```aurora
#- src/geometry.test.ar
use geometry as g;

assert(g.area(3, 4) equals 12, "area multiplies its sides");
assert(g.perimeter(3, 4) equals 14, "perimeter walks around them");
```

Run it from the root of the project:

```sh
aurora run           # 600
aurora test          # 2 passed, 0 failed in 1 file
```

---

## The name of a module

`use geometry as g;` reads `src/geometry.ar`. The path is written from the **source root**,
without the extension, and it is the module's name everywhere: `use a/b/c as x;` is the module
`a/b/c` whoever writes the line.

| Written | Read |
|---|---|
| `use geometry as g;` | `src/geometry.ar` |
| `use a/b/c as x;` | `src/a/b/c.ar` |

There is no relative form. `./x` and `../x` are not paths, and neither is a directory: a
module is a file.

The path is one word. `a / b` is a division everywhere else in the language, and it is not a
path here either.

### Where the source root is

`src`, unless the project says otherwise:

```toml
[project]
  source_root = "lib"
```

It is **relative to the directory the command was run in**, with a manifest or without. That
is the whole rule, and the price of it is that a project with modules is run from its own
root — from anywhere else `src/` means something different, and the error says where it
looked.

It belongs to `[project]` rather than to a profile for the same reason `tape_size` does: it
decides what the source *means*. Two profiles with two roots would make one `use` line name
two different files.

---

## The alias

Every import has one, and there is no form that leaves it out. Reading `g.area(3, 4)` you know
where `area` lives without scrolling anywhere.

The alias belongs to the file that wrote it. It is not a name the module chose, and it means
nothing in any other file — two files may reach the same module under two different names, and
one file may not use the same name twice.

It is a name in the file's scope like any other, so it cannot also be a value. These are
refused while compiling — shown rather than run, because a module is read before the file that
imports it, so a block on its own would be refused for the module missing before it got this
far:

```
use geometry as x;

printd x;
```

```
x is the module geometry at line 3 and column 8: reach something inside it with x.name
```

```
use geometry as x;

ident x = 1;
```

```
x is already the alias of geometry at line 3 and column 7
```

And `use` belongs above everything else, so what a file depends on is known before anything
happens in it:

```
ident n = 1;

use geometry as g;
```

```
use belongs to the top of the file at line 3 and column 1: put it above everything else
```

---

## What a module offers

**Everything it binds with `ident` at the top of the file** — a tape or a deferred scope,
which is the same thing since a scope's value is its index. Nothing else:

- a name bound inside a block or a deferred body belongs to the scope that binds it, and is
  gone when that scope is;
- a `shape`'s **name** stays in the file that declared it, and is written elsewhere as the
  module's: `g.Square` builds one, names one with `as`, and declares one with `returns`;
- an import is not passed on. If `main` uses `a` and `a` uses `b`, then `main` does not reach
  `b` — it writes its own `use b as ...;`. Every file declares what it needs, and a name used
  in a file has its origin declared in that file.

A name that is not there is refused **while compiling**, where it was asked for, saying what
the module does have:

```
module geometry has no volume at line 3 and column 4 (it has area, perimeter)
```

That is the check the language server underlines and `aurora run` refuses on. Nothing waits
for the program to reach the line.

A module that is not there is refused the same way, naming the file it looked for — which is
also what tells you that you are standing in the wrong directory:

```aurora
#- fails: module geometry is not there
use geometry as g;

printd g.area(1, 2);
```

### A shape crosses with the scope that returns it

What a scope returns hands the shape over, so whoever imports it reads the fields without
naming anything:

```
#- src/os.ar
shape Env { found, value };

ident lookup = defer {
  Env{1, 42};
};
```

```
#- src/main.ar
use os as o;

ident r = o.lookup("HOME");
printd r.found;
printd r.value;
```

Nothing in `main.ar` mentions `Env`, and nothing has to. What crossed is what `lookup`
returns — the shape and the fields it is made of — which is what turns `found` into the first
tape of the run. Nothing in `os.ar` says it either: the compiler read what the body ends with.
Writing `returns Env` on it changes nothing about what crosses; it is a declaration `os.ar` is
held to, so a change to that body that stops returning an `Env` is refused in the file that
made the mistake.

What does not cross is a scope whose shape nothing says — one ending with arithmetic, which is
a tape and not a run. It hands over the tapes and no shape, and the file reading it has to
name one.

### One import names no file

`use storage as s;` brings in what a chain keeps between one transaction and the next. It is
written as an import and is not one — there is no `src/storage.ar`, nothing is read, and
nothing crosses — because what it is to whoever uses it is exactly an import: something
reached under an alias they chose.

```
use storage as s;

ident deposit = defer { s.set(1, s.get(1) + feed(0)); };
ident balance = defer { s.get(1); };
```

Two things are reached through the alias, and there is no third:

| written | is | answers |
|---|---|---|
| `s.set(key, value)` | keeps the value under the key | the value it wrote |
| `s.get(key)` | reads what is kept under the key | it, or the neutral tape |

A key is a tape and so is a value, and a key is anything a program works out — `s.get(k)` after
`ident k = 3 * 4;` reads what `s.set(12, ...)` wrote. A key nothing was ever written under
answers the neutral tape, the same as reading past the end of a run.

`set` answers what it wrote because everything in Aurora is an expression, so
`s.set("n", 1) + 1` is two.

`storage` is the one word a module may not be called, and importing a file of that name is
refused rather than quietly shadowed.

Off a chain it is simulated for one run of a program: what a transaction sees. That is what
`aurora run` and a test read, and what a second run starts without.

### And a shape can be named

A shape's name belongs to the file that declared it, so elsewhere it is written as the
module's:

```
use geometry as g;

ident s = g.Square{4, 5};          #- built
ident t = feed(0) as g.Square;     #- claimed
ident make = defer { g.Square{1, 2}; } returns g.Square;
```

A `Square` declared here and a `Square` declared there are **two shapes**, never one: what the
compiler knows them by carries the module, and nobody can type that. And a shape of another
module is not a value there any more than a local one is here — `printd g.Square;` is refused
the same way `printd Square;` is.

---

## When a module runs

A module is a program, and its body runs — once, before whoever needs it, however many files
import it. Two modules importing a third get the same one, loaded a single time.

That means what a module prints at its top level, it prints when it is loaded:

```aurora
#- src/loud.ar
printc "loading";
ident n = 1;
```

Aurora has nothing like `if __name__ == "__main__"`. A print is a log, and a module printing
on the way in is an honest thing for it to do.

Modules that need each other in a circle are refused, with the whole chain named:

```
modules go in a circle at line 1 and column 1: one → two → one
```

---

## Modules and tests

A test file is a module like any other, and it reaches code the way any file does — by naming
it. There is no rule tying a test to a file of the same name, and nothing is in scope that was
not imported:

```
#- src/geometry.test.ar
use geometry as g;

assert(g.area(3, 4) equals 12, "the alias this file declared is the alias here");
```

It names as many modules as it needs, each one loading once and running before it. What
`.test.ar` decides is which files `aurora test` runs and which files may hold an assertion —
see [testing.md](testing.md).

---

## What is not there yet

- There is no `private`: a module offers everything it binds at the top.
- The REPL takes `use`, reading from where it was started, and brings a module in once per
  session — a second `use` of the same one is a use of what is already there.
- The editor follows an import — it underlines a module that is not there and a name a module
  does not have, and offers what a module declared after the dot — but it does not go to a
  definition in another file, and it only notices a change in a file you have open. Editing a
  module outside the editor updates what depends on it the next time you touch that file.
- The manifest does not list dependencies of any kind. The file system is the whole story
  until third-party packages exist.

The design and why it is this shape: [module_system_design.md](module_system_design.md).
