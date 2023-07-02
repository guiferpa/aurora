import { TokenTag } from "@/tokens";
import {
  BinaryOperationNode,
  BlockStatmentNode,
  DefStatmentNode,
  IfStatmentNode,
  IntegerNode,
  LogicalNode,
  ParserNode,
  PrintCallStatmentNode,
  UnaryOperationNode,
  DefFunctionStatmentNode,
  StringNode,
} from "@/parser";

export default class Evaluator {
  static compose(block: ParserNode[]): string[] {
    const out = [];

    for (const stmt of block) {
      if (
        stmt instanceof DefStatmentNode ||
        stmt instanceof DefFunctionStatmentNode
      ) {
        continue;
      }

      if (stmt instanceof IfStatmentNode) {
        Evaluator.evaluate(stmt.test) &&
          out.push(Evaluator.compose(stmt.block).join(","));
        continue;
      }

      if (stmt instanceof PrintCallStatmentNode) {
        console.log(Evaluator.evaluate(stmt.param));
        continue;
      }

      if (stmt instanceof BlockStatmentNode) {
        out.push(Evaluator.compose(stmt.block).join(","));
        continue;
      }

      out.push(`${Evaluator.evaluate(stmt)}`);
    }

    return out;
  }

  static evaluate(tree: ParserNode): any {
    if (tree instanceof BlockStatmentNode) return Evaluator.compose(tree.block);

    if (tree instanceof IntegerNode) return tree.value;

    if (tree instanceof LogicalNode) return tree.value;

    if (tree instanceof StringNode) return tree.value;

    if (tree instanceof UnaryOperationNode) {
      const { operator, expr } = tree;

      switch (operator.tag) {
        case TokenTag.OPP:
          return !this.evaluate(expr);
      }
    }

    if (tree instanceof BinaryOperationNode) {
      const { operator, left, right } = tree;

      switch (operator.tag) {
        case TokenTag.AND:
          return this.evaluate(left) && this.evaluate(right);

        case TokenTag.OR:
          return this.evaluate(left) || this.evaluate(right);

        case TokenTag.EQUAL:
          return this.evaluate(left) === this.evaluate(right);

        case TokenTag.GREATER_THAN:
          return this.evaluate(left) > this.evaluate(right);

        case TokenTag.LESS_THAN:
          return this.evaluate(left) < this.evaluate(right);

        case TokenTag.ADD:
          return this.evaluate(left) + this.evaluate(right);

        case TokenTag.SUB:
          return this.evaluate(left) - this.evaluate(right);

        case TokenTag.MULT:
          return this.evaluate(left) * this.evaluate(right);
      }
    }

    throw new Error(`Unsupported evaluate expression`);
  }
}
