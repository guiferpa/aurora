# A forma atravessa junto com a promessa

**Estado:** proposta · **Data:** 2026-08-20

Um escopo importado responde uma corrida de tapes e quem importa não tem nome para a forma
delas. Esta RFC faz **a promessa carregar a forma**: nada de nomear struct de outro módulo,
nada de sintaxe nova — o que `returns` já diz passa a atravessar.

```
# src/os.ar
struct Env { found, value };

ident lookup = defer {
  ...
} returns Env;
```

```
# src/main.ar
use os as o;

ident r = o.lookup("HOME");
printd r.found;
```

Tudo abaixo foi medido no código de hoje.

> Os blocos aqui **não** estão marcados como `aurora`: `hosting/cli/docs_test.go` executa todo
> bloco assim marcado, e nada disto compila ainda.

---

## O que já está pronto, e é quase tudo

O compilador já sabe fazer isso — em um arquivo. As três tabelas que `struct`, `as` e
`returns` deixam para trás (`parser.Declarations`) fazem a coisa inteira:

| Tabela | O que guarda |
|---|---|
| `Structs` | nome do struct → seus campos, em ordem |
| `Shapes` | nome de um ident → o struct com que ele é lido |
| `Returns` | nome de um escopo → o struct que **chamá-lo** responde |

`ident r = lookup("HOME");` já anota `Shapes["r"]` a partir de `Returns["lookup"]`, e `r.found`
já vira índice a partir de `Structs`. **Nada disso precisa mudar.** O que falta é essas duas
últimas tabelas terem entradas para o que veio de fora.

E o que não muda é maior ainda: nem IR, nem emitter, nem evaluator, nem builder. Nome de
struct nunca chegou a uma instrução — `emitStructLiteral` não lê `n.Name`, e
`emitFieldExpression` diz na cara que o índice foi resolvido no parse. É mudança de front-end,
inteira.

---

## Só a promessa atravessa

O que cruza a fronteira é, por escopo exportado que prometeu, **o nome do struct prometido e
os campos dele**. Nada mais.

Disso sai uma propriedade que vale ter de graça: **um struct só faz parte da interface de um
módulo quando uma promessa o nomeia.** Declarar um struct novo em `os.ar` não muda o que `os`
oferece; prometer com ele, sim. Ninguém exporta forma sem querer.

E como não há sintaxe para nomear um struct de fora, a restrição é invisível: não existe
programa que gostaria de nomear `os.Env` e não consegue, porque nomear não é uma coisa que se
faça.

### A chave que ninguém consegue digitar

A forma importada entra em `Structs` sob o nome qualificado do módulo — `os.Env` —, que é o
mesmo formato que os identificadores já usam e que o lexer não deixa ninguém escrever (`/` e
`.` não cabem num identificador). Dois módulos com um `Env` cada são duas entradas, sem
colisão e sem regra nova.

---

## Nós já sabemos. Sabemos tarde

A pergunta certa é: se a árvore de `math` está ali, e o `Exports` já a lê procurando
`IdentLiteral`, por que ele não saberia do `struct Prime` do mesmo arquivo? Saberia — ler
`StructDeclaration` do mesmo jeito são duas linhas, e o nó tem nome e campos.

O problema não é **se** sabemos, é **quando**:

| | O que acontece | Onde |
|---|---|---|
| 1 | `main.ar` é parseado | `resolver/resolver.go:75` |
| 2 | as linhas `use` dele são lidas da árvore | `declarationsOf` |
| 3 | `math.ar` é lido e parseado | `resolveOne` |
| 4 | o loader confere que `math` tem mesmo um `sum` | `loader.Check` |

`m.sum(1, 2)` funciona porque a pergunta dele é respondida no **passo 4**, com as duas árvores
na mão. O parser do passo 1 não precisou saber nada: escreveu `math.sum` e anotou a referência
para depois.

`m.Prime` precisaria da resposta no **passo 1**, e `math.ar` só é lido no 3.

**O parse anda de fora para dentro, e a forma precisa de dentro para fora.** Um módulo é
parseado antes dos que ele importa — a ordem que o resolver devolve é a inversa da ordem em que
ele parseia. É esse relógio que esta RFC acerta, e é por isso que a extração do conhecimento
não aparece na conta: ela é trivial.

---

## O que muda: o conhecimento chega antes do parse

Aqui está o custo, e é um só. Hoje o resolver **parseia a entrada primeiro** e depois anda
pelas dependências (`resolver/resolver.go:75`), então quando `main.ar` é parseado, `os.ar`
ainda não foi lido. Para a promessa chegar a tempo, a ordem inverte: **dependência primeiro,
depois quem importa.**

E para saber o que uma dependência é antes de parsear o arquivo, é preciso ler as linhas `use`
dele antes. Dois jeitos:

| | Como | Custo |
|---|---|---|
| **Cabeçalho** | ler as declarações `use` dos tokens, antes do parse completo | uma porta nova e um leitor de ~40 linhas — que já existe no editor, em `scanUses` |
| **Parsear duas vezes** | parsear sem as tabelas para descobrir os `use`, e de novo com elas | nenhuma linha nova; custa um parse a mais por arquivo que importa |

**Proposta: o cabeçalho.** Parsear duas vezes é mais simples e dobra a latência do language
server a cada tecla em qualquer arquivo com import, que é justamente o caminho que a camada 2
acabou de deixar rápido.

### A degradação é graciosa

Vale dizer porque é o que torna essa troca barata: **sem as tabelas, o comportamento é o de
hoje.** Um arquivo que não importa parseia igual. Um arquivo que importa e não lê campo de
promessa parseia igual. Só quem lê `r.found` de um valor que veio de fora precisa da tabela — e
sem ela recebe o erro que já existe, "nothing says which struct this value is".

Não é o invariante inteiro que cai, é uma dependência estreita: **o parse de um arquivo passa a
depender das promessas do que ele importa, e de nada mais.**

---

## Como a promessa viaja

Do jeito que as referências qualificadas já viajam: **na árvore, ao lado dela.** O parser já
devolve `ast.AST.References`, que é o que ele viu e não pode responder; ganha um irmão:

```go
// Promises is what this file's exported scopes said they answer with, with the fields of
// each — the only thing about a struct that leaves the file that declared it.
type Promise struct {
	Scope  string   // the name the scope is bound to
	Struct string   // the struct it promised
	Fields []string // and what that struct is made of
}
```

O resolver lê `tree.Promises` de cada módulo que acabou de parsear e passa adiante, num campo
novo de `ParseInput`, indexado pelo **specifier** — não pelo alias, que é do arquivo que
importa e ele descobre sozinho ao ler o próprio `use`.

---

## O que muda em cada peça

| Onde | O quê |
|---|---|
| `wire/ast` | `Promise` e `AST.Promises` |
| `parser` | anotar as promessas e as declarações de topo; ler as importadas para `Structs` e `Returns` sob o nome qualificado; aceitar o nome qualificado em `as`, `returns` e construção |
| `parser.ParseInput` | as promessas do que este arquivo importa, por specifier |
| `resolver` | ordem de dependência primeiro, e a porta de cabeçalho |
| `loader` | nada — a conferência de nome já é dele e não muda |
| `hosting/lsp` | juntar as promessas dos módulos resolvidos às formas lidas dos tokens |
| `hosting/repl` | passar as promessas dos módulos já carregados para o parse da próxima linha |
| emitter, IR, evaluator, builder | nada |

---

## Nomear um struct importado entra junto

A primeira versão desta RFC deixava isso de fora, com o argumento de que a promessa carregando
a forma tornava a nomeação desnecessária. Uma pergunta derrubou o argumento:

```
use math as m;

ident n = { 3; } returns m.Prime;
```

**É o caso em que a promessa não ajuda**, porque é a promessa que precisa nomear a forma
alheia. E deixá-lo de fora produziria uma regra que não se explica: como as promessas já
chegam antes do parse, `m.Prime` estaria disponível **se** `math` tivesse prometido com ela em
algum lugar, e indisponível se ele apenas a declarou. "Dá para nomear quando o outro módulo
por acaso promete" não é regra, é acidente.

O que mudou o cálculo é que **o caro aqui é a ordem**, e ela já está paga por tudo que veio
antes. Sobre a ordem, nomear custa:

| | |
|---|---|
| exportar as declarações de topo, e não só as prometidas | ~20 |
| o nome qualificado em `as`, em `returns` e na construção | ~60 |
| a conferência de que o módulo tem mesmo aquele struct, com posição | ~20 |

Então entra: `as m.Prime`, `returns m.Prime` e `m.Prime{1, 2}` são todos a mesma leitura de um
nome qualificado, do jeito que `m.sum` já é.

A propriedade que a versão anterior comprava — um struct só é interface quando uma promessa o
nomeia — se perde, e é uma perda real: qualquer struct de topo passa a ser alcançável de fora.
Vale menos que a coerência da regra, e um dia `private` a devolve para quem quiser.

## O que continua de fora

- **Promessa transitiva.** Se `a` promete um struct que veio de `b`, o que atravessa é a forma
  final: campos, não a origem. Ninguém precisa saber de `b`.
- **Struct declarado num módulo e usado como declaração em outro** continua não existindo como
  conceito: não há herança de declaração, há um nome qualificado que se lê. E quem preferir
  declarar o seu continua podendo — duas declarações de mesma largura são o mesmo valor, o que
  `Point{1,2} equals Pair{1,2}` já mostra.

---

## As etapas

1. **A promessa sai na árvore**: `wire/ast.Promise`, o parser anotando, e teste. Ninguém lê
   ainda.
2. **O resolver inverte a ordem**, com a porta de cabeçalho, e passa as promessas adiante.
3. **O parser lê as importadas** para as suas tabelas, sob o nome qualificado — que é onde a
   coisa passa a funcionar.
4. **O editor e a REPL**, que é onde ela passa a ser útil enquanto se escreve.
5. **Documentação**: `docs/modules.md`, `docs/language-design.md` e o roadmap, que hoje dizem
   que struct não atravessa.

Ordem de grandeza: ~280 linhas de código e ~330 de teste, dois pull requests — a ordem e as
promessas num, a nomeação no outro, já que o segundo só é possível depois do primeiro e é o que
alguém consegue revisar de uma vez.

---

## Em aberto

1. **Cabeçalho ou parsear duas vezes.** Proposta acima: cabeçalho.
2. **Uma promessa que nomeia um struct de um módulo que o importador também importa** — a
   forma atravessa duas vezes, com duas chaves diferentes, e são dois `Structs` iguais. Não
   quebra nada; é desperdício, e dá para deduplicar depois.
3. **O que a mensagem de erro diz** quando alguém lê um campo que a promessa não tem: hoje
   seria "struct os.Env has no field named x", com um nome que ninguém digitou. Talvez valha
   dizer o nome do módulo em vez do nome qualificado.
