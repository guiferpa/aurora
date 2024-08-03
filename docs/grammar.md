# Grammar

## Demand list

- [x] Add block of statements parametrized in grammar `() { ... }` it's like annonymous functions
- [ ] Doesn't forget to define built-in functions to works with `hashmap`, `list` and `string` data structures

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
| Open bracket | **O_BRK** | `{` |
| Close bracket | **O_BRK** | `}` |
| Comma | **COMMA** | `,` |
| If | **IF** | `if` |
| Colon | **COLON** | `:` |
| Semicolon | **SEMICOLON** | `;` |
| Hashmap | **HMAP** | `hashmap` |

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
      | _log
      | _id
```

### Additive expression
```
_adde -> _prie SUM _adde
      | _prie SUB _adde
      | _prie
        
```

### Boolean expression
```
_boole -> _prie EQUALS _prie
        | _prie DIFFERENT _prie
        | _prie BIGGER _prie
        | _prie SMALLER _prie
        | _log
```

### Expression
```
_expr -> _adde
      | _boole
```

### Parameters
```
_params -> _id COMMA _param
       | _id
```

### Block of statement
```
_bst -> O_BRK _stmts C_BRK
```

#### Examples
`{}`, `{ ... }`

### Block of statement parametrized
```
_bstp -> O_PAREN _params C_PAREN _bst
```

#### Examples
`() {}`, `(a) {}`, `(a, b, c) {}`


### Identification
```
_ident -> IDENT _id ASSIGN _expr
       | IDENT _id ASSIGN _bstp (It works like a function)
```

#### Examples
`ident a = 1 + 1`, `ident a = () {}`


### Hashmap item
```
_hmapi -> _id COLON _expr
       | _id COLON _bstp
```

#### Examples
`name: () {}`, `name: 10 + 90`

### Hashmap items
```
_hmapis -> _hmapi COMMA _hmapis
      -> _hmapi COMMA
```
#### Examples
`name: 20,`, `name: 10 + 90, second_name: () {},`


### Declare hashmap
```
_hmap -> HMAP _id O_BRK _hmapis C_BRK
```

#### Examples
`{ name: 20, }`, `{ name: 10 + 90, second_name: () {}, }`

### Condition
```
_if -> IF _boole _bst
```

#### Examples
`if a equals b {}`

### Statement
```
_stmt -> _expr
      | _ident
```

### Statements
```
_stmts -> _stmt SEMICOLON _stmts
       | _stmt SEMICOLON
```

### Module
```
_module -> _stmts
```
