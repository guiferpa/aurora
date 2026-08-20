# `storage` — o que sobrevive à chamada

**Estado:** proposta · **Data:** 2026-08-20

Um contrato que não guarda nada é uma calculadora. Esta RFC diz como um programa Aurora
escreve e lê o que **persiste entre duas chamadas**, e é o que falta para o primeiro contrato
de verdade.

São duas camadas, e a divisão é o desenho inteiro:

| camada | quem escreve | o que é |
|---|---|---|
| `sload` / `sstore` | só a biblioteca padrão | as palavras cruas: um slot, uma palavra |
| módulo `storage` | qualquer programa | escopos com chave, escritos em Aurora sobre as duas palavras |

Ela **não** trata de ambiente de processo (`os`), de quem chamou (`caller`), de evento nem de
chamada a outro contrato. Cada um é a sua conversa.

Tudo abaixo foi medido no código de hoje.

> Os blocos aqui **não** estão marcados como `aurora`: `hosting/cli/docs_test.go` executa todo
> bloco assim marcado no repositório, e nada disto compila ainda.

---

## Por que persistência é palavra-chave e memória não

Memória a Aurora já usa, implicitamente: o emitter escreve os locais em offsets decididos na
compilação, e ninguém pede nada. É rascunho — some no fim da chamada — e expor um offset
arbitrário só daria ao programa a chance de corromper os próprios locais.

Storage é a outra coisa. É o disco: sobrevive à chamada, custa caro (`SSTORE` de zero para
não-zero são 20k de gas), e **nenhum programa consegue chegar nele por acidente**. Não há como
o compilador adivinhar que este valor devia durar e aquele não. Quem decide é quem escreve, e
por isso precisa de palavra.

---

## As duas palavras

```
_sload  -> SLOAD _prie
_sstore -> SSTORE _prie _prie
```

Palavra e operandos, sem parênteses e sem vírgula, que é a forma que `pull`, `push`, `head` e
`tail` já têm. **Operando primário e não expressão**, de propósito: `sload 0 + 1` lê como
`(sload 0) + 1`, sem a ambiguidade que o `_pull -> PULL _expr _expr` tem hoje. O `_head` já
restringe o operando dele pelo mesmo motivo, então há precedente.

| | responde |
|---|---|
| `sload k` | a palavra guardada na chave `k`, e zero onde nada foi escrito |
| `sstore k v` | `v`, o valor que acabou de escrever |

> **Aberto:** `sstore` responder o valor **anterior** é mais útil e menos óbvio. Responder o
> escrito é o que faz `sstore k (sload k + 1)` compor.

**Só a biblioteca padrão escreve as duas.** É a regra de arquivo que já limita `assert` a
`*.test.ar`, lida do `ParseInput`: um programa comum que escreve `sstore` é recusado, e a
mensagem manda usar o módulo. O motivo não é gosto — é que um slot cru é um número mágico, e
dois programas escolhendo o mesmo número em silêncio é o modo de falha que este projeto mais
recusa.

---

## O módulo

```
use storage as s;

ident balances = s.new(1);

ident deposit = defer {
  s.set(balances, feed(0), s.get(balances, feed(0)) + feed(1));
};

ident balance_of = defer { s.get(balances, feed(0)); };
```

| | responde |
|---|---|
| `new(id)` | um escopo: a base de onde as chaves dele saem |
| `set(scope, key, value)` | o valor guardado |
| `get(scope, key)` | o valor guardado nessa chave, ou zero |

O corpo é Aurora comum, escrito sobre as duas palavras:

```
ident new = defer { feed(0); };
ident set = defer { sstore (slot_of(feed(0), feed(1))) feed(2); };
ident get = defer { sload (slot_of(feed(0), feed(1))); };
```

### `new` não escreve nada, e isso é a decisão principal

A leitura natural de `new` é "aloque um escopo": leia um contador, incremente, devolva o
próximo. **Isso está errado aqui**, e o motivo é quando o código roda.

O topo de um arquivo Aurora roda a cada chamada, não no deploy. Um `new` que incrementasse um
contador daria um escopo **diferente a cada transação** — os saldos de ontem ficariam num
escopo que ninguém mais acha. Para funcionar, ele teria que rodar uma vez só, no construtor, e
construtor com estado é uma máquina que não existe.

Então `new` é **derivação pura**: um id entra, uma base sai, sem tocar em storage. Chamar duas
vezes com o mesmo id dá o mesmo escopo, hoje e amanhã. O id é escolhido por quem escreve o
programa, como o número do slot em Solidity é escolhido pela ordem de declaração.

> **Aberto:** `new` passa a significar "nomeie um escopo", não "aloque um". `scope`, `at` ou
> `open` dizem isso melhor. O nome foi pedido como `new`; fica registrado que ele mente um
> pouco.

### Como um escopo vira slot

A EVM dá um espaço plano de 2²⁵⁶ chaves e **não hasheia nada** — a chave é o slot. O
`keccak(chave . slot)` que se vê em contrato Solidity é escolha de layout dela, não regra da
máquina.

A Aurora não tem `KECCAK256` em runtime; o keccak que ela faz hoje é em build time, para o
seletor do dispatcher. Então a derivação do beta é **aritmética**, e vem com uma fronteira
escrita:

| candidato | colide quando |
|---|---|
| `base + chave` | sempre que `b1 + k1 == b2 + k2` — barato e frouxo demais |
| `base * 2¹²⁸ + chave` | quando a chave passa de 2¹²⁸, ou o escopo passa de 2¹²⁸ |

A segunda é a proposta: com `tape_size` 32 ela é exata para toda chave de 16 bytes ou menos —
o que inclui endereço, que tem 20... **não inclui.** Um endereço não cabe em 2¹²⁸, então a
fronteira tem que ser escolhida sabendo que a chave típica é uma carteira.

**Isso é o item mais aberto desta RFC**, e ele tem uma saída limpa: quando `KECCAK256` entrar,
a derivação vira `keccak(escopo . chave)` e a fronteira some. O preço de trocar depois é que o
layout muda — o que só afeta contrato já publicado, e no beta não há nenhum.

---

## Fora da chain

Storage é **simulável**, e é por isso que ele é palavra-chave e o ambiente do processo não é.

| | `sload` / `sstore` |
|---|---|
| na chain | `SLOAD` / `SSTORE` |
| no evaluator | um mapa que o host segura, chave de tape para valor de tape |

O evaluator não toca no mundo: o mapa chega por porta, como o print já chega.

```go
// A Store is what a program keeps between calls, and where it keeps it.
type Store interface {
	Load(key []byte) []byte
	Save(key, value []byte) error
}
```

| host | o que passa |
|---|---|
| `aurora run`, REPL | um mapa em memória, vazio a cada execução |
| `aurora test` | um mapa por arquivo de teste |
| playground | um mapa em memória |
| harness diferencial | o mesmo programa nos dois lados, e as respostas comparadas |

O `aurora run` começando com o mapa vazio é honesto: uma execução local não é uma chain com
histórico, é uma simulação de **uma** chamada. Guardar isso em disco entre execuções é outra
RFC, e provavelmente uma má ideia.

---

## O que falta compilar

Medido: `WriteCode` cobre aritmética, `OpSave`, `OpIdent`, `OpLoad`, `OpGetFeed` e `OpReturn`.
O resto compila e não faz nada na chain, em silêncio.

Em ordem de dependência:

| # | o que | quem precisa |
|---|---|---|
| 1 | **`ident` dentro de escopo** — hoje o valor é descartado e o `MSTORE` acha a pilha vazia; reverte | qualquer corpo com mais de uma linha |
| 2 | **Largura de verdade** — offset, alvo de jump e tamanho do runtime são `PUSH1` e truncam em 255 | qualquer contrato com estado |
| 3 | **`OpSLoad` / `OpSStore`** | esta RFC |
| 4 | **`call`** — jump com endereço de retorno | **o módulo**: `s.get(...)` é uma chamada |
| 5 | `if` e `assert` → `JUMPI` e `REVERT` | o cofre, não esta RFC |

**1 e 2 são os dois do stand-by**, e continuam sendo desenho e não remendo. Um `sstore`
compilado dentro de um corpo que reverte é um teste verde que mente.

E **4 é o preço do módulo**: as duas palavras sozinhas compilam sem `call`; a superfície
amigável não. Dá para entregar `sload`/`sstore` funcionando na chain antes do `storage`
existir, e é o que eu faria.

---

## O que fica de fora

- **`caller`**, evento, chamada a outro contrato, `msg.value`.
- **Apagar** um slot: escrever zero é o que a EVM chama de apagar, e o reembolso de gas que
  vem com isso não é assunto de linguagem.
- **Iterar** um escopo. Um mapa esparso não se percorre; quem quer lista guarda o tamanho.
- **Layout compatível com Solidity.** Um `cast storage` de fora não vai adivinhar onde as
  coisas estão, e tudo bem — a compatibilidade que importa é a da ABI, que o dispatcher já dá.

---

## As etapas

1. **`sload` e `sstore`** como palavras: nó, opcode, a regra de arquivo que as limita à
   biblioteca, e o evaluator com a porta atrás delas.
2. **Os dois do stand-by**, sem os quais nada disso roda na chain.
3. **`OpSLoad` e `OpSStore` no backend**, e o primeiro contrato com estado no harness
   diferencial: escreve, chama de novo, lê o que escreveu — dos dois lados.
4. **`call` no backend**, e então o módulo `storage` embutido (`shared/stdlib`, `//go:embed`,
   o gancho `Builtin` no resolver, e a recusa de um `src/storage.ar` do projeto sombrear o
   embutido).

Ordem de grandeza: 1 é pequena e fechada. 2 é a conversa que está parada desde o stand-by. 3 é
onde o beta aparece. 4 é o açúcar, e pode esperar.

---

## Em aberto

1. **A derivação do slot** e a fronteira dela, que hoje não cobre um endereço.
2. **`sstore` responde o escrito ou o anterior.**
3. **`new` mente**: ele nomeia um escopo, não aloca um.
4. **Chave menor que uma palavra**: estender a tape para 32 bytes é o que a aritmética já faz,
   mas dois projetos de larguras diferentes escrevem no mesmo slot com chaves diferentes.
5. **`sload` de um slot nunca escrito responde zero**, que é o que a EVM faz — e a Aurora não
   tem valor nenhum, então zero é a única resposta possível. Vale escrever que é uma escolha
   herdada, não desenhada.
