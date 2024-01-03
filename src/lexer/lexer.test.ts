import { TokenTag } from "./tokens/tag";
import Lexer from "./lexer";
import { LexerError } from "./errors";

describe("Lexer test suite", () => {
  test("Testing unexpected token using @@a as input", () => {
    const input = Buffer.from("@@a");
    const lexer = new Lexer(input);

    expect(() => lexer.getNextToken()).toThrow(LexerError);
  });

  test("Testing __NUM__ token using 127 as input", () => {
    const input = Buffer.from("127");
    const lexer = new Lexer(input);

    const token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.NUM);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });

  test("Testing __NUM__ token using 127_000 as input", () => {
    const input = Buffer.from("127_000");
    const lexer = new Lexer(input);

    const token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.NUM);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });

  test("Testing __NUM__ token using 0_127_000 as input", () => {
    const input = Buffer.from("0_127_000");
    const lexer = new Lexer(input);

    const token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.NUM);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });

  test("Testing __OP_MUL__ token using * as input", () => {
    const input = Buffer.from("*");
    const lexer = new Lexer(input);

    const token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.OP_MUL);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });

  test("Testing __OP_DIV__ token using / as input", () => {
    const input = Buffer.from("/");
    const lexer = new Lexer(input);

    const token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.OP_DIV);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });

  test("Testing __OP_ADD__ token using + as input", () => {
    const input = Buffer.from("+");
    const lexer = new Lexer(input);

    const token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.OP_ADD);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });

  test("Testing __OP_SUB__ token using - as input", () => {
    const input = Buffer.from("-");
    const lexer = new Lexer(input);

    const token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.OP_SUB);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });

  test("Testing tokens for expression 1_000 * 2 as input", () => {
    const input = Buffer.from("1_000 * 2");
    const lexer = new Lexer(input);

    let token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.NUM);

    token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.OP_MUL);

    token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.NUM);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });

  test("Testing tokens for __PAREN_C__ and __PAREN_O__ the same time token using )( as input", () => {
    const input = Buffer.from(")(");
    const lexer = new Lexer(input);

    let token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.PAREN_C);

    token = lexer.getNextToken();
    expect(token).not.toBeNull();
    expect(token?.tag).toBe(TokenTag.PAREN_O);

    expect(lexer.hasMoreTokens()).toEqual<boolean>(false);
  });
});
