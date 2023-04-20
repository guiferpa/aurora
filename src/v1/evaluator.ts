import {TokenTag} from "./tokens";
import {
  BinaryOperationNode, 
  ParameterOperationNode, 
  ParserNode, 
} from "../v3/parser/node";

export default class Evaluator {
  static evaluate(tree: ParserNode): number {
    if (tree instanceof ParameterOperationNode) {
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
