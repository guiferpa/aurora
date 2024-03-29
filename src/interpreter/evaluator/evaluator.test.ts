import Eater from "@/eater/eater";
import Lexer from "@/lexer/lexer";
import SymTable from "@/symtable";
import Parser from "@/parser";
import Importer, { ImportClaim } from "@/importer";
import { EnvironError, Pool } from "@/environ";

import Evaluator from "./evaluator";

const execEvaluator = async (
  bucket: Map<string, string>,
  args: string[] = [],
  context: string = "main"
) => {
  const program = bucket.get(context) as string;
  const lexer = new Lexer(Buffer.from(program, "utf-8"));
  const symtable = new SymTable("global");
  const reader = {
    read: async (entry: string) => Buffer.from(bucket.get(entry) as string),
  };
  const importer = new Importer(reader);
  const claims = await importer.imports(new Eater(context, lexer.copy()));
  const alias = importer.alias(claims);
  const imports = new Map<string, ImportClaim>(
    claims.map((claim) => [claim.context, claim])
  );

  const parser = new Parser(new Eater(context, lexer.copy()), symtable);
  const tree = await parser.parse();

  const pool = new Pool(context);
  const evaluator = new Evaluator(pool, imports, alias, args);
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

        fib(6)`,
      ],
    ]);

    const expected = [undefined, 8];
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

  test("Program that import other file without alias", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `from "testing"

        hello()`,
      ],
      ["testing", `func hello() {}`],
    ]);

    expect(execEvaluator(bucket)).rejects.toThrow(Error);
  });

  test("Program that import others files with alias", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `from "testing" as t

        t.hello()`,
      ],
      [
        "testing",
        `from "a" as alpha

        func hello() {
          return alpha.num
        }`,
      ],
      ["a", `var num = 11`],
    ]);

    const expected = [undefined, 11];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that import others files with the same alias", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `from "testing" as t

        t.hello()`,
      ],
      [
        "testing",
        `from "a" as t

        func hello() {
          return t.num
        }`,
      ],
      ["a", `var num = 11`],
    ]);

    const expected = [undefined, 11];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that import others files with the deep declaration", async () => {
    const bucket = new Map<string, string>([
      ["main", `from "a" as la`],
      ["a", `from "b" as lb`],
      ["b", `var num = 20`],
    ]);

    const expected = [undefined];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that parse string to number", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var a = arg(0)

        func sum() {
          return str->num(a) + 10
        }

        sum()`,
      ],
    ]);

    const expected = [undefined, undefined, 11];
    const got = await execEvaluator(bucket, ["1"]);

    expect(got).toStrictEqual(expected);
  });

  test("Program that make operation with function from another file", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `from "math" as m

         var op = arg(0)
         var x = str->num(arg(1))
         var y = str->num(arg(2))

         func calc() {
           if op equal "+" {
             return m.add(x, y)
           }

           if op equal "-" {
             return m.sub(x, y)
           }

           if op equal "x" {
             return m.mult(x, y)
           }

           if op equal "/" {
             return m.div(x, y)
           }
         }

         return calc()`,
      ],
      [
        "math",
        `func add(x, y) {
           return x + y
         }

         func sub(x, y) {
           return x - y
         }

         func mult(x, y) {
           return x * y
         }

         func div(x, y) {
           return x / y
         }`,
      ],
    ]);

    const expected = [undefined, undefined, undefined, undefined, undefined, 0];
    const got = await execEvaluator(bucket, ["-", "1", "1"]);

    expect(got).toStrictEqual(expected);
  });

  test("Program that create at scope for let statement", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var c = "A"
        var arr = ["B", "C", "D"]

        let [b, c] = arr {
          print(c)
        }

        print(c)`,
      ],
    ]);

    const expected = [undefined, undefined, undefined, undefined];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that create return at scope for let statement", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var arr = [1, 2, 3]

        let [a, b] = arr {
          return b
        }`,
      ],
    ]);

    const expected = [undefined, 2];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that get nth item from an array", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var arr = [1, 2, 3]

        return nth(arr, 2)`,
      ],
    ]);

    const expected = [undefined, 3];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that get nth item from an array with invalid index", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var arr = [1, 2, 3]

        return nth(arr, "A")`,
      ],
    ]);

    const expected = [undefined, undefined];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Program that get nth item from an invalid value", async () => {
    const bucket = new Map<string, string>([["main", `return nth(true, 1)`]]);

    const expected = [undefined];
    const got = await execEvaluator(bucket);

    expect(got).toStrictEqual(expected);
  });
});
