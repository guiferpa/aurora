# aurora

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
$ go install github/com/guiferpa/aurora/cmd/aurora
```
> 🎈 So far there's no an easier way to download aurora binary. Use Go to install, it's the better way for while.

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
> 🎈 Unfortunately, this project there's are not contributors enough to make this doc better but be my guest to discovery how to write some code looking at [examples folder](/examples).

## Try it out

### Playground
> 🚀 Feel free to try Aurora with [playground](https://guiferpa.github.io/aurora) built with WebAssembly + Go (Aurora source code)
<img width="942" alt="Screenshot 2025-01-12 at 12 27 41 AM" src="https://github.com/user-attachments/assets/51f073de-1fde-4a68-9cf8-b608b5b83032" />

## Extra options

### Debug flag

All commands it'll show deep dive in instructions and evaluating

`$ aurora --debug`
`$ aurora --debug run ...`
