import Lexer from "@/lexer/lexer";
import { Parser } from "@/parser";

import Evaluator from "./evaluator";
import SymTable from "@/symtable";
import Environment from "@/environ";

const execEvaluator = async (
  bucket: Map<string, string>,
  args: string[] = [],
  environ: Environment = new Environment("global")
) => {
  const program = bucket.get("main") as string;
  const lexer = new Lexer(Buffer.from(program, "utf-8"));
  const symtable = new SymTable("global");
  const reader = {
    read: async (entry: string) => Buffer.from(bucket.get(entry) as string),
  };
  const parser = new Parser(reader, symtable);
  const tree = await parser.parse(lexer);
  const evaluator = new Evaluator(environ, args);
  return evaluator.evaluate(tree);
};

describe("Evaluator test suite", () => {
  test("Program that sum two numbers", async () => {
    const bucket = new Map<string, string>([["main", `1 + 1`]]);

    const expected = [2];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that calc precedence expression", async () => {
    const bucket = new Map<string, string>([["main", `10 + 20 - 3 * 20`]]);

    const expected = [-30];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program 2 that calc precedence expression", async () => {
    const bucket = new Map<string, string>([["main", `10 - 2 * 5`]]);

    const expected = [0];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program 3 that calc precedence expression", async () => {
    const bucket = new Map<string, string>([["main", `10 * 2 - 5`]]);

    const expected = [15];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that set a variable then sum it with another number", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var value = 10
        value + 20`,
      ],
    ]);

    const expected = [undefined, 30];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test('Program that set an "if" then it has condition with "false" value', async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var compare = false

        if (compare) {
          print("Testing")
          20
        }

        10`,
      ],
    ]);

    const expected = [undefined, undefined, 10];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test('Program that set an "if" then it has condition with "true" value', async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var compare = true

        if compare {
          print("Testing")
          20
        }

        10`,
      ],
    ]);

    const expected = [undefined, undefined, 10];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that get argument from CLI execution", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `func greeting(who) {
          if who {
            return who
          }

          return "World"
        }

        print("Hello")
        greeting(greeting(arg(0)))
        `,
      ],
    ]);

    const expected = [undefined, undefined, "Testing"];
    const got = await execEvaluator(bucket, ["Testing"]);

    expect(got).toStrictEqual(expected);
  });

  test("Program that get more than one argument from CLI execution", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `arg(1)
        func greeting(who) {
          if who {
            return who
          }

          return "World"
        }

        print("Hello")
        greeting(greeting(arg(0)))`,
      ],
    ]);

    const expected = ["Testing 2", undefined, undefined, "Testing"];
    const got = await execEvaluator(bucket, ["Testing", "Testing 2"]);

    expect(got).toStrictEqual(expected);
  });

  test("Program that get more than one argument from CLI execution", async () => {
    const program = `
    `;
    const bucket = new Map<string, string>([
      [
        "main",
        `arg(1)
        func greeting(who) {
          if who {
            return who
          }

          return "World"
        }

        print("Hello")
        greeting(greeting(arg(0)))`,
      ],
    ]);

    const expected = ["Testing 2", undefined, undefined, "Testing"];
    const got = await execEvaluator(bucket, ["Testing", "Testing 2"]);

    expect(got).toStrictEqual(expected);
  });

  test('Program that set an "if" then it has condition with "true" value based on argument input', async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var a = arg(0)
        var b = arg(1)

        if a equal "print" {
          return b
        }`,
      ],
    ]);

    const expected = [undefined, undefined, "Testing"];
    const got = await execEvaluator(bucket, ["print", "Testing"]);

    expect(got).toStrictEqual(expected);
  });

  test('Program that set an "if" then it has condition with "false" value based on argument input', async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var a = arg(0)
        var b = arg(1)

        if a equal "print" {
          return b
        }`,
      ],
    ]);

    const expected = [undefined, undefined, undefined];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that execute fibonacci algorithm", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `func fib(n)
        desc "This function calculate fibonacci number" {
          if n less 1 or n equal 1 {
            return n
          }

          return fib(n - 1) + fib(n - 2)
        }

        fib(25)`,
      ],
    ]);

    const expected = [undefined, 75025];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that declare an array", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var arr = [1, 2, 50]
        arr`,
      ],
    ]);

    const expected = [undefined, [1, 2, 50]];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that call map to increment one in array's items", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var arr = [1, 2, 50]

        func increment(item) {
          return item + 1
        }

        map(arr, increment)`,
      ],
    ]);

    const expected = [undefined, undefined, [2, 3, 51]];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that call filter to return all values from array less 'aquaman' string", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var arr = ["aunt-man", "batman", "aquaman", "iron man"]

        func is_not_aquaman?(item) {
          return not (item equal "aquaman")
        }

        filter(arr, is_not_aquaman?)`,
      ],
    ]);

    const expected = [undefined, undefined, ["aunt-man", "batman", "iron man"]];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that import other file", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `from "testing"

        hello()`,
      ],
      ["testing", `func hello() {}`],
    ]);

    const expected = [undefined, undefined];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });
});
