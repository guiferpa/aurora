import { TokenTag } from "./tag";

export const Terminals: [RegExp, TokenTag][] = [
  [new RegExp(/^[0-9_]+/), TokenTag.NUM],
  [new RegExp(/^!/), TokenTag.NEG],
  [new RegExp(/^true|false/), TokenTag.LOG],
  [new RegExp(/^greater/), TokenTag.REL_GT],
  [new RegExp(/^less/), TokenTag.REL_LT],
  [new RegExp(/^equal/), TokenTag.REL_EQ],
  [new RegExp(/^different/), TokenTag.REL_DIF],
  [new RegExp(/^and/), TokenTag.LOG_AND],
  [new RegExp(/^or/), TokenTag.LOG_OR],
  [new RegExp(/^var [a-zA-Z_]+ =/), TokenTag.ASSIGN],
  [new RegExp(/^[a-zA-Z_]+/), TokenTag.IDENT],
  [new RegExp(/^\+/), TokenTag.OP_ADD],
  [new RegExp(/^\-/), TokenTag.OP_SUB],
  [new RegExp(/^\*/), TokenTag.OP_MUL],
  [new RegExp(/^\//), TokenTag.OP_DIV],
  [new RegExp(/^\(/), TokenTag.PAREN_O],
  [new RegExp(/^\)/), TokenTag.PAREN_C],
  [new RegExp(/^\{/), TokenTag.BRACK_O],
  [new RegExp(/^\}/), TokenTag.BRACK_C],
];
