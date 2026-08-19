# Módulos: resolver, loader e a ligação que faltou

**Estado:** proposta · **Data:** 2026-08-19

O desenho já existe em inglês — [docs/module_system_design.md](../docs/module_system_design.md)
— e decidiu a **forma**: um arquivo é um módulo, todo import tem alias, e os três trabalhos
(resolver, loader e a ligação de nomes) são coisas separadas. O que faltava era **como
construir**, e é isso que esta RFC decide.

Ela também muda duas coisas daquele documento: **a sintaxe do specifier** e **onde a ligação
acontece**. As duas estão marcadas onde aparecem.

Tudo abaixo foi medido no código de hoje, não suposto.

> Os blocos de código aqui **não** estão marcados como `aurora` de propósito:
> `hosting/cli/docs_test.go` executa todo bloco assim marcado no repositório inteiro, e nada
> disto compila ainda.

---

## A premissa

Aurora roda o arquivo inteiro, como Python e JavaScript: o corpo de um módulo executa **uma
vez, na primeira importação, em ordem de dependência**. Isso pede duas coisas — ordem
topológica e cache por identidade — e **não** pede um environ por módulo, que é a confusão
fácil e a única parte cara do assunto.

Uma consequência que se aceita de olhos abertos: um módulo com `printb` no topo **imprime ao
ser importado**. Python resolve com `if __name__ == "__main__"`, que Aurora não tem e não
ganha aqui. Print é log; que ele apareça na carga é honesto.

---

## A sintaxe

```
use a/b/c as x;

printb x.add(1, 2);
```

- **Todo import tem alias**, e o alias é o único jeito de alcançar o que está dentro. Não há
  forma que traga um nome solto para o escopo — lendo `x.add(1, 2)` você sabe onde `add` mora
  sem rolar a tela.
- O specifier é um caminho **a partir da raiz de módulos**, que é `src/` por padrão e se muda
  no manifesto. **Não existe forma relativa**: `./x` e `../x` não são caminhos válidos.
- O `.ar` é implícito. Um specifier que aponta para um diretório é erro — um módulo é um
  arquivo.
- Os segmentos são colados: `a / b` não é caminho, é divisão em lugar errado.

> **Muda o desenho em inglês.** Lá o specifier era um texto entre aspas, com `./math`
> relativo e `math` reservado para a raiz. Some a distinção: existe uma raiz só, e todo módulo
> tem um nome canônico a partir dela — o modelo do Clojure e do Go, não o do TypeScript.

**O que essa escolha compra**, e é mais do que parece:

1. **O specifier é a identidade do módulo.** `a/b/c` não depende de quem importou, nem do
   diretório de onde o comando foi rodado, nem de symlink. Cache e detecção de ciclo passam a
   ser sobre uma string canônica em vez de sobre um caminho de arquivo.
2. **O parser não precisa saber a raiz do projeto** para escrever o nome ligado (adiante), o
   que mantém o parse puro e sem entrada nova além do módulo em que ele está.
3. Some a canonicalização de caminho, que era metade do trabalho chato do resolver.

### Onde a raiz é declarada

| | |
|---|---|
| Padrão | `src/`, relativo à raiz do projeto (o diretório do `aurora.toml`) |
| Mudança | `[project] source_root = "lib"` |
| Sem manifesto | a raiz é o diretório do arquivo que foi mandado rodar |

Vai em `[project]` pelo mesmo motivo que `tape_size` foi para lá: **não é um caminho de uma
tarefa, é o que decide o que o código-fonte significa.** Dois profiles com raízes diferentes
fariam `use a/b/c` apontar para dois arquivos, e qualquer coisa que leia o projeto inteiro —
o language server acima de todos — não teria o que responder.

Não confundir com `profiles.<nome>.source`, que continua sendo o caminho do arquivo de
entrada. Um diz **onde os nomes de módulo resolvem**; o outro, **qual arquivo rodar**.

### O que o lexer precisa

Quase nada. `a/b/c` já é lexado como `ID DIV ID DIV ID` — `/` é um caractere só
(`lexer/scanner.go`, `scanOneChar`) e não cabe num identificador (`isIdentChar`,
`lexer/scanner.go:68`, aceita letra, dígito, `_ - ? ! > <`). O parser remonta os segmentos, e
não há ambiguidade porque só se lê caminho depois de `use`.

O que muda: **`use` vira palavra-chave**, e hoje é um identificador comum
(`lexer/scanner_test.go:72` afirma isso). É uma quebra, pequena e declarável.

---

## O que não vamos fazer: um environ por módulo

Essa é a parte que precisa estar escrita antes de qualquer decisão, porque a falha é
silenciosa.

O valor de um `defer` é **um índice, contado no environ onde ele foi criado**
(`evaluator/evaluator.go:448`, `index := uint64(e.environ.DefersLength())`). Na chamada,
`EvaluateCall` (`evaluator/evaluator.go:456`) lê o ident, extrai o índice e faz
`GetDefer(deferKey(index))` **subindo a cadeia do chamador**.

Com `math` e `main` em environs próprios, `main` chamando `m.area`:

1. `GetIdent` encontra o valor — índice `0`, contado no environ de `math`;
2. `GetDefer(deferKey(0))` sobe a cadeia de **`main`** e acha o `defer 0` **de `main`**;
3. executa o corpo errado, sem erro e sem aviso.

Para consertar, o valor do `defer` teria que carregar identidade de módulo — e ele é **uma
tape** (`evaluator/evaluator.go:450`). Com `tape_size = 1` sobram oito bits para repartir
entre módulo e índice. É o mesmo formato do teto de `PUSH1` no backend: não é bug com patch
atrás, é desenho.

O caminho do meio existe: alocar todo `defer` no environ raiz (uma linha em `EvaluateDefer`)
devolve a unicidade dos índices e libera os *idents* por módulo. Sobra a metade difícil — em
runtime, `x.add` precisa saber **qual** environ consultar, e isso é identidade de módulo no
operando do IR, o passo que o desenho em inglês chama de "mais honesto" e que só se paga
quando o backend precisar dele.

**Clojure, que é o modelo de que gostamos, não faz isso.** Namespace lá é coisa de
read/compile time: `(:require [math :as m])` é um mapa de aliases *daquele arquivo*, `m/add`
resolve na leitura para a var `#'math/add`, e vars são globais. Não há escopo aninhado de
namespace em runtime. Quem tem environ por módulo é o **Python**, onde o módulo é um objeto
com `__dict__` e `m.add` é busca de atributo na hora.

**Decisão: namespace em tempo de compilação, um environ em runtime.** O evaluator não muda
uma linha.

---

## O que o código já dá de graça

| Fato | Onde | Por que importa |
|---|---|---|
| Símbolo no IR é byte cru | `emitter/emitter.go:96`, `:345`, `:431` | nome com prefixo entra sem tocar em IR, environ, evaluator ou writer EVM |
| Environ indexa por `hex(bytes)` | `evaluator/evaluator.go:364`, `:411`, `:457` | idem |
| `.`, `/` e `:` não cabem num identificador | `lexer/scanner.go:68` | um nome com prefixo nunca colide com algo escrevível |
| Conflito de ident só dentro do mesmo environ | `evaluator/evaluator.go:412` | nomes únicos bastam para o environ achatado funcionar |
| `.` já é ligado em tempo de parse, contra tabela injetada | `parser/structs.go:167` | o acessor de módulo é o mesmo mecanismo, não um novo |
| Um loader de dois módulos já existe | `hosting/cli/test.go:177` | concatenar streams e avaliar por faixas está provado |

É por isso que este trabalho é menor do que parece: **a metade de baixo do compilador não
muda.**

---

## O desenho

Três trabalhos, e o terceiro é o que faltou da última vez.

| Trabalho | Pergunta | Entrada → saída |
|---|---|---|
| **Resolver** | Que arquivos são esses, e em que ordem? | specifier de entrada → módulos em ordem topológica, com AST |
| **Loader** | Como isso vira um programa? | módulos em ordem → um stream de instruções e suas faixas |
| **Ligação** | A que declaração este nome se refere? | `x.add` → o `add` do módulo `a/b/c` |

### A descoberta que barateia tudo

A ligação tem duas metades, e elas precisam de coisas diferentes:

- **Traduzir** `x.add` para `a/b/c.add` precisa só do que está no próprio arquivo: a linha
  `use a/b/c as x;` acima. Nenhum conhecimento de outro módulo.
- **Conferir** que `add` existe mesmo em `a/b/c` precisa da tabela do outro módulo.

Separando as duas, o **parse continua independente por arquivo** — cada um pode ser lexado e
parseado sozinho, em qualquer ordem, que é o que o language server já faz — e a conferência
mora onde o conhecimento cruzado está.

> **Muda o desenho em inglês.** Lá a ligação era ou mangling no emitter, ou um operando de
> módulo no IR. É mangling, sim, mas escrito no **parse**, e sem passe de reescrita de
> árvore: o parser já visita cada nó, então ele **anota as referências qualificadas que viu**
> (`{módulo, símbolo, token}`) e o loader confere a lista contra as tabelas. Um passe de
> reescrita teria ~25 casos de `switch` para escrever e manter, com exatamente o modo de falha
> silenciosa que `EmitInstruction` já tem: nó não tratado passa sem ligar.

### A regra do prefixo

**Dentro de um módulo importado, todo identificador — ligação ou menção — é escrito
`a/b/c.nome`.** Uma referência qualificada `x.add` é escrita com o prefixo do módulo
importado, que o alias fornece. Como todas as menções dentro de um arquivo levam o mesmo
prefixo, isso é uma renomeação constante do espaço de nomes daquele arquivo: não exige que o
parser saiba o que é topo e o que é escopo interno, e não exige análise de escopo nenhuma.

**O arquivo que você mandou rodar não leva prefixo.** Ele é único por construção, e os módulos
importados levam prefixos distintos entre si, então não há colisão possível. É o que mantém
tudo que existe hoje — mensagem de erro, golden do emitter, REPL, language server — exatamente
como está. Na prática: `ParseInput` ganha um campo `Module`, e vazio significa sem prefixo.

Uma consequência que precisa estar escrita: **um `defer` importado deixa de enxergar os nomes
do chamador.** Hoje o corpo de um `defer` vê a cadeia de quem chamou
([docs/defer_scope_visibility.md](../docs/defer_scope_visibility.md)); com prefixos, o corpo
de `a/b/c` procura `a/b/c.n` e não acha o `n` que o chamador ligou. Dentro do módulo nada
muda. Isso é desejável — era exatamente o acidente que fazia a camada de namespace antiga
parecer funcionar — mas é mudança de semântica na fronteira, e vai para o roadmap junto.

### O resolver, passo a passo

1. `a/b/c` → `<raiz do projeto>/<source_root>/a/b/c.ar`; se for diretório ou não existir, erro
   nomeando o specifier e quem o importou.
2. Lê os bytes, lexa, parseia. Cacheia **pelo specifier**, então um módulo importado por três
   é lido e parseado uma vez.
3. Monta o grafo a partir dos nós `use` de cada árvore — que são nós de topo, então é
   varredura rasa.
4. **Ciclo é recusado**, com a cadeia inteira (`a/b/c → d/e → a/b/c`). O `CheckDependency` do
   linker antigo já fazia isso bem e é o pedaço que vale ressuscitar.
5. Devolve os módulos em ordem topológica: dependência antes de dependente.

### O loader, passo a passo

1. Para cada módulo, na ordem, monta a **tabela de exports**: varredura rasa dos nós de topo
   (`IdentLiteral`, `StructDeclaration`). Não é walk de árvore.
2. Confere as **referências qualificadas** anotadas pelo parser contra as tabelas; o que não
   existe vira erro com posição, que é o que o language server sublinha.
3. Confere que nenhum alias colide com um nome de topo do próprio arquivo — o alias é uma
   ligação no escopo do arquivo e segue a regra do `ident`: sem shadowing, sem redeclaração.
4. Emite cada árvore (`EmitProgram` por módulo), concatena num stream só e guarda **as faixas**
   de cada um.

---

## O stream: três formas, e uma delas é armadilha

- **N programas avaliados separadamente:** **não funciona.** `OpDefer` grava `from`/`to` como
  posições absolutas no stream em execução (`evaluator/evaluator.go:442`), então uma chamada
  que atravessa módulo aterrissa em instrução alheia. É o motivo do comentário em
  `hosting/cli/test.go:180`.
- **Uma árvore fundida, um `EmitProgram`:** labels ficam únicas sozinhas, mas perde-se de qual
  arquivo veio cada warning — `ir.Program.Warnings` não carrega arquivo, quem carrega é o
  chamador.
- **N `EmitProgram`, um stream, com faixas:** ← **é este.** Já está provado em
  `runTestFile`, mantém atribuição de erro e warning por arquivo, e cada faixa limpa os temps
  (`EvaluateRange`, `evaluator/evaluator.go:661`) — necessário porque cada `EmitProgram`
  numera labels do zero (`emitter/emitter.go:447`).

---

## Onde as peças moram

As regras de [architecture.md](../docs/contributing/architecture.md) decidem quase tudo
sozinhas: vital é puro, `shared` pode tocar no mundo mas só importa `wire` e `util`, hosting
recebe as fases prontas.

| Pacote | Tipo | O quê |
|---|---|---|
| `wire/module` | wire | a identidade de um módulo, a tabela de exports, o grafo — dado atravessando fronteira, que é a definição da categoria |
| `shared/module` | shared | resolver e loader: lê, cacheia, detecta ciclo, ordena, monta o stream |
| `parser` | vital | `use`, o alias, o prefixo e as referências anotadas |
| `hosting/cli` | hosting | pede ao loader em vez de ler um arquivo |

`shared/module` recebe lexer, parser e emitter como **portas** (tipos-função sobre `wire`),
do mesmo jeito que o evaluator recebe `Printer`. É o que o mantém fora de `hosting/cli`, para
que REPL e language server usem o mesmo loader, sem importar nenhum pacote vital.

`wire/module` existe porque `shared` não pode importar `parser`: a tabela que o loader monta e
o parser lê precisa de uma casa que os dois possam nomear. `parser.Declarations` não serve de
precedente contrário — aquilo nunca atravessa fronteira, morre no parse.

---

## As etapas

Uma por commit, cada uma compila e passa sozinha. As duas primeiras não mudam comportamento
nenhum.

1. **O emitter lê o nó, não o token.** Hoje o símbolo sai de `n.Token.GetMatch()`
   (`emitter/emitter.go:96`, `:345`, `:431`), então qualquer renomeação teria que fabricar
   token sintético. Passa a ler o campo do próprio nó, e o token fica só para posição. Três
   sítios, sem mudança de comportamento.
2. **`use` vira palavra-chave**, com o nó `UseDeclaration`, a leitura do caminho por segmentos
   e as regras do alias (obrigatório, único, só no topo do arquivo).
3. **`wire/module` e `shared/module`:** resolver, grafo, ciclo com a cadeia, ordem topológica,
   cache pelo specifier. Sem ligação ainda — devolve módulos em ordem e tem teste próprio.
4. **A ligação:** prefixo no parse, referências anotadas, conferência no loader contra as
   tabelas de export.
5. **O CLI passa pelo loader:** `run` e `test` primeiro, com as faixas por módulo.
6. **`aurora build` e o exemplo:** um projeto com dois módulos em `examples/`, e a
   documentação em `docs/`.

**Isto não cabe num pull request.** As etapas 1–3 são um (nada do que o usuário vê muda, e é
onde mora a maior parte do código novo); 4–5 é o segundo, e é o que faz a coisa existir; 6 é o
terceiro. Se o segundo passar de quarenta minutos de leitura, a conferência do loader sai
para um seu.

---

## Onde o código é tocado

Levantado arquivo por arquivo, para que o tamanho de cada etapa seja lido em vez de estimado.

**1. O emitter lê o nó**

| Onde | O que muda |
|---|---|
| `emitter/emitter.go:92` `emitIdentLiteral` | `n.Token.GetMatch()` → `n.Id` |
| `emitter/emitter.go:338` `emitCalleeLiteral` | `n.Id.Token.GetMatch()` → `n.Id.Value` |
| `emitter/emitter.go:429` `emitIdentifierLiteral` | `n.Token.GetMatch()` → `n.Value` |

Os dois campos já existem e já chegam preenchidos (`wire/ast/node.go:51` e `:200`); o emitter
só não os lê. O token continua no nó, para posição.

**2. `use` no lexer e no parser**

| Onde | O que muda |
|---|---|
| `wire/token/tag.go` | `USE` e `TagUse`; e `processableTags` (`:109`), de onde o language server tira o completar (`hosting/lsp/textdoc/validator.go:214`) |
| `lexer/scanner.go:5` `keywordTags` | entra `TagUse`, e nada mais: `a/b/c` já sai como `ID DIV ID DIV ID` |
| `wire/ast/node.go` | o nó `UseDeclaration` |
| `wire/ast/equal.go` | um caso a mais, na forma do `sameKind` |
| `parser/parser.go:768` `ParseExpr` | `token.USE` → `ParseUse`, ao lado de `STRUCT` |
| `parser/parser.go:899` `ParseExprs` | a regra do topo: `use` só antes de qualquer outro nó |
| `parser/structs.go:167` `parseField` | se a cabeça é um alias, o `.` é membro de módulo em vez de campo |

> **A referência qualificada não precisa de nó novo.** `x.add(1, 2)` chega em `parseField`
> com `x` já lido, e o que sai é um `IdentifierLiteral` com o nome já prefixado — que
> `ParseCallee` (`parser/parser.go:80`) vira `CalleeLiteral` como faria com qualquer outro
> nome. Só o `use` vira nó; a referência é um identificador comum com um nome que ninguém
> consegue digitar. A anotação que o loader confere vai numa lista à parte, não na árvore, do
> mesmo jeito que `parseField` não deixa o nome do campo em lugar nenhum.

**3. O resolver**

| Onde | O que muda |
|---|---|
| `wire/module/` | novo: id do módulo, tabela de exports, grafo |
| `shared/module/` | novo: caminho, leitura, cache pelo specifier, ciclo com a cadeia, ordem topológica |
| `shared/manifest/manifest.go:37` `Project` | o campo `source_root` |
| `shared/manifest/manifest.go:131` `refuseProfileTapeSize` | a mesma recusa para `source_root` dentro de um profile, pelo mesmo motivo |

**4. A ligação**

| Onde | O que muda |
|---|---|
| `parser/parser.go:28` `ParseInput` | o campo `Module` — vazio significa sem prefixo |
| `parser/parser.go:946` `Parse` | escreve o prefixo e devolve as referências que viu |
| `wire/ast/node.go:222` `AST` | carrega as referências qualificadas, ao lado de `Filename` |
| `shared/module/` | a conferência contra as tabelas de export |

**5. Os hosts**

| Onde | O que muda |
|---|---|
| `hosting/cli/session.go` | a porta do loader entra em `NewSessionOptions` |
| `hosting/cli/run.go:17` `Run` | pede o stream e as faixas ao loader em vez de ler um arquivo |
| `hosting/cli/build.go:31` `Build` | idem, com o stream inteiro |
| `hosting/cli/test.go:177` `runTestFile` | vira um caso do loader: o teste e a fonte dele são um módulo só |
| `hosting/cli/target.go:39` `ResolveTarget` | responde também qual é a raiz de módulos |
| `cmd/aurora/run.go:56`, `build.go:66`, `test.go:58` | montam o loader e o entregam — é o único lugar que monta |

### O que não é tocado

É o argumento mais forte a favor deste desenho.

| Pacote | Por quê |
|---|---|
| `evaluator`, `evaluator/environ` | nenhuma linha: nome é byte, e a tabela é indexada pelo hex dele |
| `builder/evm`, `wire/ir` | idem — o backend continua exatamente onde está |
| `emitter/testdata` | o golden parseia sem módulo (`emitter/golden_test.go:37`), então não há prefixo e `wide.ir` não muda |
| `hosting/repl` | segue linha a linha, sem módulo: `use` no REPL fica de fora do primeiro corte |
| `hosting/lsp` | segue parseando o arquivo sozinho; "esse módulo não existe" só chega quando ele passar pelo loader |

---

## O que fica de fora, e por quê

- **`struct` não atravessa módulo.** Exportar um `struct` exigiria a tabela de structs do
  outro módulo **durante** o parse, que é exatamente a dependência que este desenho acabou de
  eliminar. Fica local ao arquivo, e é dito em voz alta.
- **Não há `private`/`export`.** Um módulo expõe tudo que declara no topo. Marcar depois não
  quebra nada.
- **Não há reexport nem transitividade.** `main` usa `a`, `a` usa `m`; `main` não enxerga `m`.
  Cada arquivo declara os seus, como no Clojure.
- **O manifesto não lista dependências.** O sistema de arquivos é a história inteira até
  existir pacote de terceiro.
- **O backend continua em stand-by.** `aurora build` passa a montar um binário de mais de um
  módulo, e os avisos do que ele não carrega continuam valendo — nenhum teto do EVM é aberto
  aqui.

---

## Riscos

- **`use` deixa de ser um identificador.** Quebra declarada, e há teste afirmando o contrário
  hoje (`lexer/scanner_test.go:72`, `lexer/tag_test.go:102`).
- **Language server:** o parse independente por arquivo continua funcionando, que é o que ele
  precisa; mas "esse módulo não existe" e "esse símbolo não existe lá" só aparecem quando ele
  passar pelo loader.
- **Efeito colateral na carga:** o `printb` de um módulo importado imprime. Documentar.
- **Prefixo em mensagem de erro:** `identifier a/b/c.k not found` é mais informativo e mais
  feio. Vale conferir na primeira vez que aparecer de verdade.
