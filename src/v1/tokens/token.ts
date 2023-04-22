import {TokenTag} from "./types";

interface IToken {
  toString(): string
}

export class Token implements IToken {
  public readonly tag: TokenTag;

  constructor(tag: TokenTag) {
    this.tag = tag;
  }

  public toString(): string {
    return `<${this.tag}>`;
  }
}

export class TokenIdentifier extends Token {
  public readonly name: string;

  constructor(name: string) {
    super(TokenTag.IDENT);
    
    this.name = name;
  }

  public toString(): string {
    return `<${this.tag}, ${this.name}>`
  }
}

export class TokenNumber extends Token {
  public readonly value: number;

  constructor(value: number) {
    super(TokenTag.NUM);
    
    this.value = value;
  }

  public toString(): string {
    return `<${this.tag}, ${this.value}>`
  }
}

export class TokenLogical extends Token {
  public readonly value: boolean;

  constructor(value: boolean) {
    super(TokenTag.LOGICAL);
    
    this.value = value;
  }

  public toString(): string {
    return `<${this.tag}, ${this.value}>`
  }
}

