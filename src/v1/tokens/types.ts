export enum TokenTag {
  EOT = "EOT", EOF = "EOF", WHITESPACE = "WHITESPACE",
  DEF = "DEF", IDENT = "IDENT", ASSIGN = "ASSIGN",
  SEMI = "SEMI", NUM = "NUM", 
  PAREN_BEGIN = "PAREN_BEGIN", PAREN_END = "PAREN_END",
  BLOCK_BEGIN = "BLOCK_BEGIN", BLOCK_END = "BLOCK_END",
  ADD = "ADD", SUB = "SUB", MULT = "MULT", LOGICAL = "LOGICAL",
  EQUAL = "EQUAL"
}

export const TokenProduct: [RegExp, TokenTag][] = [
  [new RegExp(/^\d+/), TokenTag.NUM],
  [new RegExp(/^(true|false)/), TokenTag.LOGICAL],
  [new RegExp(/^var/), TokenTag.DEF],
  [new RegExp(/^equal/), TokenTag.EQUAL],
  [new RegExp(/^[a-z_]+/), TokenTag.IDENT],
  [new RegExp(/^\s+/), TokenTag.WHITESPACE],
  [new RegExp(/^=/), TokenTag.ASSIGN],
  [new RegExp(/^\(/), TokenTag.PAREN_BEGIN],
  [new RegExp(/^\)/), TokenTag.PAREN_END],
  [new RegExp(/^{/), TokenTag.BLOCK_BEGIN],
  [new RegExp(/^}/), TokenTag.BLOCK_END],
  [new RegExp(/^\+/), TokenTag.ADD],
  [new RegExp(/^\-/), TokenTag.SUB],
  [new RegExp(/^\*/), TokenTag.MULT],
  [new RegExp(/^;/), TokenTag.SEMI],
];

