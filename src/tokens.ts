export type TokenIdentifierType = number;

export interface Token {
	id: TokenIdentifierType;
	type: string;
	value: string;
}

export enum TokenIdentifier {
	// Effects token
	EOF = 1,
	ILLEGAL = 2,
	WHITESPACE = 3,

	// Identifier token
	IDENT = 4,
	ASSIGN = 5,
	SEMI = 6,
	NUMBER = 7,

	// Operations token
	ADD = 8,
	SUB = 9,
}

export const TokenRecordList: Record<TokenIdentifierType, string> = {
	[TokenIdentifier.EOF]: "EOF",
	[TokenIdentifier.ILLEGAL]: "ILLEGAL",
	[TokenIdentifier.WHITESPACE]: "WS",
	[TokenIdentifier.IDENT]: "IDENT",
	[TokenIdentifier.ASSIGN]: "ASSIGN",
	[TokenIdentifier.SEMI]: "SEMI",
	[TokenIdentifier.ADD]: "ADD",
	[TokenIdentifier.SUB]: "SUB",
	[TokenIdentifier.NUMBER]: "NUMBER"
};

export const TokenProductList: Record<TokenIdentifierType, RegExp> = {
	[TokenIdentifier.IDENT]: new RegExp(/^[a-z]+/),
	[TokenIdentifier.WHITESPACE]: new RegExp(/^\s+/),
	[TokenIdentifier.ASSIGN]: new RegExp(/=/),
	[TokenIdentifier.SEMI]: new RegExp(/^;/),
	[TokenIdentifier.ADD]: new RegExp(/^\+/),
	[TokenIdentifier.SUB]: new RegExp(/^\-/),
	[TokenIdentifier.NUMBER]: new RegExp(/^\d+/)
};

export const TokenSpecList = [
	[TokenProductList[TokenIdentifier.IDENT], TokenRecordList[TokenIdentifier.IDENT], TokenIdentifier.IDENT],
	[TokenProductList[TokenIdentifier.WHITESPACE], TokenRecordList[TokenIdentifier.WHITESPACE], TokenIdentifier.WHITESPACE],
	[TokenProductList[TokenIdentifier.SEMI], TokenRecordList[TokenIdentifier.SEMI], TokenIdentifier.SEMI],
	[TokenProductList[TokenIdentifier.ASSIGN], TokenRecordList[TokenIdentifier.ASSIGN], TokenIdentifier.ASSIGN],
	[TokenProductList[TokenIdentifier.NUMBER], TokenRecordList[TokenIdentifier.NUMBER], TokenIdentifier.NUMBER],
	[TokenProductList[TokenIdentifier.ADD], TokenRecordList[TokenIdentifier.ADD], TokenIdentifier.ADD],
	[TokenProductList[TokenIdentifier.SUB], TokenRecordList[TokenIdentifier.SUB], TokenIdentifier.SUB]
];

