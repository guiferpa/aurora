# Design: module system (resolver and loader)

**Status: proposal.** Nothing here is implemented. The previous attempt was removed (see below); this document exists so the next one starts from a decided shape.

---

## What went wrong the first time

Aurora had a namespace layer with `use path::to::ns as alias;` and a `linker` package. It was removed because it never worked end to end, and the half that existed got in the way. Three separate mistakes, worth naming so they are not repeated:

**1. A directory was the unit.** The entry point's *directory* defined the namespace, and every `.ar` file in it was compiled into one program. Ten independent examples in `examples/` collided on `ident a`, and `aurora run -p fib` failed with `conflict between identifiers named a`. Passing one file compiled its neighbours too.

**2. Symbol identity was never implemented.** The parser recorded `IdentifierLiteral.Namespace` and produced a `UseDeclaration`, and then *nothing read them*. The emitter ignored `use` entirely, and `m::add(1, 2)` emitted a call to `add` with no notion of where it came from. There was no namespace in the IR, none in the environ, none at call time.

**3. What made multi-file work was the accident, not the feature.** Because the whole directory was concatenated into one flat scope, names found each other by luck. The explicit mechanism (`use`) contributed nothing. That is why `examples/namespace_demo` parsed and linked but died at evaluation with `call: identifier not found`.

The lesson: **a module system is not a file-loading feature, it is a name-resolution feature.** Loading files is the easy half.

---

## Vocabulary

Three jobs that the old `linker` package mixed into one, and that should stay apart:

| Job | Question it answers | Input → output |
|---|---|---|
| **Resolver** | Which file does this import mean? | specifier (`"./math"`) → absolute path |
| **Loader** | What is in it, and in what order? | path → parsed module, dependency graph, topological order |
| **Binder** | Which declaration does this name refer to? | `m.add` → the `add` declared in module `m` |

The third is the one that was missing. It is also the one that touches the IR, the evaluator and the EVM backend.

**Why "linker" was the wrong name.** A linker resolves symbol references between *already compiled* objects, producing an executable. What the package did was discover, parse and order source files — that is module resolution and loading. The name may become right later: if each module ends up emitting its own instruction block with symbolic references to be patched, a real link step appears.

---

## What we are borrowing

### From TypeScript

- **A module is a file.** Not a directory, not a declaration. One file, one module, one scope. This alone kills mistake #1.
- **Explicit specifiers.** An import names a path; nothing arrives implicitly because it happens to sit nearby.
- **Relative vs non-relative specifiers carry different meaning.** `./math` is "the file next to me"; `math` is "something the project provides". Two syntaxes because they are two different questions.

### From Clojure

- **The alias is declared at the point of import**, and it belongs to the importing file alone. Nothing global, no ambient registry.
- **Dependencies are declared at the top**, so a file states what it needs before it uses it (`ns` + `:require`).
- **`:as` gives a short handle for a long name**, and the handle is how the symbols are reached — never by having them dumped into the local scope.

What we are **not** borrowing: Clojure's `:refer` (pulling names into the current scope) and TypeScript's `import { x }`. Both make a symbol's origin invisible at the point of use, which is the opposite of what an alias is for.

---

## The proposal

### Alias is mandatory

Every import names a parent, and the parent is the only way to reach anything inside:

```aurora
use "./math" as m;
use "./utils/strings" as s;

printb m.add(1, 2);
printc s.upper("hi");
```

There is no form that omits the alias, and no form that brings a name in bare. Reading `m.add(1, 2)`, you know where `add` lives without scrolling anywhere. This is also what keeps the binder simple: a qualified name has exactly one meaning, decided by one line in the same file.

The alias is a **binding in the file's scope**, not a keyword: it follows the same rules as `ident` — immutable, no shadowing, no redeclaration.

### Resolution rules

| Specifier | Meaning |
|---|---|
| `"./math"` | file `math.ar` next to the importing file |
| `"../shared/math"` | file `math.ar` in a sibling directory |
| `"math"` | non-relative: reserved for the project root and, later, dependencies |

The `.ar` extension is implied. A specifier that resolves to a directory is an error — a module is a file. If a directory ever gets to be a module, it will be through an explicit entry file, not by scooping up its contents.

### Loading

1. Resolve the entry file, parse it, collect its `use` declarations.
2. Resolve each dependency, recursively, caching by absolute path so a file is parsed once no matter how many modules import it.
3. Detect cycles and report them as a chain (`a.ar → b.ar → a.ar`), which the existing `CheckDependency` in the old linker already did well.
4. Order dependencies before dependents.

### Binding: the part that was missing

This is what makes the difference between a design that runs and one that only parses.

- A parsed module gets a **table of what it declares** — its top-level `ident` bindings.
- A qualified name `m.add` is resolved **at compile time**: the alias `m` points to a module, and `add` must exist in that module's table. If it does not, that is a compile error with a position (`lexer.NewError` already gives us that), and the language server underlines it.
- The IR carries **module identity**, not just a name. `OpCall` and `OpLoad` need to say *which* `add`. Two shapes are possible:
  - **Mangling**: the emitter rewrites the symbol to a unique name (`math.add`). Cheapest to build, keeps the IR shape as it is.
  - **A module operand**: the instruction carries the module id alongside the symbol. More honest, but touches the instruction encoding, the evaluator's environ and the EVM builder.

  Mangling is the pragmatic first step; the operand is where it should end up.
- The evaluator keeps **one environ per module** rather than one flat scope. A module's body runs once, in dependency order, and its bindings live in its own environ.

### What a module exposes

Everything it declares at the top level, for now. A `private`/`export` distinction can come later; it is not needed to make imports work, and adding it later breaks nothing.

---

## Open questions

Worth deciding before implementation, not during:

1. **Accessor syntax.** `m.add` reads naturally and matches most languages. `m::add` was the old syntax and stands out more, at the cost of a heavier token. Note that `.` is taken: it reads a struct field since structs landed, so a module accessor would have to share it or pick something else.
2. **Non-relative specifiers.** Do they resolve from the project root (the directory holding `aurora.toml`), or from a `src`/`lib` path the manifest declares? The manifest is the natural place to say.
3. **Cycles.** Reject outright, or allow and initialize lazily? Rejecting is simpler to explain and simpler to build.
4. **Manifest involvement.** Should `aurora.toml` list dependencies, or is the filesystem the whole story until third-party packages exist?
5. **Test files.** May a `*.test.ar` import the module it tests with the ordinary form, or does testing need something of its own?

---

## Why this stays a proposal for now

The unit of compilation is the file, and the CLI compiles exactly the file it is given. That is a coherent place to sit: everything the language does today works, and nothing pretends to work. The next step should be taken whole — resolver, loader **and** binder — because two out of three is precisely what left the last attempt broken.
