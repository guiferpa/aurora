# aurora
Aurora's just for studying programming language concepts

> ⚠ Don't use it to develop something that'll go to production environment

## Summary

- [Get started](#get-started)
  - [Install CLI](#install-cli)
  - [Using REPL mode](#using-repl-mode)
  - [Execute from file](#execute-from-file)
- [Extra options](#extra-options)
  - [Debug flag](#debug-flag)

## Get started

### Install CLI
```sh
$ npm i -g @guiferpa/aurora
```

### Using REPL mode

```sh
$ aurora
```

```sh
>> var result = 1_000 * 2;
=
>> result + 1;
= 2001
>> print(result);
2000
=
```

### Execute from file

#### Create aurora source code file

```js
var result = 10 * 20;
print(result + 1);
```

#### Execute file

```sh
$ aurora run ./<file>.ar
```

#### That's the output from evaluator
```js
201
=
```

## Extra options

### Debug flag

All commands support tree (AST) mode

`$ aurora --tree`
`$ aurora --tree run ...`

Setting **tree mode** the CLI show up the [AST (Abstract syntax tree)](https://en.wikipedia.org/wiki/Abstract_syntax_tree) from source code 
