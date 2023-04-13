import Lexer from "./lexer";
import OperatorEvaluator from "./operator-evaluator";
import OperatorParser from "./operator-parser";

describe('Operator evaluator tests', () => {
  test.skip('Given 3 result should be 3', () => {
    const program = `3`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);
    const result = OperatorEvaluator.eval(parser.parse())

    expect(result).toEqual(3);
  });

  test.skip('Given 3 + 2 result should be 5', () => {
    const program = `3 + 2`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);
    const result = OperatorEvaluator.eval(parser.parse())

    expect(result).toEqual(5);
  });

  test.skip('Given 3 + 2 + 1 result should be 6', () => {
    const program = `3 + 2 + 1`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);
    const result = OperatorEvaluator.eval(parser.parse())

    expect(result).toEqual(6);
  });

  test.skip('Given 3 - 2 result should be 1', () => {
    const program = `3 - 2`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);
    const result = OperatorEvaluator.eval(parser.parse())

    expect(result).toEqual(1);
  });

  test.skip('Given 3 - 2 - 1 result should be 0', () => {
    const program = `3 - 2 - 1`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);
    const result = OperatorEvaluator.eval(parser.parse())

    expect(result).toEqual(0);
  });
});
