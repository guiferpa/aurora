# Development guide

How to build, run, test and lint Aurora locally, and the conventions we follow. For releasing, see [releasing.md](releasing.md); for how the compiler is organized, see [compiler_pipeline_and_lowering.md](compiler_pipeline_and_lowering.md).

---

## Requirements

| Tool | Version | Needed for |
|---|---|---|
| **Go** | 1.24+ (`go.mod` pins `go 1.24.0`, `toolchain go1.24.11`) | everything |
| **golangci-lint** | v2.7.2 (pinned) | `make lint` — installed automatically into `$GOBIN` |
| **tparse** | latest | `make test` pretty output — installed automatically |
| **act** | latest | `make act`, running the GitHub workflow locally — installed automatically |

The auto-installing targets write to `$(go env GOBIN)`; make sure it is on your `PATH`.

---

## Build and run

### Fast loop (preferred while developing)

```sh
go run ./cmd/aurora run -s examples/fibonacci.ar 10   # evaluator
go run ./cmd/aurora build -s src/main.ar -o bin/main  # EVM bytecode
go run ./cmd/aurora repl
```

`go run` always compiles what is on disk, so there is no stale-binary risk.

### Building a binary

```sh
go build -o /tmp/aurora ./cmd/aurora   # plain, fast
make aurora                            # ./target/bin/aurora, rebuilt whenever sources change
make build-force                       # clean + rebuild from scratch
```

`$(BIN)/aurora` depends on the `SOURCES` list (every `*.go` in the tree plus `go.mod` and `go.sum`), so editing code rebuilds the binary and running the target twice does nothing the second time. Adding a new file is picked up too, since the list is computed on each invocation.

Note that both targets build with `-race`: great for catching data races, ~20MB and noticeably slower than a plain `go build` for benchmarking.

> macOS ships GNU Make 3.81, which compares timestamps with one-second granularity. A build triggered in the same second as the previous one can be skipped — only relevant when scripting builds back to back.

### Manual testing trap: `-s` takes a directory, not a file

The linker treats a **namespace as a directory** and parses **every `.ar` file next to the one you pass** (`linker.GetUnits`). So:

```sh
go run ./cmd/aurora run -s examples/fibonacci.ar   # fails: conflict between identifiers named a
```

…because all of `examples/*.ar` get compiled together. To exercise a single program, put it in its own directory:

```sh
mkdir -p /tmp/ar && cp examples/fibonacci.ar /tmp/ar/
go run ./cmd/aurora run -s /tmp/ar/fibonacci.ar 10   # 55
```

### Inspecting each phase

```sh
go run ./cmd/aurora run -s /tmp/ar/main.ar -l lexer,parser,evaluator
go run ./cmd/aurora build -s /tmp/ar/main.ar -o /tmp/out.bin -l builder && xxd /tmp/out.bin
```

Valid loggers: `lexer`, `parser`, `emitter`, `evaluator` (run) and `builder` (build).

### Playground (WASM)

```sh
cd playground && make build && make assets   # output in playground/dist
```

---

## Tests

```sh
go test ./...                       # everything
go test ./repl/... -run TestEditor  # one package, one test
go test ./... -race                 # race detector (CI always uses it)
make test                           # -race -cover, pretty output, writes coverage.out
make bench                          # benchmarks with -benchmem
```

### Conventions

- **Standard library only.** No testify or other assertion libraries. Compare with `bytes.Equal`/`reflect.DeepEqual` and report with `t.Errorf("got: %v, expected: %v", got, want)`.
- **Table-driven with subtests** for anything with more than two cases:

  ```go
  cases := []struct {
      name string
      op   byte
      want int
  }{
      {name: "OpGetArg_push", op: emitter.OpGetArg, want: 1},
  }
  for _, tc := range cases {
      t.Run(tc.name, func(t *testing.T) { ... })
  }
  ```

- **Tests live in the same package** (`package evaluator`, not `evaluator_test`), so they can reach unexported state — e.g. `ev.environ.SetTemp(...)`. Keep it that way; the compiler internals are the thing under test.
- **Shared helpers go in a non-test file** when other packages need them: `parser/testutil.go` exports `NamespaceEqual` and `TokenEqual` for AST comparison that ignores pointer identity.
- **Filesystem and env**: always `t.TempDir()` and `t.Setenv()`. No test may touch the developer's home, the repo tree, or a real network.
- **Fixtures are inline source strings** (`const minimalAR = "ident x = 1 + 2;\nprint x;\n"`), not files on disk.
- **Benchmarks** live in `*_bench_test.go` (see `lexer/scanner_bench_test.go`).
- Naming: `TestThing` for the simple case, `TestThing_specificBehavior` when a type has several behaviors (`TestBuild_failsWhenSourceMissing`).

### What to test per layer

| Layer | What a test should assert | Reference |
|---|---|---|
| `lexer` | token stream (tags + matches) for a source string | `lexer/scanner_test.go` |
| `parser` | AST shape, using `NamespaceEqual` | `parser/parser_test.go` |
| `emitter` | no direct tests today — covered through the evaluator | see gap below |
| `evaluator` | **source → resulting bytes**; this is the de-facto integration suite | `evaluator/evaluator_test.go` |
| `builder/evm` | emitted opcode bytes and the lowering order | `builder/evm/lowering_test.go` |
| `internal/cli` | handler behavior against a temp dir (files created, errors) | `internal/cli/build_test.go` |
| `repl` | keystroke stream → resulting line, over `io.Reader`/`io.Writer` | `repl/editor_test.go` |

**Any change to language semantics needs an evaluator test** (source in, bytes out). That is the only place that proves lexer, parser, emitter and runtime still agree with each other.

---

## Coverage

```sh
make test          # writes coverage.out
make cover-html    # opens the HTML report
go test ./... -cover
```

CI uploads the profile to Coveralls (badge in the README). **There is no gate today** — nothing fails a build because of coverage. The rules below are the working agreement, not automation:

- **Never regress a package.** If you touch a package, its coverage should not go down.
- **New code ships with tests**: aim for ~70%+ on files you add; anything below deserves a note in the PR saying why.
- **Core layers target 80%+** (`lexer`, `parser`, `emitter`, `evaluator`, `builder/evm`) — they are pure functions over bytes with no excuse for being untested.
- **Glue is best-effort**: `cmd/aurora`, `logger`, thin wrappers. Prefer testing the handler in `internal/cli` over the Cobra command.

Current baseline (`go test ./... -cover`):

| Package | Coverage |
|---|---|
| `evaluator` | 85.1% |
| `evaluator/environ` | 85.0% |
| `lexer` | 79.9% |
| `byteutil` | 71.3% |
| `repl` | 55.1% |
| `parser` | 53.5% |
| `manifest` | 51.9% |
| `internal/cli` | 45.4% |
| `builder/evm` | 34.3% |
| `emitter`, `linker`, `fileutil`, `evaluator/builtin`, `logger`, `cmd/aurora` | 0% |

**Known gap:** `emitter` has no direct unit tests — it is exercised indirectly through `evaluator` and `builder/evm`. The `linker` (namespace resolution, dependency cycles) has none at all. Both are the highest-value places to add tests right now.

---

## Lint and formatting

```sh
gofmt -l repl/    # the packages you touched: must print nothing
go vet ./...
make lint         # golangci-lint v2.7.2, default linters (no .golangci.yml in the repo)
```

> `gofmt` is **not** part of the default golangci-lint set, so CI will not catch formatting. A handful of files that predate this guide are not gofmt-clean — `gofmt -l .` lists them. Keep what you touch formatted; reformatting the rest is a change of its own, not something to smuggle into an unrelated PR.

CI runs the linter in its own `lint` job (`pipeline.yml`), with the same version and arguments as `make lint`, so a clean local run means a clean CI run. Run it before pushing anyway; `make all` chains `test`, `lint` and a clean build.

The version lives in two places — the `$(LINTER)` install target in the `Makefile` and the `version:` input in `pipeline.yml`. Bump both together.

Formatting is plain `gofmt` (tabs). `.editorconfig` sets LF endings and a final newline everywhere, plus 2-space indentation for `.ts`/`.json` (playground).

---

## Continuous integration

| Workflow | Trigger | What it does |
|---|---|---|
| `pipeline.yml` | push to `main` | **test**: `go test -race -covermode atomic` + Coveralls upload · **lint**: golangci-lint v2.7.2 (runs in parallel with the tests) |
| `playground.yml` | push to `main`, manual | builds the WASM playground and deploys to GitHub Pages |
| `release.yml` | tag `v*` | GoReleaser: archives, `.deb`, Homebrew tap |

Run the pipeline locally with `make act` (installs `act`; needs Docker).

> The test matrix lists Go `1.23` and `1.24`, but `go.mod` carries `toolchain go1.24.11`, so the 1.23 job downloads and uses 1.24.11 as well. Both entries currently test the same toolchain.

---

## Dependencies

Keep the dependency list small — the project is a study compiler, and every dependency is a thing to explain. Current direct deps: `cobra` (CLI), `BurntSushi/toml` (manifest), `fatih/color` (output), `go-ethereum` (keccak/deploy), `x/term` (REPL raw mode).

**Pin the version when adding one.** `go get pkg@latest` may upgrade the `go` directive and drop the `toolchain` line as a side effect:

```sh
go get golang.org/x/term@latest    # bumped go 1.24.0 -> 1.25.0 and removed the toolchain line
go get golang.org/x/term@v0.36.0   # keeps the module's Go version untouched
```

Always check `git diff go.mod` after a `go get`, and run `go mod tidy`.

---

## Commits and pull requests

- Prefixed messages, following the existing history: `feat:` for behavior, `chore:` for everything else (refactors, docs, tooling). Imperative, lowercase, short.
- Branch off `main`; `main` is what CI, the playground deploy and releases follow.
- Before opening a PR: `gofmt -l .`, `go test ./... -race`, `make lint`.
- Update `CHANGELOG.md` when the change is user-visible (a new keyword, a CLI flag, changed semantics), and the docs under `docs/` when you change behavior they describe.
- Docs drift is a real problem in this repo: several documents describe designs that were never implemented (tape evaluation, `state`, namespace resolution). If you implement or change one of those, fix the document in the same PR.
