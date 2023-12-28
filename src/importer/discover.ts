import { Lexer } from "@/lexer";
import { TokenTag } from "@/lexer/tokens/tag";
import { Token } from "@/lexer/tokens/token";

import Dependency from "./dependency";

export default class Discover {
  private _lookahead: Token | null = null;

  constructor(private readonly _lexer: Lexer) {}

  private _discovery(): Dependency[] {
    const deps = new Map<string, Dependency>();

    this._lookahead = this._lexer.getNextToken();

    while (this._lookahead?.tag === TokenTag.FROM) {
      this._lookahead = this._lexer.getNextToken();
      if (this._lookahead.tag !== TokenTag.STR) {
        throw new Error(
          `Missing dependency identifier at line ${this._lookahead.line}`
        );
      }

      const id = this._lookahead.value;

      this._lookahead = this._lexer.getNextToken();

      if (this._lookahead.tag !== TokenTag.AS) {
        throw new SyntaxError(
          `Unexpected token ${this._lookahead.tag} at line ${this._lookahead.line}`
        );
      }

      this._lookahead = this._lexer.getNextToken();

      if (this._lookahead.tag !== TokenTag.IDENT) {
        throw new Error(`Invalid alias at line ${this._lookahead.line}`);
      }

      const alias = this._lookahead.value;

      deps.set(id, new Dependency(id, alias));

      this._lookahead = this._lexer.getNextToken();
    }

    return Array.from(deps.values());
  }

  public run(): Dependency[] {
    return this._discovery();
  }
}
