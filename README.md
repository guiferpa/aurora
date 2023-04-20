# aurora
Aurora's just for studying programming language concepts

> ⚠ Don't use it to develop something that'll go to production environment

## Get started

### Hello world example

#### Create a file with content below

```aurora
(2 + 2) - 10 * 10;
10 + 10;
{
  20 + 20;
  100 * 2 * 10;
  {
    1 + 1;
    {
      40 + 30 * 10 * 20 + 3;
    }
  }
}
```

#### Execute this source code file

```sh
$ aurora ./<file>.ar
```

#### That's the output from evaluator
```js
BlockStatmentNode {
  tag: 'BlockStatment',
  id: 'root',
  block: [
    BinaryOperationNode {
      tag: 'BinaryOperation',
      left: [BinaryOperationNode],
      right: [BinaryOperationNode],
      operator: [Token]
    },
    BinaryOperationNode {
      tag: 'BinaryOperation',
      left: [ParameterOperationNode],
      right: [ParameterOperationNode],
      operator: [Token]
    },
    BlockStatmentNode {
      tag: 'BlockStatment',
      id: '1681959585607',
      block: [Array]
    }
  ]
}
= -96,20,40,2000,2,6043
```

