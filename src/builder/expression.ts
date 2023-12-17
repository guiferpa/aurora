import { ParserNode } from "@/parser";
import {
  BinaryOpNode,
  IdentNode,
  LogicalNode,
  NumericalNode,
} from "@/parser/node";

export default class Expression {
  public static lvalue(n: ParserNode): string {
    if (n instanceof IdentNode) {
      return n.name;
    }

    throw SyntaxError();
  }

  public static rvalue(n: ParserNode): string {
    const temps: string[] = [];

    if (n instanceof IdentNode) {
      return n.name;
    }

    if (n instanceof NumericalNode) {
      return n.value.toString();
    }

    if (n instanceof LogicalNode) {
      return `${n.value}`;
    }

    if (n instanceof BinaryOpNode) {
      const value = `t = ${Expression.rvalue(n.left)} ${
        n.op.value
      } ${Expression.rvalue(n.right)}`;
    }

    throw SyntaxError();
  }
}
