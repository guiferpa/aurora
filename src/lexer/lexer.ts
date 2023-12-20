import { Token } from "./tokens/token";
import { Terminals } from "./tokens/terminal";
import { TokenTag } from "./tokens/tag";

export default class Lexer {
  private _cursor = 0;
  private _buffer: Buffer;

  constructor(buffer: Buffer = Buffer.from("")) {
    this._buffer = buffer;
  }

  public write(buffer: Buffer) {
    this._buffer = Buffer.concat([this._buffer, buffer]);
  }

  public hasMoreTokens(): boolean {
    return this._cursor < this._buffer.length;
  }

  public getNextToken(): Token {
    if (!this.hasMoreTokens()) return new Token(TokenTag.EOF, "EOF");

    const str = this._buffer.toString("ascii", this._cursor);

    if (this._isSpace(str)) {
      return this.getNextToken();
    }

    for (const [regex, tag] of Terminals) {
      const value = this._match(regex, str);

      if (value === null) continue;

      this._cursor += value.length;

      if (tag === TokenTag.ASSIGN) {
        const [d] = value.split("=");
        return new Token(tag, d.replace(/^var/, "").trim());
      }

      if (tag === TokenTag.DECL_FN) {
        return new Token(tag, value.split(" ")[1]);
      }

      if (tag === TokenTag.STR) {
        return new Token(tag, value.replace(/\"/g, "").trim());
      }

      if (tag === TokenTag.NUM) {
        return new Token(tag, value.replace(/_/g, ""));
      }

      return new Token(tag, value);
    }

    throw new SyntaxError(`Token doesn't exist: ${str}`);
  }

  private _isSpace(str: string): boolean {
    const matched = /^\s+/.exec(str);
    if (matched === null) return false;
    this._cursor += matched[0].length;
    return true;
  }

  private _match(product: RegExp, str: string) {
    const matched = product.exec(str);
    if (matched === null) return null;
    return matched[0];
  }
}
