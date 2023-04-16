import {TokenTag} from "./types"

describe('v1.Token test suite', () => {
  test('Given EOT expected "EOT"', () => {
    expect(TokenTag.EOT).toBe("EOT");
  });

  test('Given EOF expected "EOF"', () => {
    expect(TokenTag.EOF).toBe("EOF");
  });

  test('Given WHITESPACE expected "WHITESPACE"', () => {
    expect(TokenTag.WHITESPACE).toBe("WHITESPACE");
  });

  test('Given DEF expected "DEF"', () => {
    expect(TokenTag.DEF).toBe("DEF");
  });

  test('Given IDENT expected "IDENT"', () => {
    expect(TokenTag.IDENT).toBe("IDENT");
  });

  test('Given ASSIGN expected "ASSIGN"', () => {
    expect(TokenTag.ASSIGN).toBe("ASSIGN");
  });

  test('Given SEMI expected "SEMI"', () => {
    expect(TokenTag.SEMI).toBe("SEMI");
  });

  test('Given NUM expected "NUM"', () => {
    expect(TokenTag.NUM).toBe("NUM");
  });

  test('Given ADD expected "ADD"', () => {
    expect(TokenTag.ADD).toBe("ADD");
  });

  test('Given SUB expected "SUB"', () => {
    expect(TokenTag.SUB).toBe("SUB");
  });
});
