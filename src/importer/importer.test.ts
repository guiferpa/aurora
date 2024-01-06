import Lexer from "@/lexer";
import Eater from "@/eater";

import Importer from "./importer";
import {
  AccessContextStatementNode,
  ArityStmtNode,
  AssignStmtNode,
  BlockStmtNode,
  CallFuncStmtNode,
  DeclFuncStmtNode,
  IdentNode,
  NumericalNode,
  ProgramNode,
  ReturnStmtNode,
} from "@/parser";

const createImporter = (bucket: Map<string, string>): Importer => {
  const reader = {
    read: async (entry: string) => {
      const program = bucket.get(entry) as string;
      return Buffer.from(program);
    },
  };
  return new Importer(reader);
};

const execImports = async (
  bucket: Map<string, string>,
  context: string = "main"
) => {
  const program = bucket.get(context) as string;
  const lexer = new Lexer(Buffer.from(program, "utf-8"));
  const eater = new Eater(context, lexer);
  return await createImporter(bucket).imports(eater);
};

describe("Importer mapping test suite", () => {
  test("Importer test imports with alias", async () => {
    const bucket = new Map<string, string>([
      [
        "main",
        `from "a" as t
        from "b" as tt

        var a = 1`,
      ],
      ["a", `from "b" as t`],
      ["b", `from "c" as t`],
      ["c", `var b = 2`],
    ]);

    const expected = [
      {
        context: "main",
        mapping: [
          { alias: "t", id: "a" },
          { alias: "tt", id: "b" },
        ],
        alias: new Map([
          ["t", "a"],
          ["tt", "b"],
        ]),
        program: new ProgramNode([
          new AssignStmtNode("a", new NumericalNode(1)),
        ]),
      },
      {
        context: "a",
        mapping: [{ alias: "t", id: "b" }],
        alias: new Map([["t", "b"]]),
        program: new ProgramNode([]),
      },
      {
        context: "b",
        mapping: [{ alias: "t", id: "c" }],
        alias: new Map([["t", "c"]]),
        program: new ProgramNode([]),
      },
      {
        context: "c",
        mapping: [],
        alias: new Map([]),
        program: new ProgramNode([
          new AssignStmtNode("b", new NumericalNode(2)),
        ]),
      },
    ];

    const got = await execImports(bucket);

    expect(got).toStrictEqual(expected);
  });

  test("Importer test imports with different alias", async () => {
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

    const expected = [
      {
        context: "main",
        mapping: [{ alias: "t", id: "testing" }],
        alias: new Map([["t", "testing"]]),
        program: new ProgramNode([
          new AccessContextStatementNode(
            "t",
            new CallFuncStmtNode("hello", [])
          ),
        ]),
      },
      {
        context: "testing",
        mapping: [{ alias: "alpha", id: "a" }],
        alias: new Map([["alpha", "a"]]),
        program: new ProgramNode([
          new DeclFuncStmtNode(
            "hello",
            null,
            new ArityStmtNode([]),
            new BlockStmtNode([
              new ReturnStmtNode(
                new AccessContextStatementNode("alpha", new IdentNode("num"))
              ),
            ])
          ),
        ]),
      },
      {
        context: "a",
        mapping: [],
        alias: new Map([]),
        program: new ProgramNode([
          new AssignStmtNode("num", new NumericalNode(11)),
        ]),
      },
    ];

    const got = await execImports(bucket);

    expect(got).toStrictEqual(expected);
  });
});
