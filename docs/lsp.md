# Language server (`aurorals`)

`aurorals` is the Aurora language server. It speaks LSP over stdin/stdout, so any editor with an LSP client can use it — and coloring comes from the compiler's own lexer instead of a grammar each editor would have to keep in sync.

---

## What it provides

| Capability | Method | What you get |
|---|---|---|
| **Semantic tokens** | `textDocument/semanticTokens/full` | Coloring for keywords, numbers, reels (strings), comments, operators, identifiers, calls and namespace segments |
| **Diagnostics** | `textDocument/publishDiagnostics` | Lexer and parser errors underlined where they happen, republished on every change |
| **Hover** | `textDocument/hover` | The description of the keyword under the cursor, or what an identifier was bound to |
| **Completion** | `textDocument/completion` | Language keywords plus the identifiers declared in the open document |

Document sync is **full** (`textDocumentSync: 1`): the client resends the whole file on each change.

**Scope:** the server analyses the **open document alone** — it lexes and parses it, and never evaluates. That is what diagnostics and coloring need, and it stays within what the compiler can actually do: resolution of `use ns as alias` across files is not implemented yet (see [import_design.md](import_design.md)).

**Known limitations**

- The parser stops at the first error, so **one diagnostic per pass**. Fix it and the next one appears.
- No go-to-definition, no code actions, no formatting, no incremental sync.
- Semantic token types are decided lexically. A call is anything followed by `(`, a namespace segment anything followed by `::`.

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
