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
| Print | **PRINT** | `print` |
| Echo | **ECHO** | `echo` |
| Assert | **ASSERT** | `assert` |
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
| Colon | **COLON** | `:` |
| Semicolon | **SEMICOLON** | `;` |
| Comment | **COMMENT** | `#-` |

## Naming

### `feed`, formerly `arguments`

The builtin that reads the nth value applied to a scope is **`feed`**. It used to be `arguments`.

The rename follows the design of `defer` (see [defer_and_scope_callable_philosofy.md](defer_and_scope_callable_philosofy.md)): Aurora has no functions, so it has no signature, no arity and no parameters. Execution is *the application of a vector of values to a scope*. "Argument" is a word that only means something against "parameter" — it dragged in the very vocabulary the design refuses, and it was the only word in the language to borrow from outside. Everything else — `tape`, `reel`, `pull`, `push`, `head`, `tail` — speaks of tape and scope.

`feed` says what happens: a scope is fed values, and `feed(n)` reads the nth one. It is also four characters instead of nine, in a construct that repeats several times per line.

The plural form was misleading on its own: `arguments(0)` reads *one* value, not a collection.

**Breaking change:** `arguments(n)` no longer parses as a builtin; `arguments` is now an ordinary identifier, which will not resolve.

## Terminals

| Name | Reference | Representation |
|---|---|---|
| Logical | **_log** | `true \| false` |
| Character | **_char** | *One tape (`tape_size` bytes, 8 by default)* |
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
_expr -> _print | _echo | _assert
       | _block | _if | _branch | _defer | _ident
       | _pull | _push | _head | _tail
       | _boole
```

### Boolean expression
```
_boole -> _rele OR _boole
        | _rele AND _boole
        | _rele
```

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

Left-associative, unlike the levels around it: `a - b - c` is `(a - b) - c`.

### Multiplicative expression
```
_multe -> _expoe MULT _multe
        | _expoe DIV _multe
        | _expoe
```

### Exponential expression
```
_expoe -> _unae EXPO _expoe
        | _unae
```

### Unary expression
```
_unae -> SUB _prie
       | _prie
```

### Primary expression
```
_prie -> _feed
       | O_PAREN _expr C_PAREN
       | _tape
       | _num | _reel | TRUE | FALSE
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
_print  -> PRINT _expr
_echo   -> ECHO _expr
_assert -> ASSERT O_PAREN _expr COMMA _expr C_PAREN
```

`assert` is only accepted in files named `*.test.ar`.
