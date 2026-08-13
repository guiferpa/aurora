# Changelog

All notable changes and release notes for Aurora are documented here.

---

## Unreleased

### Language

- **Every value is a tape of the same width.** Booleans are no longer a one-byte exception: `true` is a tape holding 1, indistinguishable from the number 1, and `nothing` is a tape of zeros — the same bytes as `0`.
- **Tape width is a compiler parameter**: `tape_size` in `aurora.toml` or `--tape-size` on `build`, `run` and `repl` (1 to 32 bytes, default 8, flag wins over the manifest). A literal that does not fit is a compile-time error instead of being truncated silently.
- Arithmetic now wraps at the tape width (`255 + 1` is `0` with one-byte tapes) and runs on 256-bit integers, which also fixes `^` losing precision above 2^53.
- The EVM backend emits `PUSH<n>` for the configured width, from `PUSH1` to `PUSH32`.
- The REPL prints values as numbers: with no boolean type of its own, there is no `true`/`false` to show. It also prints the value of the line that was typed, instead of every intermediate value of the expression in map order.
- **Tape operations run.** `pull`, `push`, `head` and `tail` were parsed and emitted but never evaluated, which also left tape literals like `[1, 2, 3]` broken. A tape is a shift register: `pull` shifts left with the value entering at the right, `push` shifts right with the value entering at the left, and what reaches the far end is discarded.

### Tooling

- `aurorals`, the language server, returned with semantic tokens, diagnostics, hover and completion. See [docs/lsp.md](docs/lsp.md).
- The REPL gained command history (`~/.aurora/history`) and line editing with the arrow keys.

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
