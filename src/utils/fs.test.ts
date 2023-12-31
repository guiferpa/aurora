import { noext } from "./fs";

describe("utils/fs test suite", () => {
  test("Should return filename and empty extension", async () => {
    const raw = "greeting";
    const expected = ["greeting", ""];
    const got = noext(raw);
    expect(got).toStrictEqual(expected);
  });

  test("Should return filename and 'br' extension", async () => {
    const raw = "greeting.br";
    const expected = ["greeting", "br"];
    const got = noext(raw);
    expect(got).toStrictEqual(expected);
  });

  test("Should return filename and 'ar' extension", async () => {
    const raw = "greeting.ar";
    const expected = ["greeting", "ar"];
    const got = noext(raw);
    expect(got).toStrictEqual(expected);
  });

  test("Should return 'gree.ting' filename and 'ar' extension", async () => {
    const raw = "gree.ting.ar";
    const expected = ["gree.ting", "ar"];
    const got = noext(raw);
    expect(got).toStrictEqual(expected);
  });

  test("Should return empty filename and empty extension", async () => {
    const raw = "";
    const expected = ["", ""];
    const got = noext(raw);
    expect(got).toStrictEqual(expected);
  });
});
