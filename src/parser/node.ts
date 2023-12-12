import { ParserNodeTag } from "./tag";
import { Token } from "@/lexer/tokens/token";

export class ParserNode {
  constructor(public readonly tag: ParserNodeTag) {}
}

export class IdentNode extends ParserNode {
  constructor(public readonly name: string) {
    super(ParserNodeTag.IDENT);
  }
}

export class DeclNode extends ParserNode {
  constructor(public readonly name: string, public readonly value: ParserNode) {
    super(ParserNodeTag.DECL);
  }
}

export class NumericNode extends ParserNode {
  constructor(public readonly value: number) {
    super(ParserNodeTag.NUMERIC);
  }
}

export class BinaryOpNode extends ParserNode {
  constructor(
    public readonly left: ParserNode,
    public readonly right: ParserNode,
    public readonly op: Token
  ) {
    super(ParserNodeTag.BINARY_OP);
  }
}

export class StatementNode extends ParserNode {
  constructor(public readonly value: ParserNode) {
    super(ParserNodeTag.STATEMENT);
  }
}

export class BlockStatement extends ParserNode {
  constructor(public readonly children: ParserNode[]) {
    super(ParserNodeTag.BLOCK_STATEMENT);
  }
}

export class ProgramNode extends ParserNode {
  constructor(public readonly children: ParserNode[]) {
    super(ParserNodeTag.PROGRAM);
  }
}
