import { TokenTag } from "./tag";

export class Token {
  constructor(
    public readonly line: number,
    public readonly column: number,
    public readonly tag: TokenTag,
    public readonly value: string
  ) {}
}
