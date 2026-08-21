# O IR

**Estado:** proposta · **Data:** 2026-08-21

O IR é a saída do frontend. Ele deveria descrever **o programa**.

Hoje ele descreve **uma execução dele**. O que existe em `wire/ir` é o conjunto de instruções
do evaluator — uma fita de interpretador — e o `builder/evm` é um segundo leitor que recebeu
essa fita e tenta reconstruir o programa a partir dela. Tudo que ele faz de constrangedor sai
daí: ele varre, casa padrão e supõe, porque a estrutura de que precisa não está escrita.

Isto não é crítica ao evaluator. Ele veio primeiro, e foi certo que viesse — a linguagem se
prova nele. É a constatação de que **o IR nunca foi desenhado como IR**: foi desenhado como o
que o evaluator precisava executar, e o segundo consumidor chegou depois e se virou.

São duas doenças, e vale separá-las desde já porque as curas são diferentes. O IR **não
descreve a estrutura** do programa, então o backend a reconstrói contando e casando padrão. E o
IR **nomeia operações que não define**, então cada consumidor implementa a sua leitura do nome —
e os dois já responderam coisas diferentes para o mesmo programa, que é a única promessa que
este projeto fez. Duas vezes, e as duas foram descobertas depois do fato.

A EVM aparece aqui como exemplo, nunca como motivo. O teste de cada decisão é se um backend
WASM, ARM ou x86 quereria a mesma coisa.

Tudo abaixo foi medido no código de hoje.

---

## O que o IR é hoje, medido

| o que deveria ser do programa | o que é |
|---|---|
| o nome de um valor | chave num mapa de runtime — `SetTemp(byteutil.ToHex(label))` |
| um escopo diferido | um par de posições do cursor: `from := e.cursor + 1` (evaluator.go:469) |
| um desvio | aritmética de cursor — `AddCursor(byteutil.ToUint64(right) + 1)` |
| uma chamada | laço de interpretação aninhado, com cursor salvo e restaurado |
| um nome ligado | resolvido em execução, subindo uma cadeia de `environ` |
| um argumento | escrito por índice num `environ` antes da chamada |

O `EvaluateDefer` diz tudo sozinho:

```go
from := e.cursor + 1
to := from + bodylength
e.environ.SetDefer(deferKey(index), encodeDeferBlob(from, to, returnKey))
```

O valor de um `defer` **é um par de posições do cursor de quem está executando**. Não há
programa descrito nisso; há um marcador de fita. Por isso o builder recorta corpo contando
instruções à mão (`PickDeferAtCursor`: quarenta linhas que falham de cinco jeitos e devolvem
`ok bool` por causa disso).

E o `+1` aparece em `EvaluateDefer`, `EvaluateIf`, `EvaluateJump` e `EvaluateCall`. Cada um é um
off-by-one esperando acontecer, e **nenhum significa coisa alguma para um backend**.

## O que isso custa em qualquer arquitetura

1. **Reconstruir estrutura varrendo.** Onde começa e termina um escopo é contado, não escrito.
2. **Casar padrão para entender uma operação.** Uma chamada é *uma sequência de `OpPushFeed`
   seguida de um `OpCall`*. Casar padrão foi exatamente o que quebrou o lowering antes.
3. **Refazer resolução de nome.** O `IdentManager` mapeia nome para lugar — trabalho de
   frontend escorrendo ladeira abaixo.
4. **Não poder mover nada.** Contagens relativas trancam a lista na ordem em que foi escrita.
5. **Não poder recusar nada.** Não há como verificar um IR que não afirma nada, então o único
   desfecho de um erro é o binário sair quieto.
6. **Não poder concordar.** O IR nomeia operações sem defini-las, então cada consumidor
   implementa a sua leitura do nome, e as duas vezes em que isso divergiu foram descobertas
   depois do fato.

---

# A forma

## O TAC não é o limite

Três endereços é a **forma de uma operação** — `valor = op a, b` —, não um teto sobre o
registro. O nome conta endereços, não campos: o TAC clássico já tem `goto L`, que nomeia
rótulo, e `call p, n`, que carrega contagem.

O que limita hoje é a codificação: `{label, opcode, left []byte, right []byte}`, dois campos sem
tipo. **A forma fica; a codificação sai.**

## Dois eixos, independentes

**O registro da instrução** deixa de ser dois `[]byte` e vira um struct com campos nomeados e
operandos em número livre.

**A forma do programa** deixa de ser lista chapada e vira grafo de blocos com parâmetros.

O primeiro cura "a instrução não conta o que é". O segundo cura "o programa não está
descrito". São doenças diferentes e se resolvem separadas.

## Bloco com parâmetros

Um bloco declara os valores que recebe; quem desvia para ele passa esses valores. É o que
Swift SIL, MLIR e Cranelift fazem, e dispensa o `phi`.

O parâmetro existe para resolver **o encontro dos braços de um desvio**, que é onde o IR de
hoje recorre a um truque: `emitIfExpression` faz os dois braços escreverem o **mesmo rótulo**
(`OpReturn(inl, valorDoBraço)` nos dois), e funciona porque num interpretador a última escrita
vence. Num IR isso é um valor definido duas vezes.

```
b0(vector):               ; o corpo de um defer recebe o vetor aplicado a ele
  v0 = feed  vector, 0
  v1 = feed  vector, 1
  v2 = gt    v0, v1
  br v2 -> b1, b2

b1:                       ; entao
  v3 = sub   v0, v1
  br b3(v3)               ; passa o valor do braco adiante

b2:                       ; senao
  v4 = sub   v1, v0
  br b3(v4)

b3(x):                    ; x E o valor do if
  ret x
```

## Uma chamada não é um desvio com parâmetros

Seria elegante dizer que chamada e desvio são o mesmo mecanismo. **Para a Aurora é falso**, e a
razão é a linguagem: um escopo não tem assinatura, não tem aridade e não tem lista de
parâmetros — executar é **aplicar um vetor de valores a um escopo**, e `feed(n)` lê uma posição
desse vetor.

Então o bloco de um escopo recebe uma coisa: o vetor. Não há aridade a declarar, e o `Verify`
não tem o que conferir numa chamada — a conferência de aridade vale para **desvio**, onde ela
existe. E `feed` continua sendo uma operação e não vira leitura de parâmetro:
`v0 = feed vector, 0`.

## O IR nomeia operações que não define

Esta é a segunda doença, e é mais séria que a primeira: a primeira faz o backend adivinhar
**estrutura**, esta faz ele adivinhar **significado**.

Já custou duas vezes, e as duas do mesmo jeito: alguém descobriu depois do fato, e remendou à
mão no lado que estava com a razão sobre a própria regra.

**A largura do tape.** O comentário do `WriteMask` conta a primeira:

> *"without this the same program answers two different things: at one byte, 255 + 1 is 0 in
> the evaluator and 256 on chain."*

**Ler além do que foi aplicado.** `OpGetFeed` carrega um índice e mais nada. O que ele fazia
quando o índice passava do fim do vetor estava escrito num lugar só — `evaluator/builtin/functions.go`,
num `%` — e em nenhum outro. O `builder/evm` escreveu um `CALLDATALOAD` num offset fixo, que é a
leitura óbvia de um opcode chamado "pegue o argumento n". Um escopo que lê `feed(0) + feed(1)`,
chamado com um valor, respondia `10` fora da cadeia e `5` nela.

Isso é exatamente a promessa pela qual o projeto existe, quebrada: *"the same program answers
the same thing on a chain and off it"*. As duas foram corrigidas — a segunda em #92, que
também trouxe o caso de harness que a prova. Nenhuma das duas foi **prevenida**, e é isso que
esta seção trata.

Porque não é deslize, é estrutural. As outras operações de tape — `head`, `tail`, `pull`,
`push`, `join`, `field` — não divergem ainda **só porque o backend não as implementou**; estão
todas em `pending`. No dia em que forem implementadas, cada uma é uma chance nova de o segundo
consumidor ler o nome do seu jeito.

A regra que fecha isso:

> **Uma operação do IR é definida pelo IR. Um consumidor implementa a definição.** Onde a
> definição não está escrita, a primeira implementação vira a definição por acidente — e a
> segunda chuta.

Duas coisas concretas saem daí, e nenhuma é um campo na instrução:

**A definição de cada opcode mora em `wire/ir`, ao lado do opcode.** É o mesmo argumento que o
`format.go` já faz para a impressão — *"writing the vocabulary down is part of the vocabulary,
the way a token knows how to spell itself"*. O que uma operação faz é parte do vocabulário, e
hoje mora na implementação de um dos leitores.

**O harness diferencial ganha um caso por operação definida.** É o único mecanismo que prova
que os dois consumidores concordam, e hoje ele cobre aritmética sobre argumentos e mais nada.

Qual é a semântica de cada operação não é assunto desta RFC — muda o que um programa existente
responde, e é conversa de linguagem em PR próprio, como foi a de `feed` na #92. O que ela
decide é que, seja qual for, **ela fica escrita no IR** e os dois consumidores a implementam a
partir dali.

## O que deixa de ser instrução

| hoje | vira |
|---|---|
| `OpBeginScope` | o bloco **é** o escopo |
| `OpPushFeed` | os valores viajam na própria chamada |
| `OpPreCall` | nada — está declarado e nunca foi emitido |
| `OpIf`, `OpJump`, `OpReturn` | terminadores, que não são instruções da lista |
| `OpDefer` | um valor que referencia um bloco |
| `OpIdent` | **nada**: um nome local vira número de valor no emitter |
| `OpLoad` (local) | **nada**, pelo mesmo motivo |

`OpGetFeed` **fica**, pela razão acima. Some quem só existia como protocolo de interpretador.

O caso do `ident` merece a linha: com valor numerado e nome resolvido no frontend,
`ident x = feed(0); x + feed(1);` não deixa binding nenhum, porque `x` **é** `v0`.

```
b0(vector):
  v0 = feed vector, 0
  v1 = feed vector, 1
  v2 = add  v0, v1
  ret v2
```

O conflito de nomes que o `OpIdent` hoje detecta em execução (*"conflict between identifiers
named x"*) é checagem semântica: sobe para o frontend e vira erro de compilação, que é onde
deveria estar.

Isto é SSA, na forma que dispensa `phi`. Ganhamos a propriedade de graça — cada valor definido
uma vez — sem construir maquinaria de otimização nenhuma sobre ela.

---

# O que a instrução carrega

## A regra dos dois níveis

Enriquecer o IR não pode abrir porta para o IR mentir. O critério que fecha essa porta é o do
LLVM, e vale inteiro:

> **Se descartar o fato muda o que o programa significa, ele é campo — e o verificador o
> confere contra as instruções. Se descartar não muda nada, é metadado — e descartá-lo tem
> que ser sempre seguro.**

| fato | nível | por quê |
|---|---|---|
| tipo do operando | campo | descarte e o consumidor volta a adivinhar |
| o que a instrução deixa | campo | descarte e a pilha desalinha |
| efeito (puro / escreve) | campo | descarte e reordenar vira incorreto |
| parâmetros de um bloco | campo | descarte e o desvio não sabe o que entregar |
| origem (linha, coluna) | metadado | descarte e some a linha do aviso, não o programa |

**Campo carregado é campo verificado.** Um fato que o `Verify` não consegue conferir contra as
instruções não entra como campo: ou é metadado descartável, ou não entra.

## O que entra, e o teste de três perguntas

1. **O frontend sabe?** Se não sabe, não é assunto do IR.
2. **O consumidor recupera barato?** Se recupera, não carregue — derivar não fica velho.
3. **Vale em qualquer alvo?** Se não, é saída do lowering.

Passa nas três, entra. O que reprova está na seção do que fica de fora — inclusive coisas que
pareciam boas ideias.

## A forma, em código

```go
// wire/ir/operand.go

// Kind says how an operand is read. The same bytes mean different things under different
// kinds, and nothing in a byte slice says which.
type Kind byte

const (
	Ref  Kind = iota // a value computed earlier in this block, or a parameter of it
	Imm              // the bytes themselves: a tape, a width, an index
	Name             // a name that outlives a block — a binding from another module
	Text             // bytes written for a person to read
)

// An Operand is one of the things an instruction points at, and what it is.
type Operand struct {
	kind  Kind
	value uint32 // Ref: the number of the value
	bytes []byte // Imm, Name, Text
}
```

```go
// wire/ir/instruction.go

// Effect says whether an instruction can be moved.
//
// A consumer emits a value next to whoever takes it, which is sound only for an instruction
// whose whole result is the value it leaves. A print writes, and moving it past another print
// changes what the program says.
type Effect byte

const (
	Pure   Effect = iota // the value it leaves is all it does
	Writes               // it says something, and when it says it matters
)

// An Instruction computes one value from its operands. The shape is three-address —
// `value = op operands...` — with the operand count free, so a construction of n values is
// one instruction and not a chain of n.
type Instruction struct {
	Value    uint32 // the number this defines, unique in its block
	Op       byte
	Operands []Operand
	Effect   Effect
	Origin   Origin // metadata: dropping it loses a diagnostic, never a meaning
}
```

```go
// wire/ir/block.go

// A Block is a scope, and the only unit of structure. It takes values, computes values from
// them, and ends in exactly one terminator.
//
// It replaces counting. A deferred scope used to be found by reading a length off OpDefer and
// slicing, which locked the instruction list into the order it was written in.
type Block struct {
	ID     BlockID
	Params int // values applied to it; an instruction reads one as a Ref
	Insts  []Instruction
	Term   Terminator
	Origin Origin
}

// A Terminator is how a block ends, and the only thing that decides where control goes. An
// instruction never moves control — which is what makes every instruction movable.
type Terminator struct {
	Kind    TermKind // Ret, Br, BrIf
	Cond    Operand  // BrIf
	Value   Operand  // Ret
	Targets []Target // Br: one. BrIf: two.
}

// A Target is a block and the values handed to its parameters.
type Target struct {
	Block BlockID
	Args  []Operand
}
```

```go
// wire/ir/program.go

// Program is what a source file compiled to: every scope it declared, and where to start.
type Program struct {
	Blocks      []Block
	Entry       BlockID
	Expressions []Expression // where each top-level expression's value landed
	Warnings    []diag.Warning
}
```

---

# O que isso compra: um IR verificável

É o item que só existe depois dos outros, e é o que os paga. Com operando tipado, bloco com
parâmetros e terminador, dá para escrever `Verify(Program) error` e afirmar, antes de qualquer
consumidor tocar em nada:

- toda `Ref` nomeia um valor computado antes dela no mesmo bloco, ou um parâmetro dele;
- todo alvo nomeia um bloco que existe;
- todo **desvio** entrega exatamente a quantidade de valores que o bloco de destino declara
  (uma chamada não: um escopo não tem aridade, e é decisão da linguagem);
- todo bloco termina em exatamente um terminador;
- todo operando tem o `Kind` que o opcode espera;
- nenhum valor é definido duas vezes.

Hoje **nenhuma dessas seis frases é verificável**, e é por isso que o modo de falha deste
backend sempre foi o mesmo: o binário sai, faz deploy, responde, e ignora em silêncio. O
`builder/evm/support.go` existe para minorar isso à mão, com uma lista de opcodes `handled`
mantida por disciplina humana.

Um verificador troca a lista por uma afirmação — a diferença entre um compilador que recusa e
um que entrega bytes errados.

---

## O que fica de fora, e por quê

**Pilha, memória, registrador, frame, `JUMPDEST`, `PUSH2`.** Reprovam na pergunta 3: só
significam algo num alvo. São saída do lowering.

**Contagem de leituras de um valor.** Reprova na pergunta 2: sai numa passada sobre o bloco.
Carregar seria guardar um número que pode discordar das instruções, para poupar uma varredura
barata.

**Grafo de fluxo — `preds`, `succs`.** Mesma razão. O Go carrega os seus porque muta o grafo em
dezenas de passes e recomputar sairia caro; a Aurora não muta.

**Tipo de valor.** A linguagem é não-tipada e todo valor é um tape de largura única; `shape` é
resolvido no parser e some antes daqui, de propósito. Não há tipo para carregar.

**`phi`.** A forma escolhida o dispensa.

**Otimização de qualquer espécie.** Esta RFC não propõe deixar o programa mais rápido. Propõe
deixá-lo descrito.

---

## O preço, dito com todas as letras

**O evaluator muda, e é a maior parte do trabalho.** Ele deixa de ser a forma do IR e vira um
consumidor como outro qualquer: percorre um bloco em vez de mover cursor, segue terminador em
vez de somar `+1`, indexa valor em fatia em vez de hashear rótulo em mapa, e lê parâmetro em
vez de subir cadeia de `environ`.

Nada disso acrescenta funcionalidade. O que se ganha é que **o evaluator para de ser o único
desenho possível** — hoje toda decisão do IR foi tomada por ele e o outro consumidor paga a
conta.

Duas coisas ficam onde estão: a resolução entre módulos, que é resolução de verdade e não
frontend esquecido; e o contrato com a REPL e o playground, que pedem o valor de uma expressão
de topo por vez e continuam pedindo.

E há um ganho de lado: hoje cada leitura de operando faz `hex.EncodeToString`. Um `OpAdd` custa
três alocações de string — duas leituras e uma escrita. Com valor numerado, some.

---

## As etapas

Cada uma é um PR e nenhuma muda o que um programa responde — é isso que as torna revisáveis:
mesmas respostas, caminho novo.

1. **`Operand` tipado**, ainda dentro do registro de hoje. Mecânica pura.
2. **Registro com operandos em número livre.** `shape`, tape literal e chamada deixam de
   encadear.
3. **Bloco e terminador.** `Program` passa a ser blocos; `PickDeferAtCursor` sai.
4. **Parâmetros de bloco**, para o encontro dos braços de um desvio. `OpPushFeed` sai e os
   valores passam a viajar na própria chamada. `OpGetFeed` fica.
5. **Nome resolvido no emitter.** `OpIdent` e `OpLoad` local saem; o `IdentManager` sai com
   eles. O conflito de nomes vira erro de compilação.
6. **Origem e efeito.** Solta — a origem paga sozinha e imediatamente (o aviso de backend passa
   a ter linha, e o LSP passa a ter o que sublinhar).
7. **A definição de cada opcode**, escrita em `wire/ir`, e um caso no harness diferencial por
   operação definida. Também solta, e é a única que impede a terceira divergência em vez de
   remendá-la depois.
8. **`Verify`.** Depois de todas.

A [if_and_call.md](if_and_call.md) entra depois de 4, e a ambiguidade do `OpReturn` que ela
deixa em aberto some na etapa 3. Um ponto dela fica pendente da decisão sobre `feed`: se o
índice continuar dando a volta, a convenção de frame precisa de um lugar para a contagem de
argumentos; se passar a responder zero fora do alcance, não precisa.

---

## Em aberto

1. **Escopo enxerga onde foi escrito.** É a regra da linguagem, e é a única complicação real
   da etapa 5: um nome de fora não é valor deste bloco.

   A saída comum em linguagem funcional — acrescentar o que foi capturado ao vetor de
   argumentos, o *lambda lifting* — **está fechada aqui**. A razão que vale em qualquer
   hipótese: um `defer` é valor de primeira classe, pode ser ligado e passado adiante, então os
   sítios que o aplicam não são todos conhecidos — e acrescentar argumento exige reescrever
   todos eles. E enquanto `feed(n)` der a volta em `len(vetor)`, está fechada duas vezes:
   acrescentar qualquer coisa ao vetor muda o que **todo** `feed(n)` existente significa.

   Sobra a ligação estática: o bloco carrega um ponteiro para o escopo onde foi escrito, e um
   nome de fora é lido de lá. Fica num canal separado do vetor, que é o que o mantém intacto. A
   `if_and_call.md` já tinha deixado isso fora do beta, e continua certo deixar.
2. **`Effect` com dois valores** hoje. Leitura de estado não é `Pure` nem `Writes`, e a RFC de
   storage é vizinha — vale fixar o conjunto com ela junto.
3. **Origem em toda instrução engorda todo golden.** Vale medir `emitter/testdata` antes de
   decidir se a origem entra no `Format` ou fica fora dele.
4. **`Verify` recusa ou avisa?** Recusar é o certo e quebra todo programa que hoje compila
   torto e roda assim mesmo. Talvez nasça avisando e vire recusa numa versão marcada.
5. **`emitter.Parse`** (`emitter/parser.go`) decodifica um formato serializado que ninguém
   escreve e ninguém chama. Sai antes de tudo isto, para não ser reescrito à toa.
