import { Token } from "@/tokens";

export enum ParserNodeReturnType {
  Void = "Void",
  Integer = "Integer",
  Logical = "Logical",
  Str = "String",
}

export enum ParserNodeTag {
  Integer = "Integer",
  Logical = "Logical",
  Arity = "Arity",
  Str = "String",
  BinaryOperation = "BinaryOperation",
  UnaryOperation = "UnaryOperation",
  BlockStatment = "BlockStatment",
  IfStatment = "IfStatment",
  PrintCallStatment = "PrintCallStatment",
  FunctionStatment = "FunctionStatment",
  DefStatment = "DefStatment",
}

export class ParserNode {
  public readonly tag: ParserNodeTag;
  public readonly returnType: ParserNodeReturnType;

  constructor(tag: ParserNodeTag, returnType: ParserNodeReturnType) {
    this.tag = tag;
    this.returnType = returnType;
  }
}

export class BlockStatmentNode extends ParserNode {
  public readonly id: string;
  public readonly block: ParserNode[];

  constructor(id: string, block: ParserNode[]) {
    super(ParserNodeTag.BlockStatment, ParserNodeReturnType.Void);

    this.id = id;
    this.block = block;
  }
}

export class IfStatmentNode extends ParserNode {
  public readonly id: string;
  public readonly test: ParserNode;
  public readonly block: ParserNode[];

  constructor(id: string, test: ParserNode, block: ParserNode[]) {
    super(ParserNodeTag.IfStatment, ParserNodeReturnType.Void);

    this.id = id;
    this.test = test;
    this.block = block;
  }
}

export class DefFunctionStatmentNode extends ParserNode {
  public readonly name: string;
  public readonly arity: ArityNode;
  public readonly body: ParserNode[];

  constructor(
    id: string,
    name: string,
    arity: ArityNode,
    body: ParserNode[],
    returnType: ParserNodeReturnType
  ) {
    super(ParserNodeTag.FunctionStatment, returnType);

    this.name = name;
    this.arity = arity;
    this.body = body;
  }
}

export class DefStatmentNode extends ParserNode {
  public name: string;
  public value: ParserNode;

  constructor(name: string, value: ParserNode) {
    super(ParserNodeTag.DefStatment, ParserNodeReturnType.Void);

    this.name = name;
    this.value = value;
  }
}

export class PrintCallStatmentNode extends ParserNode {
  public readonly param: ParserNode;

  constructor(param: ParserNode) {
    super(ParserNodeTag.PrintCallStatment, ParserNodeReturnType.Void);

    this.param = param;
  }
}

export class BinaryOperationNode extends ParserNode {
  public left: ParserNode;
  public right: ParserNode;
  public operator: Token;

  constructor(
    left: ParserNode,
    right: ParserNode,
    operator: Token,
    returnType: ParserNodeReturnType
  ) {
    super(ParserNodeTag.BinaryOperation, returnType);

    this.left = left;
    this.right = right;
    this.operator = operator;
  }
}

export class UnaryOperationNode extends ParserNode {
  public readonly expr: ParserNode;
  public readonly operator: Token;

  constructor(
    expr: ParserNode,
    operator: Token,
    returnType: ParserNodeReturnType
  ) {
    super(ParserNodeTag.UnaryOperation, returnType);

    this.expr = expr;
    this.operator = operator;
  }
}

export class ArityNode extends ParserNode {
  public params: string[];

  constructor(params: string[]) {
    super(ParserNodeTag.Arity, ParserNodeReturnType.Void);

    this.params = params;
  }
}

export class IntegerNode extends ParserNode {
  public value: number;

  constructor(value: number) {
    super(ParserNodeTag.Integer, ParserNodeReturnType.Integer);

    this.value = value;
  }
}

export class LogicalNode extends ParserNode {
  public value: boolean;

  constructor(value: boolean) {
    super(ParserNodeTag.Logical, ParserNodeReturnType.Logical);

    this.value = value;
  }
}

export class StringNode extends ParserNode {
  public value: string;

  constructor(value: string) {
    super(ParserNodeTag.Str, ParserNodeReturnType.Str);

    this.value = value;
  }
}
