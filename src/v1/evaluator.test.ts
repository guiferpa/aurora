import {Parser} from "../v3";
import Evaluator from "./evaluator";
import Lexer from "./lexer";

describe('v1.Evaluator test suite', () => {
  test('Given 1 + 1 expected 2', () => {
    const program = Buffer.from("1 + 1");
    const lexer = new Lexer(program);
    const parser = new Parser(lexer);
    const tree = parser.parse();
    const got = Evaluator.evaluate(tree);

    expect(got).toBe(2);
  });

  test('Given 1 - 8 + 2 expected -5', () => {
    const program = Buffer.from("1 - 8 + 2");
    const lexer = new Lexer(program);
    const parser = new Parser(lexer);
    const tree = parser.parse();
    const got = Evaluator.evaluate(tree);

    expect(got).toBe(-5);
  });
});
