export type TokenIdentifierType = number;

export interface Token {
  id: TokenIdentifierType;
  type: string;
  value: any;
}

export enum TokenIdentifier {
  EOT = -1,
  EOF = 0,
  ILLEGAL = 1,
  WHITESPACE = 2,
  DEF = 3,
  IDENT = 4,
  ASSIGN = 5,
  SEMI = 6,
  NUMBER = 7,
  ADD = 8,
  SUB = 9,
  BEGIN_BLOCK = 10,
  FINISH_BLOCK = 11,
  MULT = 12,
  NEGATIVE_NUMBER = 13
}

export const TokenRecordList: Record<TokenIdentifierType, string> = {
  [TokenIdentifier.EOT]: "EOT",
  [TokenIdentifier.EOF]: "EOF",
  [TokenIdentifier.ILLEGAL]: "ILLEGAL",
  [TokenIdentifier.WHITESPACE]: "WS",
  [TokenIdentifier.IDENT]: "IDENT",
  [TokenIdentifier.DEF]: "DEF",
  [TokenIdentifier.ASSIGN]: "ASSIGN",
  [TokenIdentifier.SEMI]: "SEMI",
  [TokenIdentifier.ADD]: "ADD",
  [TokenIdentifier.SUB]: "SUB",
  [TokenIdentifier.NUMBER]: "NUMBER",
  [TokenIdentifier.BEGIN_BLOCK]: "BEGIN_BLOCK",
  [TokenIdentifier.FINISH_BLOCK]: "FINISH_BLOCK",
  [TokenIdentifier.MULT]: "MULT",
  [TokenIdentifier.NEGATIVE_NUMBER]: "NEGATIVE_NUMBER"
};

export const TokenProductList: Record<TokenIdentifierType, RegExp> = {
  [TokenIdentifier.DEF]: new RegExp(/^var/),
  [TokenIdentifier.IDENT]: new RegExp(/^[a-z]+/),
  [TokenIdentifier.WHITESPACE]: new RegExp(/^\s+/),
  [TokenIdentifier.ASSIGN]: new RegExp(/^=/),
  [TokenIdentifier.SEMI]: new RegExp(/^;/),
  [TokenIdentifier.ADD]: new RegExp(/^\+/),
  [TokenIdentifier.SUB]: new RegExp(/^\-/),
  [TokenIdentifier.NUMBER]: new RegExp(/^\d+/),
  [TokenIdentifier.BEGIN_BLOCK]: new RegExp(/^{/),
  [TokenIdentifier.FINISH_BLOCK]: new RegExp(/^}/),
  [TokenIdentifier.MULT]: new RegExp(/^\*/),
  [TokenIdentifier.NEGATIVE_NUMBER]: new RegExp(/^\-[0-9]+/)
};

export const TokenSpecList: [RegExp, string, TokenIdentifierType][] = [
  [TokenProductList[TokenIdentifier.DEF], TokenRecordList[TokenIdentifier.DEF], TokenIdentifier.DEF],
  [TokenProductList[TokenIdentifier.IDENT], TokenRecordList[TokenIdentifier.IDENT], TokenIdentifier.IDENT],
  [TokenProductList[TokenIdentifier.WHITESPACE], TokenRecordList[TokenIdentifier.WHITESPACE], TokenIdentifier.WHITESPACE],
  [TokenProductList[TokenIdentifier.SEMI], TokenRecordList[TokenIdentifier.SEMI], TokenIdentifier.SEMI],
  [TokenProductList[TokenIdentifier.ASSIGN], TokenRecordList[TokenIdentifier.ASSIGN], TokenIdentifier.ASSIGN],
  [TokenProductList[TokenIdentifier.NUMBER], TokenRecordList[TokenIdentifier.NUMBER], TokenIdentifier.NUMBER],
  [TokenProductList[TokenIdentifier.ADD], TokenRecordList[TokenIdentifier.ADD], TokenIdentifier.ADD],
  [TokenProductList[TokenIdentifier.SUB], TokenRecordList[TokenIdentifier.SUB], TokenIdentifier.SUB],
  [TokenProductList[TokenIdentifier.BEGIN_BLOCK], TokenRecordList[TokenIdentifier.BEGIN_BLOCK], TokenIdentifier.BEGIN_BLOCK],
  [TokenProductList[TokenIdentifier.FINISH_BLOCK], TokenRecordList[TokenIdentifier.FINISH_BLOCK], TokenIdentifier.FINISH_BLOCK],
];

