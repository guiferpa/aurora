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

## A test belongs to the source of the same name

`greeting.test.ar` tests `greeting.ar`. That file is evaluated **first**, and the test runs in the same scope — so everything the source declared is already there, with no import:

```aurora
#- greeting.ar
ident hello = defer { 1; };
ident twice = defer { feed(0) * 2; };
```

```aurora
#- greeting.test.ar
assert(hello() equals 1, "hello answers 1");
assert(twice(21) equals 42, "twice doubles");
```

There is no module system yet, so this pairing is how a test reaches the code it checks. When there is one, the rule becomes "a test belongs to the module of the same name" — the same sentence.

Three consequences follow, and none of them is worked around:

- **The source has to exist.** A `.test.ar` with no `.ar` next to it is an error: it would not see the code it is meant to check.
- **Whatever the source does at its top level happens.** If `greeting.ar` prints, you see that print during the test.
- **The two files share one scope.** Declaring in the test a name the source already declares is a conflict, not a shadow:

  ```
  conflict between identifiers named a
  ```

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
assert(<condition>, <message>);
```

The condition is any expression; a tape of zeros is false and anything else is true. The message names the check, and it is what the report prints — for a passing assertion as well as a failing one.

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

Note that `aurora run` on a test file runs **only that file** — the source it belongs to is not evaluated, so anything it declares is missing. That is a job for `aurora test`.

---

## What is not here yet

- **No grouping.** One file is one test; there is no `case` or `describe`. The message of each assertion is the name of the check.
- **A test cannot reach another file** beyond the one it is paired with. That waits on the [module system](module_system_design.md).
- **No fixtures, no setup or teardown, no mocks.** A test is an ordinary Aurora file that happens to hold assertions.

A worked example lives in [examples/greeting.ar](../examples/greeting.ar) and [examples/greeting.test.ar](../examples/greeting.test.ar), and `aurora init` writes a smaller one into every new project.
