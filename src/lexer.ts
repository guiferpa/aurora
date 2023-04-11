import {Token, TokenIdentifier, TokenRecordList, TokenSpecList} from "./tokens";

export class Lexer {
	public _cursor = 0;
	public readonly _buffer: Buffer;

	constructor(buffer: any) {
		this._buffer = buffer;
	}

	public async tokenify(): Promise<any> {
		const tokens: any[] = [];

		while (this.hasMoreTokens()) {
			const token = await this.getAsyncNextToken();
			tokens.push(token);
		}

		// Info about end of token list
		tokens.push(null);

		this._cursor = 0;

		return tokens;
	}

	public hasMoreTokens(): boolean {
		return this._cursor < this._buffer.length;
	}

	public getNextToken(): Token {
		if (!this.hasMoreTokens()) return {
			id: TokenIdentifier.EOT,
			type: TokenRecordList[TokenIdentifier.EOT],
			value: ""
		}

		const str = this._buffer.toString('utf-8', this._cursor);

		for (const [regex, tokenType, tokenId] of TokenSpecList) {
			const tokenValue = this._match(regex, str);

			if (tokenValue === null) continue;

			if (tokenId === TokenIdentifier.WHITESPACE)
				return this.getNextToken();

			return {id: tokenId, type: tokenType, value: tokenValue}
		}

		throw new SyntaxError(`Unexpected token: ${str}`);
	}

	public async getAsyncNextToken(): Promise<Token | null> {
		if (!this.hasMoreTokens()) return null;

		const str = this._buffer.toString('utf-8', this._cursor);

		for (const [regex, tokenType, tokenId] of TokenSpecList) {
			const tokenValue = this._match(regex as RegExp, str);

			if (tokenValue === null) continue;

			if (tokenId === TokenIdentifier.WHITESPACE) {
				return this.getAsyncNextToken();
			}

			return { id: tokenId, type: tokenType, value: tokenValue }
		}

		throw new SyntaxError(`Unexpected token: ${str}`);
	}

	private _match(product: RegExp, str: string) {
		const matched = product.exec(str);
		if (matched === null) return null

		const value = matched[0];
		this._cursor += value.length;
		return value;
	}
}

