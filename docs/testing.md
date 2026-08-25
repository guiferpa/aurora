# Testing

Aurora has one builtin for checking things, `assert`, and one command that runs them, `aurora test`.

```sh
aurora test                        # the "main" profile
aurora test dev                    # another profile
aurora test src/greeting.test.ar   # one file
```

```
src/greeting.test.ar
  ok    hello answers 1
  ok    twice doubles
  FAIL  assertion failed: bigger_of takes the larger one

2 passed, 1 failed in 1 file
```

The command exits non-zero when an assertion fails or a file could not run, so a CI job can rely on it.

---

## A test names what it checks

A test file is a file like any other: it reaches other code the way every file does, with
`use`. The `.test.ar` in its name does two things and no more — it tells `aurora test` which
files to run, and it tells the compiler which files may hold an assertion.

```aurora
#- src/greeting.ar
ident hello = defer { 1; };
ident twice = defer { feed(0) * 2; };
```

```aurora
#- src/greeting.test.ar
use greeting as g;

assert(g.hello() equals 1, "hello answers 1");
assert(g.twice(21) equals 42, "twice doubles");
```

This used to be a pairing: `greeting.test.ar` tested `greeting.ar` because it was named after
it, the two ran as one scope, and neither imported anything. That was how a test reached code
before modules existed. It is gone, and what replaces it is not a rule about tests at all —
it is the module system, doing here what it does everywhere else.

Three things follow, and each of them is what a module already meant:

- **A test runs on its own.** It loads what it names and nothing else. A file that names
  nothing is a program of one file, and a perfectly good test of what it declares itself.
- **A module runs when it is loaded.** If the module a test names prints at its top level, you
  see that output during the test. That is the module running, not the test.
- **Two files are two scopes.** A test may bind a name the module it checks also binds; they
  are different names, and the module's is reached through the alias.

A shape crosses the way it crosses anywhere: named through the alias, or carried by what a scope returns.

```aurora
#- src/point.ar
shape Point { x, y };

ident origin = Point{0, 0};
```

```aurora
#- src/point.test.ar
use point as p;

ident here = p.Point{10, 20};

assert(here.y equals 20, "a shape is named through the alias");
assert((p.origin as p.Point).x equals 0, "and a value the module built is claimed with as");
```

The other way round does not hold: a module is compiled without its test, so it cannot lean on
anything the test declares — under `aurora run` there is no test at all.

---

## Where the command looks

With a profile — named or the default `main` — the search starts at the **directory of the profile's `source`** and goes down to the leaves. Nothing above it is considered:

```
my-project/
  aurora.toml            [profiles.main] source = "src/main.ar"
  src/
    main.ar
    main.test.ar         ← runs
    utils/
      text.ar
      text.test.ar       ← runs
  tests/
    legacy.test.ar       ← not found: above the starting point
```

With a path ending in `.test.ar`, only that file runs, and no manifest is needed.

---

## `assert`

```aurora
assert(<condition>, "<message>");
```

The condition is any expression; a tape of zeros is false and anything else is true.

The message names the check, and it is what the report prints — for a passing assertion as well as a failing one. It is written as text, not computed: it is meant for whoever reads the result, the same way a shape's field names are meant for whoever reads the source. That is also why it is not limited to the width of a tape, which a message would usually exceed.

A file is only allowed to hold assertions if it is named `*.test.ar`. In any other file it is a compile error:

```
assert can only be used in .test.ar files
```

**A failing assertion does not stop the file.** Every assertion runs, and the report shows all of them.

---

## `assert` under `aurora run`

Assertions belong to `aurora test`. Running a test file with `aurora run` warns about each one and carries on without checking it:

```
$ aurora run src/greeting.test.ar
src/greeting.test.ar:1:1: warning: assert only runs under 'aurora test'; ignored here
src/greeting.test.ar:2:1: warning: assert only runs under 'aurora test'; ignored here
```

The warning carries the position, so an editor can jump to it. A program that happens to hold an assertion that would fail is not affected: `run` exits zero.

`aurora run` on a test file runs it like any other file: what it names is loaded, and the assertions are skipped with a warning. Checking them is a job for `aurora test`.

---

## What is not here yet

- **No grouping.** One file is one test; there is no `case` or `describe`. The message of each assertion is the name of the check.
- **No fixtures, no setup or teardown, no mocks.** A test is an ordinary Aurora file that happens to hold assertions.

A test that checks another file is worked out in [examples/project](../examples/project) — `src/geometry.ar` and `src/geometry.test.ar` next to it. One that checks what it declares itself, and runs with no project around it, is [examples/assertions.test.ar](../examples/assertions.test.ar). `aurora init` writes a small one of the first kind into every new project.
