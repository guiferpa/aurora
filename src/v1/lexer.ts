import {Token, TokenIdentifier, TokenNumber, TokenProduct, TokenTag} from "./tokens";

export default class Lexer {
  public _cursor = 0;
  public readonly _buffer: Buffer;
  private line: number = 1;
  private column: number = 1;

  constructor(buffer: Buffer) {
    this._buffer = buffer;
  }

  public hasMoreTokens(): boolean {
    return this._cursor < this._buffer.length;
  }

  public getNextToken(): Token {
    if (!this.hasMoreTokens()) 
      return new Token(TokenTag.EOT, this.line, this.column)

    const str = this._buffer.toString('utf-8', this._cursor);

    for (const [regex, tag] of TokenProduct) {
      const value = this._match(regex, str);

      if (value === null) 
        continue;

      if (tag === TokenTag.WHITESPACE)
        return this.getNextToken();

      if (tag === TokenTag.NUM)
        return new TokenNumber(
          Number.parseInt(value),
          this.line,
          this.column
        );

      if (tag === TokenTag.IDENT)
        return new TokenIdentifier(value, this.line, this.column);

      return new Token(tag, this.line, this.column);
    }

    throw new SyntaxError(`Unexpected token: ${str}`);
  }

  private _match(product: RegExp, str: string) {
    const matched = product.exec(str);
    if (matched === null) return null

    const value = matched[0];
    this._cursor += value.length;
    return value;
  }
}

