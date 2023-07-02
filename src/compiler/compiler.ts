import colorize from "json-colorizer";

import { Lexer } from "@/lexer";
import {
  BinaryOperationNode,
  BlockStatmentNode,
  DefStatmentNode,
  IfStatmentNode,
  IntegerNode,
  Parser,
  ParserNode,
} from "@/parser";

export default class Compiler {
  private readonly _lexer: Lexer;
  private readonly _parser: Parser;
  private _registers: string[] = [];
  private _counter: number = 1;

  constructor(buffer: Buffer = Buffer.from("")) {
    this._lexer = new Lexer(buffer);
    this._parser = new Parser(this._lexer /*Tokenizer*/);
  }

  private _build(stmt: ParserNode): string {
    if (stmt instanceof IntegerNode) return `${stmt.value}`;

    if (stmt instanceof DefStatmentNode) {
      const inst = `${stmt.name} = ${this._build(stmt.value)}`;
      this._registers.push(`${this._registers.length}: ${inst}`);
    }

    if (stmt instanceof BinaryOperationNode) {
      const reg = `t${this._counter}`;
      const inst = `${reg} = ${this._build(stmt.left)} ${
        stmt.operator.tag
      } ${this._build(stmt.right)}`;
      this._registers.push(`${this._registers.length}: ${inst}`);
      this._counter++;

      return reg;
    }

    if (stmt instanceof IfStatmentNode) {
      const test = this._build(stmt.test);
      const inst = `if-false ${test} goto ${
        this._registers.length + stmt.block.length + 1
      }`;
      this._registers.push(`${this._registers.length}: ${inst}`);
      for (const s of stmt.block) {
        this._build(s);
      }
    }

    if (stmt instanceof BlockStatmentNode) {
      for (const s of stmt.block) {
        this._build(s);
      }
    }

    return "";
  }

  public compile(debug?: boolean): string[] {
    const tree = this._parser.parse();
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));

    this._build(tree);

    return this._registers;
  }
}
