import { ParserNode } from "@/parser";

type SymTableData = ParserNode;

export default class SymTable {
  private _table: Map<string, SymTableData>;
  public refs: Map<string, number>;

  constructor(
    private readonly id: string,
    public readonly previous: SymTable | null = null
  ) {
    this._table = new Map();
    this.refs = new Map();
  }

  public set(key: string, node: ParserNode) {
    this._table.set(key, node);
    this.refs.set(key, 0);
  }

  public mergeRefs(ctx: SymTable | null) {
    if (ctx === null) return;
    this.refs = new Map([...this.refs, ...ctx.refs]);
  }

  public has(key: string) {
    let environ: SymTable | null = this;

    while (environ !== null) {
      const payload = environ._table.get(key);
      if (payload !== undefined) {
        const refs = this.refs.get(key) as number;
        this.refs.set(key, (refs || 0) + 1);
        return;
      }

      environ = environ.previous;
    }

    throw new SyntaxError(`Symbol "${key}" not found`);
  }

  public hasAnyRef() {
    let environ: SymTable | null = this;
    const noRefs: string[] = [];

    while (environ !== null) {
      const refs = Array.from(environ.refs, ([name, count]) => ({
        name,
        count,
      }));

      noRefs.push(
        ...refs.filter(({ count }) => count === 0).map(({ name }) => name)
      );

      environ = environ.previous;
    }

    if (noRefs.length > 0) {
      const [name] = noRefs;
      throw new SyntaxError(`"${name}" was declared but not referenced`);
    }
  }
}
