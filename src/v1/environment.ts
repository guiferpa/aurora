import {ParserNode} from "../v3/parser/node";

export default class Environment {
  public readonly id: string;
  private _table: Map<string, ParserNode>; 
  public readonly prev: Environment | null;

  constructor (id: string, prev: Environment | null = null) {
    this.id = id;
    this._table = new Map();
    this.prev = prev;
  }

  public set(key: string, payload: ParserNode) {
    this._table.set(key, payload);
  }

  public query(key: string): ParserNode {
    let environ: Environment | null = this;

    while (environ !== null) {
      const payload = environ._table.get(key);
      if (payload !== undefined) return payload;

      environ = environ.prev;
    }

    throw new SyntaxError(`Symbol ${key} not found`);
  }

  public describe() {
    let stmt: Environment | null = this;

    while (stmt !== null) {
      console.log(stmt);
      stmt = stmt.prev;
    }
  }
}
