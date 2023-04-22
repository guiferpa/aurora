import colorize from "json-colorizer";

import {
  Lexer, 
  Token, 
  TokenTag, 
  TokenNumber, 
  Environment,
  TokenLogical,
  TokenIdentifier
} from "../../v1";

import {
  BinaryOperationNode, 
  RelativeOperationNode,
  BlockStatmentNode, 
  IdentifierNode, 
  IntegerNode, 
  LogicalNode, 
  ParserNode,
  NegativeOperationNode,
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
        `Unexpected end of token, expected token: ${tokenTag.toString()}`
      );

    if (tokenTag !== token?.tag)
      throw new SyntaxError(
        `Unexpected token: ${token?.toString()}`
      );

    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  /**
   * fact =>
   *  | NUM
   *  | LOGICAL
   *  | IDENT
   *  | DEF IDENT ASSIGN expr
   *  | PAREN_BEGIN expr PAREN_END
   */
  private _fact(): ParserNode {
    if (this._lookahead?.tag === TokenTag.NUM) {
      const num = this._eat(TokenTag.NUM);
      return new IntegerNode((num as TokenNumber).value);
    }

    if (this._lookahead?.tag === TokenTag.LOGICAL) {
      const logical = this._eat(TokenTag.LOGICAL);
      return new LogicalNode((logical as TokenLogical).value);
    }

    if (this._lookahead?.tag === TokenTag.IDENT) {
      const ident = (this._eat(TokenTag.IDENT) as TokenIdentifier);
      return (this._environ as Environment).query(ident.name);
    }

    if (this._lookahead?.tag === TokenTag.DEF) {
      this._eat(TokenTag.DEF);
      const ident = (this._eat(TokenTag.IDENT) as TokenIdentifier);
      this._eat(TokenTag.ASSIGN);
      const expr = this._expr();
      this._environ?.set(ident.name, expr);

      return new IdentifierNode(ident.name, expr);
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
   * rel =>
   *  | add == rel
   *  | add
   */
  private _rel(): ParserNode {
    const add = this._add();

    if (![
      TokenTag.EQUAL,
      TokenTag.GREATER_THAN,
      TokenTag.LESS_THAN
    ].includes(this._lookahead?.tag as TokenTag))
      return add;

    const comparator = this._eat(this._lookahead?.tag as TokenTag);

    return new RelativeOperationNode(add, this._rel(), comparator);
  }

  /**
   * expr =>
   *  | rel
   */
  private _expr(): ParserNode {
    return this._rel();
  }

  /**
   * neg =>
   *  | NOT expr
   *  | expr
   */
  private _neg(): ParserNode {
    if (this._lookahead?.tag === TokenTag.NEG) {
      this._eat(this._lookahead.tag);

      const expr = this._expr();

      if (
        (expr instanceof RelativeOperationNode) 
        || (expr instanceof LogicalNode)
      )
        return new NegativeOperationNode(expr);

      throw new SyntaxError(
        `Invalid negative syntax for no relative expression`
      );
    }

    return this._expr();
  }

  /**
   * blckStmt =>
   *  | BLOCK_BEGIN stmts BLOCK_END
   */
  private _block(): ParserNode {
    this._eat(TokenTag.BLOCK_BEGIN);

    const id = `${Date.now()}`;
    this._environ = new Environment(id, this._environ);
    const stmts = this._stmts(TokenTag.BLOCK_END);
    const stmt = new BlockStatmentNode(this._environ.id, stmts);

    this._eat(TokenTag.BLOCK_END);

    this._environ = this._environ.prev;

    return stmt;
  }

  /**
   * stmt =>
   *  | block
   *  | neg SEMI
   */
  private _stmt(): ParserNode {
    if (this._lookahead?.tag === TokenTag.BLOCK_BEGIN) {
      return this._block();
    }

    const expr = this._neg();
    this._eat(TokenTag.SEMI);

    return expr;
  }

  private _stmts(et?: TokenTag): ParserNode[] {
    const list = [];
  
    while (this._lookahead?.tag !== et) {
      list.push(this._stmt());
    }

    return list;
  }

  /***
   * prorgram =>
   *  | stmts
   */
  private _program(): ParserNode[] {
    return this._stmts(TokenTag.EOT);
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
