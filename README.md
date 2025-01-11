# aurora

🌌 Aurora's just for studying programming language concepts

> ⚠ Don't use it to develop something that'll go to production environment

## Summary

- [Get started](#get-started)
  - [Install CLI](#install-cli)
  - [Using REPL mode](#using-repl-mode)
  - [Execute from file](#execute-from-file)
  - [Writing some code](#writing-some-code)
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
>> var result = 1_000 * 2
=
>> result + 1
= 2001
>> print(result)
2000
=
```

### Execute from file

#### Create aurora source code file

```js
var result = 10 * 20
print(result + 1)
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

### Writing some code
> 🎈 Unfortunately this project there's no contributor enough to turn this doc better but be my guest discovering how to write some code looking at [examples folder](/examples).

## Extra options

### Debug flag

All commands support tree (AST) mode

`$ aurora --tree`
`$ aurora --tree run ...`

Setting **tree mode** the CLI show up the [AST (Abstract syntax tree)](https://en.wikipedia.org/wiki/Abstract_syntax_tree) from source code 
