import colorize from "json-colorizer";

import Lexer from "@/lexer";
import SymTable from "@/symtable";
import Parser from "@/parser";

import Generator from "./generator";
import Eater from "@/eater/eater";

export default class Builder {
  constructor(private readonly _buffer: Buffer = Buffer.from("")) {}

  public async run(debug?: boolean): Promise<string[]> {
    const lexer = new Lexer(this._buffer);
    const eater = new Eater(lexer.copy());
    const parser = new Parser(eater, new SymTable("global"));
    const tree = await parser.parse();
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));
    return Generator.run(tree);
  }
}
