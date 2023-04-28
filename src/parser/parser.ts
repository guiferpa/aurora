import colorize from "json-colorizer";

import Environment from "./environment";

import {Lexer} from "@/lexer";
import {
  isAdditiveOperatorToken,
  isLogicalOperatorToken,
  isMultiplicativeOperatorToken, 
  isRelativeOperatorToken, 
  Token, 
  TokenIdentifier, 
  TokenLogical, 
  TokenNumber, 
  TokenString,
  TokenTag
} from "@/tokens";

import {
  BinaryOperationNode,
  BlockStatmentNode,
  IdentifierNode,
  IfStatmentNode,
  IntegerNode,
  LogicalNode,
  ParserNode,
  ParserNodeReturnType,
  PrintCallStatmentNode,
  StringNode,
  UnaryOperationNode,
} from "./node";

export default class Parser {
  private readonly _lexer: Lexer;
  private _lookahead: Token | null = null;
  public _environ: Environment | null = null;

  constructor(lexer: Lexer) {
    this._lexer = lexer;
  }

  private _eat(tokenTag: TokenTag): Token {
    const token = this._lookahead;

    if (token?.tag === TokenTag.EOT)
      throw new SyntaxError(
        `Unexpected end of token, expected token: ${tokenTag}`
      );

    if (tokenTag !== token?.tag)
      throw new SyntaxError(`Unexpected token: ${token}`);

    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  /**
   * fact =>
   *  | NUM
   *  | LOGICAL
   *  | IDENT
   *  | STR
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

    if (this._lookahead?.tag === TokenTag.STR) {
      const str = this._eat(TokenTag.STR);
      return new StringNode((str as TokenString).value);
    }

    if (this._lookahead?.tag === TokenTag.IDENT) {
      const ident = (this._eat(TokenTag.IDENT) as TokenIdentifier);
      return (this._environ as Environment).query(ident.name);
    }

    if (this._lookahead?.tag === TokenTag.DEF) {
      this._eat(TokenTag.DEF);
      const ident = (this._eat(TokenTag.IDENT) as TokenIdentifier);
      this._eat(TokenTag.ASSIGN);
      const log = this._log();
      this._environ?.set(ident.name, log);

      return new IdentifierNode(ident.name, log);
    }

    this._eat(TokenTag.PAREN_BEGIN);
    const expr = this._log();
    this._eat(TokenTag.PAREN_END);

    return expr;
  }

  /**
   * mult =>
   *  | fact * mult
   *  | fact
   */
  private _mult(): ParserNode {
    const fact = this._fact();

    if (!isMultiplicativeOperatorToken(this._lookahead as Token))
      return fact;

    const operator = this._eat(this._lookahead?.tag as TokenTag);
    const mult = this._mult();

    if (
      fact.returnType !== ParserNodeReturnType.Integer
      || mult.returnType !== ParserNodeReturnType.Integer
    )
      throw new SyntaxError(
        `It's not possible use ${operator} operator with non-integer parameters`
      );

    return new BinaryOperationNode(
      fact, mult, operator, ParserNodeReturnType.Integer
    );
  }

  /**
   * add =>
   *  | mult + add
   *  | mult - add
   *  | mult
   */
  private _add(): ParserNode {
    const mult = this._mult();

    if (!isAdditiveOperatorToken(this._lookahead as Token))
      return mult;

    const operator = this._eat(this._lookahead?.tag as TokenTag);
    const add = this._add();

    if (
      mult.returnType !== ParserNodeReturnType.Integer
      || add.returnType !== ParserNodeReturnType.Integer
    )
      throw new SyntaxError(
        `It's not possible use ${operator} operator with non-integer parameters`
      );

    return new BinaryOperationNode(
      mult, add, operator, ParserNodeReturnType.Integer
    );
  }

  /**
   * rel =>
   *  | add == rel
   *  | add > rel
   *  | add < rel
   *  | add
   */
  private _rel(): ParserNode {
    const add = this._add();

    if (!isRelativeOperatorToken(this._lookahead as Token))
      return add;

    const operator = this._eat(this._lookahead?.tag as TokenTag);
    const rel = this._rel();

    return new BinaryOperationNode(
      add, rel, operator, ParserNodeReturnType.Logical
    );
  }

  /**
   * log =>
   *  | rel OR log
   *  | rel AND log
   *  | rel
   */
  private _log(): ParserNode {
    const rel = this._rel();

    if (!isLogicalOperatorToken(this._lookahead as Token))
      return rel;

    const operator = this._eat(this._lookahead?.tag as TokenTag);
    const log = this._log();

    if (
      rel.returnType !== ParserNodeReturnType.Logical
      || log.returnType !== ParserNodeReturnType.Logical
    )
      throw new SyntaxError(
        `It's not possible use ${operator} operator with non-boolean parameters`
      );

    return new BinaryOperationNode(
      rel, log, operator, ParserNodeReturnType.Logical
    );
  }

  /**
   * opp =>
   *  | !log
   *  | log
   */
  private _opp(): ParserNode {
    if (this._lookahead?.tag === TokenTag.OPP) {
      const operator = this._eat(TokenTag.OPP);
      const log = this._log();

      if (log.returnType !== ParserNodeReturnType.Logical)
        throw new SyntaxError(
          `It's not possible use ${operator} operator with non-boolean parameters`
        );

      return new UnaryOperationNode(log, operator, ParserNodeReturnType.Logical);
    }

    return this._log();
  }

  /**
   * if =>
   *  | IF PAREN_BEGIN opp PAREN_END block
   */
  private _if(): ParserNode {
    const tif = this._eat(TokenTag.IF);
    this._eat(TokenTag.PAREN_BEGIN);
    const opp = this._opp();
    this._eat(TokenTag.PAREN_END);

    if (opp.returnType !== ParserNodeReturnType.Logical)
      throw new SyntaxError(
        `It's not possible use ${opp.tag} no-boolean expression as if-condition`
      );

    const { block } = this._block();
    const id = `${tif}-${Date.now()}`;

    return new IfStatmentNode(id, opp, block);
  }

  /**
   * block =>
   *  | BLOCK_BEGIN stmts BLOCK_END
   */
  private _block(): BlockStatmentNode {
    this._eat(TokenTag.BLOCK_BEGIN);

    const id = `${Date.now()} `;
    this._environ = new Environment(id, this._environ);
    const stmts = this._stmts(TokenTag.BLOCK_END);
    const stmt = new BlockStatmentNode(this._environ.id, stmts);

    this._eat(TokenTag.BLOCK_END);

    this._environ = this._environ.prev;

    return stmt;
  }

  /**
   * call =>
   *  | IDENT PAREN_BEGIN opp PAREN_END
   */
  private _print(): ParserNode {
    this._eat(TokenTag.CALL_PRINT);
    this._eat(TokenTag.PAREN_BEGIN);
    const opp = this._opp();
    this._eat(TokenTag.PAREN_END);

    return new PrintCallStatmentNode(opp);
  }

  /**
   * stmt =>
   *  | block
   *  | if
   *  | print SEMI
   *  | opp SEMI
   */
  private _stmt(): ParserNode {
    if (this._lookahead?.tag === TokenTag.BLOCK_BEGIN) {
      return this._block();
    }

    if (this._lookahead?.tag === TokenTag.IF) {
      return this._if();
    }

    if (this._lookahead?.tag === TokenTag.CALL_PRINT) {
      const print = this._print();
      this._eat(TokenTag.SEMI);

      return print;
    }

    const opp = this._opp();
    this._eat(TokenTag.SEMI);

    return opp;
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

  public parse(): ParserNode {
    this._lookahead = this._lexer.getNextToken();

    const id = "root";
    this._environ = new Environment(id, this._environ);
    return new BlockStatmentNode(this._environ.id, this._program());
  }
}
