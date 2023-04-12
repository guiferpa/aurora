import {Token, TokenIdentifier} from "./tokens";

export default class OperatorEvaluator {
  static eval(ast: Token[]) {
    let stack: number[] = [];

    for (let index = 0; index < ast.length; index++) {
      const token = ast[index];

      if (token.id === TokenIdentifier.NUMBER) {
        stack.push(Number.parseInt(token.value));
        continue;
      }

      if (token.id === TokenIdentifier.ADD) {
        const [a, b] = stack.splice(stack.length - 2, 2);
        stack = [(a + b), ...stack];
        continue;
      }

      if (token.id === TokenIdentifier.MULT) {
        const [a, b] = stack.splice(stack.length - 2, 2);
        stack = [(a * b), ...stack];
        continue;
      }
    }

    return stack[0];
  }
}
