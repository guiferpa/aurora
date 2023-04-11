import {Lexer} from "./lexer";
import {Token, TokenIdentifier, TokenRecordList} from "./tokens";

export class Parser {
  private readonly _lexer: Lexer;
  private _lookahead: Token;

  constructor(lexer: Lexer) {
    this._lexer = lexer;
    this._lookahead = this._lexer.getNextToken();
  }

  /**
  * Grain handler
  * 
  * Grain =>
  * | NUMBER (Token)
  * | IDENT (Token)
  * ;
  */
  public _grain() {
    if (this._lookahead.id === TokenIdentifier.NUMBER) {
      return {
        type: "Grain",
        body: this._eat(TokenIdentifier.NUMBER, "_grain")
      }
    }

    return {
      type: "Grain",
      body: this._eat(TokenIdentifier.IDENT, "_grain")
    }
  }

  /**
  * Expression handler
  * 
  * Expr =>
  * | Grain
  * | Grain ADD (Token) Expr
  * | Grain SUB (Token) Expr
  * ;
  */
  public _expr() {
    const grain = this._grain();

    if (![
      TokenIdentifier.ADD, 
      TokenIdentifier.SUB,
      TokenIdentifier.MULT
    ].includes(this._lookahead.id)) {
      return {
        type: "Expr",
        body: grain
      }
    }

    const operator = this._eat(this._lookahead.id);
    const expr: any = this._expr();

    return {
      type: "Expr",
      body: {
        operator,
        grain,
        expr
      }
    }
  }

  /**
  * Variable definition handler
  * 
  * VarDef =>
  * | DEF (Token) IDENT (Token) ASSIN (Token) Expr
  * ;
  */
  public _varDef() {
    this._eat(TokenIdentifier.DEF);
    const ident = this._eat(TokenIdentifier.IDENT);
    this._eat(TokenIdentifier.ASSIGN)
    const expr = this._expr();

    return {
      type: "VarDef",
      body: {
        ident,
        expr
      }
    }
  }

  /**
  * BlockStatement handler
  * 
  * Statement =>
  * | BEGIN_BLOCK (Token) Statement * FINISH_BLOCK (Token)
  * ;
  */
  public _blockStatement(): any {
    this._eat(TokenIdentifier.BEGIN_BLOCK, "_blockStatement");

    const blockStatement = [];

    while (this._lookahead.id !== TokenIdentifier.FINISH_BLOCK) {
      blockStatement.push(this._statement());
    }

    this._eat(TokenIdentifier.FINISH_BLOCK, "_blockStatement")

    return {
      type: "BlockStatement",
      body: blockStatement
    }
  }

  /**
  * Statement handler
  * 
  * Statement =>
  * | Expr SEMI (Token)
  * | VarDef SEMI (Token)
  * | BlockStatement
  * ;
  */
  public _statement() {
    if (this._lookahead.id === TokenIdentifier.BEGIN_BLOCK) {
      return this._blockStatement();
    }

    if (this._lookahead.id === TokenIdentifier.DEF) {
      const varDef = this._varDef();
      this._eat(TokenIdentifier.SEMI, "_statement");

      return varDef;
    }

    const expr = this._expr();
    this._eat(TokenIdentifier.SEMI, "_statement");

    return expr;
  }

  /**
  * Statement list handler
  * 
  * StatementList =>
  * | Statement *
  * ;
  */
  public _statementList() {
    const statementList = [this._statement()];

    while (this._lookahead.id !== TokenIdentifier.EOT) {
      statementList.push(this._statement());
    }

    return statementList;
  }

  /**
  * Program entry point
  * 
  * Program =>
  * | StatementList
  * ;
  */
  public _program() {
    return {
      type: "Program",
      body: this._statementList()
    }
  }


  public _eat(tokenId: number, from?: string): Token {
    const token = this._lookahead;
    if (from) console.log(`From: ${from}, Lookahead: ${JSON.stringify(this._lookahead)}`);

    if (token.id === TokenIdentifier.EOT)
      throw new SyntaxError(`Unexpected end of tokens, expected token: ${TokenRecordList[tokenId]}`);

    if (token.id !== tokenId)
      throw new SyntaxError(`Unexpected token: ${token.value}`);

    this._lookahead = this._lexer.getNextToken();

    return token;
  }

  public parse() {
    return this._program();
  }
}

