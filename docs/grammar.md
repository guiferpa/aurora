# Grammar

Aurora is **expression-only**: the grammar has no separate statement category. Top-level forms and block bodies are sequences of expressions (separated by `;`). `if`/`else` and blocks are expressions that produce a value. An expression with no value to give produces a tape of zeros, which is what `false` and `0` are.

## Tokens

Every token the lexer produces. Keywords are only recognised when the word starts with a lowercase letter.

| Name | Reference | Token |
|---|---|---|
| Identification | **IDENT** | `ident` |
| If | **IF** | `if` |
| Else | **ELSE** | `else` |
| Branch | **BRANCH** | `branch` |
| Defer | **DEFER** | `defer` |
| Feed | **FEED** | `feed` |
| Print bytes | **PRINTB** | `printb` |
| Print characters | **PRINTC** | `printc` |
| Print decimal | **PRINTD** | `printd` |
| Assert | **ASSERT** | `assert` |
| Struct | **STRUCT** | `struct` |
| As | **AS** | `as` |
| True | **TRUE** | `true` |
| False | **FALSE** | `false` |
| Pull | **PULL** | `pull` |
| Push | **PUSH** | `push` |
| Head | **HEAD** | `head` |
| Tail | **TAIL** | `tail` |
| Equals | **EQUALS** | `equals` |
| Different | **DIFFERENT** | `different` |
| Bigger than | **BIGGER** | `bigger` |
| Smaller than | **SMALLER** | `smaller` |
| And | **AND** | `and` |
| Or | **OR** | `or` |
| Assignment | **ASSIGN** | `=` |
| Sum | **SUM** | `+` |
| Subtract | **SUB** | `-` |
| Multiply | **MULT** | `*` |
| Divide | **DIV** | `/` |
| Exponentiation | **EXPO** | `^` |
| Open parentheses | **O_PAREN** | `(` |
| Close parentheses | **C_PAREN** | `)` |
| Open curly bracket | **O_CUR_BRK** | `{` |
| Close curly bracket | **C_CUR_BRK** | `}` |
| Open bracket | **O_BRK** | `[` |
| Close bracket | **C_BRK** | `]` |
| Comma | **COMMA** | `,` |
| Dot | **DOT** | `.` |
| Colon | **COLON** | `:` |
| Semicolon | **SEMICOLON** | `;` |
| Comment | **COMMENT** | `#-` |

## Naming

### `feed`, formerly `arguments`

The builtin that reads the nth value applied to a scope is **`feed`**. It used to be `arguments`.

The rename follows the design of `defer` (see [defer_and_scope_callable_philosofy.md](defer_and_scope_callable_philosofy.md)): Aurora has no functions, so it has no signature, no arity and no parameters. Execution is *the application of a vector of values to a scope*. "Argument" is a word that only means something against "parameter" — it dragged in the very vocabulary the design refuses, and it was the only word in the language to borrow from outside. Everything else — `tape`, `pull`, `push`, `head`, `tail` — speaks of tape and scope.

`feed` says what happens: a scope is fed values, and `feed(n)` reads the nth one. It is also four characters instead of nine, in a construct that repeats several times per line.

The plural form was misleading on its own: `arguments(0)` reads *one* value, not a collection.

**Breaking change:** `arguments(n)` no longer parses as a builtin; `arguments` is now an ordinary identifier, which will not resolve.

### `printb` / `printc` / `printd`, formerly `print` and `echo`

There used to be two: `print` wrote the bytes of a value and `echo` wrote it as text. Neither
name said which reading it was — `print` is what every language calls writing something,
whatever the form, and `echo` says nothing at all about characters. Someone reading a program
had to know that one of them meant bytes.

The three that replaced them carry the reading in the suffix, and the third one had nowhere to
go under the old pair: reading a tape as a decimal number was possible only by counting the
bytes by hand.

`echo` also read one byte per byte, and kept it only when it fell in printable ASCII, so
every character above 127 was silently dropped — `printc "café"` used to write `caf`.
`printc` read the number a whole tape held after that, which made `printc 514` write `Ȃ`;
since text became a tape of bytes it reads the bytes as UTF-8 instead, so `printc "café"`
is right either way and a character above ASCII is written as text rather than as its
code point.

**Breaking change:** `print` and `echo` no longer parse; both are ordinary identifiers now,
which will not resolve. `print x` becomes `printb x`, `echo x` becomes `printc x`.

## Terminals

| Name | Reference | Representation |
|---|---|---|
| Logical | **_log** | `true \| false` |
| Text | **_text** | `"…"` — *its bytes, in one tape of `tape_size`* |
| Integer | **_int** | `[0-9]+` |
| Identifier | **_id** | `[a-zA-Z_?!]` |

## Non terminals

Written to match the recursive-descent parser in `parser/parser.go`. Precedence runs from
the loosest form at the top to the tightest at the bottom.

### Module
```
_module -> (_expr SEMICOLON)*
```

Every expression ends in `;`, at the top level and inside a block alike.

### Expression
```
_expr -> _print | _assert
       | _block | _if | _branch | _defer | _ident
       | _pull | _push | _head | _tail
       | _boole
```

### Boolean expression
```
_boole -> _ande (OR _ande)*
_ande  -> _rele (AND _rele)*
```

`and` binds tighter than `or`, and both are left-associative: `a and b or c` is
`(a and b) or c`. Until v0.3.2-alpha the two shared one level and recursed to the right,
which read it as `a and (b or c)` and answered false for `false and true or true`.

Neither short-circuits: both operands are evaluated before the operation runs.

### Relational expression
```
_rele -> _adde EQUALS _rele
       | _adde DIFFERENT _rele
       | _adde BIGGER _rele
       | _adde SMALLER _rele
       | _adde
```

### Additive expression
```
_adde -> _multe ((SUM | SUB) _multe)*
```

Left-associative: `a - b - c` is `(a - b) - c`.

### Multiplicative expression
```
_multe -> _expoe ((MULT | DIV) _expoe)*
```

Left-associative as well: `a / b / c` is `(a / b) / c`, and `a / b * c` is `(a / b) * c`. It
recursed to the right until v0.3.1-alpha, which grouped the other way and answered 10 for
`20 / 5 / 2`. Multiplication on its own is associative, so only a chain mixing the two tells
the groupings apart.

### Exponential expression
```
_expoe -> _unae EXPO _expoe
        | _unae
```

Right-associative, which is the convention for exponentiation: `2 ^ 3 ^ 2` is `2 ^ (3 ^ 2)`,
512 rather than 64.

### Struct

```
_struct -> STRUCT _id O_CUR_BRK _id (COMMA _id)* C_CUR_BRK
_build  -> _id O_CUR_BRK _expr (COMMA _expr)* C_CUR_BRK
_field  -> _prie DOT _id
_shape  -> _prie AS _id
```

A struct names the tapes of a run. `Point{10, 20}` is two tapes laid end to end, with
nothing in it saying a struct built it: no header, no length and no tag. A field is exactly
one tape wide, so the field at index *i* sits at `i × tape_size`.

`struct` and `as` are **directives**: they exist for the compiler to turn a name into an
index, to report a mistake where it was written, and to tell the language server what is
there. Nothing about them reaches the IR or the binary — the flow is static and the fields
are positional, so an index is all that is needed. `Point{97, 98}` and `"ab"` are the same
value, and compare equal.

`as` names the shape where the compiler cannot see it, which is above all when a value
crosses into a scope: `feed` hands over bytes and nothing else. It is a claim rather than a
check — there is nothing in a run of bytes to check against.

```
struct Point { x, y };

ident p = Point{10, 20};
printd p.x;                  # 10
printd p;                    # 10 20

ident area = defer {
  ident q = feed(0) as Point;
  q.x * q.y;
};
printd area(Point{10, 20});  # 200
```

Braces build the value, as in Go, which is also what tells a construction apart from
applying values to a scope — `Point(1, 2)` and `greet(1, 2)` are the same shape, `Point{1, 2}`
is not. It is a construction only when the name was declared, since `if flag { … }` also puts
a brace after a name.

A struct name is not a value: writing it on its own is an error, because there is nothing to
load under it.

Because the directive exists to catch mistakes, these are compile errors: a field the struct
does not have, a value whose shape nothing declared, a construction that miscounts the
and a struct name used as a value. Reading a field past the end of a value is not one — it gives the neutral value, the
way `head` saturates and `feed` wraps.

Reading a field binds tighter than any operator, so `p.x * p.y` multiplies two fields.

### Unary expression
```
_unae -> SUB _prie
       | _prie
```

A tape is unsigned, so `-x` is the value taken away from zero, wrapping at the tape width:
`-5` is the same tape as `0 - 5`, which is 251 with one-byte tapes and 2^64 - 5 with the
default eight. The compiler emits exactly that subtraction.

`-` is not a sign, and the language has none: a byte runs from 0 to 255, no bit is set aside
to mark a value negative, and no operation produces a negative value. `-5 bigger 5` is
therefore true. See [No signs, and no negative numbers](language-design.md#no-signs-and-no-negative-numbers).

Until v0.3.1-alpha the operator was parsed and then dropped on the way to the IR, so `-5`
was 5 and `10 + -5` was 15.

### Primary expression
```
_prie -> _feed
       | O_PAREN _expr C_PAREN
       | _tape
       | _num | _text | TRUE | FALSE
       | _block
       | _ident
       | _callee
       | _id
```

### Identification
```
_ident -> IDENT _id ASSIGN _expr
```

An `ident` is immutable and cannot be redeclared in the same scope.

#### Examples
`ident a = 1 + 1`, `ident r = defer { 1; }`, `ident t = [1, 2, 3]`

### Block expression
```
_block -> O_CUR_BRK (_expr SEMICOLON)* C_CUR_BRK
```

A block is an expression: it runs and produces the value of its last expression. An empty
block produces a tape of zeros.

#### Examples
`{ }`, `{ 1; 2; }`

### Defer expression
```
_defer -> DEFER _block
```

Delays a scope instead of running it. The value is a reference to the scope, which is
invoked like `r(1, 2)`. No signature, no arity.

### Callee
```
_callee -> _id O_PAREN (_expr (COMMA _expr)*)? C_PAREN
```

Applies a vector of values to a deferred scope.

### Feed
```
_feed -> FEED O_PAREN _num C_PAREN
       | FEED _num
```

Reads the nth value applied to the running scope. The index is normalized modulo the
length of the vector, so it never fails.

#### Examples
`feed(0)`, `feed(1)`

### If expression
```
_if -> IF _boole O_CUR_BRK (_expr SEMICOLON)* C_CUR_BRK (_else)?

_else -> ELSE O_CUR_BRK (_expr SEMICOLON)* C_CUR_BRK
```

An expression: it produces the value of the branch taken. Without an `else`, a failing
test produces a tape of zeros.

#### Examples
`if a equals b { 1; }`, `if a bigger b { 1; } else { 2; }`

### Branch expression
```
_branch -> BRANCH O_CUR_BRK _branchitems C_CUR_BRK

_branchitems -> _boole COLON _expr COMMA _branchitems
              | _expr SEMICOLON
```

Sugar for nested `if`/`else`. The last item is the fallback and closes with `;`.

#### Examples
```
branch {
  n equals 1: 10,
  n equals 2: 20,
  0;
}
```

### Tape expression
```
_tape -> O_BRK (_expr (COMMA _expr)*)? C_BRK
```

A tape literal holds one byte per value and cannot carry more values than the tape is
wide. Values must fit in a byte.

#### Examples
`[]`, `[1, 2, 3]`

### Tape operations
```
_pull -> PULL _expr _expr
_push -> PUSH _expr _expr
_head -> HEAD _expr _num
_tail -> TAIL _expr _num
```

A tape is a shift register: `pull` shifts left with the value entering at the right,
`push` shifts right with the value entering at the left. `head` and `tail` slice the
significant bytes, with the index taken modulo the tape width.

#### Examples
`pull t 4`, `push t 5`, `head t 2`, `tail t 2`

### Builtins
```
_print  -> (PRINTB | PRINTC | PRINTD) _expr
_assert -> ASSERT O_PAREN _expr COMMA _text C_PAREN
```

The three print builtins are three readings of the same tape, and the suffix names the
reading: `printb` the bytes, `printd` the number they spell, `printc` those bytes as UTF-8
text.

```
printb 44;      [0 0 0 0 0 0 0 44]
printd 44;      44
printc 44;      ,      the byte 44 is a comma
printc 26729;   hi     the bytes 104 and 105
```

`assert` is only accepted in files named `*.test.ar`.

Its message is a **literal**, not an expression: it is written for whoever reads the result
of a test, the same way a struct's field names are written for whoever reads the source. It
rides in the instruction as its bytes rather than as a value, which is also what lets it be
longer than a tape — a message usually is.
