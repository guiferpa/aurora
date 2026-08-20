# Language server (`aurorals`)

`aurorals` is the Aurora language server. It speaks LSP over stdin/stdout, so any editor with an LSP client can use it — and coloring comes from the compiler's own lexer instead of a grammar each editor would have to keep in sync.

---

## What it provides

| Capability | Method | What you get |
|---|---|---|
| **Semantic tokens** | `textDocument/semanticTokens/full` | Coloring for keywords, numbers, text, comments, operators, identifiers, calls, structs and fields |
| **Diagnostics** | `textDocument/publishDiagnostics` | Lexer and parser errors underlined where they happen, republished on every change |
| **Hover** | `textDocument/hover` | The description of the keyword under the cursor, what an identifier was bound to, a struct's fields, or which tape a field reads |
| **Completion** | `textDocument/completion` | Keywords as snippets, the identifiers and structs declared in the document, and — right after a `.` — the fields of a struct or what a module offers |

Document sync is **full** (`textDocumentSync: 1`): the client resends the whole file on each change.

### Completion

Three things come back, depending on where the cursor is.

**Right after a dot on a value**, the fields of that struct and nothing else. The shape comes
from what was declared — `ident p = Point{...}`, `... as Point`, or a call to a scope that
promised with `returns` — read straight from the tokens rather than from the tree, because a
document being edited hardly ever parses: the moment someone types `p.` there is no field name
yet, and that is exactly when completion is wanted. The dot is declared as a trigger character,
so the client asks on its own.

The struct does not have to be declared here. What the imported modules offer is written down
first — their structs, and what their scopes promised — under the alias this file gave them, so
`ident s = g.new_square(4, 5);` is enough for `s.` to answer with `width` and `height`. That is
the same table the parser fills, read the same way.

**Right after a dot on a module alias**, what that module offers: the names it binds with
`ident`, each described by what it is, and the structs it declares, each with its fields. A
module that could not be found offers nothing rather than everything — the diagnostic already
says it is missing.

**Everywhere else**, the keywords and whatever the document declares. A declared struct comes
back as a way of building one, with its own field names as the places to fill in:

```
struct Point { x, y };   ->   Point{${1:x}, ${2:y}}
```

A struct of an imported module comes back the same way, under the name this file has to write
it with: `g.Square{${1:width}, ${2:height}}`.

**Keywords expand into their shape**, for the forms that have one to get wrong:

| Typing | Gets |
|---|---|
| `defer` | `defer {` … `}` |
| `ident` | `ident name = value;` |
| `if` | `if condition {` … `}` |
| `branch` | `branch {` test`:` value`,` fallback`; }` |
| `struct` | `struct Name { field };` |
| `assert` | `assert(condition, "message");` |
| `printb` `printc` `printd` | `printb value;` |
| `feed` | `feed(0)` |
| `pull` `push` `head` `tail` | `pull tape value` |

Snippets are offered **only to a client that said it expands them** (`snippetSupport` at
initialize). To anyone else the placeholders would land in the buffer as the literal text
they are, so that client gets the bare keyword.

**Scope:** the server lexes and parses the open document and the files it imports, and never evaluates. The imported files arrive through a port the host fills in — the command line reads a disk, the playground reads a map it already holds, since a browser has no files — so the same package answers wherever it is put. An imported file that is open in the editor is read as it is on screen, not as it is on disk: a name just typed resolves, and one just deleted stops resolving. See [modules.md](modules.md) for what a module is.

**Known limitations**

- The parser stops at the first error, so **one diagnostic per pass**. Fix it and the next one appears.
- No go-to-definition, no code actions, no formatting, no incremental sync.
- Semantic token types are decided lexically. A call is anything followed by `(`, a struct anything after `struct` or `as`, a field anything after `.`.

---

## Install

```sh
go install github.com/guiferpa/aurora/cmd/aurorals@latest   # or: make build-force
```

Release archives (and the Homebrew tap) ship `aurorals` next to `aurora`. The binary must be on your `PATH` for the editor to launch it.

```sh
aurorals --version
aurorals --log /tmp/aurorals.log   # optional; logs go to stderr by default
```

Running `aurorals` by hand does nothing visible: it waits for LSP messages on stdin.

---

## Neovim

Two steps: teach Neovim that `.ar` is Aurora, then start the server for that filetype. Neovim enables semantic-token highlighting on its own once the server advertises it.

### Neovim 0.11+

```lua
vim.filetype.add({ extension = { ar = "aurora" } })

vim.lsp.config("aurorals", {
  cmd = { "aurorals" },
  filetypes = { "aurora" },
  root_markers = { "aurora.toml", ".git" },
})

vim.lsp.enable("aurorals")
```

### Neovim 0.9 / 0.10

```lua
vim.filetype.add({ extension = { ar = "aurora" } })

vim.api.nvim_create_autocmd("FileType", {
  pattern = "aurora",
  callback = function(args)
    vim.lsp.start({
      name = "aurorals",
      cmd = { "aurorals" },
      root_dir = vim.fs.dirname(vim.fs.find({ "aurora.toml", ".git" }, { upward = true })[1]),
    }, { bufnr = args.buf })
  end,
})
```

Neither snippet needs `nvim-lspconfig`: `aurorals` has no options to configure beyond the command itself.

### Checking it works

Open a `.ar` file and:

- `:checkhealth vim.lsp` (or `:LspInfo`) — the client should be attached to the buffer.
- `:Inspect` with the cursor on a keyword — the highlight should include an `@lsp.type.keyword` group. That group is what proves the color is coming from the server; if you only see `Normal`, the client attached but semantic tokens are off.
- Break the file on purpose (drop a `;`) — the diagnostic appears under the offending token and clears when you fix it.
- `K` on `ident` or `defer` — hover shows the keyword description.

Semantic token types map to the standard `@lsp.type.*` highlight groups, so any modern colorscheme colors Aurora with no extra work.

---

## Other editors

Any LSP client works: run `aurorals` with stdio transport and register the `.ar` extension. Only the Neovim setup is documented here for now. Note that VS Code cannot attach a server without an extension, so it needs one to be written before `.ar` files light up there.

---

## Debugging

```sh
aurorals --log /tmp/aurorals.log
tail -f /tmp/aurorals.log
```

Every incoming method and payload is logged. To exercise the server without an editor, speak the protocol directly — each message is a `Content-Length` header, a blank line, then JSON:

```sh
printf 'Content-Length: 60\r\n\r\n{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | aurorals
```
