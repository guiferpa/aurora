import {TokenIdentifier} from "./tokens";

export default class OperatorEvaluator {
  static eval(ast: any): number {
    if (ast.type === "ParameterOperation") {
      return Number.parseInt(ast.value);
    }

    if (ast.type === "BinaryOperation") {
      const {operator, left, right} = ast.value;

      switch (operator.id) {
        case TokenIdentifier.ADD:
          return Number.parseInt(left.value) + this.eval(right);

        case TokenIdentifier.SUB:
          return Number.parseInt(left.value) - this.eval(right);
      }
    }

    return NaN;
  }
}
