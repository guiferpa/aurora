import Environment from "@/environ/environ";
import { TokenTag } from "@/lexer/tokens/tag";
import { ParserNode } from "@/parser";
import {
  BinaryOpNode,
  DeclNode,
  IdentNode,
  NumericNode,
  StatementNode,
  ProgramNode,
  BlockStatement,
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

  public evaluate(tree: ParserNode): any {
    if (tree instanceof ProgramNode) return this.compose(tree.children);

    if (tree instanceof BlockStatement) return this.compose(tree.children);

    if (tree instanceof DeclNode || tree instanceof IdentNode) return "";

    if (tree instanceof StatementNode) {
      return this.evaluate(tree.value);
    }

    if (tree instanceof NumericNode) return tree.value;

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
