# O IR diz o que uma instrução faz além de deixar um valor

**Estado:** proposta · **Data:** 2026-08-26

Duas coisas travam no mesmo lugar: um shape de cinco campos não chega ao bytecode, e storage
não pode ser escrito. Esta RFC diz por que são a mesma coisa, e em que ordem sair delas.

O vocabulário está em [docs/contributing/architecture.md](../docs/contributing/architecture.md),
seção **The words**. O IR está em [docs/contributing/ir.md](../docs/contributing/ir.md), e
[why-blocks.md](../docs/contributing/why-blocks.md) ao lado diz por que ele tem essa forma.

---

## O buraco, que existe e ninguém caiu nele ainda

O builder **reordena instruções**. `ResolveOperandsOrder` (`builder/evm/lowering.go`) segura
uma instrução e a solta ao lado de quem consome o valor dela, porque a pilha da EVM quer os
operandos na ordem certa e o programa foi escrito em outra.

Isso é correto para uma instrução cujo resultado inteiro é o valor que ela deixa. Não é
correto para uma que **faz** alguma coisa:

```
printd a;
printd b;
```

Mover a segunda para antes da primeira muda o que o programa diz. Nada no IR impede.

O que segura hoje são três listas de opcodes em `builder/evm` — `passesThrough`,
`ordersItsOwn` e `produces` — e uma lista de opcodes é **a forma de errar deste
repositório**. Foi assim que um `ident` dentro de escopo quebrou: a lowering decidia quais
operandos nomeavam valores por uma lista dos quatro aritméticos, e um nome ligado dentro de
um escopo não estava nela. `ident x = feed(0); x + feed(1);` respondia 4 na chain onde o
programa responde 7. Quem escreve o opcode seguinte tem que lembrar de três listas, e não há
nada que o lembre.

**E ninguém caiu no buraco ainda**, por um acidente que vale dizer em voz alta: a única
instrução cuja ordem importa hoje é o `print`, e o `print` é justamente a que o backend
recusa a escrever — por decisão, porque uma chain não tem onde pôr um log. Então o backend
que reordena nunca viu uma instrução que não podia mover.

Storage põe uma lá. `s.set("key", v)` seguido de `s.get("key")` é a primeira dupla em que a
ordem é o programa inteiro, e ela chega ao bytecode. **Por isso o `Effect` vem antes do
storage, e não depois.**

---

## O `Effect`

Já foi desenhado na [ir.md](ir.md) com dois valores, e a questão em aberto número 2 de lá
dizia que leitura de estado não é nenhum dos dois e que valia fixar o conjunto com storage
junto. É o que esta faz.

```go
// Effect diz o que uma instrução faz além de deixar um valor.
type Effect byte

const (
	Pure    Effect = iota // o valor que ela deixa é tudo que ela faz
	Reads                 // depende de estado que outra coisa pode mudar
	Writes                // muda esse estado, ou diz algo cuja ordem importa
	Escapes               // sai do programa: outro contrato roda, e pode voltar para dentro
)
```

**A regra é uma linha:** duas instruções trocam de lugar a menos que uma delas mude algo que
a outra perceberia.

Ela começou mais estrita — *só trocam se pelo menos uma for `Pure`* — e a medição descrita
mais abaixo disse que estrita demais. Uma subtração de dois nomes, `a - b`, põe as duas
leituras na pilha na ordem que a subtração quer, o que troca dois `Reads` de lugar. Isso é
seguro (duas leituras do frame comutam) e a regra estrita recusava, no programa mais ordinário
que existe.

Então os dezesseis pares **saem** de duas perguntas em vez de serem listados:

```go
func MayCross(a, b Effect) bool { return !disturbs(a, b) && !disturbs(b, a) }

func disturbs(a, b Effect) bool { return changes(a) && notices(b) }

func changes(e Effect) bool { return e == Writes || e == Escapes }
func notices(e Effect) bool { return e != Pure }
```

Custa quatro linhas a mais que a versão estrita e responde toda célula, inclusive as que nada
alcança ainda. Uma matriz de quatro por quatro escrita à mão é que seria cara: um mantenedor
tem que segurar dezesseis casos na cabeça, e ninguém confere se eles concordam entre si.

Os outros degraus não são para a regra, são para **o que um backend tem que dizer**. Um
`Escapes` é o que faz o backend precisar de reentrância; um `Reads` é o que o evaluator tem
que simular para a promessa se manter. Um backend que não carrega um efeito recusa nomeando
ele, do jeito que `builder/evm/support.go` já recusa nomeando um opcode. E fixar os quatro
agora é o que evita renomear `Ordered` para `Writes` quando o storage chegar — o custo que a
ir.md nomeou.

**O efeito é do opcode, não da instrução.** A ir.md tinha posto um campo em `Instruction`.
Um campo é preenchido por quem constrói a instrução, então pode ser preenchido errado e nada
percebe; um opcode tem um efeito onde quer que ele apareça. Dizer uma vez é menos linha para
ler e é a única versão que não pode discordar de si mesma. No dia em que o efeito de uma
instrução depender de mais que o opcode dela, o campo chega e tem motivo.

`require` **não entra**. Ele não é instrução: é controle, e controle já mora fora das
instruções desde que o bloco tem terminador. Um `require` é um `BrIf` para um bloco que
reverte, e é por isso que ele não precisa de efeito nenhum — o mesmo argumento que a ir.md
usou para tirar o desvio de dentro da instrução.

---

## O shape de cinco campos

Off-chain um run é uma fatia de bytes de qualquer tamanho (`EvaluateJoinOver`). On-chain é
**uma palavra da pilha** (`WriteJoin`), e uma palavra tem 32 bytes. No tape padrão de oito
isso é quatro campos, e o quinto é recusado:

```
a run of 5 tapes is 40 bytes and a word is 32: a shape this wide does not reach the bytecode
```

Recusar é melhor do que escrever curto, e ainda assim é um programa que roda no evaluator e
não pode ir para uma chain — que é exatamente a promessa que o projeto existe para manter.

**Salvar o run contíguo na memória resolve**, e a pergunta difícil não é essa: é o que um
valor passa a ser. Se um run vive na memória, o valor que o nomeia é um endereço, e a
linguagem não tem tipo — nada na palavra da pilha distingue um tape de oito bytes de um
endereço de memória.

Marcar o valor com um bit custa uma máscara em toda aritmética, e o custo é em todo programa
para pagar o caso de cinco campos. **A alternativa é decidir estaticamente**, e ela ficou
barata agora: o parser já sabe qual valor é um run — é o `shapeOf`, e desde a
`shape_inference` ele sabe em mais lugares do que sabia — um bloco, um `if` cujos braços
concordam, uma chamada a um escopo que não declarou nada, e tudo isso atravessando módulo. O que falta é o IR **dizer**: um operando que é um run
carrega quantos tapes ele tem, e o backend lê em vez de adivinhar — que é a mesma frase que
resolveu o `ident` dentro de escopo.

Um run na memória é escrita de memória, e é aqui que as duas metades desta RFC se encontram:
`OpJoin` deixa de ser `Pure` no momento em que passa a escrever, e sem o `Effect` o builder
tem todo o direito de movê-lo.

---

## O que isso destrava

Storage como um mapa nativo, do jeito que foi pensado:

```
use storage as s;

s.set("key", value);
s.get("key");
```

`set` é `Writes`, `get` é `Reads`, e a regra de uma linha já basta para o backend nunca
inverter os dois. O evaluator simula com um mapa em memória, e a promessa se mantém: o mesmo
programa responde a mesma coisa dentro e fora da chain.

Isso é a RFC seguinte, não esta.

---

## Entrega

1. **`Effect` na instrução, com todo mundo `Pure` menos os três `print`.** Puramente aditivo:
   nada muda de comportamento porque nada que não é `Pure` chega ao backend hoje. O que muda é
   que a lowering passa a perguntar o efeito em vez de consultar lista, e as três listas de
   opcodes de `builder/evm` encolhem para o que é de fato sobre a pilha.
2. **A regra, medida antes de prender.** Um teste que roda a regra sobre o que a lowering
   produz hoje, para todo programa do corpus, e diz o que ela recusaria. Ele é o que decide se
   o item 1 foi aditivo, e continua valendo depois como o teste que prova que a regra é
   cumprida — sem ele, o item 1 é uma anotação em que ninguém confia.
3. ~~O IR diz quantos tapes um operando tem, quando ele é um run.~~ **O bloco diz quantos
   tapes ele retorna.** A forma estava errada, e escrever mostrou por quê: a largura já viaja
   em dois dos três lugares. Uma construção diz a dela por **ter aquele tanto de operandos**, e
   uma leitura de campo carrega a largura da run de onde ela sai — é o terceiro operando do
   `OpField`, e ele existe exatamente por isto. O único lugar em que nada dizia é **o valor que
   volta de uma chamada**, e isso é fato do escopo, não de nenhuma instrução dentro dele. Então
   é `ir.Block.Tapes`, e não um operando.

   Vale dizer de onde vem: o emitter lê `tree.Returns`, que é o que a `shape_inference` acabou
   de encher. Antes dela isso valeria só para os escopos que escreveram `returns`; agora vale
   para quase todos, sem o arquivo dizer nada.
4. **O run vai para a memória**, e um campo é lido de lá. O teto continua onde está e nada
   muda de comportamento: o que muda é onde a run mora. O harness é o juiz — tudo que passa
   hoje tem que continuar passando.
5. **O teto sai.** O retorno passa a devolver `Tapes * tape_size` bytes em vez de uma palavra,
   a recusa do `WriteJoin` some, e o harness prova um shape de cinco campos e um de oito.

O 4 e o 5 eram um só, e são dois porque o primeiro é uma troca de mecanismo com prova pronta
(todo shape que já funciona continua funcionando) e o segundo é o que muda o que a linguagem
alcança. Um de cada vez é o que deixa o harness dizer qual dos dois quebrou.

Um a um, cada um passando `make check` sozinho.

---

## O frame é estado

A pergunta foi feita com um dos nomes errados, e a resposta corrige junto. `OpSave` não mexe
no frame: ele é `Imm -> the bytes themselves`, que é como um literal chega ao programa, e isso
é `Pure`. Quem escreve no frame é o `OpIdent` — `Name, Ref -> binds the name to the value`.

Então, decidido:

| opcode | efeito | porquê |
|---|---|---|
| `OpIdent` | `Writes` | escreve o frame com um `MSTORE` |
| `OpLoad` | `Reads` | lê o frame com um `MLOAD` |
| `OpSave` | `Pure` | é um literal, e não toca em nada |

**E isso pode custar alguma coisa, ao contrário dos `print`.** Um `print` nunca chega ao
backend que reordena, então prendê-lo é grátis. Um `OpIdent` e um `OpLoad` chegam, e os dois
são movidos hoje: o `OpLoad` porque `produces(op)` o segura para soltar ao lado de quem
consome, e o `OpIdent` porque a lowering o segura quando ele é a última expressão de um
escopo.

A regra de uma linha proíbe trocar um `Reads` com um `Writes`. Se a lowering hoje faz isso em
algum programa, o item 1 da entrega **deixa de ser aditivo** — e essa é a única parte desta
RFC que não dá para decidir lendo o código, porque depende de qual ordem cada programa real
produz.

Por isso a entrega é nesta ordem, e não em outra: o item 2 existe para **medir antes de
prender**.

**A medição foi feita, e a resposta foi "uma coisa".** A lowering troca dois `OpLoad` de lugar
numa subtração de dois nomes, e essa é a ordem certa — a pilha da EVM quer os operandos
assim. Não era a lowering que estava errada: era a regra estrita, que proibia dois `Reads` de
comutarem. Com a regra derivada acima, todo programa do corpus passa, e o item 1 **é** aditivo.

O teste ficou (`hosting/cli/effect_test.go`), e agora prova a regra em vez de medi-la. Ele
carrega um guarda que vale registrar: um caso cujo programa só tem instrução `Pure` passaria
sem nunca perguntar nada à regra, então ele conta quantas não são e falha se forem menos de
duas. Esse guarda já pegou um caso meu que não provava nada.

---

## Onde a run vai morar

As duas primeiras questões em aberto eram esta, e a resposta é uma só. Ela vai escrita antes
de qualquer byte porque o `CLAUDE.md` diz que a vez do `builder/evm` se discute antes.

**Onde ela mora: no frame de quem a construiu, empacotada.** `tape_size` bytes por tape e não
uma palavra por tape — uma palavra por tape seria mais fácil de escrever e devolveria bytes
diferentes dos que o evaluator devolve, que é a única coisa que este backend não pode fazer.

**Quanto espaço: reservado estaticamente, como um nome.** Quantos `OpJoin` um escopo tem e a
largura de cada um são as duas coisas sabidas na hora de escrever — a largura é o número de
operandos do próprio join. Então cada join ganha a sua região no frame, o `Scope.Frame` cresce
pela soma, e não existe alocador, nem liberação, nem nada em tempo de execução decidindo
endereço. Uma run é uma alocação estática do mesmo naipe que um nome ligado.

**O que o valor vira: o endereço da região.** E nada na palavra da pilha distingue isso de um
tape — nem precisa. Quem sabe é o IR, estaticamente: o valor de um join é uma run porque o
join é um join, e o valor que volta de uma chamada é uma run porque `ir.Block.Tapes` diz. Foi
para isso que o passo 3 existiu.

**Tempo de vida: quem chamou copia.** Uma run só sobrevive ao escopo que a construiu sendo
retornada, e aí quem chamou copia `Tapes * tape_size` bytes para o frame dele **antes** de
mover o ponteiro de frame de volta. Isso é o que a convenção de chamada da
[if_and_call.md](if_and_call.md) já faz com valor, com a diferença de que agora tem tamanho,
e o tamanho está no bloco do chamado — que quem chama já resolve estaticamente pelo `Entry`.

**Atravessando a chain: `RETURN` de `Tapes * tape_size` bytes** a partir do endereço da run,
em vez de uma palavra do `RETURN_SCRATCH`. E o harness continua valendo sem ser mexido, o que
vale explicar porque não é óbvio: ele lê o que voltou com `SetBytes`, ou seja, como um número.
Hoje uma run de três tapes volta como uma palavra de 32 bytes com 24 bytes úteis no fim, e o
evaluator devolve os 24 — mesmo número. Devolvendo `Tapes * tape_size`, os bytes passam a ser
**os mesmos**, e não apenas o mesmo número. A comparação fica mais forte do que era.

---

## O que ficou decidido, e onde

Nada ficou em aberto. As três perguntas com que esta RFC abriu foram respondidas, duas delas
por medição e não por opinião:

| pergunta | resposta | onde |
|---|---|---|
| onde a run mora, e por quanto tempo | no frame de quem construiu, empacotada, e quem chama copia | acima, "Onde a run vai morar" |
| o que uma run que atravessa a chain é | `RETURN` de `Tapes * tape_size` bytes, os mesmos que o evaluator devolve | idem |
| se `OpLoad` e `OpIdent` são `Reads`/`Writes` | são: o frame é estado | acima, "O frame é estado" |

E duas coisas mudaram no caminho por terem sido medidas em vez de assumidas:

- **A regra era estrita demais.** Rodada sobre programas reais, ela recusava `a - b`, onde a
  lowering troca dois `Reads` de lugar porque a pilha da EVM quer os operandos assim. Duas
  leituras do frame comutam; era a regra que estava errada, não a lowering.
- **O passo 3 estava com a forma errada.** "O IR diz quantos tapes um operando tem" — mas a
  largura já viajava em dois dos três lugares, e o que faltava era fato do bloco. Virou
  `ir.Block.Tapes`.
