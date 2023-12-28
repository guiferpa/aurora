import { Lexer } from "@/lexer";
import Discover from "./discover";
import Dependency from "./dependency";

describe("Discover test suite", () => {
  test("Program that imports only one dependency", () => {
    const program = `
    from "github.com/guiferpa/testing" as testing
    `;

    const expected = [new Dependency("github.com/guiferpa/testing", "testing")];

    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const discover = new Discover(lexer);
    const got = discover.run();
    expect(got).toStrictEqual(expected);
  });

  test("Program that imports some dependencies", () => {
    const program = `
    from "github.com/guiferpa/testing" as testing
    from "github.com/guiferpa/tester" as tester
    from "github.com/guiferpa/test" as test
    `;

    const expected = [
      new Dependency("github.com/guiferpa/testing", "testing"),
      new Dependency("github.com/guiferpa/tester", "tester"),
      new Dependency("github.com/guiferpa/test", "test"),
    ];

    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const discover = new Discover(lexer);
    const got = discover.run();
    expect(got).toStrictEqual(expected);
  });

  test("Program that imports some duplicated dependencies", () => {
    const program = `
    from "github.com/guiferpa/testing" as testing
    from "github.com/guiferpa/testing" as testing
    from "github.com/guiferpa/tester" as tester
    from "github.com/guiferpa/test" as test
    `;

    const expected = [
      new Dependency("github.com/guiferpa/testing", "testing"),
      new Dependency("github.com/guiferpa/tester", "tester"),
      new Dependency("github.com/guiferpa/test", "test"),
    ];

    const lexer = new Lexer(Buffer.from(program, "utf-8"));
    const discover = new Discover(lexer);
    const got = discover.run();
    expect(got).toStrictEqual(expected);
  });
});
