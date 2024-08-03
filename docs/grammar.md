# Grammar

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

### Identification
```
_ident -> IDENT _id ASSIGN _expr
```
