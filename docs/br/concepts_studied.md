
> :balloon: This file's portuguese _(pt-BR)_ language, feel free to contribute translating to another language

# Conceitos estudados

- [Linguagens formal](https://pt.wikipedia.org/wiki/Linguagem_formal#:~:text=Entende%2Dse%20por%20linguagem%20formal,%2C%20caracter%C3%ADsticas%20e%20inter%2Drelacionamentos%20.)
- [Hierarquia de Chomsky](https://pt.wikipedia.org/wiki/Hierarquia_de_Chomsky)
- [Autômatos](#autômatos)
- Linguagem livre de contexto
- Gramática livre de contexto
- [Gramática com ambiguidade](#gramática-com-ambiguidade)
- [Lexemas](#lexemas)
- [Tokens](#tokens)
- [Análise léxica](#análise-léxica)
- Análise sintática
- Árvore sintática abstrata _(AST)_
- Análise semântica
- Otimização do código

## Autômatos

Um autômato é como uma máquina abstrata, imaginada para entender ou reconhecer padrões em sequências de símbolos. Você pode pensar nele como um modelo matemático simples que segue regras específicas para processar entradas.

Imagine um autômato como um dispositivo que recebe uma sequência de símbolos, como letras ou números, e, seguindo um conjunto de instruções simples, ele pode decidir se essa sequência está de acordo com um padrão específico ou pertence a uma linguagem.

Por exemplo, imagine um autômato que reconhece palavras que terminam com `ar`. Ele receberia uma palavra como "computar" e, seguindo suas regras, diria _Sim_, essa palavra termina com `ar`". Porém, se a palavra fosse `abacaxi`, ele diria _Não_, essa palavra não termina com `ar`".

Assim, um autômato é uma maneira de representar, de forma simplificada, máquinas ou processos que podem reconhecer ou processar padrões em sequências de símbolos, sendo uma ferramenta fundamental no estudo de linguagens formais e na construção de compiladores.

Existem diferentes tipos de autômatos, sendo os principais:

- **Autômato Finito Determinístico (AFD):** É um modelo simples com um conjunto finito de estados, uma entrada (alfabeto) e uma função de transição determinística. Reconhece linguagens regulares.

- **Autômato Finito Não-determinístico (AFN):** Similar ao AFD, mas com transições não-determinísticas, permitindo múltiplos estados de destino para uma mesma entrada em um estado particular.

- **Autômato de Pilha (AP):** Utiliza uma pilha para armazenar informações. Reconhece linguagens livres de contexto e é mais poderoso que os autômatos finitos.

- **Máquina de Turing:** É um autômato mais abstrato e poderoso, com uma fita infinita e um conjunto de regras que especificam como a máquina muda de estado dependendo do símbolo lido na fita. Pode resolver problemas decidíveis e reconhecer linguagens recursivamente enumeráveis.

Na construção de um compilador para uma linguagem de programação, diferentes tipos de autômatos são usados em etapas específicas do processo de compilação:

- **Análise Léxica (Scanner):** Geralmente é implementada usando autômatos finitos, especialmente autômatos finitos determinísticos (AFDs) ou autômatos finitos não-determinísticos (AFNs), para reconhecer tokens (como palavras-chave, identificadores, números) a partir do código fonte.

- **Análise Sintática (Parser):** Aqui, autômatos de pilha são frequentemente usados, como autômatos de pilha determinísticos (APDs), para verificar a estrutura gramatical do código fonte e criar uma árvore de análise sintática.

Os autômatos são fundamentais na compreensão da estrutura e do reconhecimento de padrões em linguagens formais, sendo essenciais na construção de compiladores para traduzir código-fonte em linguagens de programação para código executável. Cada etapa do compilador pode envolver diferentes tipos de autômatos para realizar análises específicas no processo de compilação.

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

### Como reescrever a gramática para remover sua ambiguidade?

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

🎈 Um ponto essencial para entender é que toda ambiguidade só é possível ser retirada de uma gramática devido a um comportamento esperado/regra estabelecida. No do nosso exemplo acima a regra imposta foi que a operação matemática de multiplicação deveria sempre ser considerada como prioridade na sua derivação, ou seja, ter um peso de precedência maior que a outra operação.

## Lexemas

Um lexema é uma sequência de caracteres que representa uma unidade básica de significado em um programa de computador. Em linguagens de programação, um lexema pode ser uma palavra-chave (como `if` ou `else` em muitas linguagens), um identificador (nome de variável ou função), um número, um operador ou um símbolo especial.

O reconhecimento de lexemas é uma etapa fundamental na análise léxica de um compilador. Durante essa análise, o código-fonte é dividido em lexemas, identificando palavras-chave, variáveis, constantes, operadores e outros elementos básicos da linguagem de programação. Cada lexema representa uma unidade indivisível que possui um significado específico dentro da gramática da linguagem.

Por exemplo:

- Em uma expressão matemática como `a = b + 3`, os lexemas são `a`, `=`, `b`, `+` e `3`.
- Em uma declaração de controle de fluxo como `if (x < 10) { ... }`, os lexemas são `if`, `(` , `x`, `<`, `10`, `)` e `{`.

## Tokens

Um token é uma estrutura de dados que representa um lexema reconhecido durante a análise léxica de um programa de computador. Ele é uma unidade fundamental na construção de um compilador.

Quando o analisador léxico identifica um lexema (uma sequência de caracteres com significado dentro da linguagem de programação), ele gera um token correspondente. Esse token contém informações sobre o lexema, como seu tipo e possivelmente seu valor.

Por exemplo, em uma expressão matemática simples como `a = b + 3`, os lexemas são `a`, `=`, `b`, `+` e `3`. Cada um desses lexemas seria transformado em um token durante a análise léxica. O token para "a" poderia ser do tipo identificador, o token para "=" seria do tipo operador de atribuição e assim por diante.

Os tokens são então utilizados nas fases subsequentes da compilação, como a análise sintática e a geração de código, para entender a estrutura do programa e criar representações intermediárias ou traduzir o código-fonte para outra forma, como código de máquina.

### Lexemas e Tokens são a mesma coisa?

Na verdade, lexemas e tokens são conceitos relacionados, mas não são exatamente a mesma coisa. Um lexema é a sequência de caracteres em um código-fonte que é reconhecida como uma instância de uma classe de palavras-chave, identificadores, operadores ou símbolos especiais. Por exemplo, em uma linguagem de programação, a palavra-chave `if` ou um identificador como `counter` são lexemas.

Já um token é uma estrutura de dados que contém informações sobre um lexema específico, incluindo seu tipo e valor. Durante a análise léxica, os lexemas são identificados e agrupados em tokens. Um token pode conter informações como o tipo do lexema (por exemplo, palavra-chave, identificador, número, etc.) e seu valor (por exemplo, o valor numérico de um número, ou o texto exato de um identificador).

Portanto, um lexema é a sequência de caracteres reconhecida como uma unidade léxica, enquanto um token é a estrutura de dados que representa esse lexema, associando-o a um tipo e, possivelmente, a um valor específico.

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

#### Análise léxica (Geração de tokens)

| Padrão     | Tipo                   | Símbolos  | Valor |
|------------|------------------------|-----------|-------|
| `(`        | Parênteses             | `PAREN_O` | (     |
| `1`        | Números                | `NUM`     | 1     |
| `+`        | Operações aritiméticas | `OP_ARIT` | +     |
| `2`        | Números                | `NUM`     | 2     |
| `)`        | Parênteses             | `PAREN_C` | )     |
| `*`        | Operações aritiméticas | `OP_ARIT` | *     |
| `10`       | Números                | `NUM`     | 10    |
