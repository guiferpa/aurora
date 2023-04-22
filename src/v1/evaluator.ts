import {TokenTag} from "./tokens";
import {
  BinaryOperationNode, 
  BlockStatmentNode, 
  IdentifierNode, 
  IntegerNode, 
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
        const { left, right } = stmt;
        out.push(`${Evaluator.evaluate(left) == Evaluator.evaluate(right)}`);
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
