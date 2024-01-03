import Lexer from "@/lexer";
import Eater from "@/eater";

import Importer from "./importer";

const execImporterMapping = async (
  bucket: Map<string, string>,
  pname: string = "main"
) => {
  const program = bucket.get(pname) as string;
  const lexer = new Lexer(Buffer.from(program, "utf-8"));
  const eater = new Eater(lexer);
  const reader = {
    read: async (entry: string) => {
      const program = bucket.get(entry) as string;
      return Buffer.from(program);
    },
  };
  const importer = new Importer(eater, reader);
  return await importer.mapping();
};

describe("Importer mapping test suite", () => {
  test("Import c from b from a from main", async () => {
    const bucket = new Map<string, string>([
      ["main", `from "a"`],
      ["a", `from "b"`],
      ["b", `from "c"`],
      ["c", ``],
    ]);

    const expected = [
      { id: "a", alias: "" },
      { id: "b", alias: "" },
      { id: "c", alias: "" },
    ];
    const got = await execImporterMapping(bucket);

    expect(got).toStrictEqual(expected);
  });
});
