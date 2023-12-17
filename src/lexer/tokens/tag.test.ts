import { TokenTag } from "./tag";

describe("Token tags test suite", () => {
  test('Given NUM expected "__NUM__"', () => {
    expect(TokenTag.NUM).toBe("__NUM__");
  });

  test('Given OP_ADD expected "__OP_ADD__"', () => {
    expect(TokenTag.OP_ADD).toBe("__OP_ADD__");
  });

  test('Given OP_SUB expected "__OP_SUB__"', () => {
    expect(TokenTag.OP_SUB).toBe("__OP_SUB__");
  });

  test('Given OP_MUL expected "__OP_MUL__"', () => {
    expect(TokenTag.OP_MUL).toBe("__OP_MUL__");
  });

  test('Given OP_DIV expected "__OP_DIV__"', () => {
    expect(TokenTag.OP_DIV).toBe("__OP_DIV__");
  });

  test('Given PAREN_O expected "__PAREN_O__"', () => {
    expect(TokenTag.PAREN_O).toBe("__PAREN_O__");
  });

  test('Given PAREN_C expected "__PAREN_C__"', () => {
    expect(TokenTag.PAREN_C).toBe("__PAREN_C__");
  });
});
