# A linguagem sabe qual shape, sem precisar do `returns`

**Estado:** proposta · **Data:** 2026-08-25

Hoje o `returns` é a única fonte para saber o que um escopo responde. Esta RFC diz por que isso
está errado, o que passa a ser inferido, e o que o `returns` vira.

> Os blocos aqui **não** estão marcados como `aurora`: `hosting/cli/docs_test.go` executa todo
> bloco assim marcado no repositório, e o primeiro exemplo abaixo não compila hoje — que é
> justamente o ponto.

---

## O que acontece hoje

```
shape Square { width, height };
ident new_square = defer { Square{feed(0), feed(1)}; };
ident s = new_square(30, 20);
printd s.width;
```

```
cannot read field width at line 4 and column 10: nothing says which shape this value is,
name it with 'as'
```

Acrescente `returns Square` ao `defer` e o mesmo programa responde 30.

O caminho é este: `shapeOf` para uma chamada lê `declarations.Returns[nome]`
(`parser/shapes.go`), e `Returns` só é preenchido quando o `returns` foi escrito
(`parser/parser.go`). Sem promessa, a chamada não tem shape, e ler um campo dela é recusado.

O compilador **poderia** saber. O corpo do escopo termina em `Square{...}`, ali, no mesmo
arquivo. Ele simplesmente não olha.

## E tem um segundo problema, mais fundo

O editor não usa o que a linguagem sabe. O servidor de linguagem monta a **própria** tabela de
shapes varrendo o fluxo de tokens (`hosting/lsp/textdoc/shapes.go`, `scan`), casando padrões:

```
shape N { ... }
ident p = N{
ident p = ... as N
ident f = defer { ... } returns N
```

É uma reimplementação do `shapeOf`, e mais fraca: o `shapeOf` resolve um bloco pela última
expressão, um `if` cujos braços concordam, e uma cadeia de ligações. O `scan` não resolve
nenhum dos três.

São **duas descrições da mesma coisa**, e duas descrições divergem — que é o modo de falhar que
este repositório vem encontrando de outras formas o ano todo. A diferença é que aqui a
descrição fraca é justamente a que a pessoa vê enquanto escreve.

## A decisão

**Inferir qual shape é trabalho da linguagem.** O `returns` deixa de ser fonte e vira
**restrição que o compilador confere**.

Três consequências, e a terceira é a que custa:

**O que já existe faz quase tudo.** `answersWith` (`parser/shapes.go`) já percorre um corpo e
calcula o shape da última expressão, olhando através dos braços de um `if`. Ele só usa isso
para conferir uma promessa. Separar esse passeio da checagem é o começo.

**Um shape inferido atravessa módulo.** `promises()` exporta o que estiver em `Returns`, e o
editor já lê `Tree.Promises` do módulo importado — então inferir no parser faz o editor
conhecer shapes atravessando arquivo sem que o servidor de linguagem mude uma linha.

**E o preço, dito com todas as letras:** a superfície pública de um módulo passa a ser
inferida. Mudar a última expressão de um escopo muda, calado, o que quem importa aquele arquivo
enxerga. É o mesmo preço que qualquer linguagem paga por inferência através de fronteira, e foi
escolhido de propósito: a alternativa — exigir `returns` para atravessar — devolve a obrigação
exatamente no caso que motivou esta RFC.

## O que o `returns` passa a ser

Uma declaração dirigida a quem lê, que o compilador confere.

```
ident new_square = defer { Square{feed(0), feed(1)}; } returns Square;
```

Continua conferido: um corpo que termina em outra coisa continua sendo recusado, com as mesmas
mensagens (`brokenPromise`, `describeAnswer`). Inferir **não** pode virar aceitar promessa
errada.

O que ele deixa de ser é obrigação para o compilador saber. E ele ganha um lugar onde volta a
ser a única fonte: **quando o destino não estiver no programa**. Chamar outro contrato — a
ideia de `use "<endereço>@<hash>" as c;` — resolve para bytecode, e bytecode não tem nomes. Ali
a promessa é tudo o que existe, o que é consistente com a regra: *o `returns` serve ao que não
dá para resolver em tempo de compilação*.

## Os limites, nomeados

Uma checagem que se cala onde não tem certeza é a regra deste compilador, e não uma falta. Onde
a inferência se cala:

**Referência para frente.** `ident a = defer { b(); };` escrito antes de `b` não infere, porque
o parser é de uma passada. É o comportamento de hoje, então não é regressão — mas é limite, e
está escrito aqui para ninguém descobrir sozinho.

**Recursão.** Inferir `down` consulta o que `down` responde, que ainda não existe, e responde
vazio. Não trava: é leitura de mapa, não recursão.

**Um `if` sem `else`.** Já é vazio no `shapeOf`, e é a diferença de contrato entre inferir e
conferir: para a promessa, um caminho que responde nada é **erro**; para a inferência, é
**desconhecido**.

## O que esta RFC não faz

**O tamanho de uma run para guardá-la contígua na memória.** É pergunta diferente e de outra
fase. "Qual índice é `.width`" é do parser e precisa da identidade do shape; "quantos tapes tem
a run" precisa só da contagem de campos, e o builder já resolve o destino de toda chamada
estaticamente — o `Entry` de cada escopo já carrega quantas posições ele lê, e ganharia quantos
tapes ele responde. Isso vai na RFC do `Effect`, com shape na memória como caso motivador.

**Fazer o editor parar de varrer tokens.** Ele passa a ler o que o parser sabe, e o passeio de
tokens fica como reserva — porque autocompletar tem que funcionar **enquanto se digita**, e
documento meio escrito quase nunca parseia. O que muda é o papel: deixa de ser a única fonte.

## As etapas

1. **Um passeio só.** Separar de `answersWith` a função que responde qual shape um corpo
   responde. Nada muda para quem escreve Aurora; os testes que existem são a prova.
2. **Inferir o que um escopo responde**, quando ele não prometeu. Puramente aditivo: nada que
   compila deixa de compilar, e passa a compilar o que hoje é recusado.
3. **O editor lê o que a linguagem sabe**, com o passeio de tokens como reserva.

## Em aberto

1. **Um aviso para superfície implícita.** Um escopo do topo que responde um shape sem declarar
   `returns` podia render um aviso — empurrando para superfície explícita sem obrigar. Fica de
   fora agora porque acrescenta ruído antes de a gente saber se incomoda.
2. **Ligar duas vezes.** Um nome ligado a dois `defer` diferentes: o segundo sobrescreve. Hoje
   é assim para o `returns` também, então nada muda — mas é um lugar onde inferir amplia o
   alcance de um comportamento que ninguém decidiu.
