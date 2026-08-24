# `if` e `call` em bytecode

**Estado:** implementada · **Proposta:** 2026-08-20 · **Fechada:** 2026-08-24
· PRs [#111](https://github.com/guiferpa/aurora/pull/111),
[#113](https://github.com/guiferpa/aurora/pull/113),
[#115](https://github.com/guiferpa/aurora/pull/115),
[#119](https://github.com/guiferpa/aurora/pull/119)

As duas instruções que faltavam para um contrato deixar de ser uma calculadora. As duas
compilam hoje, e com elas o backend passou a carregar a linguagem inteira.

O texto abaixo da linha é a proposta como foi escrita, e fica como registro do que se pensava
antes de escrever o código. **O que o código respondeu está aqui em cima**, porque quem lê uma
RFC implementada quer primeiro saber o que ficou de pé.

## O que ficou diferente do proposto

**Medir é escrever para algo que só conta.** A proposta dizia "percorrer as instruções e anotar
quantos bytes cada uma ocupa" — uma tabela de tamanhos ao lado do escritor. O que foi escrito
mede escrevendo, num `io.Writer` que só soma o que recebe (`counter`, `PositionsOf`). Uma
tabela é uma segunda descrição dos mesmos bytes, e duas descrições de uma coisa divergem: este
backend já errou assim mais de uma vez, e o erro é calado porque os bytes saem, só que nos
endereços errados.

**Quem move o ponteiro de frame é quem chama, e move pelo tamanho do frame *dele mesmo*.** A
proposta via duas opções e preferia a segunda — quem chama mover, precisando saber o tamanho do
frame de quem chama; ou o corpo mover, e quem chama não precisar saber. A terceira é melhor que
as duas: quem chama soma o **próprio** tamanho, então nenhum dos lados precisa saber o do
outro. Voltar é subtrair o mesmo número, e não uma cópia guardada — uma cópia teria que ficar
em algum lugar durante a chamada, e o único lugar é a pilha, embaixo de uma resposta que o
trabalho do chamado empilha por cima.

**Um escopo é escrito uma vez e entrado duas.** A proposta dizia que "o dispatcher vira um
chamador". O que ficou põe o prólogo **dentro do escopo**, com dois pontos de entrada e um
epílogo:

```
external:  JUMPDEST          <- o dispatcher salta aqui
           <prólogo>         copia a calldata para o frame
           PUSH2 <epílogo>
           PUSH2 <internal>
           JUMP
epílogo:   JUMPDEST          <- o corpo volta aqui, valor na pilha
           <responde a cadeia>
internal:  JUMPDEST          <- outro escopo salta aqui, frame já escrito
           <corpo>           feed(n) lê o frame, seja quem for que o encheu
           SWAP1; JUMP       volta para quem chamou
```

O efeito é o que a proposta queria — o corpo tem uma forma só, e nada nele sabe se veio de uma
transação ou de outro escopo — mas o `RETURN` da EVM não fica no dispatcher: fica no epílogo do
próprio escopo. Terminar um escopo é **sempre** voltar para quem chamou.

**Só escopo ligado no topo do programa pode ser chamado.** Não estava na proposta e virou
limite: um escopo escrito dentro de outro não é escrito de jeito nenhum (o nome guarda o valor
neutro), e chamar um é recusado em vez de virar salto para um endereço que escopo nenhum tem.
Essa recusa é o que torna o nome de uma chamada resolvível em tempo de compilação, e é dela que
sai a resposta da pergunta 5.

**Recursão e recursão mútua saem de graça, e agora estão provadas.** `hosting/cli/evm_harness_test.go`
tem as duas contra o evaluator.

## As perguntas em aberto, respondidas

**1 · Onde mora o ponteiro de frame.** Em `0x00`, e o conflito com o `RETURN` foi resolvido
dando à resposta um slot só dela (`RETURN_SCRATCH`, `0x20`). Os frames começam em `0x40`.
Escrever a resposta no slot do ponteiro perderia o frame no ato de responder.

**2 · Quem soma o ponteiro.** Quem chama, pelo tamanho do próprio frame. Ver acima.

**3 · O `OpReturn` com dois sentidos.** Lido da estrutura, sem mexer no IR: um braço nomeia o
`OpIf` a que pertence, e um escopo nomeia o `OpBeginScope`. E apareceu um **terceiro** sentido
que a proposta não previu — o código que escopo nenhum segura, o topo de um programa, que não
tem para quem voltar e por isso encerra a chamada. Esse não dá para ler da instrução, então é
parâmetro.

**4 · `OpPreCall`.** Saiu da lista. Os argumentos chegam à pilha pelo lowering, que os põe
imediatamente antes da chamada, e é a própria chamada que os escreve no frame — não sobrou
lugar para ele. Todo opcode que o IR declara agora tem operação atrás de si nos dois
consumidores.

**5 · Quem zera as posições que ninguém escreveu.** Quem chama, e sem nenhuma das duas saídas
que a proposta via como únicas. Ela achava que quem chama não pode saber quantas posições o
corpo lê, "porque um `defer` é valor de primeira classe e pode ser chamado através de qualquer
nome" — mas com a recusa acima o nome resolve em tempo de compilação, e o escopo nomeado diz
quantas lê, que é o mesmo número que o prólogo dele já usa. Nada viaja no frame e nenhum teto é
zerado: quem chama escreve exatamente os lugares entre o que aplicou e o que o chamado lê, que
é nada quando a chamada não é curta.

Isso não foi visto quando o `call` foi escrito. Um escopo lendo duas posições, chamado com uma,
lia o segundo valor da chamada anterior — 501 onde o programa responde 301 — e só apareceu ao
reler estas perguntas contra o código. A pergunta estava certa antes de o código existir.

**6 · Recusar em vez de truncar.** Feito: acima de 24.576 bytes o builder recusa.

## O que ficou por fazer

**O `StackDepth` não virou recusa, e foi removido.** A proposta dizia que ele provaria que os
dois braços de um desvio deixam a mesma pilha, "e vira uma recusa de compilação quando ela não
valer". Nunca virou: nenhum código o chamava, só os testes. E ao olhar de perto ele estava
**errado** — `produces` não lista `OpCall`, então ele contava toda chamada como quem come os
argumentos e não deixa nada, e cada chamada afundava a conta em um. Ninguém percebeu porque
ninguém o chamava.

Uma função errada e sem uso é pior que qualquer uma das duas coisas sozinha: ela parece uma
garantia. E o desenho dela também não dava para o que foi prometido — ela anda em linha reta,
então num `if` conta os dois braços em sequência, que é justamente o que ela deveria provar não
acontecer.

A invariante continua verdadeira **por construção**: o emitter não produz desvio desbalanceado.
Quando isso deixar de ser óbvio, o que entra é uma verificação que conhece caminhos, e não esta.

**Static link.** Um `defer` que lê nome do escopo onde foi escrito continua de fora, como a
proposta já dizia. Hoje é mais do que de fora: um escopo escrito dentro de outro não é escrito.

**`assert` → `REVERT`.** Continua de fora.

---


## O que já está de pé

> A partir daqui, o texto é a proposta de 2026-08-20, sem retoque.

Duas coisas precisam existir antes, e uma delas já foi escrita.

**A tabela de pilha.** O lowering deixou de casar padrão e passou a dizer, por opcode, o que
ele toma e o que deixa (`consumes`/`produces`, e o `StackDepth` que sai delas). É o que fez um
`ident` dentro de escopo funcionar, e é o que torna um salto **escrevível**: os dois lados de
um desvio têm que se encontrar com a mesma pilha embaixo, e sem essa tabela não há como
afirmar isso. Está pronto no branch `feat/a-stack-model-in-the-lowering`, verde, esperando
esta RFC.

**`PUSH2` no lugar de `PUSH1`** para offset de memória, alvo de salto e tamanho do runtime.
Hoje os três truncam depois de 255. E o teto novo é o último: um contrato publicado não passa
de 24.576 bytes (EIP-170), então dois bytes cobrem **todo contrato legal** — não existe um
terceiro tamanho depois deste.

O `PUSH2` fixo também é o que mantém a montagem simples: com todo push do mesmo tamanho, o
tamanho de cada instrução é conhecido **antes** dos endereços, e uma passada de medição
resolve os offsets sem iterar até estabilizar.

---

## A montagem em duas passadas

Hoje o builder escreve bytes direto, e o dispatcher calcula offset com uma constante
(`DISPATCHER_BYTES_SIZE`). Salto para frente não cabe nisso: um `if` precisa do endereço do
`else` antes de ter escrito o `else`.

Proposta: **medir, depois escrever.**

1. **Medir** — percorrer as instruções e anotar quantos bytes cada uma ocupa, sem escrever
   nada. Com push de tamanho fixo, isso é uma soma.
2. **Resolver** — a posição de cada instrução é a soma das anteriores, e um alvo de salto é a
   posição da instrução que ele nomeia.
3. **Escrever** — a mesma passada de hoje, com os endereços em mãos.

Nada disso é novo em compilador; é o que qualquer assembler faz. O que ele desbloqueia é
salto para frente, que é `if`, e chamada de escopo declarado depois, que é recursão mútua.

---

# `if`

## A forma que o IR já tem

```
OpIf     (label=inl, left=teste, right=quantas instruções pular)
...corpo do então...
OpReturn (left=inl, right=valor do então)
OpJump   (left=quantas instruções pular)
...corpo do senão...
OpReturn (left=inl, right=valor do senão)
```

Duas coisas para reparar, porque as duas mudam o desenho:

**O `right` do `OpIf` é uma contagem de instruções, não de bytes.** É o que o evaluator
precisa; o builder precisa de endereço. A passada de medição é o tradutor.

**O `OpReturn` aqui não é o `RETURN` da EVM.** Ele é o mesmo opcode que fecha um escopo, e o
builder hoje escreve `MSTORE; PUSH 0x20; PUSH 0x00; RETURN` para ele — o que encerraria a
chamada inteira no fim do primeiro braço. Dentro de um `if`, `OpReturn` significa **"o valor
deste braço é X"**, e não "responda X para quem chamou".

## O bytecode

```
        <teste>            ; deixa 1 valor
        ISZERO             ; o OpIf pula quando o teste é falso
        PUSH2 <senão>
        JUMPI
        <então>            ; deixa 1 valor
        PUSH2 <fim>
        JUMP
<senão> JUMPDEST
        <senão>            ; deixa 1 valor
<fim>   JUMPDEST
```

`JUMPI` tira `[destino, condição]` e desvia quando a condição não é zero; o `ISZERO` é o que
casa isso com a semântica do IR, que pula quando o teste é falso.

**Os dois braços deixam exatamente um valor**, e é isso que faz o `if` ser expressão: quem
está embaixo dele encontra um valor na pilha, sem saber por onde o programa passou. O
`StackDepth` é o que prova essa afirmação, e vira uma recusa de compilação quando ela não
valer.

**Um `if` sem `else` deixa o valor neutro**, que é o que o evaluator responde. O braço vazio é
um `PUSH` de zero, e não um caso especial.

`JUMPDEST` é obrigatório: a EVM recusa um salto para qualquer byte que não seja um. Ele custa
1 gas e um byte, e é o preço de existir desvio.

---

# `call`

É o mais caro dos dois, e o único que precisa de convenção nova.

## Por que a forma de hoje não estica

Três coisas, e cada uma sozinha já quebra recursão:

**Um `defer` só é alcançável de fora.** Cada um vira uma entrada do dispatcher, com um
`JUMPDEST` no topo, e nada no contrato salta para ele.

**`feed(n)` lê da calldata.** Numa chamada interna não há calldata: os argumentos são valores
que quem chamou acabou de calcular.

**Os locais têm endereço fixo por nome.** O `IdentManager` mapeia nome → offset de memória,
uma vez, para o contrato inteiro. Duas chamadas do mesmo escopo escrevem no mesmo lugar — a de
dentro sobrescreve a de fora, e a de fora continua com o valor errado. **É isso que impede
recursão**, não a falta de um `JUMP`.

## A convenção

**Um frame por ativação.** Um slot fixo de memória guarda o **ponteiro de frame**; cada escopo
sabe, em tempo de compilação, quantos slots ele quer. Entrar é somar, sair é subtrair.

O tamanho do frame **não pode sair da quantidade de argumentos**, e é fácil escrever isso
errado: um escopo da Aurora não tem aridade, então quantos valores chegam é propriedade de quem
chamou e muda de sítio para sítio. Se um local morasse depois do último argumento, ele trocaria
de endereço conforme a chamada, e o corpo tem um endereço só, compilado uma vez.

O que o corpo **sabe** de si mesmo é quantas posições ele lê: o maior índice de `feed` que
aparece nele, mais um. É desse número que sai o layout, e ele é constante de compilação.

| | onde |
|---|---|
| ponteiro de frame | um slot de memória fixo, escolhido uma vez |
| argumento `n` | `frame + n`, para `n` abaixo do que o corpo lê |
| local | `frame + (quantas posições o corpo lê) + índice` |
| endereço de retorno | na pilha da EVM |

**Uma posição que ninguém escreveu tem que ler zero.** É a regra da linguagem desde a #92:
ler além do que foi aplicado responde o tape neutro. Na calldata isso é de graça, porque a EVM
já devolve zeros além do fim. No frame não é: o ponteiro sobe e desce, então a memória de uma
ativação já serviu a outra, e um argumento que não foi enviado leria o que sobrou lá.

**Chamar:**

```
        <argumentos>       ; um valor cada, na ordem
        ...escreve cada um em frame_novo + i
        PUSH2 <volta>      ; endereço de retorno
        PUSH2 <corpo>
        JUMP
<volta> JUMPDEST
```

**Voltar:** o corpo termina com um valor na pilha e o endereço de retorno embaixo dele —
`SWAP1`, `JUMP`. O valor fica no topo para quem chamou, que é exatamente o que uma expressão
tem que deixar.

**Recursão sai de graça disso.** Cada ativação soma o ponteiro antes de escrever os
argumentos, então a chamada de dentro não vê os locais da de fora. O fundo é o gas e o custo
de memória, que cresce quadraticamente — não é um limite de profundidade escrito em lugar
nenhum, e não deve ser.

## O dispatcher vira um chamador

Aqui está a parte que simplifica em vez de complicar.

Se um corpo só sabe ler argumento do frame, então quem chama de fora precisa montar um frame
também. Então o dispatcher deixa de saltar direto para o corpo e passa a fazer o que qualquer
chamador faz: **copia os argumentos da calldata para o frame, empurra o endereço de retorno, e
salta.** O retorno dele é o único lugar onde o `RETURN` da EVM aparece.

Com isso **o corpo tem uma forma só** — nada nele sabe se veio de uma transação ou de outro
escopo — e o `feed(n)` tem um lowering só.

---

## O que fica de fora, e por quê

**Um `defer` que lê nome do escopo onde foi declarado.** A visibilidade de escopo da Aurora diz
que o corpo enxerga onde foi escrito, e com frames isso é um ponteiro para o frame de cima —
um static link. Nada aqui impede, e nada aqui resolve. O beta é escopo de topo, que só lê os
próprios argumentos e locais.

**`assert` → `REVERT`.** É pequeno e é adjacente, mas é falha e não desvio.

**Tail call.** Reaproveitar o frame quando a chamada é a última coisa do corpo é otimização, e
otimização entra depois de existir o que otimizar.

---

## As etapas

1. **`PUSH2`** nos três lugares, e a interface de offset deixando de ser `byte`.
2. **Medir, resolver, escrever** — a montagem em duas passadas, sem nenhuma instrução nova.
   Ela sozinha não muda nada do que o binário faz hoje, o que é exatamente o que a torna
   testável: mesmos bytes, caminho novo.
3. **`if`**, que precisa de 1 e 2 e de nada mais.
4. **O frame**, e `feed(n)` lendo dele; o dispatcher virando chamador. O binário continua
   respondendo o mesmo, e é isso que o harness compara.
5. **`call`**, e a recursão junto.

Ordem de grandeza: 1 e 2 são pequenas e mecânicas. 3 é média. 4 é a maior de todas, e é a que
não dá para dividir — o dia em que `feed` passa a ler do frame é o mesmo em que o dispatcher
passa a escrever nele.

---

## Em aberto

1. **Onde mora o ponteiro de frame.** Um slot fixo baixo (`0x00`) é simples e conflita com o
   `MSTORE` que o `RETURN` já usa; a Solidity reserva `0x40`, e imitar isso tem o mérito de
   ninguém se surpreender.
2. **Quem soma o ponteiro**, quem chama ou o corpo. Se for o corpo, o chamador não precisa
   saber o tamanho do frame de quem ele chama — o que é melhor, e custa uma leitura a mais.
3. **O `OpReturn` com dois sentidos.** Fecha escopo e fecha braço de `if`, e o builder precisa
   distinguir. Dá para ler da estrutura, ou para o emitter passar a dizer qual é qual — a
   segunda é mais honesta e mexe no IR.
4. **`OpPreCall` existe e não é usado por ninguém.** Ou ele é o lugar onde os argumentos são
   escritos no frame, ou ele sai da lista.
5. **Quem zera as posições que ninguém escreveu.** Quem chama sabe quantos valores está
   mandando — é constante de compilação, está escrito no fonte — mas não sabe quantas posições
   o corpo lê, porque um `defer` é valor de primeira classe e pode ser chamado através de
   qualquer nome. Quem é chamado sabe quantas lê e não sabe quantas chegaram. Então ou a
   contagem viaja num slot do frame e o corpo zera da contagem até o que ele lê, ou quem chama
   zera um teto global e ninguém precisa da contagem. A primeira é exata e custa um slot; a
   segunda é grosseira e custa memória em toda chamada.
5. **Recusar em vez de truncar.** Com `PUSH2`, um contrato acima de 64KB não é representável —
   mas ele também não é publicável. Vale um erro de compilação dizendo isso, em vez de bytes
   errados.
