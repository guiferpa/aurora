# Aurora — Alpha Release Checklist

Checklist para o lançamento da **versão alpha** da linguagem Aurora. Use como guia antes de publicar a release.

---

## 1. Código e testes

- [x] **Testes passando**  
  - `make test` (ou `go test ./...`) sem falhas.  
  - Pipeline CI (Go 1.23 e 1.24) verde.

- [x] **Testes atualmente ignorados**  
  - Todos os testes que tinham t.Skip() foram habilitados e estão passando (lowering, builder, parser, internal/cli/run).
- [x] **Lint**  
  - `make lint` (golangci-lint) sem erros.

- [x] **Build dos binários**  
  - `make all` ou `make build-force`: gera `target/bin/aurora` e `target/bin/aurorals` sem erro.

- [x] **Cobertura de testes**  
  - Revisado com `make test` + `go tool cover -func=coverage.out` e `make cover-html`.  
  - Módulos críticos: **lexer** 79.6%, **builder/evm** 34.3%, **parser** 22.4%, **emitter** 0% (cobertura indireta via evaluator/builder). Total 34.1%. Aceitável para alpha; melhorar parser e emitter em releases futuras.

---

## 2. Compilador e EVM

- [x] **Pipeline de compilação estável**  
  - Lexer → Parser → Emitter → Lowering (builder/evm) → Builder EVM → bytecode.  
  - Documentação em `docs/compiler_pipeline_and_lowering.md` alinhada com o código (Lowering explícito no diagrama).

- [x] **Lowering (Sub/Div e RPN)**  
  - Reordenação para left-assoc e RPN (ex.: `c b a - -`) funcionando e coberta por testes em `builder/evm/lowering_test.go`.

- [x] **Exemplos de compilação**  
  - `examples/evm/calc.ar` e `examples/evm/ident.ar` compilam com `aurora build -s examples/evm/calc.ar -o /tmp/calc.evm` (e ident análogo).

- [ ] **Compatibilidade de bytecode**  
  - Bytecode gerado é aceito por cliente EVM (ex.: deploy em Sepolia ou rede local) para os casos documentados.

---

## 3. CLI e experiência do usuário

- [x] **Comandos essenciais**  
  - `aurora init`, `aurora run`, `aurora build`, `aurora version`, `aurora help`, `aurora repl` existem e funcionam (verificado localmente).

- [x] **Manifest (aurora.toml)**  
  - `aurora init` cria manifest válido com `[project]` e `[profiles.main]` (source, binary); `docs/manifest.md` descreve campos e opcionais (rpc, privkey).

- [ ] **Deploy e call**  
  - Fluxo deploy → persistência em `.aurora.deploys.toml` → `aurora call` documentado e testado em testnet (ex.: Sepolia), com avisos de segurança (privkey, .gitignore).

- [x] **Instalação**  
  - README documenta `go install github.com/guiferpa/aurora/cmd/aurora@HEAD`; binários gerados por `make build-force` funcionam.

---

## 4. Documentação

- [x] **README.md**  
  - Aviso “alpha / não usar em produção” visível (“Don't use it to production”).  
  - Get started (install, manifest, repl, run) presente; links para manifest e playground.

- [x] **Manifest**  
  - `docs/manifest.md` com project, profiles, deploy state e opcionais (rpc, privkey).

- [x] **Design da linguagem**  
  - README descreve Untyped, Tapes, Reels, Arithmetic; comandos (print, echo, etc.) referenciados.

- [x] **Grammar**  
  - `docs/grammar.md` com “demand list” (itens [x] feitos; list built-ins [ ] como alpha/known gap).

- [x] **CHANGELOG (recomendado)**  
  - `CHANGELOG.md` criado com seção “Alpha (v0.13.1)”, highlights e limitações conhecidas.

---

## 5. Versão e release

- [x] **Versão da alpha**  
  - Versão no código: `version/version.go` com `0.13.1`. Definir tag (ex.: `v0.13.1-alpha`) quando for publicar.

- [ ] **Tag Git**  
  - Criar tag (ex.: `v0.13.1-alpha`) no commit que representa a alpha.  
  - Push da tag para o remoto.

- [ ] **Release no GitHub (opcional)**  
  - Criar GitHub Release com a tag; anexar binários (aurora, aurorals) para linux/darwin/windows se desejado.  
  - Descrição curta: “Aurora alpha – linguagem que compila para EVM; não usar em produção”.

---

## 6. CI/CD e qualidade

- [x] **Pipeline (`.github/workflows/pipeline.yml`)**  
  - Push para `main`; matriz Go 1.23 e 1.24; testes com race e coverage; Coveralls.

- [x] **Badges no README**  
  - Go Reference, Last commit, Go Report Card, Pipeline, Coverage presentes.

- [ ] **Playground (se aplicável)**  
  - Workflow `playground.yml` e deploy do playground (WASM) funcionando; link no README correto.

---

## 7. Legal e repositório

- [x] **LICENSE**  
  - Arquivo `LICENSE` presente na raiz do repositório.

- [x] **Avisos de segurança**  
  - `docs/manifest.md` e README avisam: não commitar `privkey` em `aurora.toml`; usar env/secrets.

---

## 8. Limitações conhecidas (alpha)

Documentar de forma visível (README ou CHANGELOG):

- [x] Lista em `CHANGELOG.md` (seção Alpha): emitter/parser/builder cobertura parcial, built-ins para list não definidos, deploy/call e bytecode para validar em testnet.  
- [x] CHANGELOG e README referenciam GitHub Issues para bugs e melhorias.

---

## Resumo rápido

| Área           | Ação principal                                      |
|----------------|-----------------------------------------------------|
| Testes         | Todos passando; testes com `t.Skip()` resolvidos    |
| Build/Lint     | `make all` e `make lint` ok                         |
| Cobertura      | Revisada; aceitável para alpha                      |
| EVM/Lowering   | Pipeline estável (doc atualizada); exemplos compilando |
| CLI            | init, run, build, version, help, repl, manifest ok  |
| Docs           | README, manifest, grammar, CHANGELOG; aviso alpha    |
| Versão         | 0.13.1 em código; pendente: tag Git e GitHub Release |
| CI             | Pipeline (Go 1.23/1.24); badges no README           |
| Legal/Segurança| LICENSE; avisos privkey em manifest e README        |
| Limitações     | Documentadas em CHANGELOG e grammar demand list     |

Quando todos os itens relevantes para a alpha estiverem marcados, o projeto está pronto para anunciar a release alpha.
