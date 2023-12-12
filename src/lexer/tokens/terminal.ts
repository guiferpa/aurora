import { TokenTag } from "./tag";

export const Terminals: [RegExp, TokenTag][] = [
  [new RegExp(/^[0-9_]+/), TokenTag.NUM],
  [new RegExp(/^var [a-zA-Z_] =/), TokenTag.DECL],
  [new RegExp(/^[a-zA-Z_]+/), TokenTag.IDENT],
  [new RegExp(/^\+/), TokenTag.OP_ADD],
  [new RegExp(/^\-/), TokenTag.OP_SUB],
  [new RegExp(/^\*/), TokenTag.OP_MUL],
  [new RegExp(/^\//), TokenTag.OP_DIV],
  [new RegExp(/^\(/), TokenTag.PAREN_O],
  [new RegExp(/^\)/), TokenTag.PAREN_C],
];
