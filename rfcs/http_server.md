# Um servidor HTTP com Aurora

**Estado:** proposta · **Data:** 2026-08-16

Tudo que está descrito aqui como "hoje" foi conferido no código, não lembrado.

---

## Duas perguntas diferentes

"Construir um servidor HTTP com Aurora" se parte em duas, e elas têm respostas opostas:

**(A) Aurora como a linguagem do handler.** O host abre o socket, fala HTTP, e aplica um
escopo Aurora a cada requisição. O programa é uma função de requisição para resposta.

**(B) Aurora escrevendo o servidor.** O programa abre o socket, lê bytes da rede, parseia o
protocolo e escreve a resposta — tudo em Aurora.

**A proposta é (A)**, e a razão não é de esforço, é de arquitetura.

(B) exige chamadas de sistema dentro do evaluator, o que quebra duas premissas de uma vez: um
pacote vital deixa de ser puro (recebe valores e devolve valores) e o erro deixa de se
resolver no host. Também exigiria tudo que (A) exige — valores maiores que uma fita, índice
calculado em runtime, laços — e ainda mais. E o backend EVM, que está parado, ficaria ainda
mais distante: `listen` não tem significado numa cadeia.

(A) usa a premissa a favor em vez de contra ela. A linguagem continua pura; quem tem socket,
timeout, endereço e erro de rede é o host, que é exatamente o lugar onde essas coisas já
moram no projeto.

---

## O que a linguagem tem hoje

| O que | Estado hoje | Onde |
|---|---|---|
| Valores entrando no programa | um vetor de palavras de 32 bytes, **estreitadas para uma fita cada** | `evaluator/environ/environ.go`, `internal/cli/args.go` |
| Lendo esses valores | `feed(n)`, com o índice dando a volta no tamanho do vetor | `evaluator/builtin/functions.go` |
| Valores saindo | o que `printb`/`printc`/`printd` escrevem, num `io.Writer` que o host escolhe | `evaluator/evaluator.go` |
| O valor de cada expressão | o host recebe, por rótulo, o que cada expressão de topo deixou | `emitter.Program.Expressions` |
| Escopo aplicável a valores | `defer { … }` produz um valor; `nome(a, b)` aplica | — |
| Recursão | funciona | `examples/recursion.ar` |
| Texto | é uma fita, então cabe **no máximo `tape_size` bytes** (32 no teto) | `parser/parser.go:153` |
| Índice de `head`/`tail`/campo | é **imediato**: resolvido em tempo de compilação | `emitter/emitter.go:266` |
| Estado mutável | não existe | `rfcs/state_management.md` (a escrever) |
| Laço | não existe; o que existe é recursão | — |

Duas dessas linhas são a história inteira desta RFC: **entrada e saída já passam pelo host**,
e **nenhum valor pode ser maior que 32 bytes**.

---

## O primeiro passo já é possível hoje

Isto é o que torna a proposta concreta em vez de uma lista de desejos. Com o que existe:

- o host compila o programa **uma vez**;
- por requisição, cria um evaluator novo com o método e o caminho como valores aplicados;
- o corpo da resposta é **o que o programa imprimiu**, capturado pelo `io.Writer` que o host
  já escolhe;
- o status é **o valor da última expressão**, que o host já sabe ler — é assim que o REPL e o
  playground mostram o valor de uma linha.

```aurora
#- Um handler. feed(0) é o método, feed(1) é o caminho.
ident method = feed(0);

if method equals "GET" {
  printc "hello";
  200;
} else {
  printc "no";
  405;
};
```

```sh
aurora serve handler.ar --port 8080
```

Nenhuma palavra nova na linguagem, nenhum opcode novo, nenhuma mudança no emitter. É um host
novo — irmão de `aurora run` — e mais nada.

**O que essa fase honestamente não faz:** caminho maior que 32 bytes, corpo de requisição,
cabeçalhos, resposta maior que o que se consegue imprimir por partes, e qualquer coisa que
dependa de olhar dentro de um texto.

---

## As paredes, na ordem em que aparecem

**1. Um valor maior que uma fita.** Uma requisição real tem quilobytes; `"Guilherme"` já não
cabe em oito. Enquanto um valor for uma corrida fixa de bytes, o programa não segura uma URL
longa, quanto mais um corpo. É o item maior do [roadmap](../docs/roadmap.md) e é o que
destrava todo o resto.

**2. Ler uma posição decidida em runtime.** `head t 2` compila e `head t i` não — o índice
entra na instrução como imediato. Sem isso não existe parsear: nem separar caminho de query,
nem achar o `:` de um cabeçalho, nem percorrer nada.

**3. Construir um valor em runtime.** Não há concatenação. A resposta hoje só existe porque é
impressa em pedaços; montar uma, guardá-la, medi-la e só então escrevê-la exige juntar.

**4. Estado entre requisições.** Um contador, uma sessão, uma tabela de rotas. Hoje não há
estado mutável nenhum. Pode ser resolvido funcionalmente — o host guarda o valor e o aplica
na requisição seguinte —, e isso é uma decisão de projeto que vale escrever antes de precisar.

**5. Concorrência.** Uma linguagem sem estado mutável é trivialmente paralela: um evaluator
por requisição, nada compartilhado. Isso é uma vantagem rara e vale explicitar como decisão,
não descobrir por acidente.

As paredes 1, 2 e 3 são **a mesma obra** — é o que o roadmap já diz — e um servidor é a melhor
razão que a linguagem já teve para encará-la: uma meta que o usuário percebe, contra a qual
testar cada pedaço.

---

## Fases

| Fase | O que entrega | O que exige da linguagem |
|---|---|---|
| **1** | `aurora serve`: método e caminho aplicados ao programa, status pelo valor da última expressão, corpo pelo que ele imprime | **nada** |
| **2** | corpo de requisição e resposta montada como valor | valor maior que uma fita (parede 1) |
| **3** | parsear caminho, query e cabeçalhos **em Aurora** | índice calculado (2) e concatenação (3) |
| **4** | contador, sessão, rota declarada em dado | estado (4) |
| **5** | um framework de verdade | nada disso; é biblioteca, e biblioteca precisa de módulos |

A fase 1 é a que prova o contrato. As outras são a linguagem crescendo, com o servidor como
critério de aceitação.

---

## Perguntas em aberto

Nenhuma delas bloqueia a fase 1, mas todas mudam o desenho se decididas tarde:

1. **Truncar por onde?** Um valor que entra é estreitado para uma fita mantendo os **últimos**
   bytes (`byteutil.PaddingTape`). Para um número é o certo; para um caminho, `/usuarios/42`
   virando `os/42` é pior que perder o fim. Vale decidir se entrada de texto trunca pela cauda.

2. **Status por valor ou por builtin?** Usar o valor da última expressão é elegante e não custa
   nada, mas amarra "o que a expressão respondeu" a "o que o HTTP devolve". A alternativa é um
   builtin explícito, que é mais claro e é mais uma palavra na linguagem.

3. **Onde mora `serve`?** Mais um comando do `aurora`, ou um binário próprio? Um comando é mais
   simples; um binário separado mantém o CLI sobre compilar e rodar.

4. **O que acontece quando o programa falha?** Proposta: 500 para o cliente, erro no log do
   host, servidor de pé — erro se resolve no host, e uma requisição ruim não derruba as outras.

5. **Isto some da EVM?** `serve` não tem significado numa cadeia. Vale dizer em voz alta que
   este é um alvo do evaluator, como o REPL, e que não cria dívida no backend.

---

## O que fica fora, de propósito

TLS, keep-alive, streaming, upload, WebSocket, roteamento com sintaxe própria, middleware.
Nada disso é interessante enquanto um valor não puder ser maior que 32 bytes.
