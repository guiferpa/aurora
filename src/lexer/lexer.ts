import {
  Token,
  TokenIdentifier,
  TokenString,
  TokenNumber,
  TokenLogical,
  TokenProduct,
  TokenTag,
  TokenArity,
  TokenDef,
  TokenDefFunction,
  TokenTyping,
} from "@/tokens";

export default class Lexer {
  public _cursor = 0;
  public _line = 1;
  public _column = 1;
  public _buffer: Buffer;

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
      return new Token(TokenTag.EOT, this._cursor, this._line, this._column);

    const str = this._buffer.toString("ascii", this._cursor);

    for (const [regex, tag] of TokenProduct) {
      const value = this._match(regex, str);

      if (value === null) continue;

      const prevCursor = this._cursor

      this._cursor += value.length;

      if (tag === TokenTag.WHITESPACE) {

        // Identifying break line
        if (value.charCodeAt(0) === 10) {
          this._line += value.length; // Amount of break line
          this._column = 1; // Reset colunm counter
        }

        return this.getNextToken();
      }

      this._column += this._cursor - prevCursor;

      if (tag === TokenTag.NUM) {
        const num = Number.parseInt(value.replace(/_/g, ""));
        return new TokenNumber(num, this._cursor, this._line, this._column);
      }

      if (tag === TokenTag.LOGICAL) 
        return new TokenLogical(value === "true", this._cursor, this._line, this._column);

      if (tag === TokenTag.STR) 
        return new TokenString(value.replace(/"/g, ""), this._cursor, this._line, this._column);

      if (tag === TokenTag.TYPING)
        return new TokenTyping(value.replace(/:/, "").trim(), this._cursor, this._line, this._column);

      if (tag === TokenTag.IDENT) 
        return new TokenIdentifier(value, this._cursor, this._line, this._column);

      if (tag === TokenTag.DEF) 
        return new TokenDef(value.split(" ")[1], this._cursor, this._line, this._column);

      if (tag === TokenTag.DEF_FUNC) {
        const [name, params] = value
          .replace(/^(func)\s/, "")
          .replace(/{/, "")
          .trim()
          .replace(/\)/, "")
          .replace(/\(/, "-")
          .split("-");

        const arity = new TokenArity(params.split(","), this._cursor, this._line, this._column);
        return new TokenDefFunction(name, arity, this._cursor, this._line, this._column);
      }

      return new Token(tag, this._cursor, this._line, this._column);
    }

    throw new SyntaxError(`Token doesn't exist: ${str}`);
  }

  private _match(product: RegExp, str: string) {
    const matched = product.exec(str);
    if (matched === null) return null;
    const value = matched[0];
    return value;
  }
}
