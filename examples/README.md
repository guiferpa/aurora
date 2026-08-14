# Examples

One file per feature. Each one opens with a comment explaining what it shows, how to run it and the output it produces — output that was pasted from an actual run, not written by hand.

## Running them

Give the path, from anywhere. A file is all the CLI needs — no project, no manifest:

```sh
aurora run examples/numbers.ar
aurora run examples/tape_size.ar --tape-size 1
```

For the other way of working — a manifest with named profiles — see [project/](project/).

## The files

| File | What it shows |
|---|---|
| [identifiers.ar](identifiers.ar) | `ident`, immutability, what a name may contain |
| [numbers.ar](numbers.ar) | decimal, `_` separators, hexadecimal |
| [arithmetic.ar](arithmetic.ar) | `+ - * / ^`, precedence, parentheses, wrapping |
| [comparison.ar](comparison.ar) | `equals`, `different`, `bigger`, `smaller` |
| [logic.ar](logic.ar) | `and`, `or`, truthiness, booleans as tapes |
| [conditionals.ar](conditionals.ar) | `if`/`else` as an expression |
| [branch.ar](branch.ar) | `branch` with several tests and a fallback |
| [scopes.ar](scopes.ar) | `{ }` blocks, nesting, what a scope sees |
| [callables.ar](callables.ar) | `defer` and `feed`: applying values to a scope |
| [recursion.ar](recursion.ar) | factorial and fibonacci |
| [tapes.ar](tapes.ar) | tape literals, `pull`, `push`, `head`, `tail` |
| [reels.ar](reels.ar) | strings as runs of tapes, `echo` |
| [tape_size.ar](tape_size.ar) | the same file compiled at different tape widths |
| [defer_capacity.ar](defer_capacity.ar) | how many scopes a tape can name, and the warning when they do not fit |
| [comments.ar](comments.ar) | `#-` |
| [assertions.test.ar](assertions.test.ar) | `assert`, which only works in `*.test.ar` |
| [project/](project/) | a manifest with profiles, and what they change |

## Reading the output

`print` writes the raw bytes of a value, which is why a result looks like `[0 0 0 0 0 0 0 42]` — every value is a tape, 8 bytes wide by default. `echo` reads those same bytes back as text.

One file is one program: the compiler's unit is the file, so these examples share a directory without their names colliding.

Background: [language design](../docs/language-design.md), [grammar](../docs/grammar.md), [manifest reference](../docs/manifest.md).
