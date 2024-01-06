import { ParserNode, ArityStmtNode } from "@/parser";
import { EnvironError } from "./errors";

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

export type NodeClaim = ParserNode;

export type EnvironClaim = NodeClaim | VariableClaim | FunctionClaim;

export type EnvironScopeType = string;

export default class Environment {
  private _table: Map<EnvironScopeType, EnvironClaim> = new Map([]);

  constructor(
    public readonly scope: EnvironScopeType,
    public previous: Environment | null = null
  ) {}

  public set(key: string, claim: EnvironClaim) {
    this._table.set(key, claim);
  }

  public query(key: string): EnvironClaim {
    let environ: Environment | null = this;

    while (environ !== null) {
      const payload = environ._table.get(key);
      if (typeof payload !== "undefined") return payload;

      environ = environ.previous;
    }

    throw new EnvironError(
      `Definition "${key}" not found at ${this.scope} scope`
    );
  }

  public getvar(key: string): VariableClaim | FunctionClaim | null {
    const claim = this.query(key);
    if (claim instanceof VariableClaim) return claim.value;
    if (claim instanceof FunctionClaim) return claim;
    return null;
  }

  public getfunc(key: string): FunctionClaim | null {
    const claim = this.query(key);
    return claim instanceof FunctionClaim ? claim : null;
  }
}
