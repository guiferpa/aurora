import { Evaluator } from "./evaluator";
import { ParserNode } from "@/parser";
import { ImportClaim } from "@/importer/importer";
import { Pool } from "@/environ";

export default class Interpreter {
  constructor() {}

  public async run(
    context: string,
    tree: ParserNode,
    imports: Map<string, ImportClaim>,
    alias: Map<string, Map<string, string>>,
    args: string[] = []
  ): Promise<string[]> {
    const pool = new Pool();
    pool.add(context);
    const evaluator = new Evaluator(pool, imports, alias, args);
    return evaluator.evaluate(tree);
  }
}
