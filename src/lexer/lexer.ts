import { Token } from "./tokens/token";
import { Terminals } from "./tokens/terminal";
import { TokenTag } from "./tokens/tag";

export default class Lexer {
  private _cursor = 0;
  private _line = 1;
  private _column = 1;
  private _buffer: Buffer;
  private _isComment: boolean = false;

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
    if (!this.hasMoreTokens())
      return new Token(this._line, this._column, TokenTag.EOF, "EOF");

    const str = this._buffer.toString("ascii", this._cursor);

    if (this._isSpace(str)) {
      return this.getNextToken();
    }

    for (const [regex, tag] of Terminals) {
      const value = this._match(regex, str);

      if (value === null) continue;

      this._cursor += value.length;
      this._column += value.length;

      if (tag === TokenTag.COMMENT) {
        this._isComment = true;
      }

      if (this._isComment) {
        return this.getNextToken();
      }

      let token = new Token(this._line, this._column, tag, value);

      if (tag === TokenTag.ASSIGN) {
        const [d] = value.split("=");
        token = new Token(
          this._line,
          this._column,
          tag,
          d.replace(/^var/, "").trim()
        );
      }

      if (tag === TokenTag.DECL_FN) {
        return new Token(this._line, this._column, tag, value.split(" ")[1]);
      }

      if (tag === TokenTag.STR) {
        token = new Token(
          this._line,
          this._column,
          tag,
          value.replace(/\"/g, "").trim()
        );
      }

      if (tag === TokenTag.NUM) {
        token = new Token(
          this._line,
          this._column,
          tag,
          value.replace(/_/g, "")
        );
      }

      return token;
    }

    throw new SyntaxError(`Token doesn't exist: ${str}`);
  }

  private _isSpace(str: string): boolean {
    const matched = /^\s+/.exec(str);
    if (matched === null) return false;
    this._cursor += matched[0].length;
    const bs = Buffer.from(matched[0], "utf8");
    for (const b of bs) {
      if (b === 10) {
        this._line++;
        this._column = 1;
        this._isComment = false;
      } else {
        this._column += 1;
      }
    }
    return true;
  }

  private _match(product: RegExp, str: string) {
    const matched = product.exec(str);
    if (matched === null) return null;
    return matched[0];
  }
}
