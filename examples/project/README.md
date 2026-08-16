# A project with profiles

The examples one level up are loose files: you run each by path, from anywhere. This directory is the other way to work — a **project**, described by an [`aurora.toml`](aurora.toml) manifest.

## What a profile is

A profile is a named set of paths and settings, so a command does not need them spelled out every time. This project has two:

| Profile | Source | Binary |
|---|---|---|
| `main` | `src/main.ar` | `bin/main` |
| `label` | `src/label.ar` | `bin/label` |

`main` is the one used when no profile is named. A project holds as many profiles as it has things to build, and the two here are unrelated programs — that is what a profile is for.

## Running it

From **this directory**:

```sh
aurora run              # the main profile: src/main.ar
aurora run label        # the label profile: src/label.ar
aurora run src/main.ar  # that file, without naming a profile
```

The argument decides what it means: something ending in `.ar` is a path, anything else is a profile name, and nothing at all is `main`. A path skips the profile — but not the project: the file is still compiled in the width the project declares, so naming it either way gives the same program.

```sh
aurora build            # writes bin/main, the binary path from the profile
aurora build label      # writes bin/label
aurora build label -o /tmp/x.bin   # -o wins over the profile
```

The manifest is found by walking up from the working directory, so these commands also work from `src/`. Running from outside the project, a path still works — `aurora run examples/project/src/main.ar` needs no manifest at all.

## Running the tests

`src/main.test.ar` tests `src/main.ar` — a test belongs to the source file of the same name, which runs first so the test sees what it declared.

```sh
aurora test              # searches from src/ down, since that is where main.ar lives
aurora test src/main.test.ar   # just that file
```

Full reference: [testing](../../docs/testing.md).

## The tape size

`tape_size` under `[project]` is how wide every value in the project is — the dialect its source is written in, not a build setting. It is commented out in this manifest, so the project is at the default of 8.

```sh
aurora run --tape-size 1    # the same program in a one-byte dialect
```

The flag beats the manifest, which beats the default of 8, and the flag lasts one run. Declaring it in the manifest is what makes the project compile the same way for whoever runs it, instead of depending on a flag someone has to remember.

It belongs to the project rather than to a profile because it decides what the source *means*: at one byte `255 + 1` is `0` and `"box"` no longer fits — a tape holds one byte, so that is how much text fits in one. A file cannot be one dialect for one profile and another for the next.

Full reference: [manifest](../../docs/manifest.md).
