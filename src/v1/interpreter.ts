import {Parser} from "../v3";
import Evaluator from "./evaluator";
import Lexer from "./lexer";

export default class Interpreter {
  private _lexer: Lexer;
  private _parser: Parser;

  constructor (buffer: Buffer = Buffer.from("")) {
    this._lexer = new Lexer(buffer);
    this._parser = new Parser(this._lexer /*Tokenizer*/);
  }

  public write(buffer: Buffer) {
    this._lexer.write(buffer);
  }

  public run(debug?: boolean): string[] {
    const tree = this._parser.parse(debug);
    return Evaluator.compose(tree.block);
  }
}
