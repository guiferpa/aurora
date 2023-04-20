import {Token} from "../../v1";

export enum ParserNodeTag {
  ParameterOperation = "ParameterOperation",
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

export class ParameterOperationNode extends ParserNode {
  public value: number;

  constructor (value: number) {
    super(ParserNodeTag.ParameterOperation);

    this.value = value;
  }
}
