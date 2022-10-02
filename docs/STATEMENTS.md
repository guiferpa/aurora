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

### Declaring a string variable

```aurora
variable str = "Hello world";
```

This type is `pointer char`

### Declaring a variable with type equals a pointer from pointer, it's used to array data struct for example

```aurora
variable arr = ["First string", "Second string"];
```

This type is `pointer pointer char`

## Function statment

```aurora
variable fn = function () { ... }
```

### Declaring with params

```aurora
variable = fn function (x: number, y: pointer char) { ... }
```

When you use params in your function you must to put type in params statements
