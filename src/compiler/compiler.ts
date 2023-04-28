import colorize from "json-colorizer";

import {Lexer} from "@/lexer";
import {Parser} from "@/parser";

export default class Compiler {
  private readonly _lexer: Lexer;
  private readonly _parser: Parser;

  constructor (buffer: Buffer = Buffer.from("")) {
    this._lexer = new Lexer(buffer);
    this._parser = new Parser(this._lexer /*Tokenizer*/);
  }

  public compile(debug?: boolean): string[] {
    const tree = this._parser.parse();
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));
    return [];
  }
}
