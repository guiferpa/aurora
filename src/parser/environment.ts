import { ParserNode } from "@/parser";

export const FuncParameterType = "__FUNC_PARAM__";

export default class Environment {
  public readonly id: string;
  private _table: Map<string, ParserNode | string>;
  public readonly prev: Environment | null;

  constructor(id: string, prev: Environment | null = null) {
    this.id = id;
    this._table = new Map();
    this.prev = prev;
  }

  public set(key: string, payload: ParserNode | string) {
    this._table.set(key, payload);
  }

  public query(key: string): ParserNode | string {
    let environ: Environment | null = this;

    while (environ !== null) {
      const payload = environ._table.get(key);
      if (payload !== undefined) return payload;

      environ = environ.prev;
    }

    throw new SyntaxError(`Definition "${key}" not found`);
  }

  public describe() {
    let stmt: Environment | null = this;

    while (stmt !== null) {
      console.log(stmt);
      stmt = stmt.prev;
    }
  }
}
