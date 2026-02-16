# Aurora — Alpha Release Checklist

Checklist para o lançamento da **versão alpha** da linguagem Aurora. Use como guia antes de publicar a release.

---

## 1. Código e testes

- [ ] **Testes passando**  
  - `make test` (ou `go test ./...`) sem falhas.  
  - Pipeline CI (Go 1.23 e 1.24) verde.

- [ ] **Testes atualmente ignorados**  
  - Decidir para cada um: habilitar e corrigir ou documentar como “known limitation” da alpha.  
  - Arquivos com `t.Skip()`:  
    - `builder/evm/lowering_test.go` (TestLowering)  
    - `builder/evm/builder_test.go`  
    - `parser/parser_test.go`  
    - `internal/cli/run_test.go`

- [ ] **Lint**  
  - `make lint` (golangci-lint) sem erros.

- [ ] **Build dos binários**  
  - `make all` ou `make build-force`: gera `target/bin/aurora` e `target/bin/aurorals` sem erro.

- [ ] **Cobertura de testes**  
  - Revisar relatório (Coveralls / `make cover-html`) e garantir que módulos críticos (lexer, parser, emitter, builder/evm) tenham cobertura aceitável para alpha.

---

## 2. Compilador e EVM

- [ ] **Pipeline de compilação estável**  
  - Lexer → Parser → Emitter → Lowering (builder/evm) → Builder EVM → bytecode.  
  - Documentação em `docs/compiler_pipeline_and_lowering.md` alinhada com o código.

- [ ] **Lowering (Sub/Div e RPN)**  
  - Reordenação para left-assoc e RPN (ex.: `c b a - -`) funcionando e coberta por testes quando possível.

- [ ] **Exemplos de compilação**  
  - Pelo menos um exemplo em `examples/evm/` (ex.: `calc.ar`, `ident.ar`) compila e, se aplicável, roda/deploy em testnet sem surpresas.

- [ ] **Compatibilidade de bytecode**  
  - Bytecode gerado é aceito por cliente EVM (ex.: deploy em Sepolia ou rede local) para os casos documentados.

---

## 3. CLI e experiência do usuário

- [ ] **Comandos essenciais**  
  - `aurora init`, `aurora run`, `aurora build`, `aurora version`, `aurora help`, `aurora repl` funcionando conforme README.

- [ ] **Manifest (aurora.toml)**  
  - `aurora init` cria manifest válido; `build`/`run`/`deploy`/`call` respeitam `source` e `binary` e opcionais (rpc, privkey) conforme `docs/manifest.md`.

- [ ] **Deploy e call**  
  - Fluxo deploy → persistência em `.aurora.deploys.toml` → `aurora call` documentado e testado em testnet (ex.: Sepolia), com avisos de segurança (privkey, .gitignore).

- [ ] **Instalação**  
  - `go install github.com/guiferpa/aurora/cmd/aurora@HEAD` (ou tag da alpha) documentado e testado.

---

## 4. Documentação

- [ ] **README.md**  
  - Aviso “alpha / não usar em produção” visível.  
  - Get started (install, manifest, repl, run) correto.  
  - Links para manifest, playground e docs principais funcionando.

- [ ] **Manifest**  
  - `docs/manifest.md` com todos os campos usados na alpha (project, profiles, deploy state).

- [ ] **Design da linguagem**  
  - Untyped, tapes, reels, aritmética e comandos (print, echo, etc.) descritos de forma consistente com o comportamento atual.

- [ ] **Grammar**  
  - `docs/grammar.md` atualizado; itens da “demand list” marcados como feitos ou “alpha / known gap”.

- [ ] **CHANGELOG (recomendado)**  
  - Criar `CHANGELOG.md` com seção “Alpha (v0.x.x)” listando mudanças relevantes e limitações conhecidas.

---

## 5. Versão e release

- [ ] **Versão da alpha**  
  - Definir número (ex.: `0.14.0-alpha` ou manter `0.13.1` e tag `v0.13.1-alpha`).  
  - Atualizar `version/version.go` se a convenção for versão no código.

- [ ] **Tag Git**  
  - Criar tag (ex.: `v0.14.0-alpha`) no commit que representa a alpha.  
  - Push da tag para o remoto.

- [ ] **Release no GitHub (opcional)**  
  - Criar GitHub Release com a tag; anexar binários (aurora, aurorals) para linux/darwin/windows se desejado.  
  - Descrição curta: “Aurora alpha – linguagem que compila para EVM; não usar em produção”.

---

## 6. CI/CD e qualidade

- [ ] **Pipeline (`.github/workflows/pipeline.yml`)**  
  - Rodando em push para `main`; matriz Go 1.23 e 1.24; testes com race e coverage; sem falhas.

- [ ] **Badges no README**  
  - Go Reference, Last commit, Go Report Card, Pipeline, Coverage; links corretos.

- [ ] **Playground (se aplicável)**  
  - Workflow `playground.yml` e deploy do playground (WASM) funcionando; link no README correto.

---

## 7. Legal e repositório

- [ ] **LICENSE**  
  - Arquivo presente e adequado ao uso (ex.: MIT, Apache-2.0).  
  - README ou CONTRIBUTING menciona licença se relevante.

- [ ] **Avisos de segurança**  
  - README ou docs avisam: não commitar `privkey` em `aurora.toml`; usar env/secrets em produção futura.

---

## 8. Limitações conhecidas (alpha)

Documentar de forma visível (README ou CHANGELOG):

- [ ] Lista curta de “não implementado” ou “comportamento alpha” (ex.: built-ins para list, alguns casos de lowering, etc.).  
- [ ] Link para issues ou docs onde usuários podem reportar bugs ou sugerir melhorias.

---

## Resumo rápido

| Área           | Ação principal                                      |
|----------------|-----------------------------------------------------|
| Testes         | Todos passando; decidir sobre testes com `t.Skip()`  |
| Build/Lint     | `make all` e `make lint` ok                         |
| EVM/Lowering   | Pipeline estável; exemplos compilando               |
| CLI            | init, run, build, deploy, call, repl conforme docs   |
| Docs           | README, manifest, grammar e aviso alpha             |
| Versão         | Versão + tag Git (+ opcional GitHub Release)        |
| CI             | Pipeline verde; badges corretos                     |
| Legal/Segurança| LICENSE; avisos sobre privkey e uso em produção      |

Quando todos os itens relevantes para a alpha estiverem marcados, o projeto está pronto para anunciar a release alpha.
