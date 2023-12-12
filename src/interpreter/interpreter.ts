import colorize from "json-colorizer";

import { Evaluator } from "./evaluator";
import { Lexer } from "@/lexer";
import { Parser } from "@/parser";
import Environment from "@/environ/environ";

export default class Interpreter {
  private _lexer: Lexer;
  private _parser: Parser;
  private _enrivon: Environment;

  constructor(buffer: Buffer = Buffer.from("")) {
    this._enrivon = new Environment("root", null);
    this._lexer = new Lexer(buffer);
    this._parser = new Parser(this._lexer, this._enrivon);
  }

  public write(buffer: Buffer) {
    this._lexer.write(buffer);
  }

  public run(debug?: boolean): string[] {
    const tree = this._parser.parse();
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));
    return Evaluator.evaluate(tree);
  }
}
