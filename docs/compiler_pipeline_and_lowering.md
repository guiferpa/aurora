# Pipeline do compilador e Lowering

## Visão geral

O resumo abaixo alinha a arquitetura do compilador Aurora com a separação entre **IR neutra**, **Lowering** e **Builder**.

---

## Pipeline atual

```
Source → Lexer → Parser → AST → Emitter (IR) → Lowering (builder/evm) → Builder EVM → Bytecode
                                    ↓
                               Evaluator (interpreta IR)
```

- **Emitter** gera uma sequência linear de instruções (IR). Ex.: `GetArg(0)`, `GetArg(1)`, `Add`.
- **Evaluator** consome a mesma IR para executar em memória.
- **Lowering** (pacote `builder/evm`) reordena a IR para a stack EVM (Sub/Div left-assoc, RPN). O Builder chama `Lowering(body)` e `Lowering(rootinsts)` antes de escrever bytecode.
- **Builder EVM** consome a IR já reordenada e emite opcodes da EVM.

---

## Conceitos

### 1. IR (Representação intermediária)

A IR fica entre AST e o backend. Hoje é uma **lista de instruções** (opcode + operandos), ainda acoplada à ordem de uma stack machine: o Emitter emite na ordem “left, right, op”, o que por acaso bate com a ordem de empilhamento da EVM.

Uma evolução desejada é uma IR mais neutra (ex.: 3-address: `t0 = a + b`), independente de stack vs registradores.

### 2. Problema da stack (EVM)

A EVM é uma **stack machine**. A ordem em que os valores entram na stack (LIFO) define o resultado das operações. Ex.:

- Sequência correta: `PUSH a`, `PUSH b`, `ADD` → stack `[a, b]` → resultado `a + b`.
- Se a ordem estiver errada, seria preciso usar `SWAP`, o que mistura regras de “como empilhar” com o Builder.

Esse tipo de decisão **não deveria ficar no Builder puro**. Deve pertencer a uma fase explícita de **Lowering**.

### 3. Lowering

**Lowering** é a passagem que adapta a IR (neutra ou quase neutra) ao **modelo da máquina alvo**.

- **Entrada:** IR (ex.: operações em termos de “left”, “right”, “result”).
- **Saída:** sequência já no formato da máquina (para EVM: ordem explícita de pushes e ops na stack).

Exemplos:

| IR (neutra) | Lowering EVM        | Lowering WASM (futuro)   |
|-------------|---------------------|---------------------------|
| `t0 = a + b`| push a, push b, add | local.get 0, local.get 1, i32.add |

Quem decide **ordem de avaliação** e **layout na stack** é o Lowering, não o Builder.

### 4. Builder

O **Builder** deve ser **mecânico**:

- Receber uma sequência já adequada ao target (saída do Lowering).
- Traduzir cada “ação” em opcodes.
- Resolver labels/offsets e serializar bytecode.

Se o Builder começa a reorganizar operandos ou a decidir quando usar SWAP, ele está fazendo **lowering escondido** e assume responsabilidade que deveria ser do Lowering.

---

## OperandManager (removido)

O **OperandManager** (push/pop de operandos no Builder) foi uma tentativa de controlar a ordem dos valores na stack dentro do próprio Builder.

- **Problema:** ele não resolve o problema arquitetural:
  - Só **Save** fazia push; **GetArg** e **Load** não. Então para `arguments(0) + arguments(1)` o manager ficava vazio e dava panic.
  - A decisão “quem empilha o quê e em que ordem” continuava espalhada (Emitter + Builder), sem uma fase única de Lowering.

Hoje o Builder **não usa** o OperandManager para Add/Sub/Mul/Div: assume que a IR já vem na ordem correta (left, right, op) e só emite o opcode. Ou seja:

- A “ordem para a stack” está **implícita** na ordem em que o **Emitter** gera a IR, não em um Lowering explícito.
- O OperandManager pode ser removido ou mantido só para uso futuro (ex.: se um dia o Lowering passar operandos explícitos ao Builder), mas **ele não é a solução** para o problema de “garantir LIFO e evitar SWAP” de forma limpa.

---

## Arquitetura alvo (resumida)

```
Lexer → Parser → AST → Emitter → IR (neutra / 3-address)

                              ↓
                         Evaluator

                              ↓
              Lowering (por target)
                    ↓              ↓
            LoweringEVM      LoweringWASM (futuro)
                    ↓              ↓
            BuilderEVM        BuilderWASM
                    ↓              ↓
            Bytecode EVM      Binário WASM
```

- **IR:** única, independente de máquina.
- **Lowering:** adapta IR ao modelo (stack EVM, locals WASM, etc.).
- **Builder:** só escreve binário a partir da sequência já baixada.

---

## Lowering (implementado)

No pacote `builder/evm` existe uma fase de **Lowering** explícita (o nome do pacote já indica que é para EVM):

- **`Lowering(insts []emitter.Instruction) []emitter.Instruction`**  
  Recebe um bloco de IR (por exemplo o body de um defer ou as rootinsts) e devolve uma nova fatia com as instruções reordenadas/adaptadas à stack da EVM.

- **Passo atual:** `reorderLeftAssoc`  
  Reordena cadeias de Sub/Div para avaliação **left-associativa** na EVM. Ex.: IR com `a - (b - c)` (right-assoc) é reescrita para a ordem equivalente a `(a - b) - c`, para que o bytecode gerado dê o resultado esperado.

O Builder chama `Lowering(body)` antes de `WriteCode` no body de cada defer, e `Lowering(rootinsts)` antes de escrever o root. O Builder continua mecânico: só emite opcodes para a sequência que recebe.

---

## Estratégia incremental

- **Curto prazo (atual):**  
  - **Lowering** (pacote evm) implementado com reordenação Sub/Div left-assoc.  
  - Builder continua mecânico; a ordem da stack é garantida pelo Lowering.

- **Médio prazo:**  
  - Definir IR mais neutra (ex.: 3-address).  
  - Estender **Lowering** com mais passes (ex.: ordem de avaliação, temporários).  
  - Builder já não usa OperandManager (removido).

- **Longo prazo:**  
  - LoweringWASM + BuilderWASM para outro target.

---

## Objetivo

- Emitter gera IR neutra.
- Evaluator interpreta a IR.
- **Lowering** adapta a IR a cada VM (ordem de stack, etc.).
- **Builder** apenas escreve binário.
- Arquitetura pronta para EVM e WASM, com responsabilidades bem separadas.

O resumo que você trouxe está alinhado com esse desenho; o **OperandManager sozinho não resolve** o problema — a solução correta é uma fase explícita de **Lowering** (no pacote evm: `Lowering`) entre IR e Builder.
