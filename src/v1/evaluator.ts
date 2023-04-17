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
          return (left as ParameterOperationNode).value + this.evaluate(right);

        case TokenTag.SUB:
          return (left as ParameterOperationNode).value - this.evaluate(right);
      }
    }

    return NaN;
  }
}
