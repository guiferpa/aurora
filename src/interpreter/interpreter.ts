import { Evaluator } from "./evaluator";
import { ParserNode } from "@/parser";
import Environment from "@/environ/environ";
import { ImportClaim } from "@/importer/importer";
import { Pool } from "@/environ";

export default class Interpreter {
  constructor(private _environ: Environment) {}

  public async run(
    tree: ParserNode,
    imports: Map<string, ImportClaim>,
    alias: Map<string, Map<string, string>>,
    args: string[] = []
  ): Promise<string[]> {
    const pool = new Pool();
    pool.add("main");
    const evaluator = new Evaluator(pool, imports, alias, args);
    return evaluator.evaluate(tree);
  }
}
