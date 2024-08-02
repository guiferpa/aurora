# Keywords

### {} - Block of instructions
> It works like a scope, everything declared into doesn't work with father scope just works in children scope
```
{
  -- It's a father scope
  ref a = 1;

  {
    -- It works
    ref a = a;
    a;
    -- 1

    ref b = a;
  }

  -- It doesn't work
  a + b;
  -- Throw panic error or compiler crash
}
``` 

### -- - It's a keyword for comment
```
-- This project is about get user id
```

### `if` - It's a conditional keyword
> This keyword needs to be passed for the only a boolean expression
```
if a equals b {
  -- ...
}
```

### `ref` - It's like a variable but immutable then it's like a ref for some value saved in memory
> It's just a way to save some value in memory like a return from some I/O processing
```
ref user_id = get_user_id();
```
> How declare block of code / function
```
ref get_user_id = () {}
```

### `desc` - It's a good way to document your code
> You can doc any definition like refs
```
ref get_user_id
desc "This ref saving user id returned from another server" = () {
  -- ...
}

-- ... or

ref user_id
desc "It's just one user id reference" = 1;
```

# Built-in functions
### `nth` - Function to get value from List data structure
```
ref list = [1, 2, 3, 4];

nth(list, 0);
-- 1
```

### `map` - Function to change values from given a item from List data structure
```
ref list = [1, 2, 3, 4];

func mult(a, b)
desc "Multiply two numbers" {
  return a * b;
}

map(list, mult);
-- [1, 4, 9, 16]
```

### `filter` - Function to filter items from a List data structure given a condition
```
ref list = [1, 2, 3, 4];

ref is_even? (a)
desc "Return true if paramter is even" = () {
  return a mod 2 equals 0;
}

filter(list, is_even?);
-- [2, 4]
```

