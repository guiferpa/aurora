import { TokenTag } from "@/lexer/tokens/tag";
import { ParserNode } from "@/parser";
import {
  BinaryOpNode,
  DeclNode,
  IdentNode,
  NumericNode,
  StatementNode,
  ProgramNode,
} from "@/parser/node";

export default class Evaluator {
  static compose(nodes: ParserNode[]): string[] {
    const out = [];

    for (const n of nodes) {
      out.push(`${Evaluator.evaluate(n)}`);
    }

    return out;
  }

  static evaluate(tree: ParserNode): any {
    if (tree instanceof ProgramNode) return Evaluator.compose(tree.children);

    if (tree instanceof DeclNode || tree instanceof IdentNode) return undefined;

    if (tree instanceof StatementNode) {
      return Evaluator.evaluate(tree.value);
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

    throw new Error(
      `Unsupported evaluate expression for [${JSON.stringify(tree)}]`
    );
  }
}
