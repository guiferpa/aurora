import colorize from "json-colorizer";

import Lexer from "@/lexer";
import SymTable from "@/symtable";
import Parser from "@/parser";
import * as utils from "@/utils";

import Generator from "./generator";

export default class Builder {
  constructor(private readonly _buffer: Buffer = Buffer.from("")) {}

  public async run(debug?: boolean): Promise<string[]> {
    const lexer = new Lexer(this._buffer);
    const parser = new Parser({ read: utils.fs.read }, new SymTable("global"));
    const tree = await parser.parse(lexer);
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));
    return Generator.run(tree);
  }
}
