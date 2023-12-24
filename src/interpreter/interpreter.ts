import colorize from "json-colorizer";

import { Evaluator } from "./evaluator";
import { Lexer } from "@/lexer";
import { Parser } from "@/parser";
import Environment from "@/environ/environ";
import SymTable from "@/symtable";

export default class Interpreter {
  private _lexer: Lexer;
  private _parser: Parser;
  private _symtable: SymTable;
  private _environ: Environment;

  constructor(buffer: Buffer = Buffer.from("")) {
    this._environ = new Environment("global");
    this._symtable = new SymTable("global");
    this._lexer = new Lexer(buffer);
    this._parser = new Parser(this._lexer, this._symtable);
  }

  public write(buffer: Buffer) {
    this._lexer.write(buffer);
  }

  public run(debug?: boolean, args: string[] = []): string[] {
    const tree = this._parser.parse();
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));
    const evaluator = new Evaluator(this._environ, args);
    return evaluator.evaluate(tree);
  }
}
