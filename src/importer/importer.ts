import Lexer, { TokenTag } from "@/lexer";
import Eater from "@/eater";
import Parser, {
  AsStmtNode,
  FromStmtNode,
  ImportStmtNode,
  ProgramNode,
} from "@/parser";

export interface Reader {
  read(entry: string): Promise<Buffer>;
}

export type AliasClaim = Map<string, Map<string, string>>;

export interface MappingClaim {
  id: string;
  alias: string;
}

export interface ImportClaim {
  context: string;
  alias: Map<string, string>;
  mapping: MappingClaim[];
  program: ProgramNode;
}

export default class Importer {
  private readonly _imported: string[] = [];

  constructor(private _reader: Reader) {}

  public async imports(eater: Eater): Promise<ImportClaim[]> {
    if (this._imported.includes(eater.context)) return [];

    const mapping: ImportClaim["mapping"] = [];
    const alias: Map<string, string> = new Map([]);

    while (eater.lookahead().tag === TokenTag.FROM) {
      eater.eat(TokenTag.FROM);
      const id = eater.eat(TokenTag.STR).value;

      eater.eat(TokenTag.AS);
      const als = eater.eat(TokenTag.IDENT).value;

      alias.set(als, id);
      mapping.push({ id, alias: als });
    }

    const parser = new Parser(eater);

    const program = await parser.parse();

    const claim = {
      context: eater.context,
      mapping,
      alias,
      program: new ProgramNode([
        ...mapping.map(
          ({ id, alias }) =>
            new ImportStmtNode(new FromStmtNode(id), new AsStmtNode(alias))
        ),
        ...program.children,
      ]),
    };

    this._imported.push(eater.context);

    if (mapping.length === 0) return [claim];

    const imports: ImportClaim[] = [];

    const promises = mapping.flatMap(async (item): Promise<void> => {
      const buffer = await this._reader.read(item.id);
      const lexer = new Lexer(buffer);
      const eater = new Eater(item.id, lexer);
      const result = await this.imports(eater);
      imports.push(...result);
    });

    await Promise.all(promises);

    return [claim, ...imports];
  }

  public alias(claims: ImportClaim[]): AliasClaim {
    const alias: AliasClaim = new Map();

    claims.forEach((claim) => {
      alias.set(claim.context, claim.alias);
    });

    return alias;
  }
}
