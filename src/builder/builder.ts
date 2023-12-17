import colorize from "json-colorizer";

import { Lexer } from "@/lexer";
import { Parser } from "@/parser";

import Generator from "./generator";
import SymTable from "@/symtable/symtable";

export default class Builder {
  constructor(private readonly _buffer: Buffer = Buffer.from("")) {}

  public run(debug?: boolean): string[] {
    const lexer = new Lexer(this._buffer);
    const parser = new Parser(lexer, new SymTable("root"));
    const tree = parser.parse();
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));
    return Generator.run(tree);
  }
}
