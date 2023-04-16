import {TokenTag} from "./tokens";
import {
  BinaryOperationNode, 
  ParameterOperationNode, 
  ParserNode, 
  ParserNodeTag
} from "../v3/parser/node";

export default class Evaluator {
  static evaluate(tree: ParserNode): number {
    if (tree.tag === ParserNodeTag.ParameterOperation) {
      return (tree as ParameterOperationNode).value;
    }

    if (tree.tag === ParserNodeTag.BinaryOperation) {
      const {operator, left, right} = (tree as BinaryOperationNode);

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
