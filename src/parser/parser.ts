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
  CallPrintStmtNode,
  CallFuncStmtNode,
  DescFuncStmtNode,
  StringNode,
  ReturnStmtNode,
  ReturnVoidStmtNode,
  CallArgStmtNode,
  CallConcatStmtNode,
  ArrayNode,
  CallMapStmtNode,
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
      throw new SyntaxError(
        `Unexpected token at line: ${this._lookahead?.line}, column: ${this._lookahead?.column}, value: ${this._lookahead?.value}`
      );

    this._lookahead = this._lexer.getNextToken();
    return token;
  }

  /**
   * _fact -> __PAREN_O__ _log __PAREN_C__
   *        | __IDENT__
   *        | __LOG__
   *        | __NUM__
   *        | __STR__
   * **/
  private _fact(): ParserNode {
    if (this._lookahead?.tag === TokenTag.PAREN_O) {
      return this._pre();
    }

    if (this._lookahead?.tag === TokenTag.CALL_ARG) {
      return this._arg();
    }

    if (this._lookahead?.tag === TokenTag.CALL_CONCAT) {
      return this._concat();
    }

    if (this._lookahead?.tag === TokenTag.CALL_MAP) {
      return this._map();
    }

    if (this._lookahead?.tag === TokenTag.IDENT) {
      return this._call();
    }

    if (this._lookahead?.tag === TokenTag.LOG) {
      return this._bool();
    }

    if (this._lookahead?.tag === TokenTag.STR) {
      return this._str();
    }

    if (this._lookahead?.tag === TokenTag.NUM) {
      return this._num();
    }

    if (this._lookahead?.tag === TokenTag.S_BRACK_O) {
      return this._arr();
    }

    throw new Error(`Unknwon token ${JSON.stringify(this._lookahead)}`);
  }

  private _map(): ParserNode {
    this._eat(TokenTag.CALL_MAP);
    this._eat(TokenTag.PAREN_O);
    const param = this._fact();
    this._eat(TokenTag.COMMA);
    const fact = this._fact();
    this._eat(TokenTag.PAREN_C);
    return new CallMapStmtNode(param, fact);
  }

  // __PAREN_O__ __PAREN_C__
  private _pre(): ParserNode {
    this._eat(TokenTag.PAREN_O);
    const log = this._log();
    this._eat(TokenTag.PAREN_C);
    return log;
  }

  // __S_BREAK_O__ __S_BREAK_C__
  private _arr(): ArrayNode {
    this._eat(TokenTag.S_BRACK_O);
    if (this._lookahead?.tag === TokenTag.S_BRACK_C) {
      this._eat(TokenTag.S_BRACK_C);
      return new ArrayNode([]);
    }

    const items = [this._log()];
    while (this._lookahead?.tag === TokenTag.COMMA) {
      this._eat(TokenTag.COMMA);
      items.push(this._log());
    }
    this._eat(TokenTag.S_BRACK_C);

    return new ArrayNode(items);
  }

  // __LOG__
  private _bool(): ParserNode {
    const log = this._eat(TokenTag.LOG);
    return new LogicalNode(log.value === "true");
  }

  // __NUM__
  private _num(): ParserNode {
    const { value } = this._eat(TokenTag.NUM);
    const num = Number.parseInt(value);
    if (Number.isNaN(num)) throw new Error(`Value ${value} is not a number`);
    return new NumericalNode(num);
  }

  // __STR__
  private _str(): ParserNode {
    const str = this._eat(TokenTag.STR);
    return new StringNode(str.value);
  }

  // __IDENT__ or __CALL_FUNC__
  private _call(): ParserNode {
    const ident = this._eat(TokenTag.IDENT);
    this._symtable?.has(ident.value);

    // @ts-ignore
    if (this._lookahead.tag === TokenTag.PAREN_O) {
      return this._callfn(ident);
    }

    return new IdentNode(ident.value);
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
   * _descfn -> __DESC_FUNC__ __STR__
   *         | NULL
   * **/
  private _descfn(): DescFuncStmtNode | null {
    if (this._lookahead?.tag === TokenTag.DESC_FUNC) {
      this._eat(TokenTag.DESC_FUNC);
      const str = this._eat(TokenTag.STR);
      return new DescFuncStmtNode(str.value);
    }

    return null;
  }

  /**
   * _callfn -> __IDENT__ __PAREN_O__ _log (__COMMA__ _log)* __PAREN_C__
   *         | __IDENT__ __PAREN_O__ __PAREN_C__
   *         | __IDENT__
   *         | _log
   * **/
  private _callfn(id: Token): ParserNode {
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
  private _param(): string {
    const id = this._eat(TokenTag.IDENT);
    return id.value;
  }

  /**
   * _arity -> (_param __COMMA__ _param)*
   *         | _param
   *         | NULL
   * **/
  private _arity(): ArityStmtNode {
    if (this._lookahead?.tag !== TokenTag.IDENT) {
      return new ArityStmtNode([]);
    }

    const params: string[] = [this._param()];

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

    return this._log();
  }

  /**
   * _if -> __IF__ _callfn _block
   * **/
  private _if(): ParserNode {
    this._eat(TokenTag.IF);
    const log = this._log();
    const block = this._block();
    return new IfStmtNode(log, block);
  }

  /**
   * _ass -> __ASS__ _callfn
   * **/
  private _ass(): ParserNode {
    if (this._lookahead?.tag === TokenTag.ASSIGN) {
      const ass = this._eat(TokenTag.ASSIGN);
      const callfn = this._log();
      this._symtable?.set(ass.value);
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
    const desc = this._descfn();

    this._symtable?.set(func.value);

    const tid = `FUNC-${Date.now()}`;
    this._symtable = new SymTable(tid, this._symtable);
    arity.params.forEach((param) => {
      this._symtable?.set(param);
    });
    const block = this._block();
    return new DeclFuncStmtNode(func.value, desc, arity, block);
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
   * _arg -> __CALL_ARG__ __PAREN_O__ _log __PAREN_C__
   * **/
  private _arg(): ParserNode {
    this._eat(TokenTag.CALL_ARG);
    this._eat(TokenTag.PAREN_O);
    const term = this._term();
    this._eat(TokenTag.PAREN_C);
    return new CallArgStmtNode(term);
  }

  /**
   * _arg -> __CALL_ARG__ __PAREN_O__ _log __PAREN_C__
   * **/
  private _concat(): ParserNode {
    this._eat(TokenTag.CALL_CONCAT);
    this._eat(TokenTag.PAREN_O);

    const params: ParserNode[] = [this._term()];

    // @ts-ignore
    while (this._lookahead.tag === TokenTag.COMMA) {
      this._eat(TokenTag.COMMA);
      params.push(this._term());
    }

    this._eat(TokenTag.PAREN_C);

    return new CallConcatStmtNode(params);
  }

  /**
   * _return -> __RETURN__ _log
   * **/
  private _return(): ParserNode {
    this._eat(TokenTag.RETURN);
    const log = this._log();
    return new ReturnStmtNode(log);
  }

  /**
   * _statment -> _block
   *            | _if
   *            | _ass
   *            | _declfunc
   *            | _print
   *            | _arg
   *            | _return
   *            | __RETURN_VOID__
   * **/
  private _statement(): ParserNode {
    if (this._lookahead?.tag === TokenTag.RETURN_VOID) {
      this._eat(TokenTag.RETURN_VOID);
      return new ReturnVoidStmtNode();
    }

    if (this._lookahead?.tag === TokenTag.RETURN) {
      return this._return();
    }

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

    return program;
  }

  public parse(): ProgramNode {
    this._lookahead = this._lexer.getNextToken();

    return this._program();
  }
}
