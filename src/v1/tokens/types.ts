export enum TokenTag {
  EOT = "EOT", EOF = "EOF", WHITESPACE = "WHITESPACE",
  DEF = "DEF", IDENT = "IDENT", ASSIGN = "ASSIGN",
  SEMI = "SEMI", NUM = "NUM", ADD = "ADD", 
  SUB = "SUB", MULT = "MULT",
}

export const TokenProduct: [RegExp, TokenTag][] = [
  [new RegExp(/^var/), TokenTag.DEF],
  [new RegExp(/^[a-z]+/), TokenTag.IDENT],
  [new RegExp(/^\s+/), TokenTag.WHITESPACE],
  [new RegExp(/^=/), TokenTag.ASSIGN],
  [new RegExp(/^\d+/), TokenTag.NUM],
  [new RegExp(/^\+/), TokenTag.ADD],
  [new RegExp(/^\-/), TokenTag.SUB],
  [new RegExp(/^\*/), TokenTag.MULT],
  [new RegExp(/^;/), TokenTag.SEMI]
];

