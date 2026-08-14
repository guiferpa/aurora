# A project with profiles

The examples one level up are loose files: you run each by path, from anywhere. This directory is the other way to work — a **project**, described by an [`aurora.toml`](aurora.toml) manifest.

## What a profile is

A profile is a named set of paths and settings, so a command does not need them spelled out every time. This project has two:

| Profile | Source | Binary | Tape size |
|---|---|---|---|
| `main` | `src/main.ar` | `bin/main` | default (8) |
| `tiny` | `src/tiny.ar` | `bin/tiny` | 1 |

`main` is the one used when no profile is named.

## Running it

From **this directory**:

```sh
aurora run              # the main profile: src/main.ar
aurora run tiny         # the tiny profile: src/tiny.ar, one-byte tapes
aurora run src/main.ar  # that file, ignoring profiles entirely
```

The argument decides what it means: something ending in `.ar` is a path, anything else is a profile name, and nothing at all is `main`.

```sh
aurora build            # writes bin/main, the binary path from the profile
aurora build tiny       # writes bin/tiny
aurora build tiny -o /tmp/x.bin   # -o wins over the profile
```

The manifest is found by walking up from the working directory, so these commands also work from `src/`. Running from outside the project, a path still works — `aurora run examples/project/src/main.ar` needs no manifest at all.

## Running the tests

`src/main.test.ar` tests `src/main.ar` — a test belongs to the source file of the same name, which runs first so the test sees what it declared.

```sh
aurora test              # searches from src/ down, since that is where main.ar lives
aurora test src/main.test.ar   # just that file
```

Full reference: [testing](../../docs/testing.md).

## What the tape size changes

`tiny` pins `tape_size = 1`, so every value holds a single byte and arithmetic wraps at 256:

```sh
aurora run tiny        # [200] then [44]  — 200 + 100 wrapped past 255
aurora run tiny -t 8   # [0 0 0 0 0 0 0 200] then [0 0 0 0 0 0 1 44]
```

(The file also echoes both values. 200 is not printable ASCII, so what you see
for it depends on your terminal; 44 comes out as a comma.)

The flag beats the profile, which beats the default of 8. Pinning it in the manifest means the project compiles the same way for whoever runs it, instead of depending on a flag someone has to remember.

Full reference: [manifest](../../docs/manifest.md).
