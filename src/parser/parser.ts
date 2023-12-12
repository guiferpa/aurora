import { TokenTag } from "@/lexer/tokens/tag";
import { Token } from "@/lexer/tokens/token";
import {
  BinaryOpNode,
  DeclNode,
  IdentNode,
  NumericNode,
  ParserNode,
  StatementNode,
  ProgramNode,
  BlockStatement,
} from "./node";

import Environment from "@/environ";

interface Lexer {
  getNextToken(): Token | null;
}

export default class Parser {
  private _lookahead: Token | null = null;

  constructor(
    private readonly _lexer: Lexer,
    private readonly _environ: Environment
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
   * _fact -> __PAREN_O__ _expr __PAREN_C__
   *        | __IDENT__
   *        | __NUM__
   * **/
  private _fact(): ParserNode {
    if (this._lookahead?.tag === TokenTag.PAREN_O) {
      this._eat(TokenTag.PAREN_O);
      const expr = this._expr();
      this._eat(TokenTag.PAREN_C);
      return expr;
    }

    if (this._lookahead?.tag === TokenTag.IDENT) {
      const ident = this._eat(TokenTag.IDENT);
      return new IdentNode(ident.value);
    }

    const { value } = this._eat(TokenTag.NUM);
    const num = Number.parseInt(value);
    if (Number.isNaN(num)) throw new Error(`Value ${value} is not a number`);
    return new NumericNode(num);
  }

  /**
   * _term -> _fact * _term
   *        | _fact / _term
   *        | _fact
   * **/
  private _term(): ParserNode {
    const fact = this._fact();

    if (this._lookahead?.tag === TokenTag.OP_MUL) {
      const op = this._eat(TokenTag.OP_MUL);
      return new BinaryOpNode(fact, this._term(), op);
    }

    if (this._lookahead?.tag === TokenTag.OP_DIV) {
      const op = this._eat(TokenTag.OP_DIV);
      return new BinaryOpNode(fact, this._term(), op);
    }

    return fact;
  }

  /**
   * _expr -> _term + _expr
   *        | _term - _expr
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
   * _decl -> __DECL__ _expr
   *        | _expr
   * **/
  private _decl(): ParserNode {
    if (this._lookahead?.tag === TokenTag.DECL) {
      const decl = this._eat(TokenTag.DECL);
      const expr = this._expr();
      return new DeclNode(decl.value, expr);
    }

    return this._expr();
  }

  /**
   * _block -> __BRACK_O__ _statements __BRACK_C__
   *         | _expr
   * **/
  private _block(): ParserNode {
    if (this._lookahead?.tag === TokenTag.BRACK_O) {
      this._eat(TokenTag.BRACK_O);
      const statements = this._statements(TokenTag.BRACK_C);
      this._eat(TokenTag.BRACK_C);
      return new BlockStatement(statements);
    }

    return this._expr();
  }

  /**
   * _statment -> _block
   *            | _decl
   * **/
  private _statement(): ParserNode {
    if (this._lookahead?.tag === TokenTag.DECL) {
      return new StatementNode(this._decl());
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
    return new ProgramNode(this._statements(TokenTag.EOF));
  }

  public parse(): ProgramNode {
    this._lookahead = this._lexer.getNextToken();

    return this._program();
  }
}
