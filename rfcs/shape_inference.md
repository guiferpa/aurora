# A linguagem sabe qual shape, sem precisar do `returns`

**Estado:** proposta · **Data:** 2026-08-25

Hoje o `returns` é a única fonte para saber qual shape um escopo retorna. Esta RFC diz por que
isso está errado, o que passa a ser descoberto, e o que o `returns` vira.

O vocabulário está em [docs/contributing/architecture.md](../docs/contributing/architecture.md),
seção **The words**: `returns` é o verbo — uma expressão *returns* um valor, um escopo *returns*
um shape —, `shape` é o que é retornado, e `field` é um tape de uma run. Não há segunda palavra
para nada disso.

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
(`parser/shapes.go`), e esse mapa só é preenchido quando o `returns` foi escrito
(`parser/parser.go`). Sem a declaração, a chamada não retorna shape nenhum, e ler um field dela
é recusado.

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

**Descobrir qual shape um escopo retorna é trabalho da linguagem.** O `returns` deixa de ser
fonte e vira **restrição que o compilador confere**.

**O que já existe faz quase tudo.** `answersWith` (`parser/shapes.go`) já percorre um corpo e
calcula qual shape ele retorna, olhando através dos braços de um `if`. Ele só usa isso para
conferir a declaração. Separar esse passeio da checagem é o começo — e o nome dele muda junto,
porque "answers" sai do vocabulário.

**Um shape descoberto atravessa módulo.** `promises()` exporta o que estiver no mapa, e o editor
já lê o que o módulo importado oferece — então descobrir no parser faz o editor conhecer shapes
atravessando arquivo sem que o servidor de linguagem mude uma linha.

**E o preço, dito com todas as letras:** a superfície pública de um módulo passa a ser
descoberta. Mudar a última expressão de um escopo muda, calado, o que quem importa aquele
arquivo enxerga. É o mesmo preço que qualquer linguagem paga por inferência através de
fronteira, e foi escolhido de propósito: a alternativa — exigir `returns` para atravessar —
devolve a obrigação exatamente no caso que motivou esta RFC.

## Declarado ou descoberto viaja junto

O que um escopo retorna é sabido de dois jeitos: foi **declarado** com `returns`, ou foi
**descoberto** do corpo. É o mesmo fato e carrega a mesma palavra — mas qual dos dois viaja ao
lado, num campo.

```go
// Returns is what a scope gives back, and how that is known.
type Returns struct {
    Scope  string
    Shape  string
    Fields []string
    // Declared says the returns was written. Worked out from the body otherwise.
    Declared bool
}
```

Isso não é enfeite. Ele paga em três lugares:

- **O editor pode mostrar a diferença** — um shape que o arquivo garante e um que ele apenas
  acabou retornando não são a mesma promessa a quem lê.
- **O aviso de superfície implícita fica possível sem maquinaria nova**: um escopo do topo que
  retorna um shape sem declarar pode render um aviso, se um dia isso incomodar.
- **A regra do vocabulário sobrevive.** Sem esse campo, alguém acabaria inventando uma segunda
  palavra para "o que foi declarado" — que é exatamente como "promise" nasceu.

## O que o `returns` passa a ser

Uma declaração dirigida a quem lê, que o compilador confere.

```
ident new_square = defer { Square{feed(0), feed(1)}; } returns Square;
```

Continua conferido: um corpo que termina em outra coisa continua sendo recusado, com as mesmas
mensagens. Descobrir **não** pode virar aceitar declaração errada.

O que ele deixa de ser é obrigação para o compilador saber. E ele ganha um lugar onde volta a
ser a única fonte: **quando o destino não estiver no programa**. Chamar outro contrato — a ideia
de `use "<endereço>@<hash>" as c;` — resolve para bytecode, e bytecode não tem nomes. Ali a
declaração é tudo o que existe, o que é consistente com a regra: *o `returns` serve ao que não
dá para resolver em tempo de compilação*.

## Os limites, nomeados

Uma checagem que se cala onde não tem certeza é a regra deste compilador, e não uma falta. Onde
a descoberta se cala:

**Referência para frente.** `ident a = defer { b(); };` escrito antes de `b` não descobre nada,
porque o parser é de uma passada. É o comportamento de hoje, então não é regressão — mas é
limite, e está escrito aqui para ninguém descobrir sozinho.

**Recursão.** Descobrir o que `down` retorna consulta o que `down` retorna, que ainda não
existe, e responde vazio. Não trava: é leitura de mapa, não recursão.

**Um `if` sem `else`.** Já é vazio no `shapeOf`, e é a diferença de contrato entre descobrir e
conferir: para uma declaração, um caminho que retorna nada é **erro**; para a descoberta, é
**desconhecido**.

## O que esta RFC não faz

**O tamanho de uma run para guardá-la contígua na memória.** É pergunta diferente e de outra
fase. "Qual índice é `.width`" é do parser e precisa do nome do shape; "quantos tapes tem a run"
precisa só da contagem de fields, e o builder já resolve o destino de toda chamada
estaticamente — o `Entry` de cada escopo já carrega quantas posições ele lê, e ganharia quantos
tapes ele retorna. Isso vai na RFC do `Effect`, com shape na memória como caso motivador.

**Fazer o editor parar de varrer tokens.** Ele passa a ler o que o parser sabe, e o passeio de
tokens fica como reserva — porque autocompletar tem que funcionar **enquanto se digita**, e
documento meio escrito quase nunca parseia. O que muda é o papel: deixa de ser a única fonte.

## As etapas

1. **O vocabulário.** `returns` em tudo, "answers" e "promise" fora, `ast.Returns` no lugar de
   `ast.Promise`. Renomeação pura, por camada, cada uma com `make check`.
2. **Um passeio só.** Separar da checagem a função que descobre qual shape um corpo retorna.
   Nada muda para quem escreve Aurora; os testes que existem são a prova.
3. **Descobrir o que um escopo retorna**, quando ele não declarou, e carregar `Declared` junto.
   Puramente aditivo: nada que compila deixa de compilar, e passa a compilar o que hoje é
   recusado.
4. **O editor lê o que a linguagem sabe**, com o passeio de tokens como reserva.

## Em aberto

1. **Um aviso para superfície implícita.** Fica de fora agora porque acrescenta ruído antes de a
   gente saber se incomoda — e com `Declared` no lugar, é uma linha quando incomodar.
2. **Ligar duas vezes.** Um nome ligado a dois `defer` diferentes: o segundo sobrescreve. Hoje é
   assim para o `returns` também, então nada muda — mas é um lugar onde descobrir amplia o
   alcance de um comportamento que ninguém decidiu.
