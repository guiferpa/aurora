import { VariableClaim } from "./environ";
import Pool from "./pool";

describe("Pool test suite", () => {
  test("Test environ query running with previous scopes", async () => {
    const pool = new Pool();
    pool.add("a");
    pool.environ().set("k", new VariableClaim("value-k"));

    pool.ahead("b", pool.environ());

    pool.ahead("c", pool.environ());
    pool.environ().set("ka", new VariableClaim("value-ka"));

    const claim = pool.environ().query("ka");

    expect(claim).not.toBeUndefined();
    expect(claim).toBeInstanceOf(VariableClaim);
    expect((claim as VariableClaim).value).toBe("value-ka");
  });

  test("Test environ query running with contexts scopes", async () => {
    const pool = new Pool();
    pool.add("a");
    pool.environ().set("k", new VariableClaim("value-k"));

    pool.add("b");

    pool.add("c");
    pool.environ().set("ka", new VariableClaim("value-ka"));

    const claim = pool.environ().query("ka");

    expect(claim).not.toBeUndefined();
    expect(claim).toBeInstanceOf(VariableClaim);
    expect((claim as VariableClaim).value).toBe("value-ka");
  });
});
