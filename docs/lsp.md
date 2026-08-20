# Language server (`aurorals`)

`aurorals` is the Aurora language server. It speaks LSP over stdin/stdout, so any editor with an LSP client can use it — and coloring comes from the compiler's own lexer instead of a grammar each editor would have to keep in sync.

---

## What it provides

| Capability | Method | What you get |
|---|---|---|
| **Semantic tokens** | `textDocument/semanticTokens/full` | Coloring for keywords, numbers, text, comments, operators, identifiers, calls, shapes and fields |
| **Diagnostics** | `textDocument/publishDiagnostics` | Lexer and parser errors underlined where they happen, republished on every change |
| **Hover** | `textDocument/hover` | The description of the keyword under the cursor, what an identifier was bound to, a shape's fields, or which tape a field reads |
| **Completion** | `textDocument/completion` | Keywords as snippets, the identifiers and shapes declared in the document, and — right after a `.` — the fields of a shape or what a module offers |
| **Go to definition** | `textDocument/definition` | Where the name under the cursor was declared, in this file or in the module it came from |
| **Rename** | `textDocument/rename`, `textDocument/prepareRename` | A name changed everywhere it is written — for the names that cannot leave the file |

Document sync is **full** (`textDocumentSync: 1`): the client resends the whole file on each change.

### Completion

Three things come back, depending on where the cursor is.

**Right after a dot on a value**, the fields of that shape and nothing else. The shape comes
from what was declared — `ident p = Point{...}`, `... as Point`, or a call to a scope that
promised with `returns` — read straight from the tokens rather than from the tree, because a
document being edited hardly ever parses: the moment someone types `p.` there is no field name
yet, and that is exactly when completion is wanted. The dot is declared as a trigger character,
so the client asks on its own.

The shape does not have to be declared here. What the imported modules offer is written down
first — their shapes, and what their scopes promised — under the alias this file gave them, so
`ident s = g.new_square(4, 5);` is enough for `s.` to answer with `width` and `height`. That is
the same table the parser fills, read the same way.

**Right after a dot on a module alias**, what that module offers: the names it binds with
`ident`, each described by what it is, and the shapes it declares, each with its fields. A
module that could not be found offers nothing rather than everything — the diagnostic already
says it is missing.

**Everywhere else**, the keywords and whatever the document declares. A declared shape comes
back as a way of building one, with its own field names as the places to fill in:

```
shape Point { x, y };   ->   Point{${1:x}, ${2:y}}
```

A shape of an imported module comes back the same way, under the name this file has to write
it with: `g.Square{${1:width}, ${2:height}}`.

**Keywords expand into their shape**, for the forms that have one to get wrong:

| Typing | Gets |
|---|---|
| `defer` | `defer {` … `}` |
| `ident` | `ident name = value;` |
| `if` | `if condition {` … `}` |
| `branch` | `branch {` test`:` value`,` fallback`; }` |
| `shape` | `shape Name { field };` |
| `assert` | `assert(condition, "message");` |
| `printb` `printc` `printd` | `printb value;` |
| `feed` | `feed(0)` |
| `pull` `push` `head` `tail` | `pull tape value` |

Snippets are offered **only to a client that said it expands them** (`snippetSupport` at
initialize). To anyone else the placeholders would land in the buffer as the literal text
they are, so that client gets the bare keyword.

### Go to definition

Four things are names, and each one answers with where it was written:

| Under the cursor | Lands on |
|---|---|
| a value | the `ident` that binds it |
| a shape | the `shape` that declares it |
| a field | the field's name inside that declaration |
| an alias, or anywhere in a `use` line | the top of the module's file |

A name reached through an alias — `g.area`, `g.Square` — is declared in that module's file and
never in this one, so the jump goes there. So does a field of a value whose shape came from a
module: `s.width` lands inside `shape Square { width, height };`, in the file that declared it.
The module's own source is lexed and asked exactly what the open document is asked, which is
why a value, a shape and a field all cross without a rule of their own.

Asking about a name where it is declared answers with itself. That is what "declared here"
looks like, and it reads differently from "not declared at all", which answers with nothing.

The binding that answers is **the one above the cursor**, so a name bound twice lands on the
one the language itself would find. A binding that only appears further down is answered with
anyway: a deferred scope runs when it is called, so its body can name something written under
it.

A module that is not there, and a name a module does not have, are jumps that do not happen.
The diagnostic already says what is wrong, and an editor that opens nothing is better than one
that opens the wrong file.

### Rename

A name is changed everywhere it is written, in one pass, and the editor is handed every edit
at once.

It refuses more than it accepts, and that is deliberate. An editor that offers a rename it
cannot finish is worse than one that offers none: the half that was renamed compiles, the half
that was not is in somebody else's file, and nothing on screen says which is which.

| Under the cursor | |
|---|---|
| a name bound inside a scope | renamed, everywhere that scope reads it |
| an alias | renamed, in the `use` line and every reach through it |
| a name bound at the top of a file | **refused** — another file may be importing it |
| a field | **refused** — it belongs to a shape, and a shape crosses |
| a name of another module | **refused** — it is declared in a file this one does not own |

The refusal carries its reason, and a client shows it. It arrives at `prepareRename`, before
the box opens, for a client that asks — and again from the rename itself, for one that does
not.

A new name is checked before anything is rewritten. Whether it is a name at all is asked of
the lexer, so a keyword, a number or two words are refused for the same reason the compiler
would refuse them; and a shape's new name has to start with a capital letter.

**Scopes are read as the language reads them.** A declaration is a name inside the block it
was written in, and the block ends at its brace. A name written where two declarations reach
it means the deeper one, and where one block declares it twice, the one above it. Where there
is none above, the one below answers: a deferred scope runs when it is called, so its body may
name something written under it.

**Scope:** the server lexes and parses the open document and the files it imports, and never evaluates. The imported files arrive through a port the host fills in — the command line reads a disk, the playground reads a map it already holds, since a browser has no files — so the same package answers wherever it is put. An imported file that is open in the editor is read as it is on screen, not as it is on disk: a name just typed resolves, and one just deleted stops resolving. See [modules.md](modules.md) for what a module is.

**Known limitations**

- The parser stops at the first error, so **one diagnostic per pass**. Fix it and the next one appears.
- Go to definition lands on a declaration, never on every place a name is used: there is no
  `textDocument/references`. Renaming knows those places, and shows them to nobody.
- A rename stops at the file. What a module offers is refused rather than followed into the
  files that import it, which is a walk of the project the server does not do yet.
- Scope is read as the file is written, so a name declared inside a deferred scope and one
  declared at the top are told apart by which comes first, not by which is visible.
- No code actions, no formatting, no incremental sync.
- Semantic token types are decided lexically. A call is anything followed by `(`, a shape anything after `shape` or `as`, a field anything after `.`.

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
