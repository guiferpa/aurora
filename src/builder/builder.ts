import colorize from "json-colorizer";

import { Lexer } from "@/lexer";
import { Parser } from "@/parser";

import Generator from "./generator";
import SymTable from "@/symtable/symtable";
import { read } from "@/fsutil";

export default class Builder {
  constructor(private readonly _buffer: Buffer = Buffer.from("")) {}

  public async run(debug?: boolean): Promise<string[]> {
    const lexer = new Lexer(this._buffer);
    const parser = new Parser({ read }, new SymTable("root"));
    const tree = await parser.parse(lexer);
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));
    return Generator.run(tree);
  }
}
