import Lexer, { Token, TokenTag } from "@/lexer";
import Eater from "@/eater";
import SymTable from "@/symtable";

import Parser from "./parser";
import {
  ArityStmtNode,
  AsStmtNode,
  AssignStmtNode,
  BinaryOpNode,
  BlockStmtNode,
  CallArgStmtNode,
  CallFuncStmtNode,
  CallPrintStmtNode,
  CallStrToNumStmtNode,
  DeclFuncStmtNode,
  FromStmtNode,
  IdentNode,
  IfStmtNode,
  ImportStmtNode,
  LogicalNode,
  NumericalNode,
  ProgramNode,
  StringNode,
} from "./node";

const execParser = async (
  bucket: Map<string, string>,
  pname: string = "main"
) => {
  const program = bucket.get(pname) as string;
  const lexer = new Lexer(Buffer.from(program, "utf-8"));
  const eater = new Eater(lexer);
  const symtable = new SymTable("global");
  const parser = new Parser(eater, symtable);
  return await parser.parse();
};

describe("Parser test suite", () => {
  test("Program that parse sum binary operation", async () => {
    const bucket = new Map<string, string>([["main", `1_000 + 10`]]);
    const expected = new ProgramNode([
      new BinaryOpNode(
        new NumericalNode(1000),
        new NumericalNode(10),
        new Token(1, 8, TokenTag.OP_ADD, "+")
      ),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });

  test("Program that parse function declaration", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var i = 0
         func hello() {}`,
      ],
    ]);
    const expected = new ProgramNode([
      new AssignStmtNode("i", new NumericalNode(0)),
      new DeclFuncStmtNode(
        "hello",
        null,
        new ArityStmtNode([]),
        new BlockStmtNode([])
      ),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });

  test("Program that parse function declaration using parameters", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var i = 2
         func hello(world) {}`,
      ],
    ]);
    const expected = new ProgramNode([
      new AssignStmtNode("i", new NumericalNode(2)),
      new DeclFuncStmtNode(
        "hello",
        null,
        new ArityStmtNode(["world"]),
        new BlockStmtNode([])
      ),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });

  test("Program that parse function declaration using parameters and body", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var i = 2
         func hello(world) {
           var a = 1
         }`,
      ],
    ]);
    const expected = new ProgramNode([
      new AssignStmtNode("i", new NumericalNode(2)),
      new DeclFuncStmtNode(
        "hello",
        null,
        new ArityStmtNode(["world"]),
        new BlockStmtNode([new AssignStmtNode("a", new NumericalNode(1))])
      ),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });

  test("Program that parse function declaration using parameters and body calling another function", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var i = 100;

        func hello(world) {
          var a = 25;
          print(world);
        }`,
      ],
    ]);
    const expected = new ProgramNode([
      new AssignStmtNode("i", new NumericalNode(100)),
      new DeclFuncStmtNode(
        "hello",
        null,
        new ArityStmtNode(["world"]),
        new BlockStmtNode([
          new AssignStmtNode("a", new NumericalNode(25)),
          new CallPrintStmtNode(new IdentNode("world")),
        ])
      ),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });

  test("Program that parse assign a variable", async () => {
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
    const expected = new ProgramNode([
      new AssignStmtNode("compare", new LogicalNode(true)),
      new IfStmtNode(
        new IdentNode("compare"),
        new BlockStmtNode([
          new CallPrintStmtNode(new StringNode("Testing")),
          new NumericalNode(20),
        ])
      ),
      new NumericalNode(10),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });

  test("Program that parse import syntax", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `from "testing"

        print(hello())`,
      ],
    ]);
    const expected = new ProgramNode([
      new ImportStmtNode(new FromStmtNode("testing"), new AsStmtNode("")),
      new CallPrintStmtNode(new CallFuncStmtNode("hello", [])),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });

  test("Program that parse import syntax with alias", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `from "testing" as t

        print(hello())`,
      ],
    ]);
    const expected = new ProgramNode([
      new ImportStmtNode(new FromStmtNode("testing"), new AsStmtNode("t")),
      new CallPrintStmtNode(new CallFuncStmtNode("hello", [])),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });

  test("Program that parse function that parse string to number", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `var a = str->num(arg(0))
        var b = str->num(arg(1))

        print(a + b)`,
      ],
    ]);
    const expected = new ProgramNode([
      new AssignStmtNode(
        "a",
        new CallStrToNumStmtNode(new CallArgStmtNode(new NumericalNode(0)))
      ),
      new AssignStmtNode(
        "b",
        new CallStrToNumStmtNode(new CallArgStmtNode(new NumericalNode(1)))
      ),
      new CallPrintStmtNode(
        new BinaryOpNode(
          new IdentNode("a"),
          new IdentNode("b"),
          new Token(4, 18, TokenTag.OP_ADD, "+")
        )
      ),
    ]);

    const got = await execParser(bucket);
    expect(got).toStrictEqual(expected);
  });
});
