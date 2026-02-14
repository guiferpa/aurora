# Defer: blob specification

This document describes the binary format and semantics of deferred-scope data stored in the evaluator: the **defer blob**, where it lives, and how it is used at definition and call time.

---

## Where defers are stored

- **Blob storage:** Each deferred scope is serialized into a **blob** (byte slice) and stored in the current **Environ** under the key `defers[key] = blob`.
- **Key:** The key is **incremental** per environ: `key = byteutil.ToHex(byteutil.FromUint64(uint64(environ.DefersLength())))` at store time (e.g. `"0000000000000000"`, `"0000000000000001"`, …).
- **Reference in ident:** The ident that “holds” the deferred value does **not** store the blob. It stores only the **reference** (the key as bytes). At call time, the evaluator resolves the ident value to a key, then does `environ.GetDefer(key)` to obtain the blob.
- **Lookup:** `GetDefer(key)` walks the environ chain (inner to outer), like `GetIdent`, so a deferred scope can be found from inner scopes that close over it.

---

## Blob layout

The blob has **no magic bytes**. It is a contiguous byte slice with the following layout:

| Offset | Size    | Field       | Description |
|--------|---------|-------------|-------------|
| `0`    | 8 bytes | **from**    | `uint64` big-endian. Index of the **first instruction** of the deferred block in the instruction array. In practice this is the index of **OpBeginScope** (the instruction immediately after OpDefer). At call time, execution runs from `from+1` through `to`. |
| `8`    | 8 bytes | **to**      | `uint64` big-endian. Index of the **last instruction** of the deferred block. In practice this is always the index of **OpReturn**. |
| `16`   | 1 byte  | **keyLen**  | Length in bytes of the **returnKey** field (0–255). Used when decoding to know how many bytes to read for the variable-length returnKey. |
| `17`   | N bytes | **returnKey** | Temp key (string, N = keyLen bytes) where the scope stores its return value when it runs. At call time, after executing the scope, the evaluator reads `environ.GetTemp(returnKey)` and copies that value to the call result temp. |

**Total length:** `17 + len(returnKey)` bytes (minimum 17 if returnKey is empty).

**Encoding:** `from` and `to` are written with `byteutil.FromUint64` (8 bytes big-endian each). Then one byte for keyLen, then the raw bytes of returnKey.

---

## Field semantics

### from

- **Definition (EvaluateDefer):** `from := e.cursor + 1`. When OpDefer is evaluated, `e.cursor` is the index of the OpDefer instruction; the next instruction is the start of the deferred block, i.e. **OpBeginScope**.
- **Meaning:** Index of the first instruction to run when the deferred scope is **called**. In the current bytecode layout this is always the OpBeginScope of that block.

### to

- **Definition (EvaluateDefer):** `to := from + scopeLen`, where `scopeLen` is the length in instructions of the deferred block (from OpDefer’s `right` operand).
- **Meaning:** Index of the last instruction of the block. In the current bytecode this is always **OpReturn**. At call time, `ExecuteInstructions(from+1, to)` runs the scope body including the final OpReturn.

### keyLen

- **Purpose:** The blob is a flat byte array. `from` and `to` have fixed size (8 bytes each). **returnKey** is a variable-length string. To decode the blob we must know how many bytes belong to returnKey; **keyLen** is that length (one byte, so returnKey can be 0–255 bytes).

### returnKey

- **Definition (EvaluateDefer):** `returnKey := byteutil.ToHex(left)`. The `left` operand of OpDefer identifies the “return slot” chosen by the compiler/emitter for this scope.
- **Purpose:** When the deferred scope is **called** (EvaluateCall), the evaluator runs the scope’s instructions. The scope’s OpReturn writes its value into a **temp** with some key. That key is the one we stored as returnKey. After execution, the caller does `retVal := e.environ.GetTemp(returnKey)` and then sets the call’s result temp from `retVal`. So **returnKey** is the name of the temp where the scope will write its return value; we need it to know where to read the result after the call.

---

## Definition and call flow (summary)

1. **OpDefer:** Compute `from`, `to`, `returnKey`; build blob with `encodeDeferBlob(from, to, returnKey)`; compute incremental key `key`; `environ.SetDefer(key, blob)`; store ref in temp as `SetTemp(label, []byte(key))`. Later that temp is assigned to an ident (e.g. `ident r = defer { ... };`).
2. **OpCall:** Get ident value (the ref); `blob := environ.GetDefer(refKey)`; decode blob to `from`, `to`, `returnKey`; push new frame with arguments; `ExecuteInstructions(from+1, to)`; read `environ.GetTemp(returnKey)` and set call result temp from it.

Scope visibility (which environs the defer body sees at call time) is described in [defer_scope_visibility.md](defer_scope_visibility.md).
