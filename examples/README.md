# Examples

One file per feature. Each one opens with a comment explaining what it shows, how to run it and the output it produces — output that was pasted from an actual run, not written by hand.

## Running them

Run from **this directory**, so the CLI finds the [`aurora.toml`](aurora.toml) next to the examples:

```sh
cd examples
aurora run -s numbers.ar        # any file
aurora run                      # the "main" profile
aurora run -p fibonacci         # another profile
aurora run -p tiny              # a profile that pins one-byte tapes
```

The manifest is discovered by walking up from the current directory, so it has to be at or above where you run the command — not next to the file you pass with `-s`.

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
| [comments.ar](comments.ar) | `#-` |
| [assertions.test.ar](assertions.test.ar) | `assert`, which only works in `*.test.ar` |

## Reading the output

`print` writes the raw bytes of a value, which is why a result looks like `[0 0 0 0 0 0 0 42]` — every value is a tape, 8 bytes wide by default. `echo` reads those same bytes back as text.

One file is one program: the compiler's unit is the file, so these examples can share a directory without their names colliding.

Background: [language design](../docs/language-design.md), [grammar](../docs/grammar.md), [manifest reference](../docs/manifest.md).
