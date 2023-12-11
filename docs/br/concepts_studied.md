
> :balloon: This file's portuguese _(pt-BR)_ language, feel free to contribute translating to another language

# Conceitos estudados

- [Linguagens formal](https://pt.wikipedia.org/wiki/Linguagem_formal#:~:text=Entende%2Dse%20por%20linguagem%20formal,%2C%20caracter%C3%ADsticas%20e%20inter%2Drelacionamentos%20.)
- [Hierarquia de Chomsky](https://pt.wikipedia.org/wiki/Hierarquia_de_Chomsky)
- [Autômatos](#autômatos)
- Gramática livre de contexto
- Gramática sensível ao contexto
- [Associatividade](#associatividade)
- Precedência de Operadores
- [Gramática com ambiguidade](#gramática-com-ambiguidade)
- [Lexemas](#lexemas)
- [Tokens](#tokens)
- [Análise léxica](#análise-léxica)
- [Análise sintática](#análise-sintática)
- Análise descendente
- Análise ascendente
- Árvore sintática abstrata _(AST)_
- Análise semântica
- Otimização do código

## Autômatos

Um autômato é como uma máquina abstrata, imaginada para entender ou reconhecer padrões em sequências de símbolos. Você pode pensar nele como um modelo matemático simples que segue regras específicas para processar entradas.

Imagine um autômato como um dispositivo que recebe uma sequência de símbolos, como letras ou números, e, seguindo um conjunto de instruções simples, ele pode decidir se essa sequência está de acordo com um padrão específico ou pertence a uma linguagem.

Por exemplo, imagine um autômato que reconhece palavras que terminam com `ar`. Ele receberia uma palavra como "computar" e, seguindo suas regras, diria _Sim_, essa palavra termina com `ar`. Porém, se a palavra fosse `abacaxi`, ele diria _Não_, essa palavra não termina com `ar`.

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

## Associatividade

A associatividade em uma gramática se dá justamente pela escrita da mesma. Dado um código ou uma cadeia de terminais `id = id = 3` onde podemos assumir que é uma construção válida como poderíamos construir uma gramática?

### Gramática

```
expr -> expr = fact
      | fact

fact -> id
      | dig

dig -> [0-9]
```

### Criando uma árvore de derivação

<img width="420" alt="Screenshot 2023-12-11 at 07 36 32" src="https://github.com/guiferpa/aurora/assets/9096630/023ffc44-c1e1-4baa-9fac-aa3ef60b7ed8">

Da forma que essa gramática foi criada a gente consegue chegar em um resultado e verficar que a mesma é valida, porem é importante lembrar que um código fonte como esse, que acabamos de criar sua árvore de derivação, espera que o extremo lado direito se resolva primeiro para só assim conseguir ser um código válido semânticamente. Na execução desse código eu teria um erro semântico dado que é impossível atribuir algo da esquerda para a direita. Entende-se isso dado que o dígito 3 por si só não recebe nenhuma atribuição, logo, isso é um comando inválido.

### Nova gramática

```
expr -> fact = expr
      | fact

fact -> id
      | dig

dig -> [0-9]
```

Básicamente oque foi feito é colocar o não-terminal (`expr`) para o lado direito entendendo que o mesmo é quem vai ditar a recursividade da nossa árvore de derivação

### Criando uma árvore de derivação para nova gramática

<img width="420" alt="Screenshot 2023-12-11 at 07 50 58" src="https://github.com/guiferpa/aurora/assets/9096630/ecf5c878-faf9-4b38-a19f-5c7c6c546a3a">

Agora a gramática está correta. Pode observar na árvore que o dígito 3 é o primeiro a ser resolvido, podendo agora ser atribuido aos demais `id`s.

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
id + id * id -> expr + expr
              | expr + expr * expr
              | expr
```

#### Método de derivação a direita

```
id + id * id -> expr * expr
              | expr + expr * expr
              | expr
```

Como ambas as derivações podem ser concluidas nós entendemos que essa gramática possui ambiguidade. Isso porque existe mais de uma forma de derivar essa cadeia de terminais _(Código fonte)_

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
id + id * id -> expr + term
              | expr + term * fact
              | term + fact * id
              | fact + id * id
              | id + id * id
```

#### Método de derivação a direita

```
id + id * id -> expr
              | term
              | term * fact
```

No terceiro passo a derivação já não consegue ter continuidade, isso porque o não-terminal `term` não tem nenhum produto que tenha o sinal de `+`.

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

## Análise sintática

A análise sintática é uma etapa crucial no processo de compilação, onde o código-fonte de um programa é analisado para determinar se está escrito de acordo com a estrutura gramatical da linguagem de programação. É como verificar se as frases em um idioma seguem a sintaxe correta desse idioma.

Imagine que você está aprendendo uma nova língua e precisa formar frases corretas. A análise sintática faz algo semelhante: verifica se as instruções ou expressões no código estão corretas em relação às regras de sintaxe da linguagem de programação.

Essa análise é realizada por um analisador sintático, também chamado de parser. O parser utiliza uma gramática (conjunto de regras) para identificar a estrutura hierárquica do código, criando uma representação chamada de "árvore sintática" ou "árvore de análise".

Essa árvore mostra como as diferentes partes do código estão organizadas e como se relacionam entre si. Assim como uma árvore de uma língua mostra a estrutura das palavras em uma frase, a árvore sintática mostra a estrutura das expressões e comandos no código.

Por exemplo, em linguagens como C, Java ou Python, a análise sintática verifica se as instruções estão corretamente organizadas com as palavras-chave, operadores e estruturas de controle conforme definido pela gramática da linguagem.

Se o código estiver bem estruturado e seguir a gramática definida, o analisador sintático cria a árvore sintática corretamente. Caso contrário, ele identifica erros de sintaxe e geralmente gera mensagens de erro para ajudar o programador a corrigir esses problemas.

Em resumo, a análise sintática é como a verificação da estrutura gramatical de um texto para garantir que está de acordo com as regras da linguagem de programação. É uma etapa crucial no processo de compilação para garantir que o código seja compreensível e possa ser traduzido para código de máquina de forma adequada.
