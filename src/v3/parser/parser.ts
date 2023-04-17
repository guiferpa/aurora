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
   *  | PAREN_BEGIN expr PAREN_END
   */
  private _fact(): ParserNode {
    if (this._lookahead?.tag === TokenTag.NUM) {
      const num = this._eat(TokenTag.NUM);
      return new ParameterOperationNode((num as TokenNumber).value);
    }

    this._eat(TokenTag.PAREN_BEGIN);
    const expr = this._expr();
    this._eat(TokenTag.PAREN_END);

    return expr;
  }

  /**
   * mult =>
   *  | fact * mult
   *  | fact
   */
  private _mult(): ParserNode {
    const left = this._fact();

    if (![TokenTag.MULT].includes(this._lookahead?.tag as TokenTag))
      return left;
    
    const operator = this._eat(this._lookahead?.tag as TokenTag);
    return new BinaryOperationNode(left, this._mult(), operator);
  }

  /**
   * add =>
   *  | mult + add
   *  | mult - add
   *  | mult
   */
  private _add(): ParserNode {
    const left = this._mult();

    if (![TokenTag.ADD, TokenTag.SUB].includes(this._lookahead?.tag as TokenTag))
      return left;
    
    const operator = this._eat(this._lookahead?.tag as TokenTag);
    return new BinaryOperationNode(left, this._add(), operator);
  }

  /**
   * expr =>
   *  | add
   */
  private _expr(): ParserNode {
    return this._add();
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

    const result = this._program();
    console.log(result);

    return result;
  }
}
