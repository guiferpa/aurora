# Design: Manifesto do Projeto Aurora (aurora.toml)

## Objetivo

Ter um arquivo de manifesto na raiz do projeto (no estilo `Cargo.toml` / `package.json`) que:

- Define metadados do projeto e ponto de entrada
- Define **profiles** (perfis) para deploy e call on-chain
- Reduz parâmetros gigantes na CLI ao testar em diferentes redes

Com isso, em vez de:

```bash
aurora deploy calc.ar https://eth-sepolia.g.alchemy.com/v2/xxx 0xabc123...
aurora call getResult 0xContractAddress... https://eth-sepolia.g.alchemy.com/v2/xxx
```

passa a ser possível:

```bash
aurora deploy                    # usa profile default (main)
aurora deploy sepolia            # profile como parâmetro
aurora call getResult            # contract do profile default
aurora call getResult sepolia
```

---

## Nome e localização do arquivo

- **Nome:** `aurora.toml`
- **Onde:** raiz do projeto (diretório atual onde o comando é executado, ou ascendentes, até achar o arquivo)

### Por que TOML e não JSON ou YAML?

| Formato | Prós | Contras |
|--------|------|--------|
| **TOML** | Comentários; legível; sem vírgula trailing; usado pelo Cargo (Rust); tipagem clara (string, int, tabelas). | Menos comum que YAML em DevOps. |
| **JSON** | Universal, toda linguagem parseia. | Sem comentários; vírgulas e aspas verbosas; ruim para humano editar config. |
| **YAML** | Comentários; muito usado (K8s, CI). | Indentação significativa (erros silenciosos); especificação complexa; ambiguidades. |

**Escolha:** TOML — manifesto é editado por humanos, comentários ajudam (ex.: `# preencher após deploy`), e a semântica é simples (chave-valor e tabelas). Se no futuro for necessário gerar/consumir por ferramentas que exijam JSON, pode-se ter um comando `aurora manifest to-json` ou suporte a ambos com convenção de nome (`aurora.toml` vs `aurora.json`).

---

## Estrutura proposta

### Seção `[package]`

Metadados do projeto e build.

```toml
[package]
name = "my-calc"
version = "0.1.0"
entry = "src/calc.ar"   # arquivo principal (default para build/run/deploy)
# description = "Calculator contract"
```

- **name:** identificador do projeto (opcional para uso futuro: registry, etc.)
- **version:** semântica do projeto (opcional)
- **entry:** caminho do arquivo principal; usado como default quando não se passa `[file]` em `build`, `run`, `deploy`

### Seção `[build]` (opcional)

```toml
[build]
output = "bin/contract.bin"   # default do -o quando em projeto com manifest
```

- **output:** caminho padrão do bytecode gerado por `aurora build` quando não se passa `-o`.

### Seção `[profile.<nome>]`

Cada profile agrupa configuração de rede e identidade para deploy/call.

```toml
[profile.local]
rpc = "http://127.0.0.1:8545"
privkey = ".env.local"   # arquivo com uma linha: hex da chave (sem 0x)
# Opcional: após primeiro deploy, pode ser preenchido para call
# contract_address = "0x..."

[profile.sepolia]
rpc = "https://eth-sepolia.g.alchemy.com/v2/MY_KEY"
privkey = ".sepolia.key"
gas_limit = 3_000_000

[profile.mainnet]
rpc = "https://eth-mainnet.g.alchemy.com/v2/MY_KEY"
privkey = ".mainnet.key"
gas_limit = 5_000_000
```

Campos por profile:

| Campo               | Obrigatório | Descrição |
|---------------------|------------|-----------|
| `rpc`     | sim        | URL do nó RPC (HTTP/WebSocket) |
| `privkey` | sim (deploy) | Caminho para arquivo que contém a chave privada em hex (uma linha, sem `0x`) |
| `contract_address`  | não        | Endereço do contrato já deployado; usado por `call` quando não passado na CLI |
| `gas_limit`         | não        | Override do gas limit no deploy (default: 3_000_000) |
| `chain_id`          | não        | Override; se omitido, usa o da rede via `eth_chainId` |

Boas práticas:

- Manter `aurora.toml` **sem** chaves privadas; só referência via `privkey`.
- Colocar `*.key`, `.env*` (ou o que guardar chaves) no `.gitignore`.

### Profile padrão: `main`

Em muitos cenários o desenvolvedor só precisa de um ambiente (ex.: só local ou só uma testnet). Para não obrigar a passar profile sempre:

- **Convenção:** o profile chamado **`main`** é o default.
- Se existir `[profile.main]`, `aurora deploy` e `aurora call <fn>` usam esse profile sem parâmetro.
- Se não existir `main` e houver **apenas um** profile no arquivo, esse único profile é o default.
- Se não existir `main` e houver **vários** profiles, o usuário deve passar o profile como parâmetro (ex.: `aurora deploy sepolia`).

**Override:** a variável de ambiente `AURORA_PROFILE` (ex.: `AURORA_PROFILE=sepolia`) pode sobrescrever o default quando o usuário não passa profile na CLI — útil em scripts ou CI.

Assim, projeto com um único profile pode ter só `[profile.main]` e nunca passar nome de profile na CLI.

---

## Comportamento da CLI com manifest

### Resolução do manifest

1. A partir do CWD, sobe diretórios até encontrar `aurora.toml` ou chegar na raiz do filesystem.
2. Se não achar: comportamento atual (todos os argumentos obrigatórios na CLI).
3. Se achar: usa `entry` e `[build].output` como defaults quando aplicável e lê o profile para deploy/call.

### Comando `build`

- `aurora build` → usa `package.entry` como arquivo e `build.output` como `-o` (se existir).
- `aurora build other.ar` → ignora entry, usa `other.ar`; `-o` opcional.
- Continua funcionando sem manifest com `aurora build arquivo.ar [-o out.bin]`.

### Profile como parâmetro (não flag)

Profile é **parâmetro posicional opcional**, não `--profile <nome>`:

- Fica claro que “qual ambiente” é um argumento do comando, não uma opção extra.
- Com um único profile (ex.: `main`), o usuário não passa nada: `aurora deploy`, `aurora call getResult`.
- Com vários: `aurora deploy sepolia`, `aurora call getResult sepolia`.
- Evita misturar com arquivo: a ordem pode ser `aurora deploy [file] [profile]` ou `aurora deploy [profile] [file]`; definimos uma ordem e documentamos (sugestão: `[file] [profile]` para deploy, `[function] [contract_address] [profile]` para call, com profile sempre por último quando opcional).

**Deploy:** `aurora deploy [file] [profile]`  
- Com manifest: se omitir ambos, usa `entry` + profile default. Se passar um arg: se parecer caminho de arquivo (contém `.ar` ou `/`), é file; senão é profile. Se passar dois: file e profile.

**Call:** `aurora call <function> [contract_address] [profile]`  
- Com manifest: se omitir contract e profile, usa `contract_address` do profile default. Se passar um arg após function: se for 0x..., é contract; senão é profile (e contract vem do profile).

### Comando `deploy`

- **Com manifest:**  
  `aurora deploy [file] [profile]`
  - Sem args: usa `package.entry` + profile default (`main` ou único).
  - Um arg: file ou profile (heurística: termina em `.ar` ou tem `/` → file; senão → profile).
  - Dois args: file e profile, nessa ordem.
  - Lê `rpc` e `privkey` do profile; opcionalmente `gas_limit` e `chain_id`.
- **Sem manifest:**  
  `aurora deploy [file] [address] [private key]` (comportamento atual).

Sugestão: quando há manifest, o `deploy` compilar o `entry` (ou o `file` passado) em memória e enviar o bytecode, em vez de enviar o `.ar` cru; assim o manifest elimina a necessidade de rodar `build -o` manualmente antes.

### Comando `call`

- **Com manifest:**  
  `aurora call <function> [contract_address] [profile]`
  - Sem args após function: usa `contract_address` do profile default.
  - Um arg: se 0x... → contract; senão → profile (contract do profile).
  - Dois args: contract e profile.
  - RPC sempre do profile.
- **Sem manifest:**  
  `aurora call [function] [contract address] [address]` (comportamento atual).

### Exemplos com manifest

```bash
# Um único profile [profile.main] — não precisa passar nada
aurora deploy
aurora call getResult

# Vários profiles: passar o nome como parâmetro
aurora deploy sepolia
aurora deploy src/other.ar sepolia
aurora call getResult sepolia
aurora call getResult 0xNovoEndereco... sepolia
```

---

## Exemplo completo de `aurora.toml`

Cenário com um único ambiente (profile `main` como default):

```toml
[package]
name = "calc"
version = "0.1.0"
entry = "examples/evm/calc.ar"

[build]
output = "bin/calc.bin"

[profile.main]
rpc = "http://127.0.0.1:8545"
privkey = ".aurora/local.key"
# contract_address = "0x..."  # após deploy, para call sem passar endereço
```

Cenário com vários ambientes (`main` segue como default; os outros são opcionais):

```toml
[package]
name = "calc"
version = "0.1.0"
entry = "examples/evm/calc.ar"

[build]
output = "bin/calc.bin"

[profile.main]
rpc = "http://127.0.0.1:8545"
privkey = ".aurora/local.key"

[profile.sepolia]
rpc = "https://eth-sepolia.g.alchemy.com/v2/..."
privkey = ".aurora/sepolia.key"
gas_limit = 3_000_000
```

Nota: expansão de variáveis de ambiente (`${ALCHEMY_KEY}`) pode ser uma fase 2; na v1 pode-se exigir a URL já expandida ou ler apenas `env("ALCHEMY_KEY")` em um campo dedicado.

---

## Resumo dos benefícios

- **Um único arquivo** descreve o projeto e os ambientes (local, testnet, mainnet).
- **CLI enxuta:** `aurora deploy` (com `main`) ou `aurora deploy sepolia`; profile como parâmetro, não flag.
- **Segurança:** chaves fora do manifest, em arquivos ignorados pelo git.
- **Compatibilidade:** sem manifest, a CLI continua recebendo todos os parâmetros como hoje.
- **Profiles reutilizáveis** entre máquinas (cada uma com seu `privkey` ou mesmo path diferente no mesmo profile).

Se quiser, o próximo passo é implementar a leitura do `aurora.toml` e a resolução de profiles no `cmd/aurora`, e então adaptar `deploy` e `call` para usá-los.
