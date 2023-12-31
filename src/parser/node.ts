import { ParserNodeTag } from "./tag";
import { Token } from "@/lexer/tokens/token";

export class ParserNode {
  constructor(public readonly tag: ParserNodeTag) {}
}

export class ParamNode extends ParserNode {
  constructor(public readonly name: string) {
    super(ParserNodeTag.PARAM);
  }
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

export class StringNode extends ParserNode {
  constructor(public readonly value: string) {
    super(ParserNodeTag.STRING);
  }
}

export class ArrayNode extends ParserNode {
  constructor(public readonly items: ParserNode[]) {
    super(ParserNodeTag.ARRAY);
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

export class ArityStmtNode extends ParserNode {
  constructor(public readonly params: string[]) {
    super(ParserNodeTag.ARITY_STMT);
  }
}

export class DescFuncStmtNode extends ParserNode {
  constructor(public readonly value: string) {
    super(ParserNodeTag.DESC_FUNC_STMT);
  }
}

export class DeclFuncStmtNode extends ParserNode {
  constructor(
    public readonly name: string,
    public readonly desc: DescFuncStmtNode | null,
    public readonly arity: ArityStmtNode,
    public readonly body: ParserNode
  ) {
    super(ParserNodeTag.DECL_FUNC_STMT);
  }
}

export class ReturnStmtNode extends ParserNode {
  constructor(public readonly value: ParserNode) {
    super(ParserNodeTag.RETURN_STMT);
  }
}

export class ReturnVoidStmtNode extends ParserNode {
  constructor() {
    super(ParserNodeTag.RETURN_VOID_STMT);
  }
}

export class BlockStmtNode extends ParserNode {
  constructor(public readonly children: ParserNode[]) {
    super(ParserNodeTag.BLOCK_STMT);
  }
}

export class CallFuncStmtNode extends ParserNode {
  constructor(
    public readonly name: string,
    public readonly params: ParserNode[]
  ) {
    super(ParserNodeTag.CALL_FUNC_STMT);
  }
}

export class CallPrintStmtNode extends ParserNode {
  constructor(public readonly param: ParserNode) {
    super(ParserNodeTag.CALL_PRINT_STMT);
  }
}

export class CallArgStmtNode extends ParserNode {
  constructor(public readonly index: ParserNode) {
    super(ParserNodeTag.CALL_ARG_STMT);
  }
}

export class CallConcatStmtNode extends ParserNode {
  constructor(public readonly values: ParserNode[]) {
    super(ParserNodeTag.CALL_CONCAT_STMT);
  }
}

export class CallMapStmtNode extends ParserNode {
  constructor(
    public readonly param: ParserNode,
    public readonly handle: ParserNode
  ) {
    super(ParserNodeTag.CALL_MAP_STMT);
  }
}

export class CallFilterStmtNode extends ParserNode {
  constructor(
    public readonly param: ParserNode,
    public readonly handle: ParserNode
  ) {
    super(ParserNodeTag.CALL_MAP_STMT);
  }
}

export class CallStrToNumStmtNode extends ParserNode {
  constructor(public readonly param: ParserNode) {
    super(ParserNodeTag.CALL_STR_TO_NUM);
  }
}

export class FromStmtNode extends ParserNode {
  constructor(public readonly id: string) {
    super(ParserNodeTag.FROM_STMT);
  }
}

export class AsStmtNode extends ParserNode {
  constructor(public readonly alias: string) {
    super(ParserNodeTag.AS_STMT);
  }
}

export class ImportStmtNode extends ParserNode {
  constructor(
    public readonly id: ParserNode,
    public readonly alias: ParserNode,
    public readonly program: ProgramNode
  ) {
    super(ParserNodeTag.IMPORT_STMT);
  }
}

export class ProgramNode extends ParserNode {
  constructor(public readonly children: ParserNode[]) {
    super(ParserNodeTag.PROGRAM);
  }
}
