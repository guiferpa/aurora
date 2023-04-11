import {Lexer} from "./lexer";
import {Token, TokenIdentifier, TokenRecordList} from "./tokens";

export class Parser {
	private readonly _lexer: Lexer;
	private _lookahead: Token | null = null;

	constructor(lexer: Lexer) {
		this._lexer = lexer;
	}

	// factor =>
	//	| NUMBER
	//	| IDENT
	public async _factor() {
		const { id: tokenId, type: tokenType } = this._lookahead as Token;

		if (tokenId === TokenIdentifier.IDENT) {
			const token = await this._eat(tokenType);
			return ["factor", [token.type, token.value]];
		}

		if (tokenId === TokenIdentifier.NUMBER) {
			const token = await this._eat(tokenType);
			return ["factor", [token.type, Number.parseInt(token.value)]];
		}

		throw new SyntaxError(`Factor: unexpected factor production`);
	}

	// expression | expr =>
	//  | factor
	//  | factor ADD expr
	//  | factor SUB expr
	public async _expr() {
		const left = await this._factor();

		if (![
			TokenRecordList[TokenIdentifier.ADD],
			TokenRecordList[TokenIdentifier.SUB]
		].includes(this._lookahead?.type as string)) {
			return left;
		}

		const operator = await this._eat(this._lookahead?.type as string)
		const right: any = await this._expr();

		return ["expr", ["operator", [operator.type, operator.value]], left, right];
	}

	// statement || stmt =>
	//	| expr ;
	//	| DEF IDENT ASSIGN expr ;
	public async _stmt() {
		if (this._lookahead?.id === TokenIdentifier.DEF) {
			await this._eat(TokenRecordList[TokenIdentifier.DEF]);
			const id = await this._eat(TokenRecordList[TokenIdentifier.IDENT]);
			await this._eat(TokenRecordList[TokenIdentifier.ASSIGN]);
			const expr = await this._expr();
			await this._eat(TokenRecordList[TokenIdentifier.SEMI]);

			return ["stmt", ["id", [id.type, id.value], expr]];
		}

		const expr = await this._expr();
		await this._eat(TokenRecordList[TokenIdentifier.SEMI]);

		return ["stmt", expr];
	}

	/***
	 * program =>
	 *	| stmt*
	 */
	public async _program() {
		const stmts = [];

		while (this._lexer.hasMoreTokens()) {
			const stmt = await this._stmt();
			stmts.push(stmt);
		}

		return ["program", stmts];
	}

	private async _eat(tokenType: string): Promise<Token> {
		const token = this._lookahead;
		console.log("T", token);

		if (token === null) {
			throw new SyntaxError(`Unexpected end of input, expected: ${tokenType}`);
		}

		if (token.type !== tokenType) {
			throw new SyntaxError(`Unexpected token: ${token.value}, expected: ${tokenType}`)
		}

		this._lookahead = await this._lexer.getAsyncNextToken();

		return token as Token;
	}

	public async parse(): Promise<any> {
		this._lookahead = await this._lexer.getAsyncNextToken();

		return await this._program();
	}
}

