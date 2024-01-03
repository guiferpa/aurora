import colorize from "json-colorizer";

import { Evaluator } from "./evaluator";
import { ParserNode } from "@/parser";
import Environment from "@/environ/environ";
import { ImportClaim } from "@/importer/importer";

export default class Interpreter {
  constructor(private _environ: Environment) {}

  public async run(
    tree: ParserNode,
    imports: Map<string, ImportClaim>,
    debug?: boolean,
    args: string[] = []
  ): Promise<string[]> {
    debug && console.log(colorize(JSON.stringify(tree, null, 2)));
    const evaluator = new Evaluator(this._environ, imports, args);
    return evaluator.evaluate(tree);
  }
}
