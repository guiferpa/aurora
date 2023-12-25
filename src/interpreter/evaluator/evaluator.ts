import Environment, { FunctionClaim, VariableClaim } from "@/environ/environ";
import { TokenTag } from "@/lexer/tokens/tag";
import { ParserNode } from "@/parser";
import {
  BinaryOpNode,
  AssignStmtNode,
  DeclFuncStmtNode,
  IdentNode,
  NumericalNode,
  ProgramNode,
  BlockStmtNode,
  LogicalNode,
  NegativeExprNode,
  RelativeExprNode,
  LogicExprNode,
  UnaryOpNode,
  IfStmtNode,
  CallPrintStmtNode,
  CallFuncStmtNode,
  StringNode,
  ReturnStmtNode,
  ReturnVoidStmtNode,
  CallArgStmtNode,
} from "@/parser/node";
import colorizeJson from "json-colorizer";

export default class Evaluator {
  constructor(
    private _environ: Environment | null,
    private readonly _args: string[] = []
  ) {}

  private merge(nodes: ParserNode[]): ParserNode[] {
    const out = [];

    for (const n of nodes) {
      if (n instanceof IfStmtNode) {
        const tested = this.evaluate(n.test);

        if (tested && n.body instanceof BlockStmtNode) {
          out.push(...this.merge(n.body.children));
          continue;
        }

        if (tested) {
          out.push(n.body);
          continue;
        }

        continue;
      }

      out.push(n);
    }

    return out;
  }

  private compose(nodes: ParserNode[]): string[] {
    const out = [];

    for (const n of this.merge(nodes)) {
      if (n instanceof ReturnVoidStmtNode) {
        return out;
      }

      if (n instanceof ReturnStmtNode) {
        return this.evaluate(n.value);
      }

      out.push(`${this.evaluate(n)}`);
    }

    return out;
  }

  public evaluate(tree: ParserNode): any {
    if (tree instanceof ProgramNode) return this.compose(tree.children);

    if (tree instanceof BlockStmtNode) return this.compose(tree.children);

    if (tree instanceof AssignStmtNode) {
      const payload = new VariableClaim(this.evaluate(tree.value));
      this._environ?.set(tree.name, payload);
      return "";
    }

    if (tree instanceof DeclFuncStmtNode) {
      const payload = new FunctionClaim(tree.arity, tree.body);
      this._environ?.set(tree.name, payload);
      return "";
    }

    if (tree instanceof IdentNode) {
      const n = this._environ?.query(tree.name);

      if (typeof n === "undefined") {
        return null;
      }

      if (n instanceof FunctionClaim)
        throw new Error(`Invalid variable claim ${tree.name}`);

      if (n instanceof VariableClaim) {
        return n.value;
      }

      return this.evaluate(n);
    }

    if (tree instanceof CallFuncStmtNode) {
      const n = this._environ?.query(tree.name);

      if (!(n instanceof FunctionClaim))
        throw new Error(`Invalid calling for function ${tree.name}`);

      if (n.arity.params.length !== tree.params.length) {
        throw new Error(
          `Wrong arity for calling symbol ${tree.name}, expected: ${n.arity.params.length} but got ${tree.params.length}`
        );
      }

      this._environ = new Environment(`FUNC-${Date.now()}`, this._environ);

      // Allocating refs/values for evaluate AST with focus only in function scope
      n.arity.params.forEach((param, index) => {
        const payload = new VariableClaim(this.evaluate(tree.params[index]));
        this._environ?.set(param, payload);
      });

      const result = this.evaluate(n.body);

      this._environ = this._environ.prev;

      return result;
    }

    if (tree instanceof CallArgStmtNode) {
      const index = this.evaluate(tree.index);

      if (this._args.length > index) {
        return this._args[index];
      }

      return null;
    }

    if (tree instanceof CallPrintStmtNode) {
      console.log(this.evaluate(tree.param));
      return "";
    }

    if (tree instanceof NegativeExprNode) return !this.evaluate(tree.expr);

    if (tree instanceof NumericalNode) return tree.value;

    if (tree instanceof LogicalNode) return tree.value;

    if (tree instanceof StringNode) return tree.value;

    if (tree instanceof UnaryOpNode) {
      const { op, right } = tree;

      switch (op.tag) {
        case TokenTag.OP_ADD:
          return +this.evaluate(right);

        case TokenTag.OP_SUB:
          return -this.evaluate(right);
      }
    }

    if (tree instanceof BinaryOpNode) {
      const { op, left, right } = tree;

      switch (op.tag) {
        case TokenTag.OP_ADD:
          return this.evaluate(left) + this.evaluate(right);

        case TokenTag.OP_SUB:
          return this.evaluate(left) - this.evaluate(right);

        case TokenTag.OP_DIV:
          return this.evaluate(left) / this.evaluate(right);

        case TokenTag.OP_MUL:
          return this.evaluate(left) * this.evaluate(right);
      }
    }

    if (tree instanceof RelativeExprNode) {
      const { op, left, right } = tree;

      switch (op.tag) {
        case TokenTag.REL_GT:
          return this.evaluate(left) > this.evaluate(right);

        case TokenTag.REL_LT:
          return this.evaluate(left) < this.evaluate(right);

        case TokenTag.REL_EQ:
          return this.evaluate(left) === this.evaluate(right);

        case TokenTag.REL_DIF:
          return this.evaluate(left) !== this.evaluate(right);
      }
    }

    if (tree instanceof LogicExprNode) {
      const { op, left, right } = tree;

      switch (op.tag) {
        case TokenTag.LOG_AND:
          return this.evaluate(left) && this.evaluate(right);

        case TokenTag.LOG_OR:
          return this.evaluate(left) || this.evaluate(right);
      }
    }

    throw new Error(
      `Unsupported evaluate expression for [${JSON.stringify(tree)}]`
    );
  }
}
