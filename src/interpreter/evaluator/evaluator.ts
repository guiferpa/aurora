import Environment from "@/environ/environ";
import { TokenTag } from "@/lexer/tokens/tag";
import { ParserNode } from "@/parser";
import {
  BinaryOpNode,
  AssignStmtNode,
  IdentNode,
  NumericalNode,
  ProgramNode,
  BlockStmtNode,
  LogicalNode,
  NegativeExprNode,
  RelativeExprNode,
  LogicExprNode,
  UnaryOpNode,
  StatementNode,
} from "@/parser/node";

export default class Evaluator {
  constructor(private readonly _environ: Environment) {}

  private compose(nodes: ParserNode[]): string[] {
    const out = [];

    for (const n of nodes) {
      out.push(`${this.evaluate(n)}`);
    }

    return out;
  }

  public evaluate(tree: ParserNode | StatementNode): any {
    if (tree instanceof ProgramNode) return this.compose(tree.children);

    if (tree instanceof BlockStmtNode) return this.compose(tree.children);

    if (tree instanceof AssignStmtNode || tree instanceof IdentNode) return "";

    if (tree instanceof NegativeExprNode) return !this.evaluate(tree.expr);

    if (tree instanceof NumericalNode) return tree.value;

    if (tree instanceof LogicalNode) return tree.value;

    if (tree instanceof UnaryOpNode) {
      const { op, right } = tree;

      switch (op.tag) {
        case TokenTag.OP_ADD:
          return +this.evaluate(right);

        case TokenTag.OP_SUB:
          return -this.evaluate(right);
      }
    }

    if (tree instanceof BinaryOpNode) {
      const { op, left, right } = tree;

      switch (op.tag) {
        case TokenTag.OP_ADD:
          return this.evaluate(left) + this.evaluate(right);

        case TokenTag.OP_SUB:
          return this.evaluate(left) - this.evaluate(right);

        case TokenTag.OP_DIV:
          return this.evaluate(left) / this.evaluate(right);

        case TokenTag.OP_MUL:
          return this.evaluate(left) * this.evaluate(right);
      }
    }

    if (tree instanceof RelativeExprNode) {
      const { op, left, right } = tree;

      switch (op.tag) {
        case TokenTag.REL_GT:
          return this.evaluate(left) > this.evaluate(right);

        case TokenTag.REL_LT:
          return this.evaluate(left) < this.evaluate(right);

        case TokenTag.REL_EQ:
          return this.evaluate(left) === this.evaluate(right);

        case TokenTag.REL_DIF:
          return this.evaluate(left) !== this.evaluate(right);
      }
    }

    if (tree instanceof LogicExprNode) {
      const { op, left, right } = tree;

      switch (op.tag) {
        case TokenTag.LOG_AND:
          return this.evaluate(left) && this.evaluate(right);

        case TokenTag.LOG_OR:
          return this.evaluate(left) || this.evaluate(right);
      }
    }

    if (tree instanceof IdentNode) {
      const node = this._environ.query(tree.name);
      if (typeof node === "string") return node;
      return this.evaluate(node);
    }

    throw new Error(
      `Unsupported evaluate expression for [${JSON.stringify(tree)}]`
    );
  }
}
