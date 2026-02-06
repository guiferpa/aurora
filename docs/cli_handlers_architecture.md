# Proposta: Arquitetura dos handlers do CLI Aurora

## Situação atual

- Um único `main.go` (~380 linhas) com todos os comandos.
- Cada `RunE` repete: `FindProjectRoot()` → `Load()` → `Profile("main")` e mistura parsing de args, leitura do manifest e lógica de negócio (build, deploy, call).
- Difícil de testar a lógica sem rodar o Cobra; difícil de reutilizar ou evoluir cada comando de forma isolada.

---

## Objetivos

- **Separação de responsabilidades:** Cobra só faz binding de flags/args e chama um handler com entradas explícitas.
- **Testabilidade:** Lógica de cada comando testável sem subprocess ou Cobra.
- **Menos duplicação:** Contexto do projeto (root + manifest + profile) obtido em um só lugar.
- **Escalabilidade:** Novos comandos ou opções não incham um único arquivo.

---

## Proposta em 3 camadas

### 1. Contexto de comando (Environ)

Um tipo que carrega uma vez o que todos os comandos que dependem do manifest precisam:

```go
// internal/cli/env.go

type Environ struct {
    Root    string           // project root (dir of aurora.toml)
    Manifest *manifest.Manifest
    Profile manifest.Profile
}

// LoadEnviron finds project root, loads manifest, and returns Environ for the given profile.
func LoadEnviron(profileName string) (*Environ, error) {
    root, err := manifest.FindProjectRoot()
    if err != nil { return nil, err }
    m, err := manifest.Load(root)
    if err != nil { return nil, err }
    prof, err := m.Profile(profileName)
    if err != nil { return nil, err }
    return &Environ{Root: root, Manifest: m, Profile: prof}, nil
}

func (e *Environ) AbsPath(path string) string { return manifest.AbsPath(e.Root, path) }
```

Comandos que precisam do manifest chamam `LoadEnviron("main")` (ou profile vindo de arg) uma vez e usam `env.AbsPath(env.Profile.Target)` etc.

### 2. Handlers puras (entrada → erro)

Cada comando tem uma função que recebe um struct de parâmetros e executa a lógica, sem Cobra e sem globals:

```go
// cmd/aurora/handlers/build.go ou internal/cli/build.go

type BuildInput struct {
    Source    string   // path to .ar (from arg or manifest)
    OutputPath string   // path to write bytecode (from -o or manifest binary)
    Loggers   []string
}

func Build(ctx context.Context, in BuildInput) error {
    bs, err := os.ReadFile(in.Source)
    if err != nil { return err }
    // ... lexer → parser → emitter → builder, write to in.OutputPath
    return nil
}
```

Deploy e Call na mesma linha:

```go
type DeployInput struct {
    BinaryPath     string
    RPC     string
    Privkey string
}

func Deploy(ctx context.Context, in DeployInput) error { ... }

type CallInput struct {
    Function        string
    ContractAddress string
    RPC             string
}

func Call(ctx context.Context, in CallInput) error { ... }
```

O `RunE` do Cobra fica fino: monta o `*Input` a partir de args/flags e do `Environ`, e chama o handler.

### 3. Um arquivo por comando (Cobra)

Cada comando em seu próprio arquivo no mesmo `package main`, só definindo o `*cobra.Command` e o `RunE` que monta o input e chama o handler:

```go
// cmd/aurora/build.go

package main

import "github.com/spf13/cobra"

func init() { rootCmd.AddCommand(buildCmd) }

var buildCmd = &cobra.Command{
    Use:   "build [file]",
    Short: "Build binary from source code",
    Args:  cobra.MaximumNArgs(1),
    RunE:  runBuild,
}

func runBuild(cmd *cobra.Command, args []string) error {
    env, err := cli.LoadEnviron("main")
    if err != nil { return err }
    source := env.AbsPath(env.Profile.Source)
    if len(args) > 0 { source = args[0] }
    outPath := output
    if outPath == "" { outPath = env.AbsPath(env.Profile.Binary) }
    return cli.Build(cmd.Context(), cli.BuildInput{
        Source:    source,
        OutputPath: outPath,
        Loggers:   loggers,
    })
}
```

`main.go` fica só com: declaração de flags globais, `rootCmd`, `PersistentPreRunE`, `init` (ou lista explícita de `AddCommand` se preferir não usar `init` em cada arquivo) e `main()`.

---

## Onde colocar os handlers

**Opção A – Tudo em `cmd/aurora/`**

- `cmd/aurora/main.go` – root, flags, requireManifest, main.
- `cmd/aurora/env.go` – `LoadEnv`, `Env`.
- `cmd/aurora/build.go`, `run.go`, `deploy.go`, `call.go`, `init.go`, etc. – cada um com o `*cobra.Command` e o `runXxx` que chama o handler.
- Handlers (`Build`, `Deploy`, `Call`, …) no mesmo pacote, em arquivos como `handlers_build.go`, `handlers_deploy.go`, ou em um único `handlers.go` se forem curtos.

Vantagem: tudo do CLI em um lugar, sem novo pacote.

**Opção B – Handlers em `internal/cli`** *(adotada)*

- `internal/cli/env.go` – `Environ`, `LoadEnviron` (resolve root e carrega manifest).
- `internal/cli/build.go` – `BuildInput`, `Build`.
- `internal/cli/deploy.go` – `DeployInput`, `Deploy`.
- `internal/cli/call.go` – `CallInput`, `Call`.
- `internal/cli/run.go` – `RunInput`, `Run`.
- `internal/cli/init.go` – `InitInput`, `Init`.
- `cmd/aurora/*.go` – só Cobra e chamadas para `cli.Build`, `cli.Deploy`, etc.

Vantagem: lógica do CLI testável com `internal/cli` sem importar `main`; testes em `internal/cli` podem mockar leitura de arquivo / RPC.

---

## Resumo da estrutura alvo (Opção B)

```
internal/cli/
  env.go         # Environ, LoadEnviron
  build.go       # BuildInput, Build
  run.go         # RunInput, Run
  deploy.go      # DeployInput, Deploy
  call.go        # CallInput, Call
  init.go        # InitInput, Init

cmd/aurora/
  main.go        # rootCmd, flags, requireManifest, main()
  build.go       # buildCmd, runBuild → cli.Build(...)
  run.go         # runCmd, runRun → cli.Run(...)
  deploy.go      # deployCmd, runDeploy → cli.Deploy(...)
  call.go        # callCmd, runCall → cli.Call(...)
  init.go        # initCmd, runInit → cli.Init(...)
  repl.go        # replCmd
  version.go     # versionCmd
  writer.go      # (existente)
```

Fluxo por comando:

1. Cobra chama `runXxx(cmd, args)`.
2. `runXxx` obtém `Environ` (se precisar de manifest), monta o `XxxInput` a partir de args/flags e do `Environ`.
3. Chama o handler `cli.Xxx(ctx, input)`.
4. O handler não conhece Cobra nem flags globais; só o `Input` e o `context.Context`.

Assim você ganha uma arquitetura mais sofisticada para os handlers: contexto único, handlers puras e testáveis, e comandos organizados por arquivo, sem um único `main.go` gigante.
