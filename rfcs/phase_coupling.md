# Fases não se conhecem: o que isso exige do projeto

**Estado:** proposta · **Data:** 2026-08-17

Veio do review do PR #38 — o `builder/evm` importa o `emitter` — e cresceu quando as regras
foram fixadas. Fica registrado aqui o que elas exigem e o que elas encontram no código de hoje.

Tudo abaixo foi medido.

---

## As regras, como ficaram

1. **Pacote não tem interdependência com pacote.** O que ligaria um ao outro se resolve por
   injeção de dependência. **Exceto pacotes auxiliares.**
2. **Pacote vital é puro:** sem I/O, e sem processamento que não devolva nada.
3. **Erro se trata na camada de hosting.** Nada de erro resolvido magicamente dentro de um
   pacote.
4. **I/O só na camada de hosting, e injeção só em arquivos do pacote `main`.** Nenhum
   entrelaçamento entre vitais, nem entre hosting e vital, que não seja por injeção.

E o encadeamento das fases:

```
(Lexer) → {cadeia de tokens} → (Parser) → {árvore abstrata} → (Emitter) → {código intermediário} → (Builder)
```

---

## O que o desenho já diz

Vale reparar no que esse encadeamento nomeia. Entre parênteses estão as **fases**; entre
chaves, os **artefatos**. São coisas de naturezas diferentes, e hoje as chaves não existem
como pacote: a cadeia de tokens mora no `lexer`, a árvore mora no `parser`, o código
intermediário mora no `emitter`.

É daí que vem o acoplamento. Não é o `builder` querendo conhecer o `emitter` — é o artefato
que ele lê estar guardado dentro de quem o produziu. Enquanto o IR morar no emitter, quem lê
IR importa o emitter.

**Injeção sozinha não resolve isso.** Se o Lexer produz uma cadeia de tokens e o Parser a
consome, os dois precisam nomear esse tipo. Injetar o *comportamento* não elimina o tipo do
*dado*. Restam dois caminhos:

- **o artefato vira pacote auxiliar** — e a regra 1 já abre essa exceção, porque um auxiliar
  não conhece nada do projeto;
- **cada fase declara a interface do que consome**, e o `main` converte — o que, para fatias,
  é uma cópia a cada fronteira, e para constantes (tags, opcodes) é uma tabela de tradução no
  `main`, onde um mapa incompleto vira bytecode errado em silêncio em vez de erro de
  compilação.

O primeiro é o que o encadeamento acima descreve. As chaves viram pacotes.

## O que as regras encontram hoje

| Regra | Estado |
|---|---|
| Fase não conhece fase | ✗ `parser→lexer`, `emitter→parser`, `evaluator→emitter`, `builder/evm→emitter` |
| Injeção só no `main` | ✗ quatro lugares montam fases, e **só um é `main`** |
| I/O só no hosting | ✗ o evaluator recebe um `io.Writer` e **escreve durante a avaliação** |
| Erro no hosting | ✓ as fases devolvem erro, não escrevem |

Os quatro lugares que montam fases:

| Onde | Pacote | Monta |
|---|---|---|
| `cmd/playground/main.go` | `main` ✓ | lexer, parser, emitter, evaluator |
| `internal/cli/compile.go` | `cli` ✗ | lexer, parser, emitter |
| `repl/repl.go` | `repl` ✗ | lexer, parser, emitter, evaluator |
| `lsp/textdoc/validator.go` | `textdoc` ✗ | lexer, parser |

E o `builder/evm` nomeia **33 constantes de opcode** do emitter. Não é a forma que o prende —
é o vocabulário.

## A regra 2 encontra o que ninguém tinha olhado

O evaluator guarda um `io.Writer` e os builtins de print escrevem nele **enquanto o programa
roda**. Foi injetado pelo host, o que atende a regra 4, mas não a 2: é I/O dentro de um pacote
vital.

É o mesmo problema que tirou os loggers de dentro das fases, e o roadmap já registra a versão
dele que sobrou — o traço do evaluator só volta se ele **devolver** o traço em vez de escrever.
Com a regra 2 escrita assim, o print cai no mesmo lugar: um evaluator puro devolveria o que o
programa disse, e quem escreve seria o host.

Isso tem um custo que precisa ser dito: hoje o print sai **na hora**, no meio da execução. Um
evaluator que devolve tudo no fim muda o que o usuário vê num programa longo, e num REPL muda
quando a linha aparece.

## Proposta

**Primeiro os artefatos, depois a montagem.**

**Etapa 1 — o código intermediário vira pacote.** `Instruction` e os opcodes saem do `emitter`
para um auxiliar (`ir`, ou o nome que se escolher). É o artefato com mais consumidores —
`builder/evm`, `evaluator`, `internal/cli`, `internal/trace`, `repl` — e o mais barato de
mover: `opcodes.go` tem 37 linhas e `instruction.go` 41, só declaração, sem comportamento.
Depois disso, `emitter → ir`, `evaluator → ir`, `builder/evm → ir`, e nenhuma fase importa
outra fase por causa do IR.

**Etapa 2 — a árvore vira pacote.** Mesmo desenho, custo bem maior: o emitter faz `switch`
sobre vinte e um tipos concretos de nó. Vale fazer depois da etapa 1, com o desenho já visto
de pé.

**Etapa 3 — a cadeia de tokens vira pacote.** `Token` e as tags saem do `lexer`.

**Etapa 4 — a montagem sobe para o `main`.** Cada host declara a interface do que precisa e o
`main` injeta as fases prontas. É a etapa que muda mais assinatura: `internal/cli` deixa de
importar `lexer`, `parser` e `emitter`, e passa a receber o que compila.

**Etapa 5 — o evaluator devolve o que o programa disse**, em vez de escrever. Depois de
decidir o que fazer com a saída na hora certa.

A ordem importa: as etapas 1 a 3 tiram a interdependência sem mexer em assinatura de host. A
4 é a que atravessa o projeto inteiro, e fica muito mais fácil quando os artefatos já têm casa.

## O que já foi resolvido

No próprio #38: o `Warnings()` do builder devolvia `emitter.Warning`, ou seja, o backend
falava de si mesmo no vocabulário da fase anterior. O builder passou a ter o seu, e o host
junta os dois para imprimir.

## O que não é problema

O `main` importar tudo é o desenho. E um pacote auxiliar ser importado por qualquer um é a
exceção da regra 1 — é o que torna os artefatos possíveis.

## Perguntas em aberto

1. **Nome dos pacotes de artefato.** `ir`, `ast`, `token`? Ou um `contract/` com os três
   dentro?
2. **O que viaja junto com o IR?** `Program`, `Expression` e `Label` moram no emitter e
   atravessam para o host. `ResolveOpCode` e `Format` são apresentação — talvez pertençam ao
   `internal/trace`.
3. **`internal/cli` continua existindo?** Ele é host mas não é `main`. Ou ele passa a receber
   tudo injetado, ou o que ele faz sobe para `cmd/aurora`.
4. **O print sai na hora ou no fim?** A regra 2 pede que o evaluator devolva; o usuário hoje vê
   sair no meio da execução.
5. **O LSP e o playground também?** Os dois montam fases; o playground já é `main`, o LSP não.
