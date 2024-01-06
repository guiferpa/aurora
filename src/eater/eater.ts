import Lexer, { Token, TokenTag } from "@/lexer";

export class EaterError extends Error {
  constructor(message: string) {
    super(message);
  }
}

export default class Eater {
  private _lookahead: Token | null = null;

  constructor(public readonly context: string, private _lexer: Lexer) {
    this._lookahead = this._lexer.getNextToken();
  }

  public eat(tag: TokenTag): Token {
    if (this._lexer === null)
      throw new EaterError("None source code at Lexer buffer");

    const token = this.lookahead();

    if (token.tag === TokenTag.EOF)
      throw new EaterError(`Unexpected end of token, expected token: ${tag}`);

    if (tag !== token.tag)
      throw new EaterError(
        `Unexpected token at line: ${this._lookahead?.line}, column: ${this._lookahead?.column}, value: ${this._lookahead?.value}`
      );

    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  public lookahead(): Token {
    if (this._lookahead === null)
      throw new EaterError("No reference for lookahead");

    return this._lookahead;
  }
}
