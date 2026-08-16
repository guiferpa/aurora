# Defer: scope visibility

This document describes how the body of a scope created with `defer` sees variables from other scopes. This is an intentional peculiarity of Aurora and is not considered a problem.

---

## Current behavior

### What the defer body sees

The defer body **sees the caller's scope** (and any outer scopes above it).

At call time, the environment chain is:

```
(defer frame, with the fed values) → caller's environment → ...
```

Because name resolution walks this chain, the defer body sees:

- the values fed by the call (`feed(0)`, `feed(1)`, etc.);
- all idents from the **caller** and from scopes above it.

**Example:** defer and call in the same scope — outer variable is visible.

```aurora
ident x = 10;
ident r = defer { printb x; };
r();   #- caller is the same scope that has x → sees x, prints 10
```

### What the defer body does not see

The defer body **does not capture** the scope **where the defer was defined**.

If the defer is **invoked from another scope**, it no longer sees the variables that existed in the definition scope — because that environment **is not stored** in the defer value.

**Example:** the same scope, called from two places. It is the caller that decides what the
body sees, so the first call works and the second does not.

```aurora
ident show = defer { printb x; };

ident caller = defer {
  ident x = 10;
  show();     #- the caller has x → prints [0 0 0 0 0 0 0 10]
};

caller();
show();       #- fails: identifier x not found — called from the top, where x does not exist
```

Note that `show` is defined where no `x` exists at all, and still prints one: nothing about
the definition site is stored.

The defer value only stores a reference to the code range and the return key (`from`, `to`, `returnKey`). The definition-time environment is not stored (no closure).

---

## Summary

| Situation | Does the defer see it? |
|-----------|------------------------|
| **Caller's scope** (and outer scopes) | Yes |
| **Definition scope** (when the call is from elsewhere) | No |

So: the defer sees "outer" variables **at call time** (the caller), but **not** the **definition** scope.

---

## Relation to the overall design

This behavior is consistent with the model described in [defer_and_scope_callable_philosofy.md](defer_and_scope_callable_philosofy.md): defer does not introduce functions or closures; it is a scope whose execution is delayed and that receives data at invocation. The environment in which the body runs is that of **call time**, not creation time.
