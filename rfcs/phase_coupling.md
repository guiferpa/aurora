# Uma fase importa a anterior; a premissa diz que não deveria

**Estado:** proposta · **Data:** 2026-08-17

Veio do review do PR #38: o `builder/evm` importa o `emitter`. Ficou como está naquele PR, e
o problema está registrado aqui.

Tudo abaixo foi medido no código.

---

## As duas leituras da mesma regra

O `CLAUDE.md` diz duas coisas que podem ser lidas como diferentes.

A **premissa 2**: *um pacote vital ou auxiliar não conhece o resto do projeto — a ponto de
alguém pegar uma parte e construir outro compilador.*

A **tabela**, logo abaixo:

| kind | who | may depend on |
|---|---|---|
| phase | `lexer`, `parser`, `emitter`, `evaluator`, `builder/evm` | **earlier phases**, auxiliaries |

Uma proíbe conhecer o resto do projeto; a outra autoriza conhecer as fases anteriores. Hoje o
código segue a tabela:

```
lexer        → (nada)
parser       → lexer
emitter      → lexer, parser
evaluator    → lexer, parser, emitter
builder/evm  → lexer, parser, emitter
```

Isso não é recente nem é de um pacote só: é a forma do projeto desde fevereiro.

## O que está acoplado, exatamente

Duas coisas bem diferentes, que vale separar antes de decidir:

| | O que é | Exemplo |
|---|---|---|
| **A forma** | a interface do dado que atravessa | `[]emitter.Instruction`, `parser.Node`, `[]lexer.Token` |
| **O vocabulário** | o significado, em constantes | `emitter.OpAdd`, `lexer.TagSum`, os tipos concretos de nó |

O `builder/evm` nomeia **33 constantes de opcode** do emitter. Não é a forma que o prende ao
emitter — é o significado.

E o IR não tem um consumidor, tem cinco: `builder/evm`, `evaluator`, `internal/cli`,
`internal/trace` e `repl`. Ele já se comporta como contrato público; só não mora num lugar
que diga isso.

## As opções

**1. Manter, e escrever que o IR é o contrato.** Custo zero. O compilador continua acusando
quando um opcode muda de número. Perde-se a premissa como está escrita: ninguém pega o
`builder/evm` sozinho, porque leva o `emitter` junto.

**2. Injetar só a forma.** Cada fase declara a interface que consome; o host converte as
fatias, já que Go não converte `[]emitter.Instruction` em `[]evm.Instruction`. Parece
progresso e não é: as 33 constantes continuam vindo do emitter. É a opção que dá trabalho e
não entrega a premissa.

**3. Injetar forma e significado.** O host passa a tabela que liga o vocabulário do IR às
operações do backend:

```go
evm.NewBuilder(insts, evm.Options{
    Writers: map[byte]evm.WriteFunc{emitter.OpAdd: evm.WriteAdd, ...},
})
```

Entrega a premissa: o builder deixa de nomear o emitter. Custa caro — a tabela migra para o
host, e o `Lowering` também classifica por opcode (`IsOperand`, `IsBinaryValueConsumer`), então
precisa do mesmo tratamento. E tem um preço que não é de linhas: **hoje um opcode renomeado é
erro de compilação; com tabela injetada vira mapa incompleto e bytecode errado em silêncio.**

**4. Tirar o contrato de dentro do emitter.** `Instruction` e os opcodes viram um pacote
auxiliar — `ir`, ou o nome que se escolher — que não conhece nada do projeto. Aí:

```
emitter      → ir
evaluator    → ir
builder/evm  → ir
```

Nenhuma fase importa outra fase, sem cerimônia de injeção e sem perder a checagem em tempo de
compilação. É o desenho que a biblioteca padrão do Go usa: `go/token` e `go/ast` existem
separados de `go/parser` justamente porque são o contrato, não a fase.

O tamanho: `emitter/opcodes.go` tem 37 linhas e `emitter/instruction.go` tem 41 — declarações,
sem comportamento. Mover é barato; o que custa é atualizar os cinco consumidores e decidir o
que mais vai junto.

## Recomendação

**Opção 4, começando pelo IR.** É a que torna a premissa verdadeira em vez de negociada, e a
única que faz isso sem trocar erro de compilação por erro de silêncio.

A AST (`parser.Node` e os tipos de nó que o emitter percorre) tem o mesmo problema e é bem
maior — o emitter faz `switch` sobre vinte e um tipos concretos. Vale decidir depois, com o IR
já fora, para ver se o mesmo desenho se sustenta.

## O que já foi resolvido

No próprio #38, um acoplamento que não era desse tipo: o `Warnings()` do builder devolvia
`emitter.Warning`, ou seja, o backend falava de si mesmo no vocabulário da fase anterior. O
builder passou a ter o seu próprio, e o host — que pode depender de tudo — é quem junta os
dois para imprimir.

## O que não é problema

O host importar todas as fases é o desenho, não um desvio: `cmd/*`, `repl`, `internal/cli` e
`lsp` existem para montar as peças, e nada depende deles.

## Perguntas em aberto

1. **Nome do pacote.** `ir`, `bytecode`, `instruction`?
2. **O que mais vai junto?** `Program`, `Expression` e `Label` moram no emitter e atravessam
   para o host. `ResolveOpCode` e `Format` são apresentação — talvez fiquem no `internal/trace`.
3. **Vale para a AST?** Mesmo problema, custo muito maior.
4. **Fica escrito onde?** Se a resposta for a opção 1, a premissa 2 precisa ser reescrita para
   dizer o que a tabela já diz. Se for a 4, a tabela é que muda: fase depende de auxiliares e
   de contratos, nunca de outra fase.
