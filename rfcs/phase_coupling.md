# Fases não se conhecem: o caminho até lá

**Estado:** aceita · **Data:** 2026-08-17

Veio do review do PR #38 — o `builder/evm` importa o `emitter` — e virou a arquitetura do
projeto. A taxonomia está decidida e escrita no [CLAUDE.md](../CLAUDE.md) e no agente de
padrões; o que fica aqui é **o caminho da árvore de hoje até ela**, com o custo de cada etapa.

Tudo abaixo foi medido.

---

## O desenho é o argumento

```
(lexer) → {tokens} → (parser) → {árvore} → (emitter) → {IR} → (builder) │ (evaluator)
```

Entre parênteses estão as **fases**; entre chaves, os **artefatos**. E nenhum artefato tem
pacote próprio: a cadeia de tokens mora no `lexer`, a árvore no `parser`, o IR no `emitter`.

É daí que vem o acoplamento. Não é o builder querendo conhecer o emitter — é **o artefato
guardado dentro de quem o produziu**. Enquanto o IR morar no emitter, quem lê IR importa o
emitter.

**Injeção sozinha não alcança isso.** Se o Lexer produz uma cadeia de tokens e o Parser a
consome, os dois precisam nomear esse tipo; injetar o *comportamento* não elimina o tipo do
*dado*. Por isso os artefatos viram uma categoria própria — **wire** — que não conhece nada do
projeto e pode ser conhecida por todos. As chaves do desenho viram pacotes.

## O que as regras encontram hoje

| Regra | Estado |
|---|---|
| Nenhum pacote conhece outro | ✗ `parser→lexer`, `emitter→parser`, `evaluator→emitter`, `builder/evm→emitter` |
| Injeção só no `main` | ✗ quatro lugares montam fases, e **só um é `main`** |
| I/O só em hosting | ✗ o evaluator escreve durante a avaliação; `logger` escreve e chama `os.Exit(2)` |
| Erro em hosting | ✗ o `logger` encerra o processo por conta própria |

Quem monta fases hoje:

| Onde | Pacote | Monta |
|---|---|---|
| `cmd/playground/main.go` | `main` ✓ | lexer, parser, emitter, evaluator |
| `internal/cli/compile.go` | `cli` ✗ | lexer, parser, emitter |
| `repl/repl.go` | `repl` ✗ | lexer, parser, emitter, evaluator |
| `lsp/textdoc/validator.go` | `textdoc` ✗ | lexer, parser |

E o `builder/evm` nomeia **33 constantes de opcode** do emitter. Não é a forma que o prende —
é o vocabulário.

## As etapas

Artefatos primeiro, montagem por último. As três primeiras tiram a interdependência **sem
tocar em assinatura de host**, o que faz a quarta — que atravessa o projeto — ficar bem menor.

**1. O IR vira `wire`.** `Instruction` e os opcodes saem do `emitter`. É o artefato com mais
consumidores — `builder/evm`, `evaluator`, `internal/cli`, `internal/trace`, `repl` — e o mais
barato de mover: 37 + 41 linhas de declaração, sem comportamento. Depois disso, `emitter`,
`evaluator` e `builder/evm` dependem do mesmo pacote e de nenhum outro.

**2. A cadeia de tokens vira `wire`.** `Token`, as tags e o erro posicionado saem do `lexer`.

> A ordem aqui era a inversa, e mudou ao aplicar: **a árvore guarda tokens** — cada nó tem um
> campo `Token`. Mover a árvore primeiro faria `wire/ast` importar o `lexer`, que é vital.
> Também não é uma mudança de arquivo inteiro: `token.go` guarda o contrato **e** a leitura, e
> `tag.go` guarda o vocabulário **e** o casamento de token, então os dois foram cortados ao
> meio.

**3. A árvore vira `wire`.** Mesmo desenho, custo maior: o emitter faz `switch` sobre vinte e
um tipos concretos de nó. A comparação de árvores vai junto — é pergunta sobre a forma, não
sobre quem a construiu.

**4. `logger` vira util de formatação.** Hoje ele escreve em `os.Stderr` e chama `os.Exit(2)`
— um pacote decidindo o fim do processo. Passa a devolver a mensagem formatada; cada hosting
escreve, e o `os.Exit` sobe para `cmd/*`. É a etapa que conserta a regra 3.

> Ao aplicar apareceu uma segunda coisa: **o código de saída 3 era inatingível.** Ele saía do
> `AssertError`, chamado pelo `aurora run` — e `run` cria o evaluator sem `Asserts`, que só o
> `aurora test` liga. Nenhuma asserção jamais falhou por ali. A função saiu junto com o
> caminho morto, em vez de mudar de casa.

**5. `fileutil`, `manifest` e `internal/trace` viram `shared`.** Eles servem a camada de
hosting, não uma interação. Util não toca no mundo; esses três tocam.

> Onde eles moram passa a dizer o que eles são: `shared/fileutil`, `shared/manifest`,
> `shared/trace` — pelo mesmo motivo que `wire/` é uma pasta. `internal/` responde outra
> pergunta (quem pode importar de fora do módulo), não a de que tipo o pacote é.

**6. A montagem sobe para o `main`.** Cada host declara a interface do que precisa e o `main`
injeta as fases prontas. É a etapa que muda mais assinatura: `internal/cli` deixa de importar
`lexer`, `parser` e `emitter`.

**7. O evaluator devolve o que o programa disse.** Os builtins de print param de escrever. Duas
coisas a decidir junto, e nenhuma é detalhe:

- a saída tem que voltar **mesmo quando a avaliação falha** — um programa que imprime três
  linhas e depois estoura não pode perder as três;
- o usuário deixa de ver a saída **na hora**. Num programa longo, e principalmente no REPL,
  muda quando a linha aparece.

## Achado ao mover a árvore

**`Next()` não é chamado em lugar nenhum.** É o único método da interface `Node`, implementado
vinte e cinco vezes, e nada no projeto o usa: a interface existe para marcar um tipo como nó, e
faz isso através de um método que ninguém chama.

Trocar por um método não exportado (`node()`) resolveria duas coisas de uma vez: some com umas
setenta e cinco linhas mortas, e **fecha o conjunto** — ninguém de fora do `wire/ast` passa a
poder inventar um nó, o que é exatamente o que a linguagem quer, já que o emitter faz `switch`
sobre a lista inteira.

Fica registrado em vez de feito: é mudança de desenho, não de lugar, e o passo era mover.

## Consequência para o que já foi feito

No #38 o `Warnings()` do builder deixou de devolver `emitter.Warning` e ganhou um tipo próprio,
com o host convertendo os dois. **Com `wire`, isso encolhe**: aviso é dado que atravessa da
fase para o host, ou seja, é wire. Um `wire.Warning` só, e nenhuma conversão. O conserto do #38
continua certo sob a regra que existia na época; a etapa 1 o simplifica.

## O que não é problema

O `main` importar tudo é o desenho. E wire e util serem importados por qualquer um é a exceção
que torna os artefatos possíveis — sem ela, nada disso fecha.

## Perguntas em aberto

1. **Nome dos pacotes.** `wire/ir`, `wire/ast`, `wire/token` — um pacote por artefato, para
   que o builder não importe a árvore junto com o IR.
2. **O que viaja junto com o IR?** `Program`, `Expression` e `Label` moram no emitter e
   atravessam para o host: são wire. `Format` desenha uma instrução para humano — é
   apresentação, e vai para `internal/trace`. `ResolveOpCode` dá nome a um opcode: dá para
   argumentar que o nome é parte do vocabulário e fica no wire.
3. **`internal/cli` continua existindo?** Ele é hosting mas não é `main`. Ou recebe tudo
   injetado, ou o que ele faz sobe para `cmd/aurora`.
4. **O print sai na hora ou no fim?** A regra pede que o evaluator devolva; o usuário hoje vê
   sair no meio da execução.
