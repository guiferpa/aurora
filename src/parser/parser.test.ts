import Lexer from "@/lexer/lexer";
import Parser from "./parser";
import Environment from "@/environ/environ";

describe("Parser test suite", () => {
  test("Parse expression with __OP_ADD__ token", () => {
    const program = `1 + 1_000`;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const environ = new Environment("root");
    const parser = new Parser(lexer, environ);
    parser.parse();
  });

  test.skip("Get function token", () => {
    const program = `
    var i = 0;
    func hello() {}
    `;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const environ = new Environment("root");
    const parser = new Parser(lexer, environ);
    parser.parse();
  });

  test.skip("Get function token using params", () => {
    const program = `
    var i = 0;
    func hello(world) {}
    `;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const environ = new Environment("root");
    const parser = new Parser(lexer, environ);
    parser.parse();
  });

  test.skip("Get function token using body", () => {
    const program = `
    var i = 0;
    func hello(world) {
      var a = 1;
    }
    `;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const environ = new Environment("root");
    const parser = new Parser(lexer, environ);
    parser.parse();
  });

  test.skip("Get function token using body calling another func", () => {
    const program = `
    var i = 0;
    func hello(world) {
      var a = 1;
      print(world);
    }
    `;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const environ = new Environment("root");
    const parser = new Parser(lexer, environ);
    parser.parse();
  });
});
