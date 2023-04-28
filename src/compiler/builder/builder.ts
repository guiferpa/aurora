import {IntegerNode, ParserNode} from "@/parser";

export default class Builder {
  static build(stmt: ParserNode): string {
    if (stmt instanceof IntegerNode)
      return `${stmt.value}`;

    return "";
  }
}
