import { FunctionClaim, Pool, VariableClaim } from "@/environ";
import { TokenTag } from "@/lexer";
import { ImportClaim } from "@/importer";
import {
  ParserNode,
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
  CallConcatStmtNode,
  ArrayNode,
  CallMapStmtNode,
  CallFilterStmtNode,
  ImportStmtNode,
  CallStrToNumStmtNode,
  AccessContextStatementNode,
} from "@/parser";

import { EvaluateError } from "../errors";

export default class Evaluator {
  constructor(
    private _pool: Pool,
    private _imports: Map<string, ImportClaim>,
    private _alias: Map<string, Map<string, string>>,
    private readonly _args: string[] = []
  ) {}

  private translate(alias: string): string {
    const table = this._alias.get(this._pool.context());

    if (typeof table === "undefined")
      throw new EvaluateError(
        `No translate table for context ${this._pool.context()}`
      );

    const result = table.get(alias);
    if (typeof result === "undefined")
      throw new EvaluateError(
        `Alias ${alias} not resolved at context ${this._pool.context()}`
      );

    return result;
  }

  private compose(nodes: ParserNode[]): any[] {
    const out = [];

    for (const n of nodes) {
      out.push(this.evaluate(n));
    }

    return out;
  }

  public evaluate(tree: ParserNode): any {
    if (tree instanceof ProgramNode) return this.compose(tree.children);

    if (tree instanceof ImportStmtNode) {
      const importing = this._imports.get(tree.id.value);
      if (typeof importing === "undefined") return;

      if (tree.alias.value !== "") {
        this._pool.push(tree.id.value);
        this.compose(importing.program.children);
        this._pool.pop();
        return;
      }

      this.compose(importing.program.children);
      return;
    }

    if (tree instanceof AccessContextStatementNode) {
      const context = this.translate(tree.alias);
      this._pool.push(context);

      const result = this.evaluate(tree.prop);

      this._pool.pop();

      return result;
    }

    if (tree instanceof BlockStmtNode) return this.compose(tree.children);

    if (tree instanceof AssignStmtNode) {
      const payload = new VariableClaim(this.evaluate(tree.value));
      this._pool.environ().set(tree.name, payload);
      return;
    }

    if (tree instanceof DeclFuncStmtNode) {
      const context = this._pool.context();
      const payload = new FunctionClaim(tree.arity, tree.body);
      this._pool.environ().set(tree.name, payload);
      return;
    }

    if (tree instanceof ReturnVoidStmtNode) {
      return;
    }

    if (tree instanceof ReturnStmtNode) {
      return this.evaluate(tree.value);
    }

    if (tree instanceof IfStmtNode) {
      const tested = this.evaluate(tree.test);

      if (tested && tree.body instanceof BlockStmtNode) {
        for (const child of tree.body.children) {
          if (
            child instanceof ReturnVoidStmtNode ||
            child instanceof ReturnStmtNode
          ) {
            return this.evaluate(child);
          }

          this.evaluate(child);
        }

        return;
      }

      return;
    }

    if (tree instanceof IdentNode) {
      const n = this._pool.environ().getvar(tree.name);
      return n;
    }

    if (tree instanceof CallFuncStmtNode) {
      const n = this._pool.environ().getfunc(tree.name);
      if (n === null) return;

      if (n.arity.params.length !== tree.params.length) {
        throw new EvaluateError(
          `Wrong arity for calling symbol ${tree.name}, expected: ${n.arity.params.length} but got ${tree.params.length}`
        );
      }

      this._pool.push(tree.callee);
      const params = n.arity.params.map(
        (param, index): [string, VariableClaim] => {
          return [param, new VariableClaim(this.evaluate(tree.params[index]))];
        }
      );
      this._pool.pop();

      const scope = `${tree.tag}[${tree.name}]-${Date.now()}`;
      this._pool.ahead(scope, this._pool.environ());

      // Allocating refs/values for evaluate AST with focus only in function scope

      params.forEach(([name, claim]) => {
        this._pool.environ().set(name, claim);
      });

      for (const child of (n.body as BlockStmtNode).children) {
        const result = this.evaluate(child);
        if (typeof result !== "undefined") {
          this._pool.back();
          return result;
        }
      }

      this._pool.back();
      return;
    }

    if (tree instanceof CallArgStmtNode) {
      const index = this.evaluate(tree.index);

      if (this._args.length > index) {
        return this._args[index];
      }

      return;
    }

    if (tree instanceof CallConcatStmtNode) {
      const strs: string[] = tree.values.map(
        (item) => `${this.evaluate(item)}`
      );

      return strs.join("");
    }

    if (tree instanceof CallMapStmtNode) {
      const arr = this.evaluate(tree.param);
      if (!Array.isArray(arr))
        throw new EvaluateError("Call map param must be an array");

      const handle = this.evaluate(tree.handle);

      if (!(handle instanceof FunctionClaim))
        throw new EvaluateError(
          `It wasn't possible call function with no callback parameter`
        );

      const out = [];

      for (const item of arr) {
        const scope = `${tree.tag}-${Date.now()}`;
        this._pool.ahead(scope, this._pool.environ());

        handle.arity.params.forEach((param) => {
          const claim = new VariableClaim(item);
          this._pool.environ().set(param, claim);
        });

        for (const child of (handle.body as BlockStmtNode).children) {
          out.push(this.evaluate(child));
        }

        this._pool.back();
      }

      return out;
    }

    if (tree instanceof CallFilterStmtNode) {
      const arr = this.evaluate(tree.param);
      if (!Array.isArray(arr))
        throw new EvaluateError("Call filter param must be an array");

      const handle = this.evaluate(tree.handle);

      if (!(handle instanceof FunctionClaim)) return;

      const out = [];

      for (const item of arr) {
        const scope = `${tree.tag}-${Date.now()}`;
        this._pool.ahead(scope, this._pool.environ());

        handle.arity.params.forEach((param) => {
          const claim = new VariableClaim(item);
          this._pool.environ().set(param, claim);
        });

        for (const child of (handle.body as BlockStmtNode).children) {
          const tested = this.evaluate(child);
          if (tested) out.push(item);
        }

        this._pool.back();
      }

      return out;
    }

    if (tree instanceof CallPrintStmtNode) {
      console.log(this.evaluate(tree.param));
      return;
    }

    if (tree instanceof CallStrToNumStmtNode) {
      const param = this.evaluate(tree.param);
      const num = Number.parseInt(this.evaluate(tree.param));
      if (Number.isNaN(num))
        throw new EvaluateError(
          `Unexpected error for parse ${param} to number`
        );
      return num;
    }

    if (tree instanceof NegativeExprNode) return !this.evaluate(tree.expr);

    if (tree instanceof NumericalNode) return tree.value;

    if (tree instanceof LogicalNode) return tree.value;

    if (tree instanceof StringNode) return tree.value;

    if (tree instanceof ArrayNode)
      return tree.items.map((item) => this.evaluate(item));

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

    throw new EvaluateError(
      `Unsupported evaluate expression for ${JSON.stringify(tree)}`
    );
  }
}
