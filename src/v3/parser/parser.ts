import {Lexer, Token, TokenTag} from "../../v1";
import {TokenNumber} from "../../v1/tokens";
import {BinaryOperationNode, ParameterOperationNode, ParserNode} from "./node";

export default class Parser {
  private readonly _lexer: Lexer;
  private _lookahead: Token | null = null;

  constructor(lexer: Lexer) {
    this._lexer = lexer;
  }

  private _eat(tokenTag: TokenTag): Token {
    const token = this._lookahead;

    if (tokenTag !== token?.tag)
      throw new SyntaxError(`Unexpected token: ${token?.toString()}`);

    if (token?.tag === TokenTag.EOT)
      throw new SyntaxError(`Unexpected end of token, expected token: ${tokenTag}`);

    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  /**
   * fact =>
   *  | NUM
   *  | IDENT
   */
  private _fact(): Token {
    return this._eat(TokenTag.NUM);
  }

  /**
   * expr =>
   *  | fact + expr
   *  | fact - expr
   *  | fact
   */
  private _expr(): ParserNode {
    const left = new ParameterOperationNode((this._fact() as TokenNumber).value);

    if (![TokenTag.ADD, TokenTag.SUB].includes(this._lookahead?.tag as TokenTag))
      return left;
    
    const operator = this._eat(this._lookahead?.tag as TokenTag);
    return new BinaryOperationNode(left, this._expr(), operator);
  }

  /***
   * prorgram =>
   *  | expr
   */
  private _program(): ParserNode {
    return this._expr();
  }

  public parse(): ParserNode {
    this._lookahead = this._lexer.getNextToken();

    return this._program();
  }
}
