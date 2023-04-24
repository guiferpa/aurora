import {Token} from "../../v1";

export enum ParserNodeReturnType {
  Void = "Void",
  Integer = "Integer",
  Logical = "Logical"
}

export enum ParserNodeTag {
  Integer = "Integer",
  Logical = "Logical",
  Identifier = "Identifier",
  BinaryOperation = "BinaryOperation",
  UnaryOperation = "UnaryOperation",
  BlockStatment = "BlockStatment",
  IfStatment = "IfStatment",
  PrintCallStatment = "PrintCallStatment"
}

export class ParserNode {
  public readonly tag: ParserNodeTag;
  public readonly returnType: ParserNodeReturnType;

  constructor (tag: ParserNodeTag, returnType: ParserNodeReturnType) {
    this.tag = tag;
    this.returnType = returnType;
  } 
}

export class BlockStatmentNode extends ParserNode {
  public readonly id: string;
  public readonly block: ParserNode[];

  constructor (id: string, block: ParserNode[]) {
    super(ParserNodeTag.BlockStatment, ParserNodeReturnType.Void);

    this.id = id;
    this.block = block;
  }
}

export class IfStatmentNode extends ParserNode {
  public readonly id: string;
  public readonly test: ParserNode;
  public readonly block: ParserNode[];

  constructor (id: string, test: ParserNode, block: ParserNode[]) {
    super(ParserNodeTag.IfStatment, ParserNodeReturnType.Void);

    this.id = id;
    this.test = test;
    this.block = block;
  }
}

export class PrintCallStatmentNode extends ParserNode {
  public readonly param: ParserNode;

  constructor (param: ParserNode) {
    super(ParserNodeTag.PrintCallStatment, ParserNodeReturnType.Void);

    this.param = param;
  }
}

export class BinaryOperationNode extends ParserNode {
  public left: ParserNode;
  public right: ParserNode;
  public operator: Token;

  constructor (
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

  constructor (
    expr: ParserNode,
    operator: Token,
    returnType: ParserNodeReturnType
  ) {
    super(ParserNodeTag.UnaryOperation, returnType);

    this.expr = expr;
    this.operator = operator;
  }
}

export class IdentifierNode extends ParserNode {
  public name: string;

  constructor (name: string) {
    super(ParserNodeTag.Identifier, ParserNodeReturnType.Void);

    this.name = name;
  }
}

export class IntegerNode extends ParserNode {
  public value: number;

  constructor (value: number) {
    super(ParserNodeTag.Integer, ParserNodeReturnType.Integer);

    this.value = value;
  }
}

export class LogicalNode extends ParserNode {
  public value: boolean;

  constructor (value: boolean) {
    super(ParserNodeTag.Logical, ParserNodeReturnType.Logical);

    this.value = value;
  }
}

