import { TokenTag } from "./tag";

export class Token {
  constructor(public readonly tag: TokenTag, public readonly value: string) {}
}
