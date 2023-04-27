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
>> var result = 10 * 20;
=
>> result + 1;
= 201
>> print(result);
200
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

All commands support debug mode

`$ aurora --debug`
`$ aurora --debug run ...`

Setting **debug mode** the CLI show up the [AST (Abstract syntax tree)](https://en.wikipedia.org/wiki/Abstract_syntax_tree) from source code 
