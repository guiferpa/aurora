# O IR diz o que uma instrução faz além de deixar um valor

**Estado:** proposta · **Data:** 2026-08-26

Duas coisas travam no mesmo lugar: um shape de cinco campos não chega ao bytecode, e storage
não pode ser escrito. Esta RFC diz por que são a mesma coisa, e em que ordem sair delas.

O vocabulário está em [docs/contributing/architecture.md](../docs/contributing/architecture.md),
seção **The words**. O IR está em [docs/contributing/ir.md](../docs/contributing/ir.md), e
[why-blocks.md](../docs/contributing/why-blocks.md) ao lado diz por que ele tem essa forma.

---

## O buraco, que existe e ninguém caiu nele ainda

O builder **reordena instruções**. `ResolveOperandsOrder` (`builder/evm/lowering.go`) segura
uma instrução e a solta ao lado de quem consome o valor dela, porque a pilha da EVM quer os
operandos na ordem certa e o programa foi escrito em outra.

Isso é correto para uma instrução cujo resultado inteiro é o valor que ela deixa. Não é
correto para uma que **faz** alguma coisa:

```
printd a;
printd b;
```

Mover a segunda para antes da primeira muda o que o programa diz. Nada no IR impede.

O que segura hoje são três listas de opcodes em `builder/evm` — `passesThrough`,
`ordersItsOwn` e `produces` — e uma lista de opcodes é **a forma de errar deste
repositório**. Foi assim que um `ident` dentro de escopo quebrou: a lowering decidia quais
operandos nomeavam valores por uma lista dos quatro aritméticos, e um nome ligado dentro de
um escopo não estava nela. `ident x = feed(0); x + feed(1);` respondia 4 na chain onde o
programa responde 7. Quem escreve o opcode seguinte tem que lembrar de três listas, e não há
nada que o lembre.

**E ninguém caiu no buraco ainda**, por um acidente que vale dizer em voz alta: a única
instrução cuja ordem importa hoje é o `print`, e o `print` é justamente a que o backend
recusa a escrever — por decisão, porque uma chain não tem onde pôr um log. Então o backend
que reordena nunca viu uma instrução que não podia mover.

Storage põe uma lá. `s.set("key", v)` seguido de `s.get("key")` é a primeira dupla em que a
ordem é o programa inteiro, e ela chega ao bytecode. **Por isso o `Effect` vem antes do
storage, e não depois.**

---

## O `Effect`

Já foi desenhado na [ir.md](ir.md) com dois valores, e a questão em aberto número 2 de lá
dizia que leitura de estado não é nenhum dos dois e que valia fixar o conjunto com storage
junto. É o que esta faz.

```go
// Effect diz o que uma instrução faz além de deixar um valor.
type Effect byte

const (
	Pure    Effect = iota // o valor que ela deixa é tudo que ela faz
	Reads                 // depende de estado que outra coisa pode mudar
	Writes                // muda esse estado, ou diz algo cuja ordem importa
	Escapes               // sai do programa: outro contrato roda, e pode voltar para dentro
)
```

**A regra é uma linha:** duas instruções só trocam de lugar se pelo menos uma for `Pure`.

Uma escada de quatro degraus com uma regra que só pergunta "é `Pure`?" parece três degraus
a mais do que a regra precisa, e é — de propósito, e o preço está medido:

- A regra é conservadora e custa **nada hoje**. O builder só move instrução para pôr operando
  na ordem da pilha; ele nunca teve motivo para trocar duas leituras de lugar. Uma regra mais
  fina é uma matriz de quatro por quatro que um mantenedor tem que segurar na cabeça, e a
  compra quando alguém medir que ela paga.
- Os outros três degraus não são para a regra, são para **o que um backend tem que dizer**.
  Um `Escapes` é o que faz o backend precisar de reentrância; um `Reads` é o que o evaluator
  tem que simular para a promessa se manter. Um backend que não carrega um efeito recusa
  nomeando ele, do jeito que `builder/evm/support.go` já recusa nomeando um opcode.
- Fixar os quatro agora é o que evita renomear `Ordered` para `Writes` quando o storage
  chegar. É o custo que a ir.md nomeou.

`require` **não entra**. Ele não é instrução: é controle, e controle já mora fora das
instruções desde que o bloco tem terminador. Um `require` é um `BrIf` para um bloco que
reverte, e é por isso que ele não precisa de efeito nenhum — o mesmo argumento que a ir.md
usou para tirar o desvio de dentro da instrução.

---

## O shape de cinco campos

Off-chain um run é uma fatia de bytes de qualquer tamanho (`EvaluateJoinOver`). On-chain é
**uma palavra da pilha** (`WriteJoin`), e uma palavra tem 32 bytes. No tape padrão de oito
isso é quatro campos, e o quinto é recusado:

```
a run of 5 tapes is 40 bytes and a word is 32: a shape this wide does not reach the bytecode
```

Recusar é melhor do que escrever curto, e ainda assim é um programa que roda no evaluator e
não pode ir para uma chain — que é exatamente a promessa que o projeto existe para manter.

**Salvar o run contíguo na memória resolve**, e a pergunta difícil não é essa: é o que um
valor passa a ser. Se um run vive na memória, o valor que o nomeia é um endereço, e a
linguagem não tem tipo — nada na palavra da pilha distingue um tape de oito bytes de um
endereço de memória.

Marcar o valor com um bit custa uma máscara em toda aritmética, e o custo é em todo programa
para pagar o caso de cinco campos. **A alternativa é decidir estaticamente**, e ela ficou
barata agora: o parser já sabe qual valor é um run — é o `shapeOf`, e desde a
`shape_inference` ele sabe em mais lugares do que sabia — um bloco, um `if` cujos braços
concordam, uma chamada a um escopo que não declarou nada, e tudo isso atravessando módulo. O que falta é o IR **dizer**: um operando que é um run
carrega quantos tapes ele tem, e o backend lê em vez de adivinhar — que é a mesma frase que
resolveu o `ident` dentro de escopo.

Um run na memória é escrita de memória, e é aqui que as duas metades desta RFC se encontram:
`OpJoin` deixa de ser `Pure` no momento em que passa a escrever, e sem o `Effect` o builder
tem todo o direito de movê-lo.

---

## O que isso destrava

Storage como um mapa nativo, do jeito que foi pensado:

```
use storage as s;

s.set("key", value);
s.get("key");
```

`set` é `Writes`, `get` é `Reads`, e a regra de uma linha já basta para o backend nunca
inverter os dois. O evaluator simula com um mapa em memória, e a promessa se mantém: o mesmo
programa responde a mesma coisa dentro e fora da chain.

Isso é a RFC seguinte, não esta.

---

## Entrega

1. **`Effect` na instrução, com todo mundo `Pure` menos os três `print`.** Puramente aditivo:
   nada muda de comportamento porque nada que não é `Pure` chega ao backend hoje. O que muda é
   que a lowering passa a perguntar o efeito em vez de consultar lista, e as três listas de
   opcodes de `builder/evm` encolhem para o que é de fato sobre a pilha.
2. **A regra, medida antes de prender.** Um teste que roda a regra sobre o que a lowering
   produz hoje, para todo programa do corpus, e diz o que ela recusaria. Ele é o que decide se
   o item 1 foi aditivo, e continua valendo depois como o teste que prova que a regra é
   cumprida — sem ele, o item 1 é uma anotação em que ninguém confia.
3. **O IR diz quantos tapes um operando tem**, quando ele é um run.
4. **O run vai para a memória**, e o teto de 32 bytes sai — com o harness diferencial provando
   um shape de cinco campos e um de oito.

Um a um, cada um passando `make check` sozinho.

---

## O frame é estado

A pergunta foi feita com um dos nomes errados, e a resposta corrige junto. `OpSave` não mexe
no frame: ele é `Imm -> the bytes themselves`, que é como um literal chega ao programa, e isso
é `Pure`. Quem escreve no frame é o `OpIdent` — `Name, Ref -> binds the name to the value`.

Então, decidido:

| opcode | efeito | porquê |
|---|---|---|
| `OpIdent` | `Writes` | escreve o frame com um `MSTORE` |
| `OpLoad` | `Reads` | lê o frame com um `MLOAD` |
| `OpSave` | `Pure` | é um literal, e não toca em nada |

**E isso pode custar alguma coisa, ao contrário dos `print`.** Um `print` nunca chega ao
backend que reordena, então prendê-lo é grátis. Um `OpIdent` e um `OpLoad` chegam, e os dois
são movidos hoje: o `OpLoad` porque `produces(op)` o segura para soltar ao lado de quem
consome, e o `OpIdent` porque a lowering o segura quando ele é a última expressão de um
escopo.

A regra de uma linha proíbe trocar um `Reads` com um `Writes`. Se a lowering hoje faz isso em
algum programa, o item 1 da entrega **deixa de ser aditivo** — e essa é a única parte desta
RFC que não dá para decidir lendo o código, porque depende de qual ordem cada programa real
produz.

Por isso a entrega é nesta ordem, e não em outra: o item 2 existe para **medir antes de
prender**. Ele anota os efeitos, roda a regra sobre o corpus de testes que já existe, e diz o
que ela recusaria. Se a resposta for "nada", prender é aditivo e o item 1 fecha. Se for
alguma coisa, o que ela recusa é um lugar onde a lowering move o que não podia mover — o
buraco desta RFC, com um programa em cima —, e aí a correção é da lowering, não da regra.

---

## Em aberto

1. **Onde na memória.** Um frame já tem `FRAME_POINTER`, `RETURN_SCRATCH` e `FRAME_BASE`, e um
   run de um escopo que já voltou não pode ser lido depois — o mesmo problema de tempo de vida
   que qualquer alocação tem. A resposta mais barata é o run viver no frame de quem o
   construiu, e um run retornado ser copiado para o frame de quem chamou, que é o que a
   convenção de chamada da [if_and_call.md](if_and_call.md) já faz com valores.
2. **O que um run que atravessa a chain é.** Hoje um shape retornado é uma palavra e o teste
   compara número com número. Passando a ser bytes, o `RETURN_TO_CHAIN` muda, e o harness
   junto.
3. ~~Se `OpLoad` e `OpSave` viram `Reads`/`Writes`.~~ **Decidido: o frame é estado, e a regra
   os prende também.** Ver abaixo.
