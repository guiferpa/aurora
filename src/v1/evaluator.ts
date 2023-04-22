import {TokenTag} from "./tokens";
import {
  BinaryOperationNode,
  BlockStatmentNode,
  IdentifierNode,
  IntegerNode,
  LogicalNode,
  NegativeOperationNode,
  ParserNode,
  RelativeOperationNode,
} from "../v3/parser/node";

export default class Evaluator {
  static compose(block: ParserNode[]): string[] {
    const out = [];

    for (const stmt of block) {
      if (stmt instanceof IdentifierNode) {
        continue;
      }

      if (stmt instanceof RelativeOperationNode) {
        out.push(`${Evaluator.relative(stmt)}`);
        continue;
      }

      if (stmt instanceof NegativeOperationNode) {
        const { expr } = stmt;

        if (expr instanceof LogicalNode) {
          out.push(`${!expr.value}`);
          continue;
        }

        out.push(`${!Evaluator.relative(expr)}`);
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

  static relative(tree: RelativeOperationNode): boolean {
    const {comparator, left, right} = tree;

    const a = left instanceof LogicalNode ? left.value : Evaluator.evaluate(left);
    const b = right instanceof LogicalNode ? right.value : Evaluator.evaluate(right);

    switch (comparator.tag) {
      case TokenTag.EQUAL:
        return a === b;

      case TokenTag.GREATER_THAN:
        return a > b;

      case TokenTag.LESS_THAN:
        return a < b;
    }

    throw new SyntaxError(
      `It was not possible evaluate relative op:
            Comparator: ${comparator}
            Left: ${left}
            Right: ${right}
        `
    );
  }

  static evaluate(tree: ParserNode): number {
    if (tree instanceof IntegerNode) {
      return tree.value;
    }

    if (tree instanceof BinaryOperationNode) {
      const {operator, left, right} = tree;

      switch (operator.tag) {
        case TokenTag.ADD:
          return this.evaluate(left) + this.evaluate(right);

        case TokenTag.SUB:
          return this.evaluate(left) - this.evaluate(right);

        case TokenTag.MULT:
          return this.evaluate(left) * this.evaluate(right);
      }
    }

    return NaN;
  }
}
