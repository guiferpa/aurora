import Environment, { VariableClaim } from "./environ";

describe("Environment test suite", () => {
  test("Test environ query running with previous scopes", async () => {
    const ec = new Environment("c");
    const eb = new Environment("b", ec);
    const ea = new Environment("a", eb);

    ec.set("ka", new VariableClaim("value-ka"));
    ea.set("k", new VariableClaim("value-k"));

    const claim = ea.query("ka");

    expect(claim).not.toBeUndefined();
    expect(claim).toBeInstanceOf(VariableClaim);
    expect((claim as VariableClaim).value).toBe("value-ka");
  });
});
