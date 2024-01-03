import SymTable from "@/symtable";
import { Token, TokenTag } from "@/lexer";
import Eater from "@/eater";

import * as node from "./node";
import { ParserError } from "./errors";

export interface Reader {
  read(entry: string): Promise<Buffer>;
}

export default class Parser {
  constructor(
    private _eater: Eater,
    private _symtable: SymTable | null = null
  ) {}

  /**
   * _fact -> __PAREN_O__ _log __PAREN_C__
   *        | __IDENT__
   *        | __LOG__
   *        | __NUM__
   *        | __STR__
   * **/
  private async _fact(): Promise<node.ParserNode> {
    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.PAREN_O) {
      return this._pre();
    }

    if (lookahead.tag === TokenTag.CALL_ARG) {
      return this._arg();
    }

    if (lookahead.tag === TokenTag.CALL_CONCAT) {
      return this._concat();
    }

    if (lookahead.tag === TokenTag.CALL_MAP) {
      return this._map();
    }

    if (lookahead.tag === TokenTag.CALL_FILTER) {
      return this._filter();
    }

    if (lookahead.tag === TokenTag.CALL_STR_TO_NUM) {
      return await this._call_str_to_num();
    }

    if (lookahead.tag === TokenTag.IDENT) {
      return this._call();
    }

    if (lookahead.tag === TokenTag.LOG) {
      return this._bool();
    }

    if (lookahead.tag === TokenTag.STR) {
      return this._str();
    }

    if (lookahead.tag === TokenTag.NUM) {
      return this._num();
    }

    if (lookahead.tag === TokenTag.S_BRACK_O) {
      return this._arr();
    }

    if (lookahead.tag === TokenTag.FROM) {
      return this._import();
    }

    throw new ParserError(`Unknown token ${lookahead.value}`);
  }

  private _from(): node.FromStmtNode {
    this._eater.eat(TokenTag.FROM);
    const str = this._str();
    return new node.FromStmtNode(str.value);
  }

  private _as(): node.AsStmtNode {
    this._eater.eat(TokenTag.AS);
    const id = this._eater.eat(TokenTag.IDENT);
    return new node.AsStmtNode(id.value);
  }

  private _import(): node.ImportStmtNode {
    const from = this._from();

    let alias = new node.AsStmtNode("");
    if (this._eater.lookahead().tag === TokenTag.AS) {
      alias = this._as();
    }

    return new node.ImportStmtNode(from, alias);
  }

  private async _call_str_to_num(): Promise<node.CallStrToNumStmtNode> {
    this._eater.eat(TokenTag.CALL_STR_TO_NUM);
    this._eater.eat(TokenTag.PAREN_O);
    const str = await this._fact();
    this._eater.eat(TokenTag.PAREN_C);
    return new node.CallStrToNumStmtNode(str);
  }

  // __CALL_MAP__
  private async _map(): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.CALL_MAP);
    this._eater.eat(TokenTag.PAREN_O);
    const param = await this._fact();
    this._eater.eat(TokenTag.COMMA);
    const fact = await this._fact();
    this._eater.eat(TokenTag.PAREN_C);
    return new node.CallMapStmtNode(param, fact);
  }

  // __CALL_FILTER__
  private async _filter(): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.CALL_FILTER);
    this._eater.eat(TokenTag.PAREN_O);
    const param = await this._fact();
    this._eater.eat(TokenTag.COMMA);
    const fact = await this._fact();
    this._eater.eat(TokenTag.PAREN_C);
    return new node.CallFilterStmtNode(param, fact);
  }

  // __PAREN_O__ __PAREN_C__
  private async _pre(): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.PAREN_O);
    const log = await this._log();
    this._eater.eat(TokenTag.PAREN_C);
    return log;
  }

  // __S_BREAK_O__ __S_BREAK_C__
  private async _arr(): Promise<node.ArrayNode> {
    this._eater.eat(TokenTag.S_BRACK_O);

    if (this._eater.lookahead().tag === TokenTag.S_BRACK_C) {
      this._eater.eat(TokenTag.S_BRACK_C);
      return new node.ArrayNode([]);
    }

    const items = [await this._log()];

    while (this._eater.lookahead().tag === TokenTag.COMMA) {
      this._eater.eat(TokenTag.COMMA);
      items.push(await this._log());
    }

    this._eater.eat(TokenTag.S_BRACK_C);

    return new node.ArrayNode(items);
  }

  // __LOG__
  private _bool(): node.ParserNode {
    const log = this._eater.eat(TokenTag.LOG);
    return new node.LogicalNode(log.value === "true");
  }

  // __NUM__
  private _num(): node.ParserNode {
    const { value } = this._eater.eat(TokenTag.NUM);
    const num = Number.parseInt(value);
    if (Number.isNaN(num))
      throw new ParserError(`Value ${value} is not a number`);
    return new node.NumericalNode(num);
  }

  // __STR__
  private _str(): node.StringNode {
    const str = this._eater.eat(TokenTag.STR);
    return new node.StringNode(str.value);
  }

  // __IDENT__ or __CALL_FUNC__
  private async _call(): Promise<node.ParserNode> {
    const ident = this._eater.eat(TokenTag.IDENT);

    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.DOT) {
      this._eater.eat(TokenTag.DOT);
      const prop = await this._call();
      return new node.AccessContextStatementNode(ident.value, prop);
    }

    if (lookahead.tag === TokenTag.PAREN_O) {
      return await this._callfn(ident);
    }

    return new node.IdentNode(ident.value);
  }

  /**
   * _uny -> __OP_ADD__ _uny
   *        | __OP_SUB__ _uny
   *        | _fact
   * **/
  private async _uny(): Promise<node.ParserNode> {
    const lookahead = this._eater.lookahead();

    if (
      lookahead.tag === TokenTag.OP_ADD ||
      lookahead.tag === TokenTag.OP_SUB
    ) {
      const op = this._eater.eat(lookahead.tag);
      const uny = await this._uny();
      return new node.UnaryOpNode(uny, op);
    }

    return await this._fact();
  }

  /**
   * _term -> _uny __OP_MUL__ _term
   *        | _uny __OP_DIV__ _term
   *        | _uny
   * **/
  private async _term(): Promise<node.ParserNode> {
    const uny = await this._uny();

    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.OP_MUL) {
      const op = this._eater.eat(TokenTag.OP_MUL);
      const term = await this._term();
      return new node.BinaryOpNode(uny, term, op);
    }

    if (lookahead.tag === TokenTag.OP_DIV) {
      const op = this._eater.eat(TokenTag.OP_DIV);
      const term = await this._term();
      return new node.BinaryOpNode(uny, term, op);
    }

    return uny;
  }

  /**
   * _expr -> _term __OP_ADD__ _expr
   *        | _term __OP_SUB__ _expr
   *        | _term
   * **/
  private async _expr(): Promise<node.ParserNode> {
    const term = await this._term();

    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.OP_ADD) {
      const op = this._eater.eat(TokenTag.OP_ADD);
      const expr = await this._expr();
      return new node.BinaryOpNode(term, expr, op);
    }

    if (lookahead.tag === TokenTag.OP_SUB) {
      const op = this._eater.eat(TokenTag.OP_SUB);
      const expr = await this._expr();
      return new node.BinaryOpNode(term, expr, op);
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
  private async _rel(): Promise<node.ParserNode> {
    const expr = await this._expr();

    const lookahead = this._eater.lookahead();

    if (
      lookahead.tag === TokenTag.REL_GT ||
      lookahead.tag === TokenTag.REL_LT ||
      lookahead.tag === TokenTag.REL_EQ ||
      lookahead.tag === TokenTag.REL_DIF
    ) {
      const op = this._eater.eat(lookahead.tag);
      const rel = await this._rel();
      return new node.RelativeExprNode(expr, rel, op);
    }

    return expr;
  }

  /**
   * _neg -> __NEG__ _rel
   *        | _rel
   * **/
  private async _neg(): Promise<node.ParserNode> {
    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.NEG) {
      this._eater.eat(TokenTag.NEG);
      const rel = await this._rel();
      return new node.NegativeExprNode(rel);
    }

    return await this._rel();
  }

  /**
   * _log -> _neg __LOG_AND__ _log
   *         | _neg __LOG_OR__ _log
   *         | _neg
   * **/
  private async _log(): Promise<node.ParserNode> {
    const neg = await this._neg();

    const lookahead = this._eater.lookahead();

    if (
      lookahead.tag === TokenTag.LOG_AND ||
      lookahead.tag === TokenTag.LOG_OR
    ) {
      const op = this._eater.eat(lookahead.tag);
      const log = await this._log();
      return new node.LogicExprNode(neg, log, op);
    }

    return neg;
  }

  /**
   * _descfn -> __DESC_FUNC__ __STR__
   *         | NULL
   * **/
  private _descfn(): node.DescFuncStmtNode | null {
    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.DESC_FUNC) {
      this._eater.eat(TokenTag.DESC_FUNC);
      const str = this._eater.eat(TokenTag.STR);
      return new node.DescFuncStmtNode(str.value);
    }

    return null;
  }

  /**
   * _callfn -> __IDENT__ __PAREN_O__ _log (__COMMA__ _log)* __PAREN_C__
   *         | __IDENT__ __PAREN_O__ __PAREN_C__
   *         | __IDENT__
   *         | _log
   * **/
  private async _callfn(id: Token): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.PAREN_O);

    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.PAREN_C) {
      this._eater.eat(TokenTag.PAREN_C);
      return new node.CallFuncStmtNode(id.value, []);
    }

    const params: node.ParserNode[] = [await this._log()];

    while (this._eater.lookahead().tag === TokenTag.COMMA) {
      this._eater.eat(TokenTag.COMMA);
      params.push(await this._log());
    }

    this._eater.eat(TokenTag.PAREN_C);

    return new node.CallFuncStmtNode(id.value, params);
  }

  /**
   * _param -> __IDENT__
   *         | NULL
   * **/
  private _param(): string {
    const id = this._eater.eat(TokenTag.IDENT);
    return id.value;
  }

  /**
   * _arity -> (_param __COMMA__ _param)*
   *         | _param
   *         | NULL
   * **/
  private _arity(): node.ArityStmtNode {
    if (this._eater.lookahead().tag !== TokenTag.IDENT) {
      return new node.ArityStmtNode([]);
    }

    const params: string[] = [this._param()];

    // @ts-ignore
    while (this._eater.lookahead().tag === TokenTag.COMMA) {
      this._eater.eat(TokenTag.COMMA);
      params.push(this._param());
    }

    return new node.ArityStmtNode(params);
  }

  /**
   * _block -> __BRACK_O__ _statements __BRACK_C__
   *         | _callfn
   * **/
  private async _block(): Promise<node.ParserNode> {
    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.BRACK_O) {
      const envId = `BLOCK-${Date.now()}`;
      this._symtable = new SymTable(envId, this._symtable);

      this._eater.eat(TokenTag.BRACK_O);
      const statements = await this._statements(TokenTag.BRACK_C);
      this._eater.eat(TokenTag.BRACK_C);

      this._symtable.previous?.mergeRefs(this._symtable);
      this._symtable = this._symtable.previous;

      return new node.BlockStmtNode(statements);
    }

    return await this._log();
  }

  /**
   * _if -> __IF__ _callfn _block
   * **/
  private async _if(): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.IF);
    const log = await this._log();
    const block = await this._block();
    return new node.IfStmtNode(log, block);
  }

  /**
   * _ass -> __ASS__ _callfn
   * **/
  private async _ass(): Promise<node.ParserNode> {
    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.ASSIGN) {
      const ass = this._eater.eat(TokenTag.ASSIGN);
      const callfn = await this._log();
      this._symtable?.set(ass.value);
      return new node.AssignStmtNode(ass.value, callfn);
    }

    return this._log();
  }

  /**
   * _declfunc -> __DECL_FN__ __PAREN_O__ _arity __PAREN_C__ _block
   * **/
  private async _declfunc(): Promise<node.ParserNode> {
    const func = this._eater.eat(TokenTag.DECL_FN);
    this._eater.eat(TokenTag.PAREN_O);
    const arity = this._arity();
    this._eater.eat(TokenTag.PAREN_C);
    const desc = this._descfn();

    this._symtable?.set(func.value);

    const tid = `FUNC-${Date.now()}`;
    this._symtable = new SymTable(tid, this._symtable);
    arity.params.forEach((param) => {
      this._symtable?.set(param);
    });
    const block = await this._block();
    return new node.DeclFuncStmtNode(func.value, desc, arity, block);
  }

  /**
   * _print -> __CALL_PRINT__ __PAREN_O__ _log __PAREN_C__
   * **/
  private async _print(): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.CALL_PRINT);
    this._eater.eat(TokenTag.PAREN_O);
    const log = await this._log();
    this._eater.eat(TokenTag.PAREN_C);
    return new node.CallPrintStmtNode(log);
  }

  /**
   * _arg -> __CALL_ARG__ __PAREN_O__ _log __PAREN_C__
   * **/
  private async _arg(): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.CALL_ARG);
    this._eater.eat(TokenTag.PAREN_O);
    const term = await this._term();
    this._eater.eat(TokenTag.PAREN_C);
    return new node.CallArgStmtNode(term);
  }

  /**
   * _arg -> __CALL_ARG__ __PAREN_O__ _log __PAREN_C__
   * **/
  private async _concat(): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.CALL_CONCAT);
    this._eater.eat(TokenTag.PAREN_O);

    const params: node.ParserNode[] = [await this._term()];

    while (this._eater.lookahead().tag === TokenTag.COMMA) {
      this._eater.eat(TokenTag.COMMA);
      params.push(await this._term());
    }

    this._eater.eat(TokenTag.PAREN_C);

    return new node.CallConcatStmtNode(params);
  }

  /**
   * _return -> __RETURN__ _log
   * **/
  private async _return(): Promise<node.ParserNode> {
    this._eater.eat(TokenTag.RETURN);
    const log = await this._log();
    return new node.ReturnStmtNode(log);
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
  private async _statement(): Promise<node.ParserNode> {
    const lookahead = this._eater.lookahead();

    if (lookahead.tag === TokenTag.RETURN_VOID) {
      this._eater.eat(TokenTag.RETURN_VOID);
      return new node.ReturnVoidStmtNode();
    }

    if (lookahead.tag === TokenTag.RETURN) {
      return this._return();
    }

    if (lookahead.tag === TokenTag.CALL_PRINT) {
      return this._print();
    }

    if (lookahead.tag === TokenTag.DECL_FN) {
      return this._declfunc();
    }

    if (lookahead.tag === TokenTag.ASSIGN) {
      return this._ass();
    }

    if (lookahead.tag === TokenTag.IF) {
      return this._if();
    }

    return this._block();
  }

  private async _statements(eot: TokenTag): Promise<node.ParserNode[]> {
    const list = [];

    while (this._eater.lookahead().tag !== eot) {
      list.push(await this._statement());
    }

    return list;
  }

  private async _program(): Promise<node.ProgramNode> {
    const program = new node.ProgramNode(await this._statements(TokenTag.EOF));
    return program;
  }

  public async parse(): Promise<node.ProgramNode> {
    const program = await this._program();
    return program;
  }
}
