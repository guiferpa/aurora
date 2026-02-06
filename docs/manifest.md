# Aurora manifest reference (`aurora.toml`)

The Aurora manifest is a TOML file named `aurora.toml` that describes your project and how the CLI should build, run, deploy, and call your program. It is created by `aurora init` and must live at the root of your project (or in a parent directory of where you run Aurora commands).

The manifest has three scopes: **`[project]`**, **`[profiles.<name>]`**, and **`[deploys.<name>]`**. This document describes every section and field, including those not created by default.

---

## Location and discovery

- **Filename:** `aurora.toml`
- **Location:** Project root (the directory that contains `aurora.toml`).
- **Discovery:** When you run a command (e.g. from `my-project/src/`), the CLI walks up the directory tree until it finds `aurora.toml`. That directory is the project root; paths in the manifest are relative to it.

---

## `[project]`

Project-wide metadata. Used for identification and future tooling (e.g. a registry).

| Field      | Required | Default (from `aurora init`) | Description |
|------------|----------|------------------------------|-------------|
| **`name`** | Yes      | Base name of the folder where `aurora init` was run | Project identifier. Keep it short and valid for use in tooling (e.g. no spaces). |
| **`version`** | No   | `"0.1.0"` | Project version. Semantic versioning (e.g. `1.2.3`) is recommended. |

**Why:** `name` and `version` give the project an identity and allow scripts or future features (e.g. publishing) to refer to it in a stable way.

---

## `[profiles.<name>]`

Profiles define how to build and run your program and, optionally, how to deploy and call it on a chain. The default profile created by `aurora init` is **`main`** (`[profiles.main]`). You can add others (e.g. `[profiles.sepolia]`, `[profiles.local]`).

### Fields created by default

These are written by `aurora init` and are enough for **build** and **run**.

| Field        | Required | Default   | Description |
|--------------|----------|-----------|-------------|
| **`source`** | Yes      | `"src/main.ar"` | Path to the main Aurora source file, relative to the project root. Used by `aurora build`, `aurora run`, and (when building before deploy) by `aurora deploy` when you don’t pass a file path. |
| **`binary`** | Yes      | `"bin/main"`    | Path where the compiled binary (bytecode) is written, relative to the project root. Used by `aurora build` when you don’t pass `-o`, and by `aurora deploy` as the contract bytecode to send. The name usually matches the source filename without extension (e.g. `main.ar` → `bin/main`). |

**Why:**  
- **`source`** centralizes the entry point so `aurora build` and `aurora run` can be used without arguments.  
- **`binary`** centralizes the build output so `aurora build` and `aurora deploy` agree on where the bytecode lives.

### Optional fields (on-chain)

These are **not** created by `aurora init`. Add them when you want to use **deploy** or **call** on a network.

| Field         | Required for      | Description |
|---------------|-------------------|-------------|
| **`rpc`**     | **deploy**, **call** | URL of the Ethereum (or compatible) node RPC endpoint. Examples: `http://127.0.0.1:8545` (local), `https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY` (Sepolia). Used to send transactions (deploy) and to perform read-only calls (call). |
| **`privkey`** | **deploy**        | Path to a file that contains the deployer’s private key in hex (one line, no `0x` prefix), relative to the project root. The CLI reads this file to sign deploy transactions. **Never** put the key directly in `aurora.toml`; use a path and add that file to `.gitignore`. |

**Why:**  
- **`rpc`** and **`privkey`** keep deploy configuration in one place per profile (e.g. main, sepolia) instead of long CLI arguments.

**Security:**  
- Do not commit files referenced by `privkey`. Add them (e.g. `*.key`, `.aurora/`) to `.gitignore`.  
- Prefer environment-specific paths or env vars for RPC URLs if they contain secrets.

---

## `[deploys.<name>]`

Deploy state for each profile. **Do not edit this section by hand.** It is written and overwritten by the CLI whenever you run **`aurora deploy`** for that profile.

The section name matches the profile name (e.g. `[deploys.main]` for profile `main`). After a deploy, the CLI updates the corresponding `[deploys.<name>]` with the new contract address and the exact time of the deploy. Every new deploy of a binary overwrites these values.

| Field           | Written by   | Description |
|-----------------|--------------|-------------|
| **`contract_address`** | CLI (deploy) | Contract address (e.g. `0x...`) of the last deployment. Used by **`aurora call`** to target the contract for this profile. Overwritten on each deploy. |
| **`deployed_at`**       | CLI (deploy) | Timestamp of the exact moment of the deploy (e.g. RFC3339). Overwritten on each deploy. |

**Why:**  
- The contract address for **call** comes from **`[deploys.<name>].contract_address`**, not from the profile.  
- You should **not** add or maintain a contract address manually; the CLI keeps it in sync with the last deploy.  
- **`deployed_at`** gives you an audit trail of when the current address was deployed.

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

---

## Example: after deploy (CLI has written `[deploys.main]`)

```toml
[project]
name = "myapp"
version = "0.1.0"

[profiles.main]
source = "src/main.ar"
binary = "bin/main"
rpc = "http://127.0.0.1:8545"
privkey = ".aurora/local.key"

[deploys.main]
contract_address = "0x1234567890abcdef..."
deployed_at = "2025-02-05T14:30:00Z"
```

The **`[deploys.main]`** block is created or overwritten by **`aurora deploy`**. Use **`aurora call <function>`** and the CLI will use **`deploys.main.contract_address`** as the contract.

---

## Example: multiple profiles

You can define several profiles and, after deploying for each, have separate `[deploys.<name>]` entries.

```toml
[project]
name = "myapp"
version = "0.1.0"

[profiles.main]
source = "src/main.ar"
binary = "bin/main"
rpc = "http://127.0.0.1:8545"
privkey = ".aurora/local.key"

[profiles.sepolia]
source = "src/main.ar"
binary = "bin/main"
rpc = "https://eth-sepolia.g.alchemy.com/v2/..."
privkey = ".aurora/sepolia.key"

[deploys.main]
contract_address = "0x..."
deployed_at = "2025-02-05T14:30:00Z"

[deploys.sepolia]
contract_address = "0x..."
deployed_at = "2025-02-05T15:00:00Z"
```

---

## Summary

| Scope                | Purpose |
|----------------------|---------|
| **`[project]`**      | Project identity: `name`, `version`. |
| **`[profiles.<name>]`** | Build and chain config per environment: `source`, `binary`, and optionally `rpc`, `privkey`. Do **not** put contract address here. |
| **`[deploys.<name>]`**  | Last deploy state per profile: `contract_address`, `deployed_at`. Written and overwritten by the CLI on each deploy; do not maintain by hand. |

**Profile fields:** `source`, `binary` (default from init); `rpc`, `privkey` (optional, for on-chain).  
**Deploy fields:** `contract_address`, `deployed_at` (CLI-only; used by **call** for the contract address).

For the main project README and getting started, see the [Project manifest](../README.md#project-manifest) section.
