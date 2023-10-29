import { TokenTag } from "./types";

interface IToken {
  toString(): string;
}

export class Token implements IToken {
  public readonly tag: TokenTag;
  public readonly cursor: number;
  public readonly line: number;
  public readonly column: number;

  constructor(tag: TokenTag, cursor: number, line: number, column: number) {
    this.tag = tag;
    this.cursor = cursor;
    this.line = line;
    this.column = column;
  }

  public toString(): string {
    return `<${this.tag}>`;
  }
}

export class TokenTyping extends Token {
  public readonly value: string;

  constructor(value: string, cursor: number, line: number, column: number) {
    super(TokenTag.TYPING, cursor, line, column);

    this.value = value;
  }

  public toString(): string {
    return `<${this.tag}, ${this.value}>`;
  }
}

export class TokenReturn extends Token {
  constructor(cursor: number, line: number, column: number) {
    super(TokenTag.RETURN, cursor, line, column);
  }

  public toString(): string {
    return `<${this.tag}>`;
  }
}

export class TokenArity extends Token {
  public readonly params: string[];

  constructor(params: string[], cursor: number, line: number, column: number) {
    super(TokenTag.ARITY, cursor, line, column);

    this.params = params;
  }

  public toString(): string {
    return `<${this.tag}, ${this.params.join(", ")}>`;
  }
}

export class TokenDefFunction extends Token {
  public readonly name: string;
  public readonly arity: TokenArity;

  constructor(name: string, arity: TokenArity, cursor: number, line: number, column: number) {
    super(TokenTag.DEF_FUNC, cursor, line, column);

    this.name = name;
    this.arity = arity;
  }

  public toString(): string {
    return `<${this.tag}, ${this.arity}>`;
  }
}

export class TokenDef extends Token {
  public readonly name: string;

  constructor(name: string, cursor: number, line: number, column: number) {
    super(TokenTag.DEF, cursor, line, column);

    this.name = name;
  }

  public toString(): string {
    return `<${this.tag}, ${this.name}>`;
  }
}

export class TokenIdentifier extends Token {
  public readonly name: string;

  constructor(name: string, cursor: number, line: number, column: number) {
    super(TokenTag.IDENT, cursor, line, column);

    this.name = name;
  }

  public toString(): string {
    return `<${this.tag}, ${this.name}>`;
  }
}

export class TokenNumber extends Token {
  public readonly value: number;

  constructor(value: number, cursor: number, line: number, column: number) {
    super(TokenTag.NUM, cursor, line, column);

    this.value = value;
  }

  public toString(): string {
    return `<${this.tag}, ${this.value}>`;
  }
}

export class TokenLogical extends Token {
  public readonly value: boolean;

  constructor(value: boolean, cursor: number, line: number, column: number) {
    super(TokenTag.LOGICAL, cursor, line, column);

    this.value = value;
  }

  public toString(): string {
    return `<${this.tag}, ${this.value}>`;
  }
}

export class TokenString extends Token {
  public readonly value: string;

  constructor(value: string, cursor: number, line: number, column: number) {
    super(TokenTag.STR, cursor, line, column);

    this.value = value;
  }

  public toString(): string {
    return `<${this.tag}, ${this.value}>`;
  }
}

export function isRelativeOperatorToken(token: Token) {
  return [TokenTag.EQUAL, TokenTag.GREATER_THAN, TokenTag.LESS_THAN].includes(
    token.tag
  );
}

export function isLogicalOperatorToken(token: Token) {
  return [TokenTag.OR, TokenTag.AND].includes(token.tag);
}

export function isAdditiveOperatorToken(token: Token) {
  return [TokenTag.ADD, TokenTag.SUB].includes(token.tag);
}

export function isMultiplicativeOperatorToken(token: Token) {
  return [TokenTag.MULT].includes(token.tag);
}
