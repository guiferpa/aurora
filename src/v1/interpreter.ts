import {Parser} from "../v3";
import Evaluator from "./evaluator";
import Lexer from "./lexer";

export default class Interpreter {
  private readonly _buffer: Buffer;

  constructor (buffer: Buffer) {
    this._buffer = buffer;
  }

  public run(debug?: boolean): string[] {
    const lexer = new Lexer(this._buffer); // Tokenizer
    const parser = new Parser(lexer);
    const tree = parser.parse(debug);
    return Evaluator.compose(tree.block);
  }
}
