# Aurora manifest reference (`aurora.toml`)

The Aurora manifest is a TOML file named `aurora.toml` that describes your project and how the CLI should build, run, deploy, and call your program. It is created by `aurora init` and lives at the root of your project.

A manifest lets you name a **profile** instead of repeating paths and settings on every command. It is not always required: `run` and `build` also take a path, and a path needs no project at all.

Deploy state (contract address, tx hash, deployed-at per profile) is stored in a **separate hidden file** (`.aurora.deploys.toml`) so that `aurora.toml` stays clean and editable.

The manifest has two scopes in `aurora.toml`: **`[project]`** and **`[profiles.<name>]`**. This document also describes the deploy state file.

---

## Location and discovery

- **Filename:** `aurora.toml`
- **Location:** Project root (the directory that contains `aurora.toml`).
- **Discovery:** When you run a command (e.g. from `my-project/src/`), the CLI walks up the directory tree until it finds `aurora.toml`. That directory is the project root; paths in the manifest are relative to it.

---

## Selecting a profile

`run` and `build` take **one argument**, and its shape decides what it means:

| Command | What it compiles | Manifest |
|---|---|---|
| `aurora run` | the `main` profile | required |
| `aurora run dev` | the `dev` profile | required |
| `aurora run src/app.ar` | that file | **not used** |

A path is anything ending in `.ar`; anything else is read as a profile name. So a loose file runs from anywhere, with no project around it, and the manifest only comes into play when you name a profile — or name nothing, which means `main`.

An argument that looks like a path but has no `.ar` extension is rejected rather than looked up as a profile:

```
$ aurora run ./app
"./app" is neither a profile name nor an .ar file
```

`build` follows the same rule and adds `-o` for the output path:

```sh
aurora build                          # profile main: source -> binary
aurora build dev                      # profile dev
aurora build src/app.ar               # writes ./app, next to where you are
aurora build src/app.ar -o bin/app    # -o wins in every case
```

Without `-o`, a profile gives the output path (`binary`); a loose file has no profile to ask, so the binary takes the source's name in the working directory.

**`deploy` and `call` are different:** they read `rpc` and `privkey` from a profile, so they always need a manifest and still select the profile with `-p/--profile`.

---

## `[project]`

Project-wide metadata. Used for identification and future tooling (e.g. a registry).

| Field      | Required | Default (from `aurora init`) | Description |
|------------|----------|------------------------------|-------------|
| **`name`** | Yes      | Base name of the folder where `aurora init` was run | Project identifier. Keep it short and valid for use in tooling (e.g. no spaces). |
| **`version`** | No   | `"0.1.0"` | Project version. Semantic versioning (e.g. `1.2.3`) is recommended. |
| **`tape_size`** | No | `8` | Width in bytes of every value in the project, from 1 to 32. A literal that does not fit is rejected at compile time. Overridden by `--tape-size` on the command line. See [language-design.md](language-design.md#tape-size-how-many-bytes-a-value-has). |
| **`source_root`** | No | `"src"` | Directory a module name resolves from: `use a/b/c as x;` reads `<source_root>/a/b/c.ar`. Relative to the directory the command was run in. See [modules.md](modules.md). |

**Why:** `name` and `version` give the project an identity and allow scripts or future features (e.g. publishing) to refer to it in a stable way.

**`source_root` is the project's for the same reason `tape_size` is**: it decides what the source means rather than what one task does with it. Two profiles with two roots would make one `use` line name two different files, and leave anything reading the project as a whole with nothing to answer. It is refused inside a profile, with the message saying where to move it.

Note that it is relative to **where the command was run**, with a manifest or without — which is one rule with no exception, and why a project with modules is run from its own root.

**`tape_size` is the project's and not a profile's** because it is not a path or a setting: it is the dialect the source is written in. At one byte `255 + 1` is `0` and `"Gui"` does not compile; at eight both answer otherwise. Two profiles with two widths made one file mean two things, and left everything that reads the project as a whole — the language server above all — with no width to answer for. A `tape_size` left inside a profile is refused, with the message saying where to move it.

---

## `[profiles.<name>]`

Profiles define how to build and run your program and, optionally, how to deploy and call it on a chain. The default profile created by `aurora init` is **`main`** (`[profiles.main]`). You can add others (e.g. `[profiles.sepolia]`, `[profiles.local]`).

### Fields created by default

These are written by `aurora init` and are enough for **build** and **run**.

| Field        | Required | Default   | Description |
|--------------|----------|-----------|-------------|
| **`source`** | Yes      | `"src/main.ar"` | Path to the Aurora source file, relative to the project root. Used by `aurora run` and `aurora build` when this profile is selected. |
| **`binary`** | Yes      | `"bin/main"`    | Path where the compiled bytecode is written, relative to the project root. Used by `aurora build` when you don’t pass `-o`, and read by `aurora deploy` as the bytecode to send — so **`deploy` needs a `build` first**. The name usually matches the source filename without extension (e.g. `main.ar` → `bin/main`). |

**Why:**  
- **`source`** centralizes the entry point so `aurora run` and `aurora build` work with a profile name, or with no argument at all.  
- **`binary`** centralizes the build output so `aurora build` and `aurora deploy` agree on where the bytecode lives.

### Optional fields (on-chain)

These are **not** created by `aurora init`. Add them when you want to use **deploy** or **call** on a network.

To **deploy**, you need a **wallet** (to sign the deploy transaction) and an **RPC** endpoint. Configure them in the profile:

| Field         | Required for      | Description |
|---------------|-------------------|-------------|
| **`rpc`**     | **deploy**, **call** | URL of the Ethereum (or compatible) node RPC endpoint. Examples: `http://127.0.0.1:8545` (local), `https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY` (Sepolia). Used to send transactions (deploy) and to perform read-only calls (call). |
| **`privkey`** | **deploy**        | **Wallet private key** in hex (no `0x` prefix). This is the key of the wallet that will pay for gas and own the deploy transaction. Used to sign deploy transactions. **Keep `aurora.toml` out of version control** if it contains `privkey`, or use environment variable substitution / a secrets manager. |

**Why:**  
- **`rpc`** and **`privkey`** (wallet key) keep deploy configuration in one place per profile (e.g. main, sepolia) instead of long CLI arguments.

**Security: name the value instead of holding it.**

A manifest is tracked and a key is not, so a value may be written as a reference to the
environment and read at load time:

```toml
[profiles.main]
  source = "src/main.ar"
  binary = "bin/main"
  rpc = "${{ AURORA_RPC }}"
  privkey = "${{ AURORA_PRIVKEY }}"
```

Where it is read from, in order:

1. **`.env` beside the manifest**, which is what a project keeps out of what it tracks. It
   holds `NAME=value` a line at a time; `#` opens a comment, `export` in front is allowed, and
   a value may be quoted when it means the spaces at its ends. Nothing else is interpreted — a
   value here is a secret or an address, and both mean their own bytes.
2. **The environment the command runs in**, when `.env` does not have the name.

That order is what lets a project be cloned and run: the project says what it needs beside
itself, and a machine that knows better — a build server holding the real key, someone running
against another chain — still gets to say so for whatever the project did not write down.

**A name nothing sets is refused**, not read as empty. An empty value reaches a deploy as a key
that is not a key, and the failure that follows says nothing about the manifest that caused it.

The braces are doubled on purpose: `${NAME}` and `$NAME` are left alone, so a manifest that
passes through a shell is not quietly rewritten by it.

**Only `rpc` and `privkey` are read this way.** A path is a path and a name is a name — neither
is a thing a project would keep somewhere else — and a reference written anywhere else stays as
it was written. A comment is not a value either, so a manifest may show the form beside the
setting it explains without being asked to resolve the example.

`.env` is in the repository's `.gitignore`. If you write a key straight into `aurora.toml`
instead, do not commit that file.

---

## Deploy state file (`.aurora.deploys.toml`)

Deploy state is stored in a **hidden file** at the project root: **`.aurora.deploys.toml`**. This file is **generated and managed by the Aurora CLI; do not edit it.**

**Purpose:** It stores the last deploy result per profile (contract address, transaction hash, deployed-at timestamp). The CLI uses it so that **`aurora call`** knows which contract to target for each profile. On every **`aurora deploy`**, the file is regenerated with the updated state for the profile you deployed, **while keeping the state for other profiles** unchanged.

| Field               | Written by   | Description |
|---------------------|--------------|-------------|
| **`contract_address`** | CLI (deploy) | Contract address (e.g. `0x...`) of the last deployment. Used by **`aurora call`** to target the contract for this profile. Overwritten on each deploy for that profile. |
| **`tx_hash`**         | CLI (deploy) | Transaction hash (e.g. `0x...`) of the deploy transaction. Useful for looking up the deploy on an explorer. |
| **`deployed_at`**     | CLI (deploy) | Timestamp of the exact moment of the deploy (RFC3339). |

**Why a separate file:**  
- **`aurora.toml`** stays clean and fully editable (comments, formatting) and is not rewritten on deploy.  
- Deploy state is isolated and regenerated only when you run deploy; other profiles’ state is preserved.

---

## Example: build and run only (default after `aurora init`)

```toml
[project]
name = "myapp"
version = "0.1.0"

[profiles.main]
source = "src/main.ar"
binary = "bin/main"
```

```sh
aurora run        # runs src/main.ar
aurora build      # writes bin/main
```

A working example lives in [`examples/project/`](../examples/project/), with a second profile that pins the tape size.

---

## Example: after deploy

Your **`aurora.toml`** stays as below (deploy state is not written here):

```toml
[project]
name = "myapp"
version = "0.1.0"

[profiles.main]
source = "src/main.ar"
binary = "bin/main"
rpc = "http://127.0.0.1:8545"
privkey = "<hex private key, no 0x>"
```

After **`aurora deploy`**, the CLI creates or updates **`.aurora.deploys.toml`** (at the project root) with the contract address, tx hash, and deployed-at for that profile. Use **`aurora call <function>`** and the CLI will read the contract address from the deploy state file.

---

## Example: multiple profiles

You can define several profiles and, after deploying for each, have separate deploy state entries in `.aurora.deploys.toml`.

```toml
[project]
name = "myapp"
version = "0.1.0"

[profiles.main]
source = "src/main.ar"
binary = "bin/main"
rpc = "http://127.0.0.1:8545"
privkey = "<hex private key, no 0x>"

[profiles.sepolia]
source = "src/main.ar"
binary = "bin/main"
rpc = "https://eth-sepolia.g.alchemy.com/v2/..."
privkey = "<hex private key, no 0x>"
```

Deploy state for each profile lives in **`.aurora.deploys.toml`** (created/updated by the CLI on deploy). Select the profile with `-p`:

```sh
aurora build sepolia          # bytecode for that profile
aurora deploy -p sepolia      # sends what build wrote
aurora call getResult -p sepolia
```

---

## Summary

| Scope / file              | Purpose |
|---------------------------|---------|
| **`[project]`**            | Project identity and dialect: `name`, `version`, `tape_size`. |
| **`[profiles.<name>]`**    | Build and chain config per environment: `source`, `binary`, and optionally `rpc`, `privkey`. Do **not** put contract address here. |
| **`.aurora.deploys.toml`** | Last deploy state per profile: `contract_address`, `tx_hash`, `deployed_at`. Generated by the CLI on deploy; do not edit. Used by **call** for the contract address. |

**Profile fields:** `source`, `binary` (default from init); `rpc`, `privkey` (optional).  
**Deploy state file:** `.aurora.deploys.toml` holds `contract_address`, `tx_hash`, `deployed_at` per profile (CLI-only).

**How a command finds its profile:** `run` and `build` take it as the argument (or a path, which skips the manifest entirely); `deploy` and `call` take `-p`.

For the main project README and getting started, see the [Project manifest](../README.md#project-manifest) section.
