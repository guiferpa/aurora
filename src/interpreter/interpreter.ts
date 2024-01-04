import { Evaluator } from "./evaluator";
import { ParserNode } from "@/parser";
import Environment from "@/environ/environ";
import { ImportClaim } from "@/importer/importer";

export default class Interpreter {
  constructor(private _environ: Environment) {}

  public async run(
    tree: ParserNode,
    imports: Map<string, ImportClaim>,
    alias: Map<string, string>,
    args: string[] = []
  ): Promise<string[]> {
    const evaluator = new Evaluator(this._environ, imports, alias, args);
    return evaluator.evaluate(tree);
  }
}
