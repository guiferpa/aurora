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
    if (!this.hasMoreTokens()) return new Token(TokenTag.EOT);

    const str = this._buffer.toString("utf-8", this._cursor);

    for (const [regex, tag] of TokenProduct) {
      const value = this._match(regex, str);

      if (value === null) continue;

      if (tag === TokenTag.WHITESPACE) return this.getNextToken();

      if (tag === TokenTag.NUM) {
        const num = Number.parseInt(value);
        return new TokenNumber(num);
      }

      if (tag === TokenTag.LOGICAL) {
        const logical = value === "true";
        return new TokenLogical(logical);
      }

      if (tag === TokenTag.STR) return new TokenString(value.replace(/"/g, ""));

      // TODO: def func typing
      //
      if (tag === TokenTag.TYPING) {
        console.log(tag, value);
        return new TokenTyping(value.replace(/:/, "").trim());
      }

      if (tag === TokenTag.IDENT) return new TokenIdentifier(value);

      if (tag === TokenTag.DEF) return new TokenDef(value.split(" ")[1]);

      if (tag === TokenTag.DEF_FUNC) {
        const [name, params] = value
          .replace(/^(func)\s/, "")
          .replace(/\)/, "")
          .replace(/\(/, "-")
          .split("-");

        const arity = new TokenArity(params.split(","));
        return new TokenDefFunction(name, arity);
      }

      return new Token(tag);
    }

    throw new SyntaxError(`Unexpected token: ${str}`);
  }

  private _match(product: RegExp, str: string) {
    const matched = product.exec(str);
    if (matched === null) return null;

    const value = matched[0];
    this._cursor += value.length;
    return value;
  }
}
