# `syscall`, o mundo, e o que falta para um contrato de verdade

**Estado:** proposta · **Data:** 2026-08-20

Esta RFC faz duas coisas:

1. **vira o eixo do `syscall`** — ele deixa de ser a porta para *fora* da chain e passa a ser
   a porta para **o mundo em que o programa roda**, sendo a chain um desses mundos;
2. **lista o que o código compilado precisa suportar** para a primeira versão beta ter um
   contrato que guarda estado e responde a quem chamou.

Ela **substitui** `syscall_and_stdlib.md` (PR #72), que resolvia só variável de ambiente. Tudo
que aquela decidiu sobre mecanismo continua valendo e está reafirmado aqui; o que muda é o
alcance e uma pergunta que ficou aberta lá.

Tudo abaixo foi medido no código de hoje.

> Os blocos aqui **não** estão marcados como `aurora`: `hosting/cli/docs_test.go` executa todo
> bloco assim marcado no repositório, e nada disto compila ainda.

---

## A virada

A #72 dizia: `env` como módulo faz o `use` ser **a declaração de que este programa é
off-chain**. Estava certo sobre o ambiente e errado sobre o mecanismo, porque tratava
"alcançar o mundo" e "não rodar na chain" como a mesma coisa.

Não são. Um contrato faz o tempo todo coisas que só o mundo dele responde — quem chamou, o que
está guardado no slot 3 — e nada disso o torna menos contrato. **O que a chain não tem é o
ambiente de um processo, não "o mundo".**

Então:

| módulo | o mundo que ele alcança | na chain | fora dela |
|---|---|---|---|
| `os` | o processo: variável de ambiente | **recusado, por decisão** | de verdade |
| `chain` | a chain: quem chamou, o que está guardado | de verdade (`CALLER`, `SLOAD`, `SSTORE`) | **simulado pelo host** |

A linha de `use` continua sendo a declaração — só que agora ela declara **qual mundo** o
programa precisa, e é isso que decide onde ele pode rodar. `use os` é um programa que a chain
não roda. `use chain` é um programa que roda nos dois lugares.

### E é aqui que a premissa da linguagem aparece inteira

A Aurora existe para que **a mesma chamada seja simulada fora da chain**. Um módulo `chain`
que só funcionasse na chain seria a negação disso.

Então ele funciona nos dois lados, e a diferença é quem responde:

| | `caller()` | `get(k)` / `set(k, v)` |
|---|---|---|
| na chain | `CALLER` | `SLOAD` / `SSTORE` |
| no evaluator | o endereço que o host configurou | um mapa que o host segura |

O harness diferencial (`hosting/cli/evm_harness_test.go`) já faz as duas metades: sobe uma EVM
em memória, faz o deploy, chama, e compara com o evaluator. **Simular a chain fora dela deixa
de ser uma promessa da documentação e vira uma linha de teste.**

---

## O que continua da #72

Reafirmado, não reaberto:

- **`syscall` é palavra-chave, e só a biblioteca padrão pode escrevê-la.** É a mesma regra de
  arquivo que limita `assert` a `*.test.ar`, lida do `ParseInput`. Um programa comum que a
  escreve é recusado, e a mensagem diz que o caminho para o mundo é o módulo.
- **O conjunto de chamadas é fechado**, e o parser recusa um nome que não existe dizendo o que
  existe.
- **A porta.** O evaluator não toca no mundo; um syscall chega como porta, igual ao print.
- **Módulo embutido pelo resolver**, com `Builtin func(module.ID) ([]byte, bool)` nas Options,
  fonte em `shared/stdlib` via `//go:embed`. O módulo é parseado, prefixado, carregado e
  avaliado como qualquer outro — **é um módulo comum que mora dentro do binário**.
- **Sombreamento é recusado**: se existe `src/chain.ar` no projeto e `chain` é embutido, o
  import é recusado em vez de silenciosamente vencer.
- **O valor tem que caber numa tape**, e o que não cabe é recusado em runtime dizendo os dois
  números.

---

## O que muda

### `syscall` recebe mais de um valor

Era uma pergunta em aberto na #72 ("chega para o ambiente; a primeira chamada com dois
argumentos vai pedir uma forma que não existe"). A chain é essa chamada:

```
syscall sstore feed(0) feed(1)
```

Proposta: a forma é `syscall <nome> <expr>*`, com a aridade **conferida no parser** contra a
tabela de chamadas — o conjunto é fechado, então cada nome já diz quantos valores quer, e
errar a conta é erro de compilação com a mesma cara dos outros.

### A lista `by decision` do backend encolhe

`OpSyscall` **não** entra inteira nela. O que entra é `os`: um módulo, não uma instrução. O
`aurora build` recusa um programa que alcança o processo dizendo **qual módulo** o torna
impossível — que era o ganho da #72 e continua sendo, agora com escopo certo.

---

## A superfície do módulo `chain`

Proposta mínima, uma pergunta por escopo:

| | responde |
|---|---|
| `chain.caller()` | quem chamou |
| `chain.get(key)` | o que está guardado nessa chave, ou o valor neutro |
| `chain.set(key, value)` | guarda, e responde o valor guardado |

Três syscalls por trás: `caller`, `sload`, `sstore`.

**Uma chave é uma tape, e um slot é uma palavra.** Com `tape_size` 32 a correspondência é
direta. Abaixo disso a tape é estendida para a palavra, do mesmo jeito que um resultado
aritmético já é cortado de volta para a largura da tape — a regra existe e está provada nos
dois lados.

> **Aberto:** `set` responder o valor guardado é o que faz dele uma expressão, e a Aurora só
> tem expressões. A alternativa é responder o valor **anterior**, que é mais útil e menos
> óbvio.

### O que a superfície mínima não dá, e por quê

Um token guarda saldo **por carteira**, e isso é um mapa: o slot é `keccak(chave . slot)`.

Hoje o keccak da Aurora acontece em **build time**, em Go, para o seletor do dispatcher
(`builder/evm/writer.go:169`). Não existe `KECCAK256` em runtime. Ou seja: mapa é um degrau a
mais, e ele tem nome — `syscall keccak`, ou uma derivação de slot escrita no próprio módulo
`chain` em cima de uma chamada de hash.

Fora do beta. Registrado aqui para não parecer esquecimento.

---

# Parte 2 — o que o código compilado precisa suportar

O estado de hoje, medido: `WriteCode` cobre **aritmética, `OpSave`, `OpIdent`, `OpLoad`,
`OpGetFeed` e `OpReturn`**. Tudo o mais compila e **não faz nada na chain** — silenciosamente,
que é a parte perigosa.

Em ordem de dependência:

| # | o que falta | por que o beta precisa | tamanho |
|---|---|---|---|
| 1 | **`ident` dentro de escopo** — hoje o valor que ele deveria guardar é descartado e o `MSTORE` acha a pilha vazia; o contrato **reverte** | qualquer corpo de `defer` com mais de uma linha | design |
| 2 | **Largura de verdade** — offset de memória, alvo de jump e tamanho do runtime são todos `PUSH1` e truncam depois de 255 | qualquer contrato que passe de 255 bytes, o que é todo contrato com estado | design |
| 3 | **`if` → `JUMPI`** | não transferir mais do que se tem | escrever |
| 4 | **`call` → jump com endereço de retorno** | decompor, e **recursão** | escrever |
| 5 | **`assert` → `REVERT`** | desfazer o que falhou; é a primitiva de falha que a linguagem **já tem** | escrever |
| 6 | **`sload` / `sstore` / `caller`** | estado e identidade | escrever (depois desta RFC) |
| 7 | **`OpJoin` / `OpField`** — shape como corrida de palavras na memória | devolver mais de um valor de uma chamada | escrever |
| 8 | **`KECCAK256` em runtime** | mapa: saldo por carteira | escrever |
| 9 | **`LOG0`–`LOG4`** — não estão nem na tabela de opcodes | evento `Transfer` | escrever |

**1 e 2 são os dois do stand-by**, e continuam sendo o que eles sempre foram: desenho, não
remendo. Nada abaixo deles vale a pena antes deles — um `if` compilado dentro de um corpo que
reverte é um teste verde que mente.

**5 merece uma frase.** A linguagem já tem `assert`, e ele já significa "isto tem que valer".
Na chain, "isto tem que valer" é `REVERT`. Não é preciso inventar `require`: é o mesmo `assert`
com o backend fazendo o que a palavra sempre disse — e vira mais um lugar onde a mesma linha
responde a mesma coisa dentro e fora da chain, que é a tese inteira.

---

## Os três degraus

| degrau | precisa de | o que prova |
|---|---|---|
| **contador** — `increment()`, `get()` | 1, 2, 3, 6 | que a chain guarda algo entre duas chamadas, e que o evaluator responde igual |
| **cofre** — `deposit()`, `withdraw()` só para o dono | + 4, 5 | identidade e falha: um saque de quem não é dono reverte nos dois lados |
| **token** — `mint`, `transfer`, `balance_of` | + 7, 8, 9 | saldo por carteira e evento |

**Proposta de alvo do beta: o cofre.**

O contador é pouco — prova estado e nada mais. O token é longe: mapa e evento são dois
subsistemas que não existem em nenhuma forma. O cofre é o menor programa que é
inegavelmente um contrato: **guarda valor, sabe quem está falando com ele, e recusa**. E os
três verbos dele são exatamente os três primeiros syscalls desta RFC.

O que faz dele um beta da *Aurora*, e não um contrato qualquer: o mesmo arquivo roda no
`aurora run` contra uma chain simulada e na EVM contra uma de verdade, e o harness compara as
duas respostas. Ninguém mais faz essa frase.

---

## O que fica de fora do beta

- **Mapa e evento** (8 e 9), que são o token.
- **Escrever no ambiente**, listar variáveis, arquivo, relógio, rede.
- **`msg.value`** e transferência de ether: o cofre guarda um número, não saldo nativo.
- **Constructor com argumentos.** O deploy de hoje não passa nada.

---

## As etapas

1. **`syscall` vira palavra-chave**, com a regra de arquivo, a aridade conferida contra a
   tabela, o nó e o opcode.
2. **A porta no evaluator**, com a chain simulada por trás dela — um mapa e um endereço que o
   host configura — e `os` sobre `os.LookupEnv`.
3. **`shared/stdlib` e o `Builtin` no resolver**, com `chain.ar` e `os.ar` embutidos e a recusa
   do sombreamento.
4. **O backend**: os dois do stand-by primeiro, depois `if`, `call`, `assert`, e então os três
   syscalls da chain.
5. **O cofre no harness diferencial**, que é o que fecha o beta.

Ordem de grandeza: 1–3 são a RFC #72 com uma tabela a mais (~400 linhas de código, ~500 de
teste, dois pull requests). 4 é o trabalho de verdade, e os dois primeiros itens dele precisam
de conversa antes de código.

---

## Em aberto

1. **`set` responde o valor guardado ou o anterior.**
2. **Nomes do módulo**: `chain` ou `evm`; `get`/`set` ou `load`/`store`.
3. **`caller` fora da chain**: o host configura um endereço, mas o REPL e o `aurora run` sem
   configuração respondem o quê — zero, ou uma recusa dizendo que não há quem chamou?
4. **A chave abaixo de 32 bytes**: estender para a palavra (proposta) é o que a aritmética já
   faz, mas dois projetos de larguras diferentes escrevem no mesmo slot com chaves diferentes.
   Isso é problema de quem simula, não da chain.
5. **`os` na chain**: recusar no `build` (proposta) ou compilar e reverter na chamada.
