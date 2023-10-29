import colorize from "json-colorizer";

import Environment, { FuncParameterType } from "./environment";

import { Lexer } from "@/lexer";
import {
  isAdditiveOperatorToken,
  isLogicalOperatorToken,
  isMultiplicativeOperatorToken,
  isRelativeOperatorToken,
  Token,
  TokenDef,
  TokenDefFunction,
  TokenIdentifier,
  TokenLogical,
  TokenNumber,
  TokenString,
  TokenTag,
  TokenTyping,
} from "@/tokens";

import {
  ArityNode,
  BinaryOperationNode,
  BlockStatmentNode,
  DefStatmentNode,
  DefFunctionStatmentNode,
  IfStatmentNode,
  IntegerNode,
  LogicalNode,
  ParserNode,
  ParserNodeReturnType,
  PrintCallStatmentNode,
  StringNode,
  UnaryOperationNode,
  ReturnStatmentNode,
} from "./node";

const types: Map<string, ParserNodeReturnType> = new Map([
  ["void", ParserNodeReturnType.Void],
  ["bool", ParserNodeReturnType.Logical],
  ["int", ParserNodeReturnType.Integer],
  ["str", ParserNodeReturnType.Str],
]);

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
      throw new SyntaxError(`Unexpected token: ${token} at [line: ${token?.line}, column: ${token?.column}, cursor: ${token?.cursor}]`);

    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  /**
   * fact =>
   *  | NUM
   *  | LOGICAL
   *  | IDENT
   *  | STR
   *  | DEF ASSIGN expr
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
      const ident = this._eat(TokenTag.IDENT) as TokenIdentifier;
      console.log(this._environ?.query(ident.name));
      return this._environ?.query(ident.name) as ParserNode;
    }

    if (this._lookahead?.tag === TokenTag.DEF) {
      const tdef = this._eat(TokenTag.DEF) as TokenDef;
      this._eat(TokenTag.ASSIGN);
      const log = this._log();
      this._environ?.set(tdef.name, log);

      return new DefStatmentNode(tdef.name, log);
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

    if (!isMultiplicativeOperatorToken(this._lookahead as Token)) return fact;

    const operator = this._eat(this._lookahead?.tag as TokenTag);
    const mult = this._mult();

    if (
      fact.returnType !== ParserNodeReturnType.Integer ||
      mult.returnType !== ParserNodeReturnType.Integer
    )
      throw new SyntaxError(
        `It's not possible use ${operator} operator with non-integer parameters`
      );

    return new BinaryOperationNode(
      fact,
      mult,
      operator,
      ParserNodeReturnType.Integer
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

    if (!isAdditiveOperatorToken(this._lookahead as Token)) return mult;

    const operator = this._eat(this._lookahead?.tag as TokenTag);
    const add = this._add();

    if (
      mult.returnType !== ParserNodeReturnType.Integer ||
      add.returnType !== ParserNodeReturnType.Integer
    )
      throw new SyntaxError(
        `It's not possible use ${operator} operator with non-integer parameters`
      );

    return new BinaryOperationNode(
      mult,
      add,
      operator,
      ParserNodeReturnType.Integer
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

    if (!isRelativeOperatorToken(this._lookahead as Token)) return add;

    const operator = this._eat(this._lookahead?.tag as TokenTag);
    const rel = this._rel();

    return new BinaryOperationNode(
      add,
      rel,
      operator,
      ParserNodeReturnType.Logical
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

    if (!isLogicalOperatorToken(this._lookahead as Token)) return rel;

    const operator = this._eat(this._lookahead?.tag as TokenTag);
    const log = this._log();

    if (
      rel.returnType !== ParserNodeReturnType.Logical ||
      log.returnType !== ParserNodeReturnType.Logical
    )
      throw new SyntaxError(
        `It's not possible use ${operator} operator with non-boolean parameters`
      );

    return new BinaryOperationNode(
      rel,
      log,
      operator,
      ParserNodeReturnType.Logical
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

      return new UnaryOperationNode(
        log,
        operator,
        ParserNodeReturnType.Logical
      );
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

  private _rtrn(): ReturnStatmentNode {
    this._eat(TokenTag.RETURN);
    const stmt = this._stmt();

    return new ReturnStatmentNode(stmt);
  }

  private _typing(): ParserNodeReturnType {
    const token = this._eat(TokenTag.TYPING) as TokenTyping;
    const t = types.get(token.value);
    if (t === undefined)
      throw new SyntaxError(`Invalid type named ${token.value}`);
    return t;
  }

  /**
   * func =>
   *  | FUNC IDENT arity BLOCK_BEGIN stmts BLOCK_END
   */
  private _defFunc(): DefFunctionStatmentNode {
    const tdfunc = this._eat(TokenTag.DEF_FUNC) as TokenDefFunction;

    const rtype = ParserNodeReturnType.Void;

    const id = `${Date.now()}`;
    this._environ = new Environment(id, this._environ);

    const stmts = this._stmts(TokenTag.BLOCK_END);

    this._eat(TokenTag.BLOCK_END);

    const arity = new ArityNode(tdfunc.arity.params);

    arity.params.forEach((param, index) => {
      this._environ?.set(param, `${FuncParameterType}${index}`);
    });

    const fn = new DefFunctionStatmentNode(
      this._environ.id,
      tdfunc.name,
      arity,
      stmts,
      rtype
    );

    fn.isValid();

    this._environ = this._environ.prev;

    this._environ?.set(fn.name, fn);

    return fn;
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
   *  | func SEMI
   *  | print SEMI
   *  | opp SEMI
   *  | rtrn SEMI
   */
  private _stmt(): ParserNode {
    if (this._lookahead?.tag === TokenTag.RETURN) {
      return this._rtrn();
    }

    if (this._lookahead?.tag === TokenTag.BLOCK_BEGIN) {
      return this._block();
    }

    if (this._lookahead?.tag === TokenTag.DEF_FUNC) {
      return this._defFunc();
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
