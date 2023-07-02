import { Lexer } from "@/lexer";
import Parser from "./parser";

describe("Testing parser cases", () => {
  test("Get function token", () => {
    const program = `
    var i = 0;
    func hello() {}
    `;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const parser = new Parser(lexer);
    parser.parse();
  });

  test("Get function token using params", () => {
    const program = `
    var i = 0;
    func hello(world) {}
    `;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const parser = new Parser(lexer);
    parser.parse();
  });

  test("Get function token using body", () => {
    const program = `
    var i = 0;
    func hello(world) {
      var a = 1;
    }
    `;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const parser = new Parser(lexer);
    parser.parse();
  });

  test("Get function token using body calling another func", () => {
    const program = `
    var i = 0;
    func hello(world) {
      var a = 1;
      print(world);
    }
    `;
    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const parser = new Parser(lexer);
    parser.parse();
  });
});
