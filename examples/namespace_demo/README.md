# Namespace demo

This example shows the **namespace syntax** (implicit import and optional alias).

## Structure

- **main.ar** — entrypoint: `use math as m;` then `print m::add(1, 2);`
- **math/add.ar** — defines `add` in the `math` namespace (path `math/` under root)

Namespace resolution: `math` → `<root>/math/*.ar`, so `math::add` is the symbol `add` from that path.

## Run

From the repo root (with a built `aurora` binary):

```sh
aurora run -s examples/namespace_demo/main.ar
```

**Note:** The namespace **resolver** is not implemented yet. The example **parses and compiles** correctly, but the evaluator fails with `call: identifier not found` because it does not load `math/add.ar` or resolve the alias `m` to `math`. Once the resolver is in place, this example should run and print `3`.
