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

export class NumericalNode extends ParserNode {
  constructor(public readonly value: number) {
    super(ParserNodeTag.NUMERICAL);
  }
}

export class LogicalNode extends ParserNode {
  constructor(public readonly value: boolean) {
    super(ParserNodeTag.LOGICAL);
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

export class UnaryOpNode extends ParserNode {
  constructor(public readonly right: ParserNode, public readonly op: Token) {
    super(ParserNodeTag.UNARY_OP);
  }
}

export class NegativeExprNode extends ParserNode {
  constructor(public readonly expr: ParserNode) {
    super(ParserNodeTag.NEG_EXPR);
  }
}

export class RelativeExprNode extends ParserNode {
  constructor(
    public readonly left: ParserNode,
    public readonly right: ParserNode,
    public readonly op: Token
  ) {
    super(ParserNodeTag.RELATIVE_EXPR);
  }
}

export class LogicExprNode extends ParserNode {
  constructor(
    public readonly left: ParserNode,
    public readonly right: ParserNode,
    public readonly op: Token
  ) {
    super(ParserNodeTag.LOGIC_EXPR);
  }
}

export class IfStmtNode extends ParserNode {
  constructor(
    public readonly test: ParserNode,
    public readonly body: ParserNode
  ) {
    super(ParserNodeTag.IF_STMT);
  }
}

export class AssignStmtNode extends ParserNode {
  constructor(public readonly name: string, public readonly value: ParserNode) {
    super(ParserNodeTag.ASSIGN_STMT);
  }
}

export class BlockStmtNode extends ParserNode {
  constructor(public readonly children: ParserNode[]) {
    super(ParserNodeTag.BLOCK_STMT);
  }
}

export class ProgramNode extends ParserNode {
  constructor(public readonly children: ParserNode[]) {
    super(ParserNodeTag.PROGRAM);
  }
}
