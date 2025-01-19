# aurora

[![Go Reference](https://pkg.go.dev/badge/github.com/guiferpa/aurora.svg)](https://pkg.go.dev/github.com/guiferpa/aurora)
[![Last commit](https://img.shields.io/github/last-commit/guiferpa/aurora)](https://img.shields.io/github/last-commit/guiferpa/aurora)
[![Go Report Card](https://goreportcard.com/badge/github.com/guiferpa/aurora)](https://goreportcard.com/report/github.com/guiferpa/aurora)
![Pipeline workflow](https://github.com/guiferpa/aurora/actions/workflows/pipeline.yml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/guiferpa/aurora/badge.svg?branch=main)](https://coveralls.io/github/guiferpa/aurora?branch=main)

🌌 Aurora's just for studying programming language concepts

> ⚠ Don't use it to develop something that'll go to production environment

## Summary

- [Get started](#get-started)
  - [Install CLI](#install-cli)
  - [Using REPL mode](#using-repl-mode)
  - [Execute from file](#execute-from-file)
  - [Writing some code](#writing-some-code)
- [Try it out](#try-it-out)
  - [Playground](#playground)
- [Extra options](#extra-options)
  - [Debug flag](#debug-flag)

## Get started

### Install CLI
```sh
go install -v github.com/guiferpa/aurora/cmd/aurora@main
```
> 🎈 So far there's no an easier way to download aurora binary. Use Go to install, it's the better way for while.

### Using REPL mode

```sh
aurora repl
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
aurora run ./<file>.ar
```

#### That's the output from evaluator
```java
[0 0 0 0 0 0 0 201]
```

### Writing some code
> 🎈 Unfortunately, this project there's are not contributors enough to make this doc better but be my guest to discovery how to write some code looking at [examples folder](/examples).

## Try it out

### Playground
> 🚀 Feel free to try Aurora with [playground](https://guiferpa.github.io/aurora) built with WebAssembly + Go (Aurora source code)
<img width="942" alt="Screenshot 2025-01-12 at 12 27 41 AM" src="https://github.com/user-attachments/assets/51f073de-1fde-4a68-9cf8-b608b5b83032" />

## Extra options

```sh
aurora help

Usage:
  aurora [command]

Available Commands:
  build       Build binary from source code
  completion  Generate the autocompletion script for the specified shell
  eval        Evaluate aurora binary file built by build command
  help        Help about any command
  repl        Enter in Read-Eval-Print Loop mode
  run         Run program directly from source code
  version     Show toolbox version

Flags:
  -h, --help   help for aurora

Use "aurora [command] --help" for more information about a command.
```

### Debug flag

All commands it'll show deep dive in instructions and evaluating

`$ aurora repl --debug`
`$ aurora run --debug ...`
