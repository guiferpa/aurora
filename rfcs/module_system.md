# Módulos: resolver, loader e a ligação que faltou

**Estado:** proposta · **Data:** 2026-08-19

O desenho já existe em inglês — [docs/module_system_design.md](../docs/module_system_design.md)
— e decidiu a **forma**: um arquivo é um módulo, todo import tem alias, e os três trabalhos
(resolver, loader e a ligação de nomes) são coisas separadas. O que faltava era **como
construir**, e é isso que esta RFC decide.

Tudo abaixo foi medido no código de hoje, não suposto.

> Os blocos de código aqui **não** estão marcados como `aurora` de propósito:
> `hosting/cli/docs_test.go` executa todo bloco assim marcado no repositório inteiro, e nada
> disto compila ainda.

---

## O que muda em relação ao documento de desenho

Comparados lado a lado, para o documento em inglês ser corrigido quando isto for aceito.

| Assunto | Documento de desenho | Aqui |
|---|---|---|
| Specifier | texto entre aspas, com `./math` relativo e `math` reservado | caminho nu a partir de uma raiz só: `use a/b/c as x;` |
| Raiz | pergunta em aberto: raiz do projeto ou caminho declarado no manifesto | `src/` por padrão, `[project] source_root` para mudar |
| Chave do cache | caminho absoluto | o specifier, que já é canônico |
| Onde a ligação acontece | mangling no emitter, ou um operando de módulo no IR | prefixo escrito no **parse**; a conferência no loader |
| Quem faz o quê | resolver = caminho; loader = parse, grafo, ordem; binder = nomes | resolver = descoberta, parse, grafo, ordem; loader = exports, conferência, emissão, stream |
| Onde os pacotes moram | não dizia | `resolver` e `loader` são **vitais**, na raiz, ao lado das outras fases |
| Environ | um por módulo | **igual** — e a primeira versão desta RFC divergia disso, errado; ver abaixo |
| Acessor | pergunta em aberto: `.` ou `::` | `.`, sem token novo |
| Ciclo | pergunta em aberto: recusar ou inicializar preguiçoso | recusar, com a cadeia |
| Arquivo de teste | pergunta em aberto | é parte do módulo que testa |

O que **não** muda: um arquivo é um módulo, alias obrigatório, nada de `:refer`, o `.ar`
implícito, diretório é erro, e um módulo expõe tudo que declara no topo.

---

## A premissa

Aurora roda o arquivo inteiro, como Python e JavaScript: o corpo de um módulo executa **uma
vez, na primeira importação, em ordem de dependência**.

Uma consequência aceita de olhos abertos: um módulo com `printb` no topo **imprime ao ser
importado**. Python resolve com `if __name__ == "__main__"`, que Aurora não tem e não ganha
aqui. Print é log; que ele apareça na carga é honesto.

---

## Módulo e namespace, que não são a mesma coisa

As duas palavras aparecem juntas o tempo todo e respondem perguntas diferentes:

- **Módulo** é a unidade de **carga**. Responde *de onde o código vem*: um arquivo, lido,
  parseado e executado uma vez.
- **Namespace** é a unidade de **resolução**. Responde *a que declaração este nome se refere*:
  uma tabela de nomes que alguém possui.

Em Aurora as duas coincidem — um arquivo tem exatamente um namespace — mas não coincidem em
toda linguagem, e é aí que os dois modelos que citamos se separam.

**Clojure: independentes.** `ns` cria um namespace; o arquivo é só onde ele foi digitado. Um
namespace pode nascer no REPL sem arquivo nenhum, e `(in-ns 'foo)` troca o corrente sem
carregar coisa alguma. `require` é a carga, `:as` é o apelido local daquele arquivo. E os vars
são **globais**: `#'math/add` é alcançável de qualquer lugar pelo nome completo, sempre — o
alias só encurta a escrita. Resolução em tempo de leitura, valor num espaço global.

**Python: a mesma coisa, e em runtime.** `import a.b.c as x` executa o arquivo e liga `x` a um
objeto cujo `__dict__` é o namespace; `x.add` é **busca de atributo na hora da execução**.
Trocar o que está em `x.add` depois muda o que a chamada faz.

**Aurora fica no meio, de propósito:** o nome é resolvido em **tempo de compilação**, como no
Clojure — quem lê `x.add` sabe onde `add` mora sem rodar nada — e o namespace **existe em
runtime como um environ próprio**, como no Python. O prefixo do módulo é o que liga as duas
pontas: é escrito no parse e é a chave de roteamento no evaluator.

De nenhum dos dois pegamos o que traz um nome solto para o escopo — `:refer` no Clojure,
`from x import y` no Python. Os dois escondem a origem do nome no ponto de uso, que é o
contrário do que um alias obrigatório existe para dar.

---

## A sintaxe

```
use a/b/c as x;

printb x.add(1, 2);
```

- **Todo import tem alias**, e o alias é o único jeito de alcançar o que está dentro. Lendo
  `x.add(1, 2)` você sabe onde `add` mora sem rolar a tela.
- O specifier é um caminho **a partir da raiz de módulos**. **Não existe forma relativa**:
  `./x` e `../x` não são caminhos válidos.
- O `.ar` é implícito. Um specifier que aponta para um diretório é erro — um módulo é um
  arquivo.
- Os segmentos são colados: `a / b` não é caminho, é divisão em lugar errado.

**O que essa escolha compra**, e é mais do que parece:

1. **O specifier é a identidade do módulo.** `a/b/c` não depende de quem importou, nem do
   diretório de onde o comando foi rodado, nem de symlink. Cache e detecção de ciclo passam a
   ser sobre uma string canônica em vez de sobre um caminho de arquivo.
2. **O parser não precisa saber a raiz do projeto** para escrever o nome prefixado, o que
   mantém o parse puro e sem entrada nova além do módulo em que ele está.
3. Some a canonicalização de caminho, que era metade do trabalho chato do resolver.

### Onde a raiz é declarada

| | |
|---|---|
| Padrão | `src/` |
| Mudança | `[project] source_root = "lib"` |
| Com manifesto | a raiz é relativa à raiz do projeto (o diretório do `aurora.toml`) |
| Sem manifesto | continua sendo `src/`, relativa ao diretório do arquivo de entrada |

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
(`lexer/scanner_test.go:72` afirma isso). É uma quebra, pequena e aceita.

---

## O environ: um por módulo, indexado pelo prefixo

> **Esta seção foi reescrita.** A primeira versão desta RFC propunha um environ raiz achatado
> e argumentava contra um por módulo. O argumento estava errado — ele refutava *uma* maneira
> de fazer environ por módulo, não a que está aqui. O documento de desenho já dizia "um
> environ por módulo", e é isso mesmo.

### Por que o argumento antigo não valia

Ele era: o valor de um `defer` é um índice contado no environ que o criou
(`evaluator/evaluator.go:448`), e a chamada acha o corpo subindo a cadeia de quem chamou
(`evaluator/evaluator.go:456`) — logo `main` chamando `x.area` acharia o `defer 0` de `main`.
Consertar exigiria identidade de módulo dentro do valor, que é **uma tape**
(`evaluator/evaluator.go:450`), e com `tape_size 1` sobram oito bits.

O erro está no "logo". Isso só acontece se a resolução continuar subindo a cadeia do chamador.
**Se o nome diz de que módulo ele é, a resolução vai direto no environ daquele módulo** — e aí
o índice do `defer` só precisa ser único dentro do módulo, que ele já é. Nada é empacotado
dentro da tape, porque a identidade vem do **nome**, não do valor.

### O modelo

Um índice de environs, na raiz: **id do módulo → environ daquele módulo**. O corpo de cada
módulo roda com o environ dele na base da cadeia, e é lá que os `ident` e os `defer` do topo
daquele módulo ficam.

A busca de um nome tem dois saltos, e é isso que faz o desenho parecer DNS:

1. **a cadeia**, exatamente como hoje — escopo local, e o escopo dinâmico que um `defer` vê do
   chamador ([docs/defer_scope_visibility.md](../docs/defer_scope_visibility.md));
2. **o environ do módulo que o nome nomeia**, que o prefixo entrega. Um nome sem prefixo cai no
   environ da base, que é o do arquivo de entrada — ou seja, o comportamento de hoje, intacto.

Cada caso, conferido contra o código:

| Caso | Como resolve |
|---|---|
| `a/b/c.add` chamado do módulo B | não está na cadeia de B → environ de `a/b/c` → ident → índice → `defers` **do mesmo environ** |
| corpo de um `defer` de A rodando a partir de B | referencia `a/b/c.k`, não está na cadeia → environ de A. **O defer importado continua enxergando o módulo dele** |
| `ident` local dentro de um bloco ou corpo | ligação nunca é roteada: vai no environ corrente, e a cadeia acha |
| escopo dinâmico dentro do mesmo módulo | a cadeia vem primeiro, então continua valendo como está documentado |
| programa de um arquivo só | nomes sem prefixo, environ base, nada muda |

**É por isso que o prefixo é uniforme** (adiante): ele não serve só para não colidir, ele é a
**chave de roteamento**. Sem ele, o evaluator precisaria carregar "qual módulo está
executando agora", salvar e restaurar isso em cada chamada — estado a mais no lugar mais
quente do interpretador.

### O índice é um mapa, não uma árvore

Ponto a discutir: um índice **plano**, de id para environ, e não uma árvore por segmento de
caminho. No DNS a delegação de `c.b.a` para `b.a` existe de verdade; aqui `a/b` e `a/b/c` não
têm parentesco nenhum — `a/b` pode nem existir como arquivo, e um nome de um não é alcançável
pelo outro. Uma árvore afirmaria um vínculo que a linguagem não dá. A busca é hierárquica no
**nome**; a estrutura que a guarda pode ser um mapa.

---

## A regra do prefixo

**Dentro de um módulo importado, todo identificador — ligação ou menção — é escrito
`a/b/c.nome`.** Uma referência qualificada `x.add` é escrita com o prefixo do módulo
importado, que o alias fornece.

Como todas as menções dentro de um arquivo levam o mesmo prefixo, isso é uma renomeação
constante do espaço de nomes daquele arquivo: **não exige que o parser saiba o que é topo e o
que é escopo interno**, e não exige análise de escopo nenhuma. E o nome resultante é
impronunciável — `/` e `.` não cabem num identificador —, então nunca colide com algo que
alguém escreveu.

**O arquivo que você mandou rodar não leva prefixo.** Ele é único por construção, e os módulos
importados levam prefixos distintos entre si, então não há colisão possível. É o que mantém
tudo que existe hoje — mensagem de erro, golden do emitter, REPL, language server — exatamente
como está. Na prática: `ParseInput` ganha um campo `Module`, e vazio significa sem prefixo.

---

## O que o código já dá de graça

| Fato | Onde | Por que importa |
|---|---|---|
| Símbolo no IR é byte cru | `emitter/emitter.go:96`, `:345`, `:431` | o prefixo entra sem tocar em IR nem no writer EVM |
| Environ indexa por `hex(bytes)` | `evaluator/evaluator.go:364`, `:411`, `:457` | o nome prefixado é chave sem nenhuma conversão |
| `.`, `/` e `:` não cabem num identificador | `lexer/scanner.go:68` | um nome com prefixo nunca colide com algo escrevível |
| O blob de um `defer` **não** é uma tape | `evaluator/evaluator.go:26` | o que tem largura fixa é só o índice; o blob é bytes livres |
| `.` já é ligado em tempo de parse, contra tabela injetada | `parser/structs.go:167` | o acessor de módulo é o mesmo mecanismo, não um novo |
| Um loader de dois módulos já existe | `hosting/cli/test.go:177` | concatenar streams e avaliar por faixas está provado |

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

O parser já visita cada nó, então ele **anota as referências qualificadas que viu**
(`{módulo, símbolo, token}`) e o loader confere a lista contra as tabelas. Um passe de
reescrita de árvore teria ~25 casos de `switch` para escrever e manter, com exatamente o modo
de falha silenciosa que `EmitInstruction` já tem: nó não tratado passa sem ligar.

### O resolver, passo a passo

1. `a/b/c` → `<source_root>/a/b/c.ar`; se for diretório ou não existir, erro nomeando o
   specifier e quem o importou.
2. Pede os bytes pela porta de leitura, lexa, parseia. Cacheia **pelo specifier**, então um
   módulo importado por três é lido e parseado uma vez.
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
   de cada um, com o id do módulo de cada faixa — que é o que diz ao evaluator em qual environ
   aquela faixa roda.

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

As faixas são **limites de cursor, não fatias**: o stream inteiro vai em toda chamada, e só o
cursor anda entre `from` e `to`.

---

## Onde as peças moram

`resolver` e `loader` são **vitais**, na raiz, ao lado das outras fases. O argumento é que
eles não precisam do mundo: se a leitura de um arquivo chega por uma porta, os dois são
funções puras que recebem valores e devolvem valores — que é a definição da categoria — e
**fazem parte do pipeline**, que é onde eles pertencem.

| Pacote | Tipo | O quê |
|---|---|---|
| `resolver` | vital | descobre, ordena, recusa ciclo; recebe a porta de leitura, o lexer e o parser |
| `loader` | vital | exports, conferência, emissão, stream e faixas; recebe o emitter |
| `wire/module` | wire | id, tabela de exports, grafo, módulo em ordem — o que cruza entre os dois |
| `parser` | vital | `use`, o alias, o prefixo e as referências anotadas |
| `evaluator/environ` | vital | o índice de environs por módulo |
| `hosting/cli` | hosting | monta a porta de leitura e pede o stream, em vez de ler um arquivo |

**A porta de leitura é o que prova o desenho.** O playground roda em `js/wasm`, sem sistema de
arquivos: com a leitura numa porta, ele entrega módulos de memória e usa o mesmo resolver.
Se o resolver lesse disco sozinho, não haveria como.

O pipeline deixa de ser uma linha reta, porque resolver e loader **conduzem** outras fases:

```
(resolver) → {módulos em ordem} → (loader) → {stream + faixas} → (evaluator) │ (builder)
    ↑ porta de leitura, lexer, parser            ↑ emitter
```

Isso muda o diagrama de [architecture.md](../docs/contributing/architecture.md), que é a fonte
da verdade sobre a estrutura — atualizar aquele documento faz parte do trabalho, não é efeito
colateral dele.

---

## As etapas

Uma por commit, cada uma compila e passa sozinha. As duas primeiras não mudam comportamento
nenhum.

1. **O emitter lê o nó, não o token.** Hoje o símbolo sai de `n.Token.GetMatch()`, então
   qualquer renomeação teria que fabricar token sintético. Passa a ler o campo do próprio nó,
   e o token fica só para posição.
2. **`use` vira palavra-chave**, com o nó `UseDeclaration`, a leitura do caminho por segmentos
   e as regras do alias (obrigatório, único, só no topo do arquivo).
3. **`wire/module` e `resolver`:** grafo, ciclo com a cadeia, ordem topológica, cache pelo
   specifier, porta de leitura. Sem ligação ainda — devolve módulos em ordem e tem teste
   próprio.
4. **A ligação:** prefixo no parse, referências anotadas, conferência no `loader` contra as
   tabelas de export.
5. **O environ por módulo:** o índice na raiz, o segundo salto da busca, e a faixa que sabe em
   qual módulo roda.
6. **O `loader` monta o stream** e o CLI passa a pedir a ele: `run` e `test` primeiro.
7. **`aurora build`, o exemplo e a documentação:** um projeto com dois módulos em `examples/`,
   `docs/` e o diagrama de `architecture.md`.

---

## Esforço e refatoração

Estimativa, não medida — as etapas 1 e 2 são as únicas que dá para cravar. O que está entre
parênteses é código de teste, contado à parte porque aqui ele é metade do trabalho.

| Etapa | Código novo | Código mexido | Risco |
|---|---|---|---|
| 1. emitter lê o nó | — | 3 linhas (+30) | nenhum: os campos já chegam preenchidos |
| 2. `use` no lexer e no parser | ~120 (+200) | ~20 | baixo, e isolado no parse |
| 3. `wire/module` + `resolver` | ~250 (+250) | — | médio: é pacote novo, e a porta de leitura é desenho novo |
| 4. a ligação | ~150 (+250) | ~40 no parser | médio: é onde um erro vira nome que não resolve |
| 5. environ por módulo | ~80 (+200) | ~60 no evaluator | **o mais alto**: mexe na busca de nome, que é o lugar mais quente |
| 6. `loader` + CLI | ~150 (+200) | ~120 nos hosts | médio: três comandos deixam de ler arquivo |
| 7. build, exemplo, docs | ~50 (+100) | ~200 de markdown | baixo |

Ordem de grandeza: **~800 linhas de código e ~1200 de teste**, em três ou quatro pull
requests. As etapas 1–3 cabem num (nada que o usuário veja muda); 4–6 é o segundo e é o que
faz a coisa existir — se passar de quarenta minutos de leitura, a etapa 5 sai sozinha para um
terceiro; 7 fecha.

**As refatorações de verdade**, que é o que custa mais do que parece:

1. **`hosting/cli` deixa de ler arquivo.** `Run`, `Build` e `Test` hoje fazem `os.ReadFile` e
   chamam lexer, parser e emitter na mão. Passam a pedir o stream ao loader, e a leitura vira
   uma porta montada em `cmd/aurora`. É a maior mudança de host, e ela **melhora** o que já
   estava lá: hoje as três funções repetem o mesmo bloco de quatro fases.
2. **A busca de nome no evaluator ganha um segundo salto.** É aditivo — a cadeia continua
   sendo consultada primeiro —, mas é o coração do interpretador, e é onde os testes precisam
   ser mais duros.
3. **`EvaluateRange` ganha o módulo da faixa.** A assinatura é usada pelo REPL
   (`hosting/repl/repl.go:168`) e pelo `aurora test`; a forma atual continua existindo, com
   módulo vazio.
4. **O emitter lê o nó em vez do token**, que é preparação e não tem volta atrás.
5. **`docs/contributing/architecture.md`**, porque o pipeline deixa de ser uma linha reta.

---

## Onde o código é tocado

Levantado arquivo por arquivo, para o tamanho de cada etapa ser lido em vez de estimado.

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
> consegue digitar.

**3. O resolver**

| Onde | O que muda |
|---|---|
| `wire/module/` | novo: id do módulo, tabela de exports, grafo, módulo em ordem |
| `resolver/` | novo: caminho, porta de leitura, cache pelo specifier, ciclo com a cadeia, ordem topológica |
| `shared/manifest/manifest.go:37` `Project` | o campo `source_root` |
| `shared/manifest/manifest.go:131` `refuseProfileTapeSize` | a mesma recusa para `source_root` dentro de um profile, pelo mesmo motivo |

**4. A ligação**

| Onde | O que muda |
|---|---|
| `parser/parser.go:28` `ParseInput` | o campo `Module` — vazio significa sem prefixo |
| `parser/parser.go:946` `Parse` | escreve o prefixo e devolve as referências que viu |
| `wire/ast/node.go:222` `AST` | carrega as referências qualificadas, ao lado de `Filename` |
| `loader/` | novo: a conferência contra as tabelas de export |

**5. O environ por módulo**

| Onde | O que muda |
|---|---|
| `evaluator/environ/environ.go` | o índice `id → *Environ` na raiz, e como se entra num módulo |
| `evaluator/environ/environ.go:47` `GetIdent` | o segundo salto: não achou na cadeia, vai no environ do módulo que o nome nomeia |
| `evaluator/evaluator.go:456` `EvaluateCall` | ident e `defer` são procurados no mesmo environ, o do dono do nome |
| `evaluator/evaluator.go:659` `EvaluateRange` | a faixa passa a saber em qual módulo roda |

**6. O loader e os hosts**

| Onde | O que muda |
|---|---|
| `loader/` | exports, emissão, concatenação, faixas |
| `hosting/cli/session.go` | as portas do resolver e do loader entram em `NewSessionOptions` |
| `hosting/cli/run.go:17` `Run` | pede o stream e as faixas, em vez de ler um arquivo |
| `hosting/cli/build.go:31` `Build` | idem, com o stream inteiro |
| `hosting/cli/test.go:177` `runTestFile` | vira um caso do loader: o teste e a fonte dele são um módulo só |
| `hosting/cli/target.go:39` `ResolveTarget` | responde também qual é a raiz de módulos |
| `cmd/aurora/run.go:56`, `build.go:66`, `test.go:58` | montam resolver e loader e os entregam — é o único lugar que monta |

### O que não é tocado

| Pacote | Por quê |
|---|---|
| `builder/evm`, `wire/ir` | nome é byte, e o prefixo é só outro nome; o backend continua onde está |
| `emitter/testdata` | o golden parseia sem módulo (`emitter/golden_test.go:37`), então não há prefixo e `wide.ir` não muda |
| `hosting/repl` | segue linha a linha, sem módulo: `use` no REPL fica de fora do primeiro corte |
| `hosting/lsp` | segue parseando o arquivo sozinho; "esse módulo não existe" só chega quando ele passar pelo loader |

---

## O que fica de fora, e por quê

- **`struct` não atravessa módulo.** Exportar um `struct` exigiria a tabela de structs do
  outro módulo **durante** o parse, que é exatamente a dependência que este desenho acabou de
  eliminar. Fica local ao arquivo, e é dito em voz alta.
- **Não há `private`/`export`.** Um módulo expõe tudo que declara no topo. Com environ por
  módulo isso vira fácil de adicionar depois, porque a fronteira passa a existir de verdade.
- **Não há reexport nem transitividade.** `main` usa `a`, `a` usa `m`; `main` não enxerga `m`.
  Cada arquivo declara os seus, como no Clojure.
- **O manifesto não lista dependências.** O sistema de arquivos é a história inteira até
  existir pacote de terceiro.
- **O backend continua em stand-by.** `aurora build` passa a montar um binário de mais de um
  módulo, e os avisos do que ele não carrega continuam valendo — nenhum teto do EVM é aberto
  aqui.

---

## Riscos

- **A busca de nome é o lugar mais quente do evaluator.** O segundo salto só roda quando o
  primeiro falha, mas é ali que um erro vira programa que responde a coisa errada em silêncio.
  É a etapa que pede mais teste e o pull request que pede mais leitura.
- **`use` deixa de ser um identificador.** Quebra aceita, e há teste afirmando o contrário
  hoje (`lexer/scanner_test.go:72`, `lexer/tag_test.go:102`).
- **Language server:** o parse independente por arquivo continua funcionando, que é o que ele
  precisa; mas "esse módulo não existe" e "esse símbolo não existe lá" só aparecem quando ele
  passar pelo resolver.
- **Prefixo em mensagem de erro:** `identifier a/b/c.k not found` é mais informativo e mais
  feio. Vale conferir na primeira vez que aparecer de verdade.

---

## Em aberto

O que ainda não foi batido o martelo, e é o que vale discutir antes de começar:

1. **O índice de environs é mapa ou árvore.** Proposta acima: mapa, porque `a/b` e `a/b/c` não
   têm parentesco.
2. **Sem manifesto, `src/` é relativo a quê.** Proposta: ao diretório do arquivo de entrada.
3. **Transitividade, `use` só no topo, o que um módulo expõe, alias dividindo espaço de nomes
   com `ident`, `struct` local ao arquivo.** Todos propostos acima; nenhum discutido.
