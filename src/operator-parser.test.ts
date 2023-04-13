import Lexer from "./lexer";

import OperatorParser from "./operator-parser";
import {TokenIdentifier, TokenRecordList} from "./tokens";

describe('Operator parser tests', () => {
  test('Given 1 result should be 1 as ParameterOperation', () => {
    const program = `1`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);

    expect(parser.parse()).toMatchObject({
      type: "ParameterOperation",
      value: "1"
    });
  });

  test('Given 1 + 2 result should be 1 + 2 as BinaryOperation', () => {
    const program = `1 + 2`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);

    expect(parser.parse()).toMatchObject({
      type: "BinaryOperation",
      operator: {
        id: TokenIdentifier.ADD,
        type: TokenRecordList[TokenIdentifier.ADD],
        value: "+"
      },
      value: {
        type: "ParameterOperation",
        value: "1"
      },
      node: {
        type: "ParameterOperation",
        value: "2"
      }
    });
  });

  test.skip('Given 1 - 2 + 3 result should be 1 - 2 + 3 as BinaryOperation with 3 layers', () => {
    const program = `1 - 2 + 3`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);

    expect(parser.parse()).toMatchObject({
      test: true
    });
  });

  test.skip('Given 1 + 2 + 3 result should be 1 + 2 + 3 as BinaryOperation with 3 layers', () => {
    const program = `1 + 2 + 3`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);

    expect(parser.parse()).toMatchObject({
      type: "BinaryOperation",
      value: {
        operator: {
          id: TokenIdentifier.ADD,
          type: TokenRecordList[TokenIdentifier.ADD],
          value: "+"
        },
        right: {
          type: "ParameterOperation",
          value: "3",
          token: {
            id: TokenIdentifier.NUMBER,
            type: TokenRecordList[TokenIdentifier.NUMBER]
          }
        },
        left: {
          type: "BinaryOperation",
          value: {
            operator: {
              id: TokenIdentifier.ADD,
              type: TokenRecordList[TokenIdentifier.ADD],
              value: "+"
            },
            left: {
              type: "ParameterOperation",
              value: "1",
              token: {
                id: TokenIdentifier.NUMBER,
                type: TokenRecordList[TokenIdentifier.NUMBER]
              }
            },
            right: {
              type: "ParameterOperation",
              value: "2",
              token: {
                id: TokenIdentifier.NUMBER,
                type: TokenRecordList[TokenIdentifier.NUMBER]
              }
            }
          }
        }
      }

    });
  });

  test.skip('Given 1 + 2 * 3 result should be 1 + 2 * 3 as BinaryOperation with 3 layers', () => {
    const program = `1 + 2 * 3`;
    const lexer = new Lexer(Buffer.from(program));
    const parser = new OperatorParser(lexer);

    expect(parser.parse()).toMatchObject({
      type: "BinaryOperation",
      value: {
        operator: {
          id: TokenIdentifier.ADD,
          type: TokenRecordList[TokenIdentifier.ADD],
          value: "+"
        },
        left: {
          type: "ParameterOperation",
          value: "1",
          token: {
            id: TokenIdentifier.NUMBER,
            type: TokenRecordList[TokenIdentifier.NUMBER]
          }
        },
        right: {
          type: "BinaryOperation",
          value: {
            operator: {
              id: TokenIdentifier.MULT,
              type: TokenRecordList[TokenIdentifier.MULT],
              value: "*"
            },
            left: {
              type: "ParameterOperation",
              value: "2",
              token: {
                id: TokenIdentifier.NUMBER,
                type: TokenRecordList[TokenIdentifier.NUMBER]
              }
            },
            right: {
              type: "ParameterOperation",
              value: "3",
              token: {
                id: TokenIdentifier.NUMBER,
                type: TokenRecordList[TokenIdentifier.NUMBER]
              }
            }
          }
        }
      }
    });
  });

  describe('Operator parser testing precedence tree generator', () => {
    test('Given 1 - 2 + 3 result should be [1, 2, -, 3, +]', () => {
      const program = `1 - 2 + 3`;
      const lexer = new Lexer(Buffer.from(program));
      const parser = new OperatorParser(lexer);
      const tree = parser.precedenceTree();

      expect(tree).toEqual([
        {"id": 7, "type": "NUMBER", "value": "1"},
        {"id": 7, "type": "NUMBER", "value": "2"},
        {"id": 9, "type": "SUB", "value": "-"},
        {"id": 7, "type": "NUMBER", "value": "3"},
        {"id": 8, "type": "ADD", "value": "+"}
      ]);
    });

    test('Given 2 + 3 * 5 result should be [2, 3, 5, *, +]', () => {
      const program = `2 + 3 * 5`;
      const lexer = new Lexer(Buffer.from(program));
      const parser = new OperatorParser(lexer);
      const tree = parser.precedenceTree();

      expect(tree).toEqual([
        {"id": 7, "type": "NUMBER", "value": "2"},
        {"id": 7, "type": "NUMBER", "value": "3"},
        {"id": 7, "type": "NUMBER", "value": "5"},
        {"id": 12, "type": "MULT", "value": "*"},
        {"id": 8, "type": "ADD", "value": "+"}
      ]);
    });
  });
});

