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
	public async _factor() {
		if ([
			TokenRecordList[TokenIdentifier.NUMBER],
		].includes(this._lookahead?.type as string)) {
			const factor = await this._eat(this._lookahead?.type as string);

			return { 
				type: 'factor',
				node: factor
			}
		}

		throw new SyntaxError(`Factor: unexpected factor production`);
	}

	// expression | expr =>
	//  | factor
	//  | factor ADD expr
	//  | factor SUB expr
	public async _expr() {
		let left: any = null;

		left = await this._factor();
	
		while ([
			TokenRecordList[TokenIdentifier.ADD],
			TokenRecordList[TokenIdentifier.SUB]
		].includes(this._lookahead?.type as string)) {
			const operator = await this._eat(this._lookahead?.type as string)

			const right = await this._expr();

			left = {
				type: 'expr',
				node: {
					operator,
					left,
					right
				}
			}
		}

		return left;
	}

	// statement || stmt =>
	//	| expr ;
	public async _stmt() {
		const expr = await this._expr();
		await this._eat(TokenRecordList[TokenIdentifier.SEMI]);

		return {
			type: 'stmt',
			node: expr
		}
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

		return {
			type: "program",
			body: stmts
		}
	}

	private async _eat(tokenType: string): Promise<Token> {
		const token = this._lookahead;

		if (token === null) {
			throw new SyntaxError(`Unexpected end of input, expected: ${tokenType}`);
		}

		if (token.type !== tokenType) {
			throw new SyntaxError(`Unexpected token: ${token.value}, expected: ${tokenType}`)
		}

		this._lookahead = await this._lexer.getNextToken();

		return token as Token;
	}

	public async parse(): Promise<any> {
		this._lookahead = await this._lexer.getNextToken();

		return await this._program();
	}
}

