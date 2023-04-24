import {TokenTag} from "./tokens";
import {
  BinaryOperationNode,
  BlockStatmentNode,
  IdentifierNode,
  IntegerNode,
  LogicalNode,
  ParserNode,
  UnaryOperationNode,
} from "../v3/parser/node";

export default class Evaluator {
  static compose(block: ParserNode[]): string[] {
    const out = [];

    for (const stmt of block) {
      if (stmt instanceof IdentifierNode) {
        continue;
      }

      if (stmt instanceof BlockStatmentNode) {
        out.push(Evaluator.compose(stmt.block).join(','));
        continue;
      }

      out.push(`${Evaluator.evaluate(stmt)}`);
    }

    return out;
  }

  static evaluate(tree: ParserNode): any {
    if (tree instanceof IntegerNode)
      return tree.value;

    if (tree instanceof LogicalNode)
      return tree.value;

    if (tree instanceof UnaryOperationNode) {
      const { operator, expr } = tree;

      switch (operator.tag) {
        case TokenTag.OPP:
          return !this.evaluate(expr);
      }
    }

    if (tree instanceof BinaryOperationNode) {
      const {operator, left, right} = tree;

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

    throw new Error(`Unsupported evalute expression`);
  }
}
