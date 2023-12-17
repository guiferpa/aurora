import {
  AssignStmtNode,
  BinaryOpNode,
  BlockStmtNode,
  IdentNode,
  NumericalNode,
  ParserNode,
  ProgramNode,
  RelativeExprNode,
} from "@/parser/node";

export default class Generator {
  private static _ops: string[] = [];
  private static _counter: number = 1;

  private static _lvalue(n: ParserNode): string {
    if (n instanceof IdentNode) {
      return n.name;
    }

    throw SyntaxError(`Invalid LValue: ${JSON.stringify(n)}`);
  }

  private static _temp(): string {
    const t = `_t${this._counter}`;
    this._counter++;
    return t;
  }

  private static _rvalue(n: ParserNode): string {
    if (n instanceof IdentNode) {
      return n.name;
    }

    if (n instanceof NumericalNode) {
      return n.value.toString();
    }

    if (n instanceof BinaryOpNode) {
      const left = this._rvalue(n.left);
      const right = this._rvalue(n.right);
      const temp = this._temp();
      this._ops.push(`${temp} = ${left} ${n.op.value} ${right}`);
      return temp;
    }

    if (n instanceof RelativeExprNode) {
      const left = this._rvalue(n.left);
      const right = this._rvalue(n.right);
      const temp = this._temp();
      this._ops.push(`${temp} = ${left} ${n.op.value} ${right}`);
      return temp;
    }

    throw SyntaxError(`Invalid RValue: ${JSON.stringify(n)}`);
  }

  private static _block(n: BlockStmtNode): void {}

  private static _ass(n: AssignStmtNode): void {
    const id = new IdentNode(n.name);
    const op = `${this._lvalue(id)} = ${this._rvalue(n.value)}`;
    this._ops.push(op);
  }

  public static run(program: ProgramNode): string[] {
    for (const child of program.children) {
      if (child instanceof AssignStmtNode) {
        this._ass(child);
        continue;
      }

      if (child instanceof BlockStmtNode) {
        this._block(child);
        continue;
      }
    }

    return this._ops;
  }
}
