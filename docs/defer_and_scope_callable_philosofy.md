# Design: Scopes, `defer`, and Execution

## Problem

The language is built around a single fundamental concept: **scope**.

* Every `{}` block is a scope
* Every scope is **auto-executable**
* Every scope **must return a value** (the last expression)
* `ident` is the only binding mechanism
* Functions do not exist as a native language concept

Given this model, the following need arises:

> How can a scope be defined without executing it immediately, allowing it to be executed later and receive data at execution time, **without introducing functions, signatures, or arity**?

The challenge is to delay execution while preserving the principle that *a scope is just a scope*.

---

## Solution: `defer`

The **`defer`** keyword solves this problem by transforming a scope into a **deferred executable value**.

```lang
ident r = defer {
  arguments(0) + arguments(1);
};

r(1, 2); // 3
```

### Semantics of `defer`

* `defer` **does not create a function**
* `defer` **does not define a signature**
* `defer` **does not define arity**
* `defer` **does not define a contract**
* `defer` only **delays the execution of a scope**

The result of `defer { ... }` is a value that contains:

> **a pointer to a scope**

No metadata about parameters or argument count is stored.

---

## Executing a deferred scope

When a scope created with `defer` is invoked:

```lang
r(1, 2);
```

The runtime:

1. Evaluates the deferred value (`r`)
2. Pushes the provided values onto the stack as a vector
3. Creates a new execution frame
4. Executes the scope
5. Returns the value of the last expression

The scope itself remains unaware of how many arguments were passed.

---

## `arguments` as a builtin function

`arguments` is a **builtin function provided by the runtime**, not a language-level function and not a user-definable construct.

```lang
arguments(index)
```

### Properties of `arguments`

* `arguments` is globally available inside any executing scope
* It is **not declared** using `ident`
* It cannot be shadowed or overridden
* It does not expose arity or argument count
* It does not perform validation

Internally, `arguments` accesses the **argument vector of the current execution frame**.

---

## Argument access semantics

* Arguments are stored as a sequential structure
* Access is **always normalized using modulo with the argument vector length**
* Negative indices are normalized by absolute value before modulo
* Access never fails and never throws

Example:

```lang
arguments(0)
arguments(10)
arguments(-1)
```

All expressions return a valid value according to the normalization rules.

---

## Philosophy: scope is not a function

In this language, **functions are not a concept**.

Traditional functions imply:

* Signatures
* Arity
* Contracts
* Usage validation

Scopes, by definition, are:

* Blind
* Neutral
* Auto-executable
* Independent of declarative intent

Execution is defined as:

> **the application of a vector of values to a scope**

—not as a function call.

The presence of the `arguments` builtin does not introduce function semantics; it only provides a runtime access mechanism to execution data.

---

## Pros of this model

### Conceptual simplicity

* A single core concept: scope
* A single execution modifier: `defer`
* Builtins provide runtime data access without contracts

### Simple and predictable runtime

* No arity checks
* No parameter validation
* Uniform execution frames
* Minimal bytecode and metadata

### High composability

* Scopes are values
* Execution is data application
* Builtins are pure runtime primitives

---

## Cons (assumed by design)

### Silent bugs

* Incorrect argument usage does not fail immediately
* Logical errors are not enforced by the language
* Responsibility lies with the programmer

### Learning curve

* Requires abandoning traditional function-based thinking

### Reduced static tooling

* Lack of signatures limits static analysis and inference

---

## Core principle

> **`defer` does not introduce functions into the language.
> It turns scopes into executable values, while `arguments` provides runtime access to execution data without creating contracts or arity.**

This design prioritizes:

* conceptual coherence
* implementation simplicity
* runtime predictability
