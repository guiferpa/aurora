# `syscall`, e o primeiro módulo da biblioteca padrão

**Estado:** proposta · **Data:** 2026-08-20

Ler variável de ambiente, off-chain. A forma foi decidida na conversa: um módulo `os` com
dois escopos, `get_env` e `lookup_env`, sobre uma palavra-chave `syscall`. Esta RFC diz como
construir isso — e registra **duas coisas que a premissa da linguagem decide sozinha**, uma
das quais muda o `lookup_env`.

Tudo abaixo foi medido no código de hoje.

> Os blocos aqui **não** estão marcados como `aurora`: `hosting/cli/docs_test.go` executa todo
> bloco assim marcado no repositório, e nada disto compila ainda.

---

## Por que módulo, e não uma palavra-chave solta

A premissa da Aurora é que **o mesmo programa responde a mesma coisa dentro e fora da chain**.
Ler o ambiente é exatamente o que uma chain não consegue reproduzir.

Se `env` fosse palavra-chave, qualquer programa poderia ser não-portável e o backend só
descobriria instrução por instrução, no fim da compilação. Sendo módulo, **o `use` é a
declaração**: este programa é off-chain, dito no topo do arquivo, e o `aurora build` pode
falar disso pelo nome antes de compilar qualquer coisa.

A fronteira do módulo passa a ser onde a portabilidade se perde. É um lugar melhor para essa
linha do que "qualquer expressão em qualquer lugar".

---

## Duas coisas que a premissa decide

### O valor tem que caber numa tape

`HOME=/Users/guiferpa` são 16 bytes. Com `tape_size` 8 não cabe.

**Proposta: recusar em runtime, dizendo os dois números.** Truncar um caminho é como se produz
uma resposta errada com cara de certa, e a mensagem já tem irmã — o compilador recusa um
literal de texto que não cabe, com essas palavras.

```
the value of HOME is 16 bytes and a tape is 8
```

### O nome também, e é o preço do módulo

Um módulo exporta um `defer`, e um `defer` recebe valores em runtime:

```
ident get_env = defer { syscall getenv feed(0); };
```

`feed(0)` é uma tape, então **o nome da variável é uma tape**. Com `tape_size` 8 dá para ler
`HOME` e `PATH`; `AURORA_RPC_ENDPOINT` (19 bytes) precisa de um projeto com 19 ou mais.

Não há como ter as duas coisas. Um nome escrito **dentro da instrução** pelo emitter — como o
`assert` já faz com a mensagem dele — não teria limite nenhum, mas aí ele é imediato e não
pode ser argumento de nada, o que mata o módulo. Escolhemos o módulo, então herdamos o teto.

É o mesmo teto que o [roadmap](../docs/roadmap.md) chama de o maior de todos, "texto maior que
uma tape". Quando ele cair, o `os` melhora sozinho e sem mudança nenhuma.

---

## `lookup_env` não pode devolver um par

Em Go, `os.LookupEnv` devolve `(string, bool)`. Aqui não devolve, e não é por falta de tuplas:

- um escopo responde **um** valor;
- e **um `struct` não atravessa módulo**, o que foi decidido no sistema de módulos. Um
  resultado de dois campos declarado em `os.ar` não é legível de fora — quem chama precisaria
  de `as Result`, e `Result` não está declarado no arquivo dele.

Então os dois escopos respondem uma pergunta cada, e juntos dão o mesmo que o par de Go:

| | responde |
|---|---|
| `get_env(name)` | o valor, e o valor neutro quando a variável não existe — igual ao `Getenv` do Go |
| `lookup_env(name)` | se ela existe |

```
use os as os;

if os.lookup_env("HOME") {
  printc os.get_env("HOME");
};
```

> **Aberto:** com essa forma, `lookup_env` responde a metade `ok` do `LookupEnv` do Go. O nome
> foi pedido assim; se soar como se devolvesse o valor, `has_env` diz o que ele faz.

---

## `syscall`, a palavra que só a biblioteca escreve

```
syscall getenv feed(0)
```

Palavra-chave, um nome de chamada, e um valor. O nome não é um valor — é como `as Point`
nomeia um struct — e o conjunto é fechado, então o parser recusa o que não conhece dizendo o
que existe.

**Só a biblioteca padrão pode escrevê-la.** É o mecanismo que já existe para o `assert`, que
só é aceito em `*.test.ar` (`parser/parser.go:867`): uma regra de arquivo, lida do
`ParseInput`. Um programa comum escrevendo `syscall` é recusado, e a mensagem diz que o
caminho para o mundo é o módulo.

Duas chamadas para começar:

| Chamada | Responde |
|---|---|
| `getenv` | o valor da variável, ou o valor neutro |
| `hasenv` | se ela existe |

> **Aberto:** `hasenv` diz o que faz, e `lookup_env` é o que o módulo exporta em cima dela. Se
> preferir simetria, os dois nomes se encontram no meio.

---

## A porta

O evaluator não toca no mundo. Um syscall chega como porta, do mesmo jeito que um print:

```go
// A Syscall is how a program reaches the world it is running in.
type Syscall interface {
	Call(name string, arg []byte) ([]byte, error)
}
```

| Host | O que passa |
|---|---|
| `aurora run`, `aurora test`, REPL | uma implementação sobre `os.LookupEnv` |
| playground | nada — um navegador não tem ambiente, e um syscall ali responde que não há mundo para alcançar |
| teste | um mapa |

O playground respondendo "não há mundo" em vez de "a variável não existe" é a diferença entre
um programa que sabe onde está e um que acha que o ambiente está vazio.

---

## Como um módulo embutido chega ao resolver

`os` não é arquivo do projeto. O resolver hoje resolve um specifier para um caminho e pede os
bytes pela porta de leitura — então basta ele perguntar antes se aquele módulo é um que o
**host fornece**:

```go
type Options struct {
	SourceRoot string
	Read       Read
	Parse      Parse
	// Builtin answers the source of a module the host provides, when it provides one.
	Builtin func(module.ID) ([]byte, bool)
}
```

São umas oito linhas no resolver, e o resto funciona sem saber de nada: o módulo é parseado,
prefixado, carregado e avaliado como qualquer outro — **é um módulo comum que mora dentro do
binário**. A fonte dele viaja num `//go:embed`, em `shared/stdlib`, que é o que serve a camada
de hosting sem pertencer a uma interação.

O playground ganha o mesmo módulo pelo mesmo caminho, e ali ele compila; o que não funciona é
a chamada, que responde que não há mundo.

### Nome reservado, e o que acontece se o projeto tiver `src/os.ar`

Sombrear em silêncio é o modo de falha que este projeto mais recusa. **Proposta: se o
specifier é de um módulo embutido e o arquivo também existe, o import é recusado** —

```
os is a module Aurora provides: rename yours
```

Custa uma tentativa de leitura por import embutido, que é nada, e troca um mistério por uma
frase.

---

## A chain

`OpSyscall` entra na lista `by decision` do `builder/evm/support.go`, ao lado de `printb` e
`assert`: uma chain não tem ambiente, e isso não expira — é a única lista onde a palavra "yet"
seria mentira.

E como a não-portabilidade agora está declarada no topo do arquivo, o `aurora build` pode
dizer **qual módulo** a torna impossível, em vez de listar instruções.

---

## O que fica de fora

- **Escrever no ambiente.** Só afeta processos filhos, e a linguagem não tem processos.
- **Listar tudo.** Exigiria montar uma sequência em runtime, que a linguagem não faz.
- **Qualquer outro syscall.** Arquivo, relógio e rede são cada um a sua conversa; o que esta
  RFC abre é o caminho, não a lista.

---

## As etapas

1. **`syscall` vira palavra-chave**, com a regra de arquivo que a limita à biblioteca, o nó, o
   opcode e a recusa de um nome que não existe. Sem porta ainda: um programa comum não pode
   escrevê-la, e a biblioteca ainda não existe.
2. **A porta no evaluator**, e o `getenv`/`hasenv` por trás dela, com os hosts passando o que
   cada um tem.
3. **`shared/stdlib` e o `Builtin` no resolver**, com o `os.ar` embutido e a recusa do
   sombreamento.
4. **A documentação e o exemplo**: `docs/stdlib.md`, o roadmap, e a lista `by decision` do
   backend.

Ordem de grandeza: ~400 linhas de código e ~500 de teste, em dois pull requests — 1–2 num,
3–4 no outro.

---

## Em aberto

1. **`lookup_env` ou `has_env`**, já que ele responde se existe e não o valor.
2. **`hasenv` como nome de syscall**, e se ele deve casar com o do módulo.
3. **O valor que não cabe**: recusar (proposta) ou truncar.
4. **`syscall` recebe um valor só.** Chega para o ambiente; a primeira chamada com dois
   argumentos vai pedir uma forma que não existe.
