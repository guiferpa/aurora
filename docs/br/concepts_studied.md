
> :balloon: This file's portuguese _(pt-BR)_ language, feel free to contribute translating to another language

# Conceitos estudados

- [Linguagens formal](https://pt.wikipedia.org/wiki/Linguagem_formal#:~:text=Entende%2Dse%20por%20linguagem%20formal,%2C%20caracter%C3%ADsticas%20e%20inter%2Drelacionamentos%20.)
- [Teoria dos autômatos](https://pt.wikipedia.org/wiki/Teoria_dos_aut%C3%B4matos)
- [Hierarquia de Chomsky](https://pt.wikipedia.org/wiki/Hierarquia_de_Chomsky)
- Linguagem livre de contexto
- Gramática livre de contexto
- [Gramática com ambiguidade](#gramática-com-ambiguidade)
- [Lexemas](#lexemas)
- Análise léxica _(Scanning)_
- Token _(Chave<Tipo> : Valor)_
- Análise sintática
- Árvore sintática abstrata _(AST)_
- Análise semântica
- Otimização do código

## Gramática com ambiguidade

Toda gramática ela pode ter ambiguidade ou não, para verificar se uma gramática tem ambiguidade pode ser usado métodos de derivação a esquerda e a direita. Um gramática que possui ambiguidade é impossível ser lida dado que existe o cenário onde a mesma pode ser lida de mais de uma forma.

Seguindo o exemplo da gramática abaixo podemos atestar que ela possui ambiguidade com a seguinte derivação

#### Gramática
```
expr -> expr + expr
      | expr * expr
      | (expr)
      | id
```

#### Método de derivação a esquerda

```
id + id * id -> expr * id
              | expr * expr
              | expr
```

#### Método de derivação a direita

```
id + id * id -> id + expr
              | expr + expr
              | expr
```

Como ambas as derivações podem ser concluidas nós entendemos que essa gramática possui ambiguidade.

### Como elaborar a gramática para remover sua ambiguidade?

Neste caso, dado como exemplo acima, nós poderíamos tratar a precedência dessa gramática. Neste caso vamos dar um peso maior para a operação de multiplicação e diferencia a precedência entre ambas, multiplicação e adição. A gramática reformulada ficaria assim:

### Gramática coma precedência nas operações

#### Gramática

```
expr -> expr + term
      | term

term -> term * fact
      | fact

fact -> (expr)
      | id
```

Agora vamos aplicar os métodos de derivação

#### Método de derivação a esquerda

```
id + id * id -> expr + term * id
              | expr * id (A partir desta etapa não é possivel prosseguir com a derivação)
```

#### Método de derivação a direita

```
id + id * id -> id + term * fact
              | expr + term
              | expr
```

Boa, tiramos a ambiguidade da nossa gramática, conseguimos derivar com o método de derivação a direita 🎆

🎈 Um ponto essencial de entender é que toda ambiguidade só é possível ser retirada de uma gramática devido a um comportamento esperado/regra estabelecida. No do nosso exemplo acima a regra imposta foi que a operação matemática de multiplicação deveria sempre ser considerada como prioridade na sua derivação, ou seja, ter um peso de precedência maior que a outra operação.

## Lexemas

Um lexema é uma sequência de caracteres que representa uma unidade básica de significado em um programa de computador. Em linguagens de programação, um lexema pode ser uma palavra-chave (como `if` ou `else` em muitas linguagens), um identificador (nome de variável ou função), um número, um operador ou um símbolo especial.

O reconhecimento de lexemas é uma etapa fundamental na análise léxica de um compilador. Durante essa análise, o código-fonte é dividido em lexemas, identificando palavras-chave, variáveis, constantes, operadores e outros elementos básicos da linguagem de programação. Cada lexema representa uma unidade indivisível que possui um significado específico dentro da gramática da linguagem.

Por exemplo:

- Em uma expressão matemática como `a = b + 3`, os lexemas são `a`, `=`, `b`, `+` e `3`.
- Em uma declaração de controle de fluxo como `if (x < 10) { ... }`, os lexemas são `if`, `(` , `x`, `<`, `10`, `)` e `{`.

## Análise léxica

Uma análise léxica é onde o compilador escaneia todos os tokens que fazem sentido existir na gramática e passa a dar sentido a eles, os tokens. Indo para prática e considerando uma gramática simples.

#### Gramática

```
expr -> expr + term
      | term

term -> term * fact
      | fact

fact -> (expr)
      | id
```

#### Lexemas

| Padrão (RegEx)    | Tipo                   |
|-------------------|------------------------|
| `(`, `)`          | Parênteses             |
| `+`, `*`          | Operações aritiméticas |
| `[0-9]+`          | Números                |

Vamos analisar léxicamente o seguinte código:

#### Código

```
(1 + 2) * 10
```

#### Análise léxica (Tokens)

| Padrão     | Tipo                   | Símbolos |
|------------|------------------------|----------|
| `(`        | Parênteses             | `PAREN`  |
| `1`        | Números                | `NUMBER` |
| `+`        | Operações aritiméticas | `OP_ARI` |
| `2`        | Números                | `NUMBER` |
| `)`        | Parênteses             | `PAREN`  |
| `*`        | Operações aritiméticas | `OP_ARI` |
| `10`       | Números                | `NUMBER` |
