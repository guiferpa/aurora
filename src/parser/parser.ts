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
   * _ass -> __ASS__ _log
   *        | _log
   * **/
  private _ass(): ParserNode {
    if (this._lookahead?.tag === TokenTag.ASSIGN) {
      const ass = this._eat(TokenTag.ASSIGN);
      const expr = this._log();
      this._symtable?.set(ass.value, expr);
      return new AssignStmtNode(ass.value, expr);
    }

    return this._log();
  }

  /**
   * _block -> __BRACK_O__ _statements __BRACK_C__
   *         | _log
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
   * _statment -> _block
   *            | _ass
   * **/
  private _statement(): ParserNode {
    if (this._lookahead?.tag === TokenTag.ASSIGN) {
      return this._ass();
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
