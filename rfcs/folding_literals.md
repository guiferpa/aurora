# Dobrar o que já está escrito

**Estado:** proposta · **Data:** 2026-08-22

Um terço do que o emitter produz existe para dar nome a um número que já era conhecido.

Esta RFC diz o que é dobrar um literal, por que não é otimização, e o que ela exige que ainda
não está de pé. Ela foi escrita **depois de a mudança ser implementada e revertida** — o que
está aqui é medido, e o que ela custa foi visto acontecer.

---

## O tamanho, medido

No `emitter/testdata/wide.ir`, que é saída real para um programa que usa quase tudo da
linguagem:

| | |
|---|---|
| instruções no total | **59** |
| que são `OpSave` | **20** — 34% |
| desses, lidos exatamente uma vez | **19** |

Um terço do IR são instruções cujo trabalho inteiro é pôr um valor conhecido sob um rótulo,
para que a instrução seguinte o cite.

Concreto, `ident t = [1, 2, 3];`:

```
0x303137 OpSave imm 0x0000000000000000 -
0x303138 OpSave imm 0x0000000000000001 -
0x303139 OpSave imm 0x0000000000000002 -
0x303230 OpSave imm 0x0000000000000003 -
0x303231 OpPull ref 0x303137 ref 0x303138 ref 0x303139 ref 0x303230
```

Dobrado:

```
0x303231 OpPull imm 0x00 imm 0x01 imm 0x02 imm 0x03
```

## Por que não é otimização

A [ir.md](ir.md) é explícita em não propor otimização, e esta RFC não a contradiz.

`OpSave imm 0x0A` afirma: *"compute o valor dez e ponha sob este rótulo."* **Não há nada a
computar.** O dez já era conhecido quando o parser leu o caractere. A instrução descreve uma
computação que não existe, e o IR existe para descrever o programa.

Menos instruções é consequência, não o objetivo. Se o objetivo fosse velocidade, o lugar de
discutir seria outro.

E há um ganho estrutural que interessa mais: **o escalonador de pilha tem menos o que mover.**
Hoje `OpSave` vira um `PUSH`, e o lowering precisa afundar esse push até ficar colado em quem
o consome. Com o valor sendo operando, o push acontece onde é preciso e não há o que
escalonar — o que tira trabalho da fase que já quebrou duas vezes.

## Como fica

Um lugar só decide, no emitter:

```go
// operandFor answers the operand a node is worth, emitting for it only when there is
// something to emit.
func operandFor(tc *int, insts *[]ir.Instruction, node ast.Node, tapeSize int) ir.Operand {
	switch n := node.(type) {
	case ast.NumberLiteral:
		return ir.Imm(n.Value, tapeSize)
	case ast.TextLiteral:
		return ir.ImmOf(n.Value, tapeSize)
	case ast.BooleanLiteral:
		return ir.ImmOf(n.Value, tapeSize)
	}
	return ir.RefTo(EmitInstruction(tc, insts, node, tapeSize))
}
```

Isso é reconhecer tipo de nó, e é legítimo **no emitter**: ele tem a AST na mão e o trabalho
dele é esse. É diferente de um backend reconhecendo padrão numa lista de instruções.

---

## O que já está de pé

**Os dois kinds separados.** Um `Imm` é valor do programa e um `Const` é número que a operação
toma sobre si. Sem essa separação, dobrar tornaria `imm` ambíguo — o mesmo kind significando
"um valor" numa aritmética e "um índice" num `head`.

**A leitura de operando num lugar só.** `e.value` responde o que um operando vale: `Ref` se
procura, `Imm` é o valor. As 26 funções `Evaluate*` recebem operandos em vez de bytes.

Com esses dois, o evaluator já aceita um `Imm` em posição de valor hoje.

## O que falta, e é o motivo desta RFC existir

**O backend conta com `OpSave` virar um `PUSH`.** Sem `OpSave`, ninguém empilha. A tentativa
derrubou **114 casos em seis pacotes**, e o diff de bytecode é direto:

```
esperado: 6700000000FFFFFFFF 6700000000FFFFFFFF 01 ...
obtido:                                         01 ...
```

Os dois `PUSH` sumiram e o `ADD` soma o que não está lá.

Materializar um `Imm` no backend não é escrever um `PUSH` — é decidir **onde** escrevê-lo, e
isso é escalonamento de pilha. Para uma soma tanto faz; para uma subtração com o imediato à
direita seria preciso `PUSH B; SWAP1; SUB`, porque o produtor do outro operando já rodou.

Que é exatamente o que `docs/compiler_pipeline_and_lowering.md` diz não ser do Builder:

> *"Se o Builder começa a reorganizar operandos ou a decidir quando usar SWAP, ele está fazendo
> lowering escondido."*

**O pré-requisito real é o lowering que sabe o que cada instrução toma e deixa** — o branch
`feat/a-stack-model-in-the-lowering`, que declara `consumes`/`produces` por opcode e nunca foi
mergeado. Com ele, um operando `Imm` é mais um caso do mesmo cálculo, e não uma exceção
escrita à mão no writer.

**E há mais que o backend.** A tentativa quebrou também a REPL e o `hosting/cli`, e eu não
rastreei o porquê antes de reverter. Antes de tentar de novo, isso precisa ser entendido —
provavelmente uma expressão de topo que é só um literal deixa de ter rótulo, e `ir.Expression`
reporta valor por rótulo.

---

## A ordem

1. **Mergear `feat/a-stack-model-in-the-lowering`**, que é pré-requisito e está verde há tempo.
2. **Entender o que quebrou fora do backend**, começando pela expressão de topo sem rótulo.
3. **Dobrar**, com o `operandFor` acima.

O passo 1 não é desta RFC e não deve ser tratado como detalhe dela: aquele branch mexe no
lowering inteiro e merece a sua própria leitura.

## Em aberto

1. **Uma expressão de topo que é só um literal.** `10;` no REPL precisa de um rótulo para ser
   reportada. Ou `OpSave` sobrevive para esse caso, ou `ir.Expression` passa a poder carregar
   um valor em vez de um rótulo. A segunda é mais limpa e mexe no contrato da REPL.
2. **Dobrar só onde nenhum backend lê** — argumentos de chamada, itens de tape, campos de
   shape — foi considerado como meio-termo e recusado: funciona, e cria uma regra a lembrar
   sobre quais posições podem ser dobradas. Regra a lembrar é a doença.
3. **Se o `head`/`tail` deveriam dobrar também.** O comprimento deles já é `Const`, mas o tape
   sobre o qual operam pode ser literal — `head [1,2,3] 2`. Cabe, e não foi medido.
