import {Token, TokenIdentifier, TokenRecordList, TokenSpecList} from "./tokens";

export default class Lexer {
  public _cursor = 0;
  public readonly _buffer: Buffer;

  constructor(buffer: any) {
    this._buffer = buffer;
  }

  public tokenify(): Token[] {
    const tokens: Token[] = [];

    while (this.hasMoreTokens()) {
      const token = this.getNextToken();
      tokens.push(token);
    }

    this._cursor = 0;

    return tokens;
  }

  public hasMoreTokens(): boolean {
    return this._cursor < this._buffer.length;
  }

  public getNextToken(): Token {
    if (!this.hasMoreTokens()) return {
      id: TokenIdentifier.EOT,
      type: TokenRecordList[TokenIdentifier.EOT],
      value: ""
    }

    const str = this._buffer.toString('utf-8', this._cursor);

    for (const [regex, tokenType, tokenId] of TokenSpecList) {
      const tokenValue = this._match(regex, str);

      if (tokenValue === null) continue;

      if (tokenId === TokenIdentifier.WHITESPACE)
        return this.getNextToken();

      return {id: tokenId, type: tokenType, value: tokenValue}
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

