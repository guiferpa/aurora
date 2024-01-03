import Eater from "@/eater";
import { TokenTag } from "@/lexer";
import Lexer from "@/lexer/lexer";
import {
  AsStmtNode,
  FromStmtNode,
  IdentNode,
  ProgramNode,
  StringNode,
} from "@/parser";
import Parser from "@/parser/parser";
import SymTable from "@/symtable/symtable";

export interface Reader {
  read(entry: string): Promise<Buffer>;
}

export interface MappingClaim {
  id: string;
  alias: string;
}

export interface ImportClaim {
  mapping: MappingClaim;
  program: ProgramNode;
}

export default class Importer {
  constructor(private _eater: Eater, private _reader: Reader) {}

  private _str(): StringNode {
    const str = this._eater.eat(TokenTag.STR);
    return new StringNode(str.value);
  }

  private _from(): FromStmtNode {
    this._eater.eat(TokenTag.FROM);
    const str = this._str();
    return new FromStmtNode(str.value);
  }

  private _ident(): IdentNode {
    const ident = this._eater.eat(TokenTag.IDENT);
    return new IdentNode(ident.value);
  }

  private _as(): AsStmtNode {
    this._eater.eat(TokenTag.AS);
    const ident = this._ident();
    return new AsStmtNode(ident.name);
  }

  private _map(): MappingClaim {
    const from = this._from();

    let alias = new AsStmtNode("");
    if (this._eater.lookahead().tag === TokenTag.AS) {
      alias = this._as();
    }

    return {
      id: from.value,
      alias: alias.value,
    };
  }

  public async mapping(): Promise<MappingClaim[]> {
    const result: MappingClaim[] = [];

    while (this._eater.lookahead().tag === TokenTag.FROM) {
      const n = this._map();

      const buffer = await this._reader.read(n.id);
      const lexer = new Lexer(buffer);
      const eatertemp = this._eater;
      this._eater = new Eater(lexer);

      const ns = await this.mapping();

      result.push(...[n, ...ns]);

      this._eater = eatertemp;
    }

    return result;
  }

  public async imports(): Promise<
    [Map<string, ImportClaim>, Map<string, string>]
  > {
    const mapping = await this.mapping();
    const translate = new Map<string, string>();

    const imports = await Promise.all(
      mapping.map(async (item): Promise<[string, ImportClaim]> => {
        const symtable = new SymTable("global");
        const buffer = await this._reader.read(item.id);
        const lexer = new Lexer(buffer);
        const parser = new Parser(new Eater(lexer), symtable);
        if (item.alias !== "") {
          translate.set(item.alias, item.id);
        }
        return [item.id, { mapping: item, program: await parser.parse() }];
      })
    );

    return [new Map(imports), translate];
  }
}
