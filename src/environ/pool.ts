import Environment, { EnvironClaim, EnvironScopeType } from "./environ";

export type EnvironContextType = string;

export default class Pool {
  private readonly _environs: Map<EnvironContextType, Environment> = new Map();
  private readonly _history: EnvironContextType[] = [];
  private _ctx: EnvironContextType = "";

  constructor() {}

  public add(context: EnvironContextType) {
    const environ = new Environment(`CONTEXT[${context}]-${Date.now()}`);
    this._environs.set(context, environ);
    this.change(context);
  }

  public change(context: EnvironContextType) {
    this._ctx = context;
  }

  public push(context: EnvironContextType) {
    this._history.push(this._ctx);
    if (!this._environs.has(context)) {
      this.add(context);
      return;
    }
    this.change(context);
  }

  public pop() {
    const context = this._history.pop();
    if (typeof context === "undefined") return;
    this.change(context);
  }

  public ahead(scope: EnvironScopeType, previous: Environment | null) {
    this._environs.set(this._ctx, new Environment(scope, previous));
  }

  public back() {
    const previous = this.environ().previous;
    if (previous === null) throw new Error(`Recursive environment overflow`);
    this._environs.set(this._ctx, previous);
  }

  public environ(): Environment {
    const environ = this._environs.get(this._ctx);
    if (typeof environ === "undefined")
      throw new Error(`Environ at context ${this._ctx} doesn't exist`);
    return environ;
  }

  public set(key: string, claim: EnvironClaim) {
    return this.environ().set(`${this._ctx}#${key}`, claim);
  }

  public query(key: string): EnvironClaim {
    return this.environ().query(`${this._ctx}#${key}`);
  }

  public context(): EnvironContextType {
    return this._ctx;
  }
}
