# Language statements

## Variable statment

```aurora
variable x = 1;
```

All variable type given for value then inference type

### Declaring a variable with type equals a pointer

```aurora
variable p = pointer x;
```


## Function statment

```aurora
variable fn = function () { ... }
```

### Declaring with params

```aurora
variable = fn function (x: number, y: pointer char) { ... }
```

When you use params in your function you must to put type in params statements
