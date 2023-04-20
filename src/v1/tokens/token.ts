import {TokenTag} from "./types";

interface IToken {
  toString(): string
}

export class Token implements IToken {
  public readonly tag: TokenTag;
  public readonly line: number;
  public readonly column: number;

  constructor(tag: TokenTag, line: number, column: number) {
    this.tag = tag;
    this.line = line;
    this.column = column;
  }

  public toString(): string {
    return `<${this.tag}>`;
  }
}

export class TokenIdentifier extends Token {
  public readonly name: string;

  constructor(name: string, line: number, column: number) {
    super(TokenTag.IDENT, line, column);
    
    this.name = name;
  }

  public toString(): string {
    return `<${this.tag}, ${this.name}>`
  }
}

export class TokenNumber extends Token {
  public readonly value: number;

  constructor(value: number, line: number, column: number) {
    super(TokenTag.NUM, line, column);
    
    this.value = value;
  }

  public toString(): string {
    return `<${this.tag}, ${this.value}>`
  }
}

