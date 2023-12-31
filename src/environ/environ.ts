import { ParserNode, ArityStmtNode } from "@/parser";

export const FuncParameterType = "__FUNC_PARAM__";

export class VariableClaim {
  constructor(public readonly value: any) {}
}

export class FunctionClaim {
  constructor(
    public readonly arity: ArityStmtNode,
    public readonly body: ParserNode
  ) {}
}

export type Payload = ParserNode | VariableClaim | FunctionClaim;

export default class Environment {
  public readonly id: string;
  private _table: Map<string, Payload>;
  public readonly prev: Environment | null;

  constructor(id: string, prev: Environment | null = null) {
    this.id = id;
    this._table = new Map();
    this.prev = prev;
  }

  public set(key: string, payload: Payload) {
    this._table.set(key, payload);
  }

  public query(key: string): Payload {
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
