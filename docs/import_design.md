# Design: Import System for Aurora

## Goal

Allow Aurora files to use code from other namespaces, enabling:
- Separation of source code and tests
- Code reuse
- Modular organization

---

## Namespace dependency system (specification)

The following is the canonical specification for how namespace dependencies work in Aurora. Implementation should follow these rules.

### 1. Implicit import at use site

There is **no separate import step**. Import/linking is **implicit**: when you call a symbol from another namespace, that namespace is resolved and linked at the point of use.

In languages where you import first and then use symbols:

```
import "std/fs/io";
open_file(...);   // function from the imported namespace
```

In Aurora it works like this:

```aurora
std::fs::io::open_file(...);
```

The compiler resolves the namespace `std::fs::io` and links to the appropriate module when it sees this call. No prior `import` statement is required.

### 2. Optional alias with `use`

Any namespace can be given an **optional alias** via `use ... as` to shorten long names:

```aurora
use std::fs::io as io;
io::open_file(...);
```

- Without alias: use the full path at each call, e.g. `std::fs::io::open_file(...)`.
- With alias: `use std::fs::io as io;` then `io::open_file(...)`.

The alias is purely for readability and convenience; resolution still follows the same path rules.

### 3. Discovery: namespace path = filesystem path

Source code for a namespace is discovered by **mapping the namespace to a directory path** under the project root. The path used in the namespace (the sequence of segments separated by `::`) is the same as the path of directories and subdirectories on disk.

Rules:

- Namespace `std::fs::io` → compiler looks under `<root>/std/fs/io/` for `*.ar` modules.
- Namespace `std::fs::io::async` → compiler looks under `<root>/std/fs/io/async/` for `*.ar` modules.

In general: namespace `a::b::c` corresponds to path `<root>/a/b/c/`, and the compiler loads the relevant `*.ar` files from that directory (or subdirectories, depending on the exact discovery rules). The namespace is always at the level of directories and subdirectories under `<root>`.

### 4. Root: entrypoint and manifest

The **root** for resolving all namespaces is defined by the **compilation entrypoint**: by default, all modules are expected to live under the same root as the **file used as the entrypoint** of the compilation. In practice, the root is the **directory that contains the project manifest (`aurora.toml`)** (or the directory of the entrypoint file when no manifest is used).

- For now, this specification applies to **programmer-defined namespaces** only: code organized in directories under the project root for scalability.
- **Built-in language packages** and **third-party dependencies** will be specified later; they are out of scope for this section.

Summary: `<root>/` is the base for every namespace path. Example: if root is `/project`, then `std::fs::io` is resolved to `/project/std/fs/io/*.ar`.

---

## Legacy / alternative: explicit import (reference)

The sections below describe an alternative design based on **explicit `import`** and path syntax with `/`. That model may be deprecated in favor of the **namespace dependency system** above. Kept for reference and migration.

## Exemplo de Uso

```aurora
# math.ar
ident sum = {
  ident x = arguments(0);
  ident y = arguments(1);
  x + y;
};

ident multiply = {
  ident x = arguments(0);
  ident y = arguments(1);
  x * y;
};
```

```aurora
# math.test.ar
import math;

group "math operations" {
  case "sum should work" {
    ident result = math.sum(2, 3);
    assert result equals 5;
  };
  
  case "multiply should work" {
    ident result = math.multiply(3, 4);
    assert result equals 12;
  };
}
```

## Proposta de Sintaxe

### Import com alias opcional
```aurora
import math;                    # alias: math (derivado do path)
import utils/helpers;           # alias: helpers (último segmento)
import lib/string as str;       # alias: str (explícito)
import a/b/c;                   # alias: c (último segmento)

ident result = math.sum(2, 3);
ident text = str.concat("hello", "world");
ident helper = helpers.someFunction();
```

**Regras**:
- Alias é **opcional** - pode usar `import X;` ou `import X as Y;`
- Se não houver alias, usa o **último segmento do path** como alias
  - `import math;` → alias `math`
  - `import utils/helpers;` → alias `helpers`
  - `import a/b/c;` → alias `c`
- Caminhos sempre partem da **raiz** onde o compilador está sendo executado
- Não há suporte a caminhos relativos (`./`, `../`)
- **Sem aspas**: O caminho é um identificador/path, não uma string
- **Extensão automática**: O sistema automaticamente adiciona `.ar` ao final
- Uso: `alias.function()` ou `alias.ident`

## Resolução de Caminhos

1. **Caminhos sempre partem da raiz**: `math`, `utils/helpers`, `lib/string`
   - Resolvidos a partir do diretório onde o compilador está sendo executado
   - Exemplo: Se executar `aurora run src/app.ar` de `/project`, o import `math` busca `/project/math.ar`
   
2. **Sem caminhos relativos**: Não há suporte a `./` ou `../`
   - Todos os imports são resolvidos a partir da raiz de execução

3. **Extensão automática**: O sistema sempre adiciona `.ar` ao final do caminho
   - `import math as math` → busca `math.ar`
   - `import utils/helpers as helpers` → busca `utils/helpers.ar`

## Estrutura de Dados

### AST - Novo Node Type

```go
type ImportStatementNode struct {
    Path      string      // Caminho do arquivo (ex: "math" ou "utils/helpers")
    Alias     string      // Alias (ex: "math" ou "helpers" - derivado do path se não especificado)
    Token     lexer.Token // Token do import para linha/coluna
}

func (isn ImportStatementNode) Next() Node {
    return nil
}

// Helper para extrair último segmento do path
func extractLastSegment(path string) string {
    parts := strings.Split(path, "/")
    return parts[len(parts)-1]
}
```

### Módulo com Imports

```go
type ModuleNode struct {
    Name    string   // Nome do módulo (atualmente sempre "main")
    Imports []ImportStatementNode  // Lista de imports
    Stmts   []Node   // Statements do módulo
}
```

## Fluxo de Processamento

### 1. Lexer
- Adicionar token `IMPORT` para palavra-chave `import`
- Adicionar token `AS` para palavra-chave `as` (se usar alias)

### 2. Parser
- Reconhecer `import` como statement em qualquer lugar do código
- Parsear caminho (identificador ou path com `/`)
- Parsear alias opcional (se presente)
- Adicionar imports ao `ModuleNode` conforme aparecem
- **Ordem**: Imports podem aparecer em qualquer lugar, mas são processados na ordem de declaração

### 3. Resolver de Imports (novo componente)
- Resolver caminhos relativos ao arquivo atual
- Carregar arquivo importado
- Verificar ciclos de import (prevenir import circular)
- Parsear arquivo importado
- Retornar AST do arquivo importado

### 4. Emitter
- Processar imports na ordem em que aparecem no código
- Emitir código dos arquivos importados quando encontrados
- **Namespace com alias**: Identificadores importados ficam prefixados com o alias
- Exemplo: `math.sum()` -> busca `sum` no namespace `math`
- Criar escopo/namespace para cada alias importado
- **Importação dinâmica**: Imports podem aparecer no meio do código, mas o código importado é executado quando encontrado

### 5. Evaluator
- Executar código importado quando o import é encontrado
- Identificadores importados ficam disponíveis para uso após o import
- **Escopo**: Imports executados em ordem, código importado disponível após execução

## Detalhes de Implementação

### Resolver de Imports

```go
type ImportResolver struct {
    rootDir    string              // Diretório raiz (onde o compilador foi executado)
    visited    map[string]bool     // Arquivos já visitados (prevenir ciclos)
    resolved   map[string]AST      // Cache de ASTs resolvidos
}

func NewImportResolver(rootDir string) *ImportResolver {
    return &ImportResolver{
        rootDir:  rootDir,
        visited:  make(map[string]bool),
        resolved: make(map[string]AST),
    }
}

func (ir *ImportResolver) Resolve(path string) (AST, error) {
    // 1. Adicionar extensão .ar se não tiver
    if !strings.HasSuffix(path, ".ar") {
        path = path + ".ar"
    }
    
    // 2. Resolver caminho a partir da raiz
    fullPath := filepath.Join(ir.rootDir, path)
    
    // 3. Normalizar caminho (resolver .., ., etc)
    fullPath = filepath.Clean(fullPath)
    
    // 3. Verificar se já foi visitado (ciclo)
    if ir.visited[fullPath] {
        return AST{}, fmt.Errorf("circular import detected: %s", path)
    }
    
    // 4. Verificar cache
    if ast, ok := ir.resolved[fullPath]; ok {
        return ast, nil
    }
    
    // 5. Carregar e parsear arquivo
    ir.visited[fullPath] = true
    defer delete(ir.visited, fullPath)
    
    bs, err := os.ReadFile(fullPath)
    if err != nil {
        return AST{}, fmt.Errorf("failed to read import %s: %w", path, err)
    }
    
    tokens, err := lexer.GetFilledTokens(bs)
    if err != nil {
        return AST{}, fmt.Errorf("failed to tokenize import %s: %w", path, err)
    }
    
    ast, err := parser.NewWithFilename(tokens, fullPath).Parse()
    if err != nil {
        return AST{}, fmt.Errorf("failed to parse import %s: %w", path, err)
    }
    
    // 6. Resolver imports do arquivo importado (recursivo)
    // ... processar imports do arquivo importado
    
    ir.resolved[fullPath] = ast
    return ast, nil
}
```

### Modificações no Parser

```go
func (p *pr) getModule() (ModuleNode, error) {
    // Parsear statements (que podem incluir imports em qualquer lugar)
    imports := make([]ImportStatementNode, 0)
    stmts := make([]Node, 0)
    
    for p.GetLookahead().GetTag().Id != lexer.EOF {
        // Verificar se é um import
        if p.GetLookahead().GetTag().Id == lexer.IMPORT {
            imp, err := p.getImport()
            if err != nil {
                return ModuleNode{}, err
            }
            imports = append(imports, imp)
            stmts = append(stmts, imp) // Adicionar import como statement também
            if _, err := p.EatToken(lexer.SEMICOLON); err != nil {
                return ModuleNode{}, err
            }
        } else {
            // Parsear statement normal
            stmt, err := p.getStmt()
            if err != nil {
                return ModuleNode{}, err
            }
            stmts = append(stmts, stmt)
            if _, err := p.EatToken(lexer.SEMICOLON); err != nil {
                return ModuleNode{}, err
            }
        }
    }
    
    return ModuleNode{"main", imports, stmts}, nil
}

func (p *pr) getImport() (ImportStatementNode, error) {
    token, err := p.EatToken(lexer.IMPORT)
    if err != nil {
        return ImportStatementNode{}, err
    }
    
    // Parsear caminho (identificador ou path com /)
    // Exemplo: math, utils/helpers, lib/string/helper
    path := ""
    pathToken, err := p.EatToken(lexer.ID)
    if err != nil {
        return ImportStatementNode{}, fmt.Errorf("expected import path at line %d, column %d",
            p.GetLookahead().GetLine(), p.GetLookahead().GetColumn())
    }
    path = string(pathToken.GetMatch())
    
    // Continuar parseando se houver / (path com múltiplas partes)
    for p.GetLookahead().GetTag().Id == lexer.DIV {
        p.EatToken(lexer.DIV) // consumir "/"
        nextPart, err := p.EatToken(lexer.ID)
        if err != nil {
            return ImportStatementNode{}, fmt.Errorf("expected path segment after '/' at line %d, column %d",
                p.GetLookahead().GetLine(), p.GetLookahead().GetColumn())
        }
        path = path + "/" + string(nextPart.GetMatch())
    }
    
    // Parsear alias opcional
    alias := ""
    if p.GetLookahead().GetTag().Id == lexer.AS {
        _, err = p.EatToken(lexer.AS) // consumir "as"
        if err != nil {
            return ImportStatementNode{}, err
        }
        
        aliasToken, err := p.EatToken(lexer.ID)
        if err != nil {
            return ImportStatementNode{}, fmt.Errorf("expected alias identifier after 'as' at line %d, column %d",
                p.GetLookahead().GetLine(), p.GetLookahead().GetColumn())
        }
        alias = string(aliasToken.GetMatch())
    } else {
        // Se não houver alias, usar último segmento do path
        parts := strings.Split(path, "/")
        alias = parts[len(parts)-1]
    }
    
    return ImportStatementNode{
        Path:  path,
        Alias: alias,
        Token: token,
    }, nil
}

// getStmt() também precisa reconhecer imports
func (p *pr) getStmt() (Node, error) {
    lookahead := p.GetLookahead()
    if lookahead.GetTag().Id == lexer.IMPORT {
        return p.getImport()
    }
    if lookahead.GetTag().Id == lexer.PRINT {
        return p.getPrint()
    }
    if lookahead.GetTag().Id == lexer.ASSERT {
        return p.getAssert()
    }
    expr, err := p.getExpr()
    if err != nil {
        return StatementNode{}, err
    }
    return StatementNode{expr}, nil
}
```

### Modificações no Emitter

```go
func (e *Emitter) Emit(ast AST) ([]Instruction, error) {
    insts := make([]Instruction, 0)
    
    // rootDir é o diretório onde o compilador foi executado
    resolver := NewImportResolver(rootDir)
    
    // Emitir statements na ordem (imports podem aparecer em qualquer lugar)
    for _, stmt := range ast.Module.Stmts {
        // Verificar se é um import
        if imp, ok := stmt.(ImportStatementNode); ok {
            // Resolver e emitir import quando encontrado
            importedAST, err := resolver.Resolve(imp.Path)
            if err != nil {
                return nil, err
            }
            
            // Emitir código do arquivo importado com namespace/alias
            importedInsts, err := e.EmitWithNamespace(importedAST, imp.Alias)
            if err != nil {
                return nil, err
            }
            
            // Adicionar instruções importadas na posição do import
            insts = append(insts, importedInsts...)
        } else {
            // Emitir statement normal
            stmtInsts, err := e.EmitStatement(stmt)
            if err != nil {
                return nil, err
            }
            insts = append(insts, stmtInsts...)
        }
    }
    
    return insts, nil
}

// EmitWithNamespace emite código com prefixo de namespace
func (e *Emitter) EmitWithNamespace(ast AST, namespace string) ([]Instruction, error) {
    // Identificadores do arquivo importado são prefixados com namespace
    // Exemplo: `sum` vira `math.sum` quando namespace é "math"
    // Isso é feito durante a emissão de OpLoad e OpCall
}
```

## Considerações

### 1. Escopo com Namespace
- Identificadores importados ficam em namespace separado (alias)
- Uso: `alias.function()` ou `alias.ident`
- Não há conflitos de nomes porque cada import tem seu próprio namespace
- Identificadores no módulo atual não precisam de prefixo

### 2. Ordem de Execução
- Imports são executados na ordem em que aparecem no código
- Cada import cria um namespace isolado quando encontrado
- Código importado fica disponível imediatamente após o import
- Exemplo: Se `import math;` aparece na linha 5, o código de `math.ar` é executado na linha 5, e `math.sum()` pode ser usado a partir da linha 6

### 3. Performance
- Cache de ASTs resolvidos
- Evitar re-parsear mesmo arquivo múltiplas vezes

### 4. Erros
- Arquivo não encontrado (a partir da raiz, com extensão .ar)
- Import circular
- Erro de parse no arquivo importado
- Alias duplicado (mesmo alias usado duas vezes)
- Path inválido (caracteres não permitidos no path)
- Path vazio (import sem path)

### 5. Testes
- Arquivo de teste importa código fonte
- Código fonte não pode importar arquivos de teste (`.test.ar`)

## Fases de Implementação

### Fase 1: Básico (MVP)
- [ ] Adicionar tokens `IMPORT` e `AS` no lexer
- [ ] Adicionar `ImportStatementNode` no parser
- [ ] Modificar `getStmt()` para reconhecer imports
- [ ] Modificar `ModuleNode` para incluir imports
- [ ] Implementar `ImportResolver` com resolução a partir da raiz
- [ ] Modificar emitter para processar imports na ordem de declaração
- [ ] Implementar resolução de identificadores com namespace (`alias.function`)
- [ ] Modificar evaluator para suportar namespaces e imports dinâmicos
- [ ] Testes básicos (imports no início, meio e fim do código)

### Fase 2: Melhorias
- [ ] Detecção de alias duplicados
- [ ] Melhor tratamento de erros
- [ ] Validação de caminhos (prevenir acesso fora da raiz)
- [ ] Validação de caracteres permitidos no path

### Fase 3: Avançado (futuro)
- [ ] Import seletivo (se necessário)
- [ ] Módulos como pacotes

## Exemplo Completo

```aurora
# utils/math.ar
ident add = {
  ident a = arguments(0);
  ident b = arguments(1);
  a + b;
};

ident subtract = {
  ident a = arguments(0);
  ident b = arguments(1);
  a - b;
};
```

```aurora
# utils/string.ar
ident concat = {
  ident a = arguments(0);
  ident b = arguments(1);
  # ... implementação
};
```

```aurora
# app.ar
import utils/math;              # alias: math (derivado do path)

ident result1 = math.add(5, 3);
ident result2 = math.subtract(10, 4);

import utils/string as str;     # alias: str (explícito) - pode aparecer no meio do código

ident result3 = str.concat("hello", "world");
```

```aurora
# app.test.ar
import app;                     # alias: app (derivado do path)

group "app tests" {
  case "should use imported functions" {
    import utils/math;          # import pode aparecer dentro de um group
    
    ident result = math.add(2, 3);
    assert result equals 5;
  };
  
  case "should use app functions" {
    ident result = app.result1;
    assert result equals 8;
  };
}
```

**Estrutura de diretórios** (executando de `/project`):
```
/project
  ├── app.ar
  ├── app.test.ar
  └── utils/
      ├── math.ar
      └── string.ar
```

**Comandos**:
```bash
cd /project
aurora run app.ar              # Importa utils/math (alias: math) e utils/string (alias: str)
aurora run app.test.ar         # Importa app (alias: app) e utils/math (alias: math)
```

**Exemplos de alias**:
- `import math;` → alias: `math`
- `import utils/helpers;` → alias: `helpers`
- `import a/b/c;` → alias: `c`
- `import lib/string as str;` → alias: `str` (explícito)

## Questões em Aberto

1. **Alias duplicados**: Permitir mesmo alias para imports diferentes?
   - **Recomendação**: Erro - cada alias deve ser único no mesmo arquivo
   - Exemplo: `import math; import utils/math;` → erro (ambos teriam alias `math`)

2. **Imports em arquivos de teste**: Podem importar outros arquivos de teste?
   - **Recomendação**: Sim, mas com cuidado

3. **Ordem de imports**: Importância da ordem?
   - **Recomendação**: Ordem de declaração importa (dependências)
   - Imports são executados quando encontrados no código
   - Código importado fica disponível imediatamente após o import
   - Exemplo: `import math; ident x = math.sum(1, 2);` - `math` está disponível na linha seguinte

4. **Imports dinâmicos**: Suporte futuro?
   - **Recomendação**: Não no MVP, considerar depois

5. **Resolução de identificadores com namespace**: Como implementar `math.sum()`?
   - **Recomendação**: 
     - No parser: `math.sum` vira um `CalleeLiteralNode` com namespace
     - No emitter: Criar instrução especial ou modificar `OpLoad`/`OpCall` para incluir namespace
     - No evaluator: Buscar identificador no namespace correto

6. **Identificadores no módulo atual**: Podem ter mesmo nome que alias?
   - **Recomendação**: Sim, porque são namespaces diferentes
   - Exemplo: `import math; ident math = 10;` - `math` (alias) vs `math` (ident)

7. **Derivação de alias**: Como funciona quando não há alias explícito?
   - **Recomendação**: 
     - Extrair último segmento do path usando `strings.Split(path, "/")`
     - Exemplo: `import utils/helpers;` → path: `utils/helpers`, alias: `helpers`
     - Exemplo: `import math;` → path: `math`, alias: `math`

8. **Parsing do path**: Como parsear paths com `/`?
   - **Recomendação**: 
     - Parsear ID seguido de `/` seguido de ID (repetir)
     - Usar token `DIV` (`/`) que já existe no lexer
     - Exemplo: `utils/helpers` → `ID` + `DIV` + `ID`

9. **Imports em qualquer lugar**: Como isso afeta a execução?
   - **Recomendação**: 
     - Imports são tratados como statements normais
     - Quando encontrado, o código importado é executado imediatamente
     - Namespace fica disponível após a execução do import
     - Exemplo: `ident x = 1; import math; ident y = math.sum(2, 3);` - `math` disponível após linha 2

