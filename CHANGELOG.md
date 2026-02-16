# Changelog

All notable changes and release notes for Aurora are documented here.

---

## Alpha (v0.13.1)

First alpha release: Aurora compiles source code to EVM bytecode. Use for study and experimentation only; **do not use in production**.

### Highlights

- **Pipeline:** Lexer → Parser → Emitter → Lowering (builder/evm) → Builder EVM → bytecode. See [docs/compiler_pipeline_and_lowering.md](docs/compiler_pipeline_and_lowering.md).
- **Lowering:** Reordering for left-associative Sub/Div and RPN stack order; covered by tests.
- **CLI:** `aurora init`, `aurora run`, `aurora build`, `aurora version`, `aurora help`, `aurora repl`, `aurora deploy`, `aurora call`.
- **Manifest:** `aurora.toml` with `[project]` and `[profiles]`; optional `rpc` and `privkey` for deploy/call. See [docs/manifest.md](docs/manifest.md).
- **Language:** Untyped; tapes (arrays), reels (strings), arithmetic; `if`/block expressions; deferred callables; `print`, `echo`, `assert`, `arguments`.

### Known limitations (alpha)

- **Emitter** has no direct unit tests (covered indirectly via evaluator and builder/evm).
- **Parser** and **builder/evm** coverage is partial; some grammar paths (e.g. tape operations, list built-ins) are not fully exercised.
- **Built-ins for list:** Not yet defined (see [docs/grammar.md](docs/grammar.md) demand list).
- **Deploy/call:** Documented and implemented; test on Sepolia or local chain for your use case. Do not commit `privkey` in `aurora.toml`; use env/secrets.
- **Bytecode compatibility:** Bytecode is generated for the documented pipeline; validate on your target chain (e.g. Sepolia) before relying on it.

### Reporting

- Bugs and feature requests: [GitHub Issues](https://github.com/guiferpa/aurora/issues).
- Project and docs: [README](README.md), [manifest](docs/manifest.md), [grammar](docs/grammar.md).
