# State Management in Aurora

## Overview

Aurora provides state management through **stateful functions** (functions ending with `!`) that encapsulate mutable state. Each stateful function maintains its own private state that persists for the lifetime of the process, similar to RAM memory.

## Core Concepts

### Stateful Functions (`!`)

Functions ending with `!` are **state modifiers** that maintain their own private state:

```aurora
ident counter! = { state + 1; };
```

- Each `!` function has its own isolated `state`
- State is initialized with zeroed bytes (8 bytes = 0)
- State persists for the entire process lifetime
- Stateful functions are modifiers and don't return values for use in expressions

### The `state` Keyword

The `state` keyword is used to:
- Access the current state within a stateful function
- Read the state of a specific stateful function from outside

## Syntax

### Declaring a Stateful Function

```aurora
ident <name>! = { <body> };
```

The last expression in the body returns a value that updates the function's state.

### Accessing State Internally

Within a stateful function, use `state` to access the current state:

```aurora
ident increment! = { state + 1; };
```

### Accessing State Externally

To read a stateful function's state from outside, use `state :<functionName>`:

```aurora
ident current = state :increment;
```

### Calling Stateful Functions

Stateful functions are called like regular functions, but they modify state rather than return values:

```aurora
increment!();  // Modifies state :increment
```

## Rules

### 1. State Initialization

- State starts with zeroed bytes (8 bytes = 0)
- No explicit initialization is required
- First update sets the initial value

### 2. State Update

- The **last expression** in the function body returns a value that updates the state
- If the last expression doesn't return a value (e.g., `print`), state receives an empty byte array

```aurora
ident counter! = { state + 1; };  // Last expression updates state
```

### 3. Multiple Expressions

Only the last expression updates the state:

```aurora
ident complex! = {
  ident temp = state * 2;  // Intermediate expression
  temp + 10;              // Last expression updates state
};
```

### 4. Nested Blocks

Blocks without `ident` are auto-executable. The last expression of the innermost block updates the state:

```aurora
ident block! = {
  {
    state + 1;  // Last expression of inner block updates state
  };
};
```

### 5. Reading State in Regular Functions

Regular functions (without `!`) can read and return states:

```aurora
ident counter! = { state + 1; };
ident getCounter = { state :counter; };

counter!();
ident value = getCounter();  // Returns 1
```

## Examples

### Example 1: Simple Counter

```aurora
ident inc! = { state + 1; };

inc!();
inc!();
ident current = state :inc;  // current = 2
```

### Example 2: Conditional State Update

```aurora
ident conditional! = {
  if state equals 0 { 100; } else { state + 1; };
};

conditional!();  // state :conditional = 100
conditional!();  // state :conditional = 101
conditional!();  // state :conditional = 102
```

### Example 3: Multiple Independent States

```aurora
ident counter! = { state + 1; };
ident accumulator! = { state + 10; };

counter!();
counter!();
accumulator!();
accumulator!();

ident count = state :counter;      // 2
ident acc = state :accumulator;    // 20
```

### Example 4: Function Returning State

```aurora
ident inner! = { state + 5; };
ident getInner = { state :inner; };

inner!();
ident value = getInner();  // Returns 5
```

### Example 5: Complex State Logic

```aurora
ident calculate! = {
  ident base = state;
  if base bigger 10 {
    base * 2;
  } else {
    base + 5;
  };
};

calculate!();  // state :calculate = 5
calculate!();  // state :calculate = 10
calculate!();  // state :calculate = 20
```

### Example 6: Combining States

```aurora
ident x! = { state + 1; };
ident y! = { state + 10; };
ident sum = {
  state :x + state :y;  // Returns sum of both states
};

x!();
y!();
y!();
ident total = sum();  // 21 (1 + 20)
```

### Example 7: State with No Return Value

```aurora
ident printState! = {
  print state;
  // print doesn't return a value, so state receives empty byte array
};

printState!();
ident result = state :printState;  // Empty byte array
```

### Example 8: Nested Blocks

```aurora
ident nested! = {
  {
    state + 1;  // Last expression updates state
  };
};

nested!();  // state :nested = 1
```

## Important Notes

1. **Stateful functions are modifiers**: They don't return values for use in expressions. They modify state as a side effect.

2. **State isolation**: Each `!` function has its own isolated state. States don't interfere with each other.

3. **Last expression rule**: Always ensure the last expression in a stateful function returns a value if you want to update the state meaningfully.

4. **State lifetime**: State persists for the entire process lifetime, similar to RAM memory.

5. **Reading state**: Use regular functions (without `!`) to read and return state values for use in expressions.

## Best Practices

- Use descriptive names for stateful functions to indicate what state they manage
- Keep stateful functions focused on a single responsibility
- Use regular functions to read state when you need the value in expressions
- Document stateful functions to clarify what state they manage and how they update it
