import {Token} from "../../v1";

export enum ParserNodeTag {
  Integer = "Integer",
  Logical = "Logical",
  Identifier = "Identifier",
  BinaryOperation = "BinaryOperation",
  BlockStatment = "BlockStatment"
};

export class ParserNode {
  public readonly tag: ParserNodeTag;

  constructor (tag: ParserNodeTag) {
    this.tag = tag;
  } 
}

export class BlockStatmentNode extends ParserNode {
  public id: string;
  public block: ParserNode[];

  constructor(id: string, block: ParserNode[]) {
    super(ParserNodeTag.BlockStatment);

    this.id = id;
    this.block = block;
  }
}

export class BinaryOperationNode extends ParserNode {
  public left: ParserNode;
  public right: ParserNode;
  public operator: Token;

  constructor (
    left: ParserNode, 
    right: ParserNode, 
    operator: Token
  ) {
    super(ParserNodeTag.BinaryOperation);

    this.left = left;
    this.right = right;
    this.operator = operator;
  }
}

export class IdentifierNode extends ParserNode {
  public name: string;
  public value: ParserNode;

  constructor (name: string, value: ParserNode) {
    super(ParserNodeTag.Identifier);

    this.name = name;
    this.value = value;
  }
}

export class IntegerNode extends ParserNode {
  public value: number;

  constructor (value: number) {
    super(ParserNodeTag.Integer);

    this.value = value;
  }
}

export class LogicalNode extends ParserNode {
  public value: boolean;

  constructor (value: boolean) {
    super(ParserNodeTag.Logical);

    this.value = value;
  }
}

