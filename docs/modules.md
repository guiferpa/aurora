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
#- src/main.test.ar
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
- a `struct`'s **name** stays in the file that declared it, and is written elsewhere as the
  module's: `g.Square` builds one, names one with `as`, and promises one with `returns`;
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

### A shape crosses with the promise that names it

A scope that says what it answers with hands the shape over, so whoever imports it reads the
fields without naming anything:

```
#- src/os.ar
struct Env { found, value };

ident lookup = defer {
  Env{1, 42};
} returns Env;
```

```
#- src/main.ar
use os as o;

ident r = o.lookup("HOME");
printd r.found;
printd r.value;
```

Nothing in `main.ar` mentions `Env`, and nothing has to. What crossed is the promise — the
struct and the fields it is made of — which is what turns `found` into the first tape of the
run.

A scope that promised nothing hands over a run of tapes and no shape, exactly as it did
before: the file reading it has to name one.

### And a shape can be named

A struct's name belongs to the file that declared it, so elsewhere it is written as the
module's:

```
use geometry as g;

ident s = g.Square{4, 5};          #- built
ident t = feed(0) as g.Square;     #- claimed
ident make = defer { g.Square{1, 2}; } returns g.Square;
```

A `Square` declared here and a `Square` declared there are **two shapes**, never one: what the
compiler knows them by carries the module, and nobody can type that. And a struct of another
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

A test and the file it tests are **one module written in two files**. What the source declared
is in scope for the test — its bindings, its structs, and the modules it brought in:

```
#- src/main.test.ar
assert(g.area(3, 4) equals 12, "the alias the source declared is the alias here");
```

A test also names modules on its own, with a `use` at its own top, including one its source
never mentioned.

---

## What is not there yet

- A `struct` cannot be exported.
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
