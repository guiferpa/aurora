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
$ go install github/com/guiferpa/aurora/cmd/aurora
```

### Using REPL mode

```sh
$ aurora repl
```

```java
>> ident a = 1_000;
>> a + 1;
= 1001
```

### Execute from file

#### Create aurora source code file

```java
ident result = 10 * 20;
print result + 1;
```

#### Execute file

```sh
$ aurora run ./<file>.ar
```

#### That's the output from evaluator
```java
[0 0 0 0 0 0 0 201]
```

### Writing some code
> 🎈 Unfortunately this project there's no contributor enough to turn this doc better but be my guest discovering how to write some code looking at [examples folder](/examples).

## Extra options

### Debug flag

All commands it'll show deep dive in instructions and evaluating

`$ aurora --debug`
`$ aurora --debug run ...`
