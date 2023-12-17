import {
  AssignStmtNode,
  BinaryOpNode,
  BlockStmtNode,
  IdentNode,
  IfStmtNode,
  NumericalNode,
  ParserNode,
  ProgramNode,
  RelativeExprNode,
} from "@/parser/node";

export default class Generator {
  private static _ops: string[] = [];
  private static _tempcounter: number = 1;
  private static _labelcounter: number = 1;

  private static _lvalue(n: ParserNode): string {
    if (n instanceof AssignStmtNode) {
      return n.name;
    }

    throw SyntaxError(`Invalid LValue: ${JSON.stringify(n)}`);
  }

  private static _temp(): string {
    const t = `_t${this._tempcounter}`;
    this._tempcounter++;
    return t;
  }

  private static _label(): string {
    const l = `_l${this._labelcounter}`;
    this._labelcounter++;
    return l;
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

  private static _compose(ns: ParserNode[]): void {
    for (const n of ns) {
      if (n instanceof AssignStmtNode) {
        this._ass(n);
        continue;
      }

      if (n instanceof BlockStmtNode) {
        this._block(n);
        continue;
      }

      if (n instanceof IfStmtNode) {
        this._if(n);
        continue;
      }
    }
  }

  private static _block(n: BlockStmtNode): void {
    this._compose(n.children);
  }

  private static _ass(n: AssignStmtNode): void {
    const op = `${this._lvalue(n)} = ${this._rvalue(n.value)}`;
    this._ops.push(op);
  }

  private static _if(n: IfStmtNode): void {
    const l = this._label();

    const test = this._rvalue(n.test);
    this._ops.push(`if-false ${test} goto ${l}`);
    if (!(n.body instanceof BlockStmtNode))
      throw SyntaxError(`IfStatement body must be a BlockStatement`);

    this._block(n.body);
    this._ops.push(`${l}:`);
  }

  public static run(program: ProgramNode): string[] {
    this._compose(program.children);
    return this._ops;
  }
}
