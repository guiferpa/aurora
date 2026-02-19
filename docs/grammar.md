# Grammar

Aurora is **expression-only**: the grammar has no separate statement category. Top-level forms and block bodies are sequences of expressions (separated by `;`). `if`/`else` and blocks are expressions that produce a value. The keyword `nothing` denotes the universal neutral value (8 zero bytes).

## Demand list

- [x] Add block of statements parametrized in grammar `() { ... }` it's like annonymous functions
- [ ] Doesn't forget to define built-in functions to works with `list`
- [x] Construct `IF` as Expression instead of a Statement which every final line in his block must be a valid return
- [x] Construct `BLOCK` as Expression instead of a Statement which every final line in his block must be a valid return

## Tokens

| Name | Reference | Token |
|---|---|---|
| Identificator | **IDENT** | `ident` |
| Assignment | **ASSIGN** | `=` |
| Open parentheses | **O_PAREN** | `(` |
| Close parentheses | **C_PAREN** | `)` |
| Equals | **EQUALS** | `equals` |
| Different | **DIFFERENT** | `different` |
| Bigger than | **BIGGER** | `bigger` |
| Smaller than | **SMALLER** | `smaller` |
| Sum | **SUM** | `+` |
| Substract | **SUB** | `-` |
| Comment | **COMMENT** | `--` |
| Open curly bracket | **O_CUR_BRK** | `{` |
| Close curly bracket | **C_CUR_BRK** | `}` |
| Open bracket | **O_BRK** | `[` |
| Close bracket | **C_BRK** | `]` |
| Comma | **COMMA** | `,` |
| If | **IF** | `if` |
| Colon | **COLON** | `:` |
| Semicolon | **SEMICOLON** | `;` |

## Terminals

| Name | Reference | Representation |
|---|---|---|
| Logical | **_log** | `true \| false` |
| Character | **_char** | *Represented for 8 bytes encoded given UTF-32* |
| Integer | **_int** | `[0-9]+` |
| Identifier | **_id** | `[a-zA-Z_?!]` |

## Non terminals

### Primary expression
```
_prie -> O_PAREN _expr C_PAREN
      | _num
```

### Unary expression
```
_unae -> SUB _unae
       | _prie
```

### Exponential expression
```
_expoe -> _unae EXPO _expoe
        | _unae
```

### Multiplicative expression
```
_multe -> _expoe MULT _multe
        | _expoe DIV _multe
        | _expoe
```

### Additive expression
```
_adde -> _multe SUM _adde
      | _multe SUB _adde
      | _multe
        
```

### Boolean expression
```
_boole -> _adde EQUALS _boole
        | _adde DIFFERENT _boole
        | _adde BIGGER _boole
        | _adde SMALLER _boole
        | _adde
```

### If expression
```
_ife -> IF _boole O_CUR_BRK _stmts C_CUR_BRK
```

#### Examples
`if a equals b {}`

### Block expression
```
_ble -> O_CUR_BRK _stmts C_CUR_BRK
```

#### Examples
`{}`, `{ ... }`

### Block parametrized
```
_blep -> O_PAREN _params C_PAREN _bst
```

#### Examples
`() {}`, `(a) {}`, `(a, b, c) {}`

### Parameters
```
_params -> _id COMMA _param
       | _id
```

### List item
```
_lsti -> _id COLON _expr
      | _expr
```

#### Examples
`name: () {}`, `name: 10 + 90`, `() {}`, `10 + 90`

### List items
```
_lstis -> _lsti COMMA _lstis
      | _lsti COMMA
```

#### Examples
`name: 20,`, `name: 10 + 90, () {},`

### List expression
```
_lste -> O_BRK _lstis C_BRK
```

#### Examples
`[ name: 20, ]`, `[ name: 10 + 90, second_name: () {}, ]`

### Expression
```
_expr -> _boole
      | _ife
      | _ble
      | _blep
      | _lste
```

### Identification
```
_ident -> IDENT _id ASSIGN _expr
```

#### Examples
`ident a = 1 + 1`, `ident a = () {}`, `ident a = { b: 1, }`

### Statement
```
_stmt -> _expr
      | _ident
```

`ident a = 1 + 1`, `1 + 1`

### Statements
```
_stmts -> _stmt SEMICOLON _stmts
       | _stmt SEMICOLON
```

### Module
```
_module -> _stmts
```
