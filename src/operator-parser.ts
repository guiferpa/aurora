import Lexer from "./lexer";
import {Token, TokenIdentifier, TokenRecordList, TokenIdentifierType} from "./tokens";

export default class OperatorParser {
  private readonly _lexer: Lexer;
  private _lookahead: Token;
  private _match = {
    primary: [TokenIdentifier.NUMBER],
    seconday: [TokenIdentifier.MULT],
    tertiary: [
      TokenIdentifier.ADD,
      TokenIdentifier.SUB
    ]
  };
  private _precedenceTable: Map<TokenIdentifierType, number> = new Map([
    [TokenIdentifier.ADD, 1],
    [TokenIdentifier.SUB, 1],
    [TokenIdentifier.MULT, 2],
    [TokenIdentifier.NUMBER, 3]
  ]);

  constructor(lexer: Lexer) {
    this._lexer = lexer;
    this._lookahead = this._lexer.getNextToken();
  }

  private _eat(id: TokenIdentifierType) {
    const token = this._lookahead;

    if (token.id === TokenIdentifier.EOT)
      throw new SyntaxError(`Unexpected end of token, expected token: ${TokenRecordList[id]}`);

    if (id !== token.id)
      throw new SyntaxError(`Unexpected token type: ${token.id}`);

    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  /***
  * Primary =>
  *   | NUMBER (Token)
  */
  private _primary(): any {
    const token = this._eat(TokenIdentifier.NUMBER);

    return {
      type: "ParameterOperation",
      value: token.value
    };
  }

  /***
  * Secondary =>
  *   | Primary (1)
  *   | Primary * Primary (1 x 1)
  *   | Primary / Primary (2 / 10)
  */
  private _secondary(): any {
    const value = this._primary();

    if (!this._match.seconday.includes(this._lookahead.id)) {
      return value;
    }

    const operator = this._eat(this._lookahead.id);
    const node = this._primary();

    return {
      type: "BinaryOperation",
      operator, value, node
    };
  }

  /***
  * Tertiary =>
  *   | Secondary
  *   | Secondary + Secondary (1 + 1)
  *   | Secondary - Secondary (1 - 1)
  */
  private _tertiary(): any {
    const value = this._secondary();

    if (!this._match.tertiary.includes(this._lookahead.id)) {
      return value;
    }

    const operator = this._eat(this._lookahead.id);
    const node = this._tertiary();

    return {
      type: "BinaryOperation",
      operator, value, node
    };
  }

  public precedenceTree(): Token[] {
    const stack: Token[] = [];
    const tree: Token[] = [];

    while (true) {
      if (this._lookahead.id === TokenIdentifier.EOT && stack.length === 0)
        break

      if (stack.length === 0) {
        stack.push(this._lookahead);
        this._lookahead = this._lexer.getNextToken();
        continue;
      }

      if (this._lookahead.id === TokenIdentifier.EOT) {
        const token = stack.pop() as Token;
        tree.push(token);
        continue;
      }

      const lookaheadPrecedence = this._precedenceTable.get(this._lookahead.id) as number;
      const stackedPrecedence = this._precedenceTable.get(stack[stack.length - 1].id) as number;
      
      console.log('-------------------------------');
      console.log(`Stack: ${JSON.stringify(stack)}`);
      console.log(`Tree: ${JSON.stringify(tree)}`);
      console.log(`Lookahead: ${this._lookahead.value}`, '<==>', `Stack: ${stack[stack.length - 1].value}`);
      console.log('-------------------------------');
      if (lookaheadPrecedence > stackedPrecedence) {
        stack.push(this._lookahead);
        this._lookahead = this._lexer.getNextToken();
        continue;
      }

      const token = stack.pop() as Token;
      tree.push(token);
    }

    return tree;
  }

  public parse(): any {
    return this._tertiary();
  }
}
