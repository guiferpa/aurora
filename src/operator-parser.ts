import Lexer from "./lexer";
import {Token, TokenIdentifier} from "./tokens";

export default class OperatorParser {
  private readonly _lexer: Lexer;
  private readonly _precedenceTable: Map<string, number>;

  constructor(lexer: Lexer) {
    this._lexer = lexer;
    this._precedenceTable = new Map([
      ["$", 0],
      ["ADD", 1],
      ["SUB", 1],
      ["MULT", 2],
      ["NUMBER", 3],
      ["IDENT", 3]
    ]);
  }

  public parse(): Token[] {
    const stack: (Token | null)[] = [null]; // Assign initial symbol
    const tree: Token[] = [];
    let lookahead = this._lexer.getNextToken();
    
    while (lookahead.id !== TokenIdentifier.EOT) {
      const item = stack[stack.length - 1];

      if (item === null) {
        stack.push(lookahead);
        lookahead = this._lexer.getNextToken();
        continue;
      }

      const itemPrecedence = this._precedenceTable.get(item.type);
      const lookaheadPrecedence = this._precedenceTable.get(lookahead.type);

      if (!itemPrecedence || !lookaheadPrecedence) {
        lookahead = this._lexer.getNextToken();
        continue;
      }

      if (lookaheadPrecedence > itemPrecedence) {
        stack.push(lookahead);
        lookahead = this._lexer.getNextToken();
        continue;
      }

      const token = stack.pop();
      if (!token) throw new Error(`Undefined token popped from stack`);

      tree.push(token);
    }

    for (let index = stack.length - 1; index >= 0; index--) {
      const item = stack[index]; // Removes null item
      if (item) tree.push(item);
    }

    return tree;
  }
}
