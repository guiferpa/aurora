import Lexer from "@/lexer/lexer";
import { Parser } from "@/parser";

import Evaluator from "./evaluator";
import SymTable from "@/symtable";
import Environment from "@/environ";

describe("Evaluator test suite", () => {
  test("Program that sum two numbers", () => {
    const program = `
    1 + 1
    `;

    const expected = [2];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program that calc precedence expression", () => {
    const program = `
    10 + 20 - 3 * 20
    `;

    const expected = [-30];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program 2 that calc precedence expression", () => {
    const program = `
    10 - 2 * 5
    `;

    const expected = [0];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program 3 that calc precedence expression", () => {
    const program = `
    10 * 2 - 5
    `;

    const expected = [15];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program that set a variable then sum it with another number", () => {
    const program = `
    var value = 10;
    value + 20;
    `;

    const expected = [undefined, 30];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test('Program that set an "if" then it has condition with "false" value', () => {
    const program = `
    var compare = false;

    if (compare) {
      print("Testing");
      20;
    }

    10;
    `;

    const expected = [undefined, undefined, 10];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test('Program that set an "if" then it has condition with "true" value', () => {
    const program = `
    var compare = true;

    if (compare) {
      print("Testing");
      20;
    }

    10;
    `;

    const expected = [undefined, undefined, 10];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program that get argument from CLI execution", () => {
    const program = `
    func greeting(who) {
      if who {
        return who
      }

      return "World"
    }

    print("Hello")
    greeting(greeting(arg(0)))
    `;

    const expected = [undefined, undefined, "Testing"];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ, ["Testing"]);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program that get more than one argument from CLI execution", () => {
    const program = `
    arg(1)
    func greeting(who) {
      if who {
        return who
      }

      return "World"
    }

    print("Hello")
    greeting(greeting(arg(0)))
    `;

    const expected = ["Testing 2", undefined, undefined, "Testing"];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ, ["Testing", "Testing 2"]);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program that get more than one argument from CLI execution", () => {
    const program = `
    arg(1)
    func greeting(who) {
      if who {
        return who
      }

      return "World"
    }

    print("Hello")
    greeting(greeting(arg(0)))
    `;

    const expected = ["Testing 2", undefined, undefined, "Testing"];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ, ["Testing", "Testing 2"]);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test('Program that set an "if" then it has condition with "true" value based on argument input', () => {
    const program = `
    var a = arg(0)
    var b = arg(1)

    if a equal "print" {
      return b
    }
    `;

    const expected = [undefined, undefined, "Testing"];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ, ["print", "Testing"]);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test('Program that set an "if" then it has condition with "false" value based on argument input', () => {
    const program = `
    var a = arg(0)
    var b = arg(1)

    if a equal "print" {
      return b
    }
    `;

    const expected = [undefined, undefined, undefined];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ, ["no-print", "Testing"]);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program execute fibonacci algorithm", () => {
    const program = `
    func fib(n)
    desc "This function calculate fibonacci number" {
      if n less 1 or n equal 1 {
        return n
      }

      return fib(n - 1) + fib(n - 2)
    }

    fib(25)
    `;

    const expected = [undefined, 75025];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program that declare an array", () => {
    const program = `
    var arr = [1, 2, 50]
    arr
    `;

    const expected = [undefined, [1, 2, 50]];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program that call map to increment one in array's items", () => {
    const program = `
    var arr = [1, 2, 50]

    func increment(item) {
      return item + 1
    }

    map(arr, increment)
    `;

    const expected = [undefined, undefined, [2, 3, 51]];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });

  test("Program that call filter to return all values from array less 'aquaman' string", () => {
    const program = `
    var arr = ["aunt-man", "batman", "aquaman", "iron man"]

    func is_not_aquaman?(item) {
      return not (item equal "aquaman")
    }

    filter(arr, is_not_aquaman?)
    `;

    const expected = [undefined, undefined, ["aunt-man", "batman", "iron man"]];

    const lexer = new Lexer(Buffer.from(program));
    const symtable = new SymTable("root");
    const parser = new Parser(lexer, symtable);
    const environ = new Environment("root");
    const evaluator = new Evaluator(environ);
    const got = evaluator.evaluate(parser.parse());

    expect(got).toStrictEqual(expected);
  });
});
