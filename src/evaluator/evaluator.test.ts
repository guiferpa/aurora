import {Lexer} from "@/lexer";
import {Parser} from "@/parser";

import Evaluator from "./evaluator";

describe('Evaluator test suite', () => {
  test('Program that sum two numbers', () => {
    const program = `
    1 + 1;
    `;

    const expected = ["2"];

    const lexer = new Lexer(Buffer.from(program));
    const parser = new Parser(lexer);
    const got = Evaluator.compose(parser.parse().block);

    expect(got).toStrictEqual(expected);
  });

  test('Program that calc precedence expression', () => {
    const program = `
    10 + 20 - 3 * 20;
    `;

    const expected = ["-30"];

    const lexer = new Lexer(Buffer.from(program));
    const parser = new Parser(lexer);
    const got = Evaluator.compose(parser.parse().block);

    expect(got).toStrictEqual(expected);
  });

  test('Program that set a vairbale then sum it with another number', () => {
    const program = `
    var value = 10;
    value + 20;
    `;

    const expected = ["30"];

    const lexer = new Lexer(Buffer.from(program));
    const parser = new Parser(lexer);
    const got = Evaluator.compose(parser.parse().block);

    expect(got).toStrictEqual(expected);
  });
});
