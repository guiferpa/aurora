import { TokenTag } from "@/lexer/tokens/tag";
import { Token } from "@/lexer/tokens/token";
import {
  BinaryOpNode,
  AssignStmtNode,
  IdentNode,
  NumericalNode,
  ParserNode,
  ProgramNode,
  BlockStmtNode,
  LogicalNode,
  NegativeExprNode,
  RelativeExprNode,
  LogicExprNode,
  UnaryOpNode,
  IfStmtNode,
  DeclFuncStmtNode,
  ArityStmtNode,
  ParamNode,
  CallPrintStmtNode,
  CallFuncStmtNode,
} from "./node";

import SymTable from "@/symtable";

interface Lexer {
  getNextToken(): Token | null;
}

export default class Parser {
  private _lookahead: Token | null = null;

  constructor(
    private readonly _lexer: Lexer,
    private _symtable: SymTable | null
  ) {}

  private _eat(tokenTag: TokenTag): Token {
    const token = this._lookahead;

    if (token?.tag === TokenTag.EOF)
      throw new SyntaxError(
        `Unexpected end of token, expected token: ${tokenTag}`
      );

    if (tokenTag !== token?.tag)
      throw new SyntaxError(`Unexpected token: ${this._lookahead?.value}`);

    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  /**
   * _fact -> __PAREN_O__ _log __PAREN_C__
   *        | __IDENT__
   *        | __LOG__
   *        | __NUM__
   * **/
  private _fact(): ParserNode {
    if (this._lookahead?.tag === TokenTag.PAREN_O) {
      this._eat(TokenTag.PAREN_O);
      const log = this._log();
      this._eat(TokenTag.PAREN_C);
      return log;
    }

    if (this._lookahead?.tag === TokenTag.IDENT) {
      const ident = this._eat(TokenTag.IDENT);
      this._symtable?.has(ident.value);
      return new IdentNode(ident.value);
    }

    if (this._lookahead?.tag === TokenTag.LOG) {
      const log = this._eat(TokenTag.LOG);
      return new LogicalNode(log.value === "true");
    }

    const { value } = this._eat(TokenTag.NUM);
    const num = Number.parseInt(value);
    if (Number.isNaN(num)) throw new Error(`Value ${value} is not a number`);
    return new NumericalNode(num);
  }

  /**
   * _uny -> __OP_ADD__ _uny
   *        | __OP_SUB__ _uny
   *        | _fact
   * **/
  private _uny(): ParserNode {
    if (
      this._lookahead?.tag === TokenTag.OP_ADD ||
      this._lookahead?.tag === TokenTag.OP_SUB
    ) {
      const op = this._eat(this._lookahead.tag);
      const uny = this._uny();
      return new UnaryOpNode(uny, op);
    }

    return this._fact();
  }

  /**
   * _term -> _uny __OP_MUL__ _term
   *        | _uny __OP_DIV__ _term
   *        | _uny
   * **/
  private _term(): ParserNode {
    const uny = this._uny();

    if (this._lookahead?.tag === TokenTag.OP_MUL) {
      const op = this._eat(TokenTag.OP_MUL);
      return new BinaryOpNode(uny, this._term(), op);
    }

    if (this._lookahead?.tag === TokenTag.OP_DIV) {
      const op = this._eat(TokenTag.OP_DIV);
      return new BinaryOpNode(uny, this._term(), op);
    }

    return uny;
  }

  /**
   * _expr -> _term __OP_ADD__ _expr
   *        | _term __OP_SUB__ _expr
   *        | _term
   * **/
  private _expr(): ParserNode {
    const term = this._term();

    if (this._lookahead?.tag === TokenTag.OP_ADD) {
      const op = this._eat(TokenTag.OP_ADD);
      return new BinaryOpNode(term, this._expr(), op);
    }

    if (this._lookahead?.tag === TokenTag.OP_SUB) {
      const op = this._eat(TokenTag.OP_SUB);
      return new BinaryOpNode(term, this._expr(), op);
    }

    return term;
  }

  /**
   * _rel ->  _expr __REL_GT__ _rel
   *        | _expr __REL_LT__ _rel
   *        | _expr __REL_EQ__ _rel
   *        | _expr __REL_DIF__ _rel
   *        | _expr
   * **/
  private _rel(): ParserNode {
    const expr = this._expr();

    if (
      this._lookahead?.tag === TokenTag.REL_GT ||
      this._lookahead?.tag === TokenTag.REL_LT ||
      this._lookahead?.tag === TokenTag.REL_EQ ||
      this._lookahead?.tag === TokenTag.REL_DIF
    ) {
      const op = this._eat(this._lookahead.tag);
      const rel = this._rel();
      return new RelativeExprNode(expr, rel, op);
    }

    return expr;
  }

  /**
   * _neg -> __NEG__ _rel
   *        | _rel
   * **/
  private _neg(): ParserNode {
    if (this._lookahead?.tag === TokenTag.NEG) {
      this._eat(TokenTag.NEG);
      return new NegativeExprNode(this._rel());
    }

    return this._rel();
  }

  /**
   * _log -> _neg __LOG_AND__ _log
   *         | _neg __LOG_OR__ _log
   *         | _neg
   * **/
  private _log(): ParserNode {
    const neg = this._neg();

    if (
      this._lookahead?.tag === TokenTag.LOG_AND ||
      this._lookahead?.tag === TokenTag.LOG_OR
    ) {
      const op = this._eat(this._lookahead.tag);
      const log = this._log();
      return new LogicExprNode(neg, log, op);
    }

    return neg;
  }

  /**
   * _callfn -> __IDENT__ __PAREN_O__ _log (__COMMA__ _log)* __PAREN_C__
   *         | __IDENT__ __PAREN_O__ __PAREN_C__
   *         | _log
   * **/
  private _callfn(): ParserNode {
    if (this._lookahead?.tag !== TokenTag.IDENT) {
      return this._log();
    }

    const id = this._eat(TokenTag.IDENT);
    this._eat(TokenTag.PAREN_O);

    // @ts-ignore
    if (this._lookahead.tag === TokenTag.PAREN_C) {
      this._eat(TokenTag.PAREN_C);
      return new CallFuncStmtNode(id.value, []);
    }

    const params: ParserNode[] = [this._log()];

    // @ts-ignore
    while (this._lookahead.tag === TokenTag.COMMA) {
      this._eat(TokenTag.COMMA);
      params.push(this._log());
    }

    this._eat(TokenTag.PAREN_C);

    return new CallFuncStmtNode(id.value, params);
  }

  /**
   * _param -> __IDENT__
   *         | NULL
   * **/
  private _param(): ParserNode {
    const id = this._eat(TokenTag.IDENT);
    return new ParamNode(id.value);
  }

  /**
   * _arity -> (_param __COMMA__ _param)*
   *         | _param
   *         | NULL
   * **/
  private _arity(): ParserNode {
    if (this._lookahead?.tag !== TokenTag.IDENT) {
      return new ArityStmtNode([]);
    }

    const params: ParserNode[] = [this._param()];

    // @ts-ignore
    while (this._lookahead?.tag === TokenTag.COMMA) {
      this._eat(TokenTag.COMMA);
      params.push(this._param());
    }

    return new ArityStmtNode(params);
  }

  /**
   * _block -> __BRACK_O__ _statements __BRACK_C__
   *         | _callfn
   * **/
  private _block(): ParserNode {
    if (this._lookahead?.tag === TokenTag.BRACK_O) {
      const envId = `BLOCK-${Date.now()}`;
      this._symtable = new SymTable(envId, this._symtable);

      this._eat(TokenTag.BRACK_O);
      const statements = this._statements(TokenTag.BRACK_C);
      this._eat(TokenTag.BRACK_C);

      this._symtable.previous?.mergeRefs(this._symtable);
      this._symtable = this._symtable.previous;

      return new BlockStmtNode(statements);
    }

    return this._callfn();
  }

  /**
   * _if -> __IF__ _callfn _block
   * **/
  private _if(): ParserNode {
    this._eat(TokenTag.IF);
    const callfn = this._callfn();
    const block = this._block();
    return new IfStmtNode(callfn, block);
  }

  /**
   * _ass -> __ASS__ _callfn
   * **/
  private _ass(): ParserNode {
    if (this._lookahead?.tag === TokenTag.ASSIGN) {
      const ass = this._eat(TokenTag.ASSIGN);
      const callfn = this._callfn();
      this._symtable?.set(ass.value, callfn);
      return new AssignStmtNode(ass.value, callfn);
    }

    return this._log();
  }

  /**
   * _declfunc -> __DECL_FN__ __PAREN_O__ _arity __PAREN_C__ _block
   * **/
  private _declfunc(): ParserNode {
    const func = this._eat(TokenTag.DECL_FN);
    this._eat(TokenTag.PAREN_O);
    const arity = this._arity();
    this._eat(TokenTag.PAREN_C);
    const block = this._block();
    this._symtable?.set(func.value, arity);
    return new DeclFuncStmtNode(func.value, arity, block);
  }

  /**
   * _print -> __CALL_PRINT__ __PAREN_O__ _log __PAREN_C__
   * **/
  private _print(): ParserNode {
    this._eat(TokenTag.CALL_PRINT);
    this._eat(TokenTag.PAREN_O);
    const log = this._log();
    this._eat(TokenTag.PAREN_C);
    return new CallPrintStmtNode(log);
  }

  /**
   * _statment -> _block
   *            | _if
   *            | _ass
   *            | _declfunc
   *            | _print
   * **/
  private _statement(): ParserNode {
    if (this._lookahead?.tag === TokenTag.CALL_PRINT) {
      return this._print();
    }

    if (this._lookahead?.tag === TokenTag.DECL_FN) {
      return this._declfunc();
    }

    if (this._lookahead?.tag === TokenTag.ASSIGN) {
      return this._ass();
    }

    if (this._lookahead?.tag === TokenTag.IF) {
      return this._if();
    }

    return this._block();
  }

  private _statements(eot: TokenTag): ParserNode[] {
    const list = [];

    while (this._lookahead?.tag !== eot) {
      list.push(this._statement());
    }

    return list;
  }

  private _program(): ProgramNode {
    const program = new ProgramNode(this._statements(TokenTag.EOF));

    // Check if there are some declaration not referenced
    // this._symtable?.hasAnyRef();

    return program;
  }

  public parse(): ProgramNode {
    this._lookahead = this._lexer.getNextToken();

    return this._program();
  }
}
