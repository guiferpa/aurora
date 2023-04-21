import colorize from "json-colorizer";

import {Lexer, Token, TokenTag, TokenNumber, Environment} from "../../v1";
import {TokenIdentifier} from "../../v1/tokens";
import {
  BinaryOperationNode, 
  BlockStatmentNode, 
  IdentifierNode, 
  ParameterOperationNode, 
  ParserNode
} from "./node";

export default class Parser {
  private readonly _lexer: Lexer;
  private _lookahead: Token | null = null;
  private _environ: Environment | null = null;

  constructor(lexer: Lexer) {
    this._lexer = lexer;
  }

  private _eat(tokenTag: TokenTag): Token {
    const token = this._lookahead;

    if (token?.tag === TokenTag.EOT)
      throw new SyntaxError(
        `Unexpected end of token, expected token: ${tokenTag.toString()}, on line: ${token.line}, column: ${token.column}`
      );

    if (tokenTag !== token?.tag)
      throw new SyntaxError(
        `Unexpected token: ${token?.toString()}, on line: ${token?.line}, column: ${token?.column}`
      );


    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  /**
   * fact =>
   *  | NUM
   *  | IDENT
   *  | DEF IDENT ASSIGN expr
   *  | PAREN_BEGIN expr PAREN_END
   */
  private _fact(): ParserNode {
    if (this._lookahead?.tag === TokenTag.NUM) {
      const num = this._eat(TokenTag.NUM);
      return new ParameterOperationNode((num as TokenNumber).value);
    }

    if (this._lookahead?.tag === TokenTag.IDENT) {
      const ident = this._eat(TokenTag.IDENT);
      return (this._environ as Environment)
        .query((ident as TokenIdentifier).name);
    }

    if (this._lookahead?.tag === TokenTag.DEF) {
      this._eat(TokenTag.DEF);
      const ident = this._eat(TokenTag.IDENT);
      this._eat(TokenTag.ASSIGN);
      const expr = this._expr();
      this._environ?.set((ident as TokenIdentifier).name, expr);

      return new IdentifierNode((ident as TokenIdentifier).name);
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

    if (![
      TokenTag.ADD, 
      TokenTag.SUB
    ].includes(this._lookahead?.tag as TokenTag))
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

  /**
   * exprStmt =>
   *  | expr SEMI
   */
  private _exprStmt(): ParserNode {
    const add = this._add();
    this._eat(TokenTag.SEMI);
    return add;
  }

  /**
   * blckStmt =>
   *  | BLOCK_BEGIN stmt* BLOCK_END
   */
  private _blckStmt(): ParserNode {
    this._eat(TokenTag.BLOCK_BEGIN);

    const id = `${Date.now()}`;
    this._environ = new Environment(id, this._environ);
    const block = this._stmtList(TokenTag.BLOCK_END);
    const stmt = new BlockStatmentNode(this._environ.id, block);

    this._eat(TokenTag.BLOCK_END);

    this._environ = this._environ.prev;

    return stmt;
  }

  /**
   * stmt =>
   *  | blckStmt
   *  | exprStmt
   */
  private _stmt(): ParserNode {
    if (this._lookahead?.tag === TokenTag.BLOCK_BEGIN) {
      return this._blckStmt();
    }

    return this._exprStmt();
  }

  private _stmtList(et?: TokenTag): ParserNode[] {
    const list = [];
  
    while (this._lookahead?.tag !== et) {
      list.push(this._stmt());
    }

    return list;
  }

  /***
   * prorgram =>
   *  | stmt*
   */
  private _program(): ParserNode[] {
    return this._stmtList(TokenTag.EOT);
  }

  public parse(): BlockStatmentNode {
    this._lookahead = this._lexer.getNextToken();

    const id = "root";
    this._environ = new Environment(id, this._environ);
    const tree = new BlockStatmentNode(this._environ.id, this._program());

    console.log(colorize(JSON.stringify(tree, null, 2)));

    return tree;
  }
}
