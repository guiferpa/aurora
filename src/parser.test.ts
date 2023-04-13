import Lexer from "./lexer";
import {TokenIdentifier, TokenRecordList} from "./tokens";
import {Parser} from "./parser";

describe('AST parse testing', () => {
  describe('Numeric handler', () => {
    test('Single and inline numeric', () => {
      const program = `
      1;
      `

      const expected = {
        type: "Program",
        body: [
          {
            type: "Expr",
            body: {
              type: "Grain",
              body: {
                id: TokenIdentifier.NUMBER,
                type: TokenRecordList[TokenIdentifier.NUMBER],
                value: "1"
              }
            }
          }
        ]
      }

      const lexer = new Lexer(Buffer.from(program));
      const parser = new Parser(lexer);
      const got = parser.parse();

      expect(got).toMatchObject(expected);
    });

    test('Single and inline variable declaration to numeric value', () => {
      const program = `
      var solution = 042;

    `;

      const expected = {
        type: "Program",
        body: [
          {
            type: "VarDef",
            body: {
              ident: {
                id: TokenIdentifier.IDENT,
                type: TokenRecordList[TokenIdentifier.IDENT],
                value: "solution"
              },
              expr: {
                type: "Expr",
                body: {
                  type: "Grain",
                  body: {
                    id: TokenIdentifier.NUMBER,
                    type: TokenRecordList[TokenIdentifier.NUMBER],
                    value: "042"
                  }
                }
              }
            }
          }
        ]
      };

      const lexer = new Lexer(Buffer.from(program));
      const parser = new Parser(lexer);
      const got = parser.parse();

      expect(got).toMatchObject(expected);
    });

    test('Multi line block statement', () => {
      const program = `
      var solution = 042;

      {
        var blocked = 10 + solution + 20;
      }
    `;

      const expected = {
        type: "Program",
        body: [
          {
            type: "VarDef",
            body: {
              ident: {
                id: 4,
                type: "IDENT",
                value: "solution"
              },
              expr: {
                type: "Expr",
                body: {
                  type: "Grain",
                  body: {
                    id: 7,
                    type: "NUMBER",
                    value: "042"
                  }
                }
              }
            }
          },
          {
            type: "BlockStatement",
            body: [
              {
                type: "VarDef",
                body: {
                  ident: {
                    id: 4,
                    type: "IDENT",
                    value: "blocked"
                  },
                  expr: {
                    type: "Expr",
                    body: {
                      operator: {
                        id: 8,
                        type: "ADD",
                        value: "+"
                      },
                      grain: {
                        type: "Grain",
                        body: {
                          id: 7,
                          type: "NUMBER",
                          value: "10"
                        }
                      },
                      expr: {
                        type: "Expr",
                        body: {
                          operator: {
                            id: 8,
                            type: "ADD",
                            value: "+"
                          },
                          grain: {
                            type: "Grain",
                            body: {
                              id: 4,
                              type: "IDENT",
                              value: "solution"
                            }
                          },
                          expr: {
                            type: "Expr",
                            body: {
                              type: "Grain",
                              body: {
                                id: 7,
                                type: "NUMBER",
                                value: "20"
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                }
              }
            ]
          }
        ]
      }

      const lexer = new Lexer(Buffer.from(program));
      const parser = new Parser(lexer);
      const got = parser.parse();

      expect(got).toMatchObject(expected);
    });
  });
});
