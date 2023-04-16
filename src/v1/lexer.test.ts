import {TokenIdentifier, TokenNumber, TokenTag} from "./tokens";
import Lexer from "./lexer";

describe("v1.Lexer test suite", () => {
  test("Given 'var result = 1;' expected [DEF, IDENT, ASSIGN, NUM, SEMI]", () => {
    const program = Buffer.from("var result = 1;");
    const lexer = new Lexer(program);

    let token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.DEF);

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.IDENT);
    expect((token as TokenIdentifier).name).toBe("result");

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.ASSIGN);

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.NUM);
    expect((token as TokenNumber).value).toBe(1);

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.SEMI);

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.EOT);
  });

  test("Given 'result = 1' expected [IDENT, ASSIGN, NUM]", () => {
    const program = Buffer.from("result = 1");
    const lexer = new Lexer(program);

    let token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.IDENT);
    expect((token as TokenIdentifier).name).toBe("result");

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.ASSIGN);

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.NUM);
    expect((token as TokenNumber).value).toBe(1);

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.EOT);
  });

  test("Given '1 = result' expected [NUM, ASSIGN, IDENT]", () => {
    const program = Buffer.from("1 = result");
    const lexer = new Lexer(program);

    let token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.NUM);
    expect((token as TokenNumber).value).toBe(1);

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.ASSIGN);

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.IDENT);
    expect((token as TokenIdentifier).name).toBe("result");

    token = lexer.getNextToken();
    expect(token.tag).toBe(TokenTag.EOT);
  });
});
