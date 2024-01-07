import { ParserNode } from "@/parser";
import { ImportClaim } from "@/importer/importer";
import { Pool } from "@/environ";

import { Evaluator } from "./evaluator";

export default class Interpreter {
  constructor(private readonly _pool: Pool) {}

  public async run(
    tree: ParserNode,
    imports: Map<string, ImportClaim>,
    alias: Map<string, Map<string, string>>,
    args: string[] = []
  ): Promise<string[]> {
    const evaluator = new Evaluator(this._pool, imports, alias, args);
    return evaluator.evaluate(tree);
  }
}
