# `returns`: o que um escopo promete, e o compilador confere

**Estado:** proposta · **Data:** 2026-08-20

Hoje um valor composto sai de um escopo e volta a ser bytes: quem chama escreve `as Result` e
o compilador acredita. `returns` é a outra ponta — **o escopo promete a forma, e a promessa é
conferida onde ela é escrita**.

```
struct Person { name };

{
  ident p = Person{"Joana"};
  10;
} returns Person;
```

Isto é um erro: o bloco diz que responde `Person` e termina num número.

Tudo abaixo foi medido no código de hoje.

> Os blocos aqui **não** estão marcados como `aurora`: `hosting/cli/docs_test.go` executa todo
> bloco assim marcado, e nada disto compila ainda.

---

## O que foi medido

Quatro sondas, e as quatro decidem alguma coisa:

| Escrito hoje | O que acontece |
|---|---|
| `ident r = { P{1}; } as P;` | **não parseia** — `unexpected token as` |
| `ident f = defer { if c { P{1}; } else { P{2}; }; };` e `f() as P` | funciona: `as` é uma alegação sobre o valor, e o valor é o que o ramo produziu |
| `ident r = if c { P{1}; } else { P{2}; }; r.a` | erro: nada diz de que struct isso é |
| `ident r = f(); r.a` | erro: idem, mesmo que `f` só responda `P` |

A primeira linha diz que `returns` é gramática nova, e não uma variação de algo que já existe:
um bloco no nível de expressão é lido direto em `ParseBlockExpr` (`parser/parser.go:692`), sem
passar pelo posfixo onde `as` e `.` moram.

As duas últimas são o que **`returns` conserta**, e são a razão dele valer mais que um `as`
para blocos.

---

## Duas coisas, e a segunda é o prêmio

**A conferência.** `as` não pode ser conferido: uma corrida de bytes não tem nada dentro para
checar contra. Mas a **última expressão de um bloco** é uma coisa sintática, e o compilador
sabe olhar para ela — `shapeOf` já responde por uma construção, por um `as` e por um nome cuja
forma foi anotada (`parser/structs.go`). Um bloco que promete `Person` e termina em `10` é
recusado antes de rodar.

**O nome não precisa mais ser alegado na chamada.** Se `f` prometeu, `f()` tem forma:

```
struct Result { failed, value };

ident divide = defer {
  if feed(1) equals 0 { Result{1, 0}; } else { Result{0, feed(0) / feed(1)}; };
} returns Result;

ident r = divide(10, 2);
printd r.value;
```

Sem `returns`, a penúltima linha precisa de `as Result` — e ninguém confere se `divide`
realmente responde isso. Com ele, a alegação vira promessa, e some do lugar onde ela era
repetida uma vez por chamada.

---

## `as` e `returns` servem a duas pessoas diferentes

**O compilador não ignora o `as`** — e é justamente por isso que ele importa. Ele lê e age
sobre a alegação: é o `as` que transforma `p.x` num índice de tape, e sem ele `p.x` nem
compila. O que ele não faz é **conferir**.

Isso é pior do que ignorar. Se fosse ignorado, um `as` errado seria inofensivo; como ele é
acreditado, um `as` errado lê a tape errada e o programa continua — o que a documentação já
diz com todas as letras: *"A wrong claim reads the wrong tapes"*
([language-design.md](../docs/language-design.md)).

Daí as duas palavras servirem a momentos diferentes:

| | serve a | e o compilador |
|---|---|---|
| `as` | **quem escreve**: é como o programador diz, para si e para quem ler depois, o que aqueles bytes são | acredita, e usa |
| `returns` | **o acordo**: é o que garante que o escopo não respondeu algo diferente do combinado | confere, e recusa |

E `returns` nomeia um **struct** porque struct é a única forma que a linguagem tem hoje de
dizer o que uma corrida de tapes é. A palavra é sobre o acordo, não sobre struct: se um dia
houver outra maneira de dar forma a uma corrida, `returns` nomeia essa também, sem mudar de
sentido.

---

## O que `shapeOf` precisa aprender

**A conferência vale o que `shapeOf` enxerga**, e hoje ele não enxerga o caso que motiva tudo
isto: um `if` escolhendo entre dois resultados. A lista, em ordem de necessidade:

| Forma | Hoje | Precisa |
|---|---|---|
| `Result{...}` | ✓ | — |
| `x as Result` | ✓ | — |
| um nome anotado | ✓ | — |
| `if c { Result{...}; } else { Result{...}; }` | ✗ | os dois ramos concordando |
| `{ ...; Result{...}; }` | ✗ | a última expressão do bloco |
| `f()` onde `f` prometeu | ✗ | a tabela nova |

**Um `if` sem `else` não promete nada.** Quando o teste falha, aquele caminho responde o
valor neutro, e o valor neutro não é um `Person` — então prometer ali seria prometer o que
metade das execuções não cumpre. A recusa diz isso:

```
this block answers with Person and its if has no else: one path answers with nothing
```

E um ramo que responde outra coisa é recusado nomeando o ramo, não o bloco:

```
this block answers with Person and the else answers with a number
```

E o que **não** dá para enxergar continua não dando: um valor vindo de `feed`, ou de qualquer
lugar que o compilador não acompanha, segue precisando de `as`. `returns` não substitui `as` —
ele tira `as` de onde havia uma promessa a cumprir, e deixa onde há mesmo uma alegação a fazer.

---

## A forma

`returns` é lido no fim de um bloco, que é onde os dois casos se encontram:

```
{ ... } returns Person;
defer { ... } returns Person;
```

Um `defer` é um bloco com uma palavra na frente (`ast.DeferExpression` guarda um
`BlockExpression`), então ler `returns` no fim de `ParseBlockExpr` cobre os dois sem caso
especial. O corpo de um `if` não passa por ali — ele é lido com `ParseExprs` direto —, então
`if c { ... } returns P` não é gramática, e é bom que não seja.

### A gramática ganha uma produção, e só

```
block := "{" exprs "}" [ "returns" ID ]
```

É tudo. **Um `if` e um `branch` nunca recebem `returns`** — os corpos deles são lidos com
`ParseExprs` direto (`parser/parser.go`), sem passar por `ParseBlockExpr`, então
`if c { ... } returns P` não é gramática e o erro é o comum, de um token onde se esperava
outro. Não há ambiguidade a resolver porque não há segunda produção.

O que um `if` e um `branch` ganham não é sintaxe, é **serem enxergados**: quando um deles é a
última expressão de um bloco que prometeu, a promessa olha através dele. Isso é regra de
forma, não de gramática, e mora no `shapeOf`.

### E `branch` sai de graça

`branch` não é um nó. `ParseBranch` devolve um `ast.IfExpression` — ele é desaçucarado em
`if`s aninhados no parse, e o último item da lista, o que não tem teste, vira o `Else` mais
interno. Não existe `BranchExpression` na árvore para ninguém percorrer.

Então o que `shapeOf` aprender sobre `if` vale para `branch` sem uma linha a mais. E como um
`branch` sempre termina com aquele item sem teste, ele **sempre tem else** — a regra abaixo
nunca o recusa por falta de saída, só por um dos ramos responder outra coisa.

Nada disso chega ao emitter. Como `struct`, `returns` é declaração: ela é lida, confere,
anota, e morre no fim da compilação. O binário não sabe que existiu.

---

## Onde a anotação mora

Ao lado das outras duas, em `parser.Declarations`, que é a tabela que `struct` e `as` já
deixam para trás:

| Tabela | Chave → valor |
|---|---|
| `Structs` | nome do struct → seus campos |
| `Shapes` | nome de um ident → a struct com que ele é lido |
| **`Returns`** | nome de um ident → a struct que **chamá-lo** responde |

`ident divide = defer { ... } returns Result;` anota `Returns["divide"]`, no mesmo lugar em
que `ident p = Point{1,2}` já anota `Shapes["p"]`. Daí `shapeOf` de uma chamada é uma consulta.

---

## O que ele não resolve

**Um struct continua não atravessando módulo.** `returns` é uma anotação de arquivo, como o
`struct` que ele nomeia — então um escopo importado pode prometer o que quiser, e o arquivo
que importa não tem nome para a forma prometida. Isso é outra conversa, e é a mesma em que a
RFC do `syscall` esbarrou.

**E não é um tipo.** Nada impede um escopo de responder outra coisa em runtime se o compilador
não conseguiu enxergar o que ele faz — `{ f_de_outro_lugar(); } returns Result` passa pela
conferência assim que alguém escrever um `as` lá dentro. A promessa vale o que `shapeOf`
alcança, e isso está escrito aqui de propósito.

---

## As etapas

1. **A palavra-chave e a gramática**: token, tag, leitura no fim do bloco, e a recusa onde não
   é bloco. Sem conferência ainda — o campo entra na árvore e ninguém lê.
2. **`shapeOf` enxerga mais**: bloco, `if`/`else` com os dois ramos concordando, e a
   conferência da promessa, com o erro posicionado.
3. **A tabela `Returns`** e a forma na chamada, que é o que tira o `as` do lugar onde ele era
   repetido.
4. **Documentação e exemplo**: `docs/language-design.md`, ao lado de "A shape coming back out",
   e o roadmap.

Ordem de grandeza: ~250 linhas de código e ~300 de teste, um pull request — ou dois, se a
etapa 2 crescer.

---

## Em aberto

1. **A promessa pode ser exigida?** Hoje um escopo sem `returns` continua legal e quem chama
   escreve `as`. Um dia isso pode virar aviso; não agora.
**Decidido, e medido:** `branch` entra junto com `if` porque **é** `if` — ele vira
`ast.IfExpression` no parse, então não há segunda travessia a escrever. Um `if` sem `else` não
promete nada. E a gramática ganha uma produção só, no bloco.

**Decidido:** o nome é `returns`. `answers` foi considerado e recusado — um escopo está de
fato **devolvendo** um valor ao escopo de fora, e essa é a palavra para isso.
