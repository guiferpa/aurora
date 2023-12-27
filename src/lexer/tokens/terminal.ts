import { TokenTag } from "./tag";

export const Terminals: [RegExp, TokenTag][] = [
  [new RegExp(/^;/), TokenTag.COMMENT],
  [new RegExp(/^[0-9_]+/), TokenTag.NUM],
  [new RegExp(/^\"(.*?)"/), TokenTag.STR],
  [new RegExp(/^arg/), TokenTag.CALL_ARG],
  [new RegExp(/^concat/), TokenTag.CALL_CONCAT],
  [new RegExp(/^print/), TokenTag.CALL_PRINT],
  [new RegExp(/^map/), TokenTag.CALL_MAP],
  [new RegExp(/^filter/), TokenTag.CALL_FILTER],
  [new RegExp(/^if/), TokenTag.IF],
  [new RegExp(/^not/), TokenTag.NEG],
  [new RegExp(/^true/), TokenTag.LOG],
  [new RegExp(/^false/), TokenTag.LOG],
  [new RegExp(/^greater/), TokenTag.REL_GT],
  [new RegExp(/^less/), TokenTag.REL_LT],
  [new RegExp(/^equal/), TokenTag.REL_EQ],
  [new RegExp(/^different/), TokenTag.REL_DIF],
  [new RegExp(/^and/), TokenTag.LOG_AND],
  [new RegExp(/^or/), TokenTag.LOG_OR],
  [new RegExp(/^desc/), TokenTag.DESC_FUNC],
  [new RegExp(/^return void/), TokenTag.RETURN_VOID],
  [new RegExp(/^return/), TokenTag.RETURN],
  [new RegExp(/^var [a-zA-Z_]+(\s?)=/), TokenTag.ASSIGN],
  [new RegExp(/^func [a-zA-Z_><\-!?]+/), TokenTag.DECL_FN],
  [new RegExp(/^\+/), TokenTag.OP_ADD],
  [new RegExp(/^\-/), TokenTag.OP_SUB],
  [new RegExp(/^\*/), TokenTag.OP_MUL],
  [new RegExp(/^\//), TokenTag.OP_DIV],
  [new RegExp(/^\(/), TokenTag.PAREN_O],
  [new RegExp(/^\)/), TokenTag.PAREN_C],
  [new RegExp(/^\{/), TokenTag.BRACK_O],
  [new RegExp(/^\}/), TokenTag.BRACK_C],
  [new RegExp(/^\[/), TokenTag.S_BRACK_O],
  [new RegExp(/^\]/), TokenTag.S_BRACK_C],
  [new RegExp(/^\,/), TokenTag.COMMA],
  [new RegExp(/^[a-zA-Z_><\-!?]+/), TokenTag.IDENT],
];
