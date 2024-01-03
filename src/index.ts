import { Command } from "commander";

import pkg from "../package.json";

import Lexer, { LexerError } from "@/lexer";
import SymTable, { SymtableError } from "@/symtable";
import Parser, { ParserError } from "@/parser";
import Environment, { EnvironError } from "@/environ";
import Interpreter, { InterpreterError, EvaluateError } from "@/interpreter";
import Builder from "@/builder";

import * as utils from "@/utils";
import { repl } from "@/repl";
import Eater from "./eater/eater";
import Importer from "./importer/importer";

function handleError(err: Error) {
  if (err instanceof LexerError) {
    console.log(`![LexerError]: ${err.message}`);
    return;
  }
  if (err instanceof SymtableError) {
    console.log(`![SymtableError]: ${err.message}`);
    return;
  }
  if (err instanceof ParserError) {
    console.log(`![ParserError]: ${err.message}`);
    return;
  }
  if (err instanceof EnvironError) {
    console.log(`![EnvironError]: ${err.message}`);
    return;
  }
  if (err instanceof InterpreterError) {
    console.log(`![InterpreterError]: ${err.message}`);
    return;
  }
  if (err instanceof EvaluateError) {
    console.log(`![EvaluateError]: ${err.message}`);
    return;
  }
  console.log(`![Error]: ${err.message}`);
}

function run() {
  const program = new Command();

  program.name(pkg.name).description(pkg.repository).version(pkg.version);

  program
    .option("-t, --tree", "tree flag to show AST", false)
    .option("-a, --args <string>", "pass arguments for runtime", "")
    .action(async function () {
      const options = program.opts();

      const optArgs = options.args.split(",");

      const r = repl();

      const environ = new Environment("global");
      const interpreter = new Interpreter(environ);

      r.on("line", async function (chunk) {
        const buffer = Buffer.from(chunk);
        const lexer = new Lexer(buffer);
        const symtable = new SymTable("global");
        const eater = new Eater(lexer.copy());
        const parser = new Parser(eater, symtable);
        const importer = new Importer(new Eater(lexer.copy()), {
          read: utils.fs.read,
        });

        try {
          const [imports, alias] = await importer.imports();
          const tree = await parser.parse();
          const result = await interpreter.run(
            tree,
            imports,
            alias,
            options.tree as boolean,
            optArgs
          );
          console.log(`= ${result}`);
        } catch (err) {
          if (err instanceof Error) {
            handleError(err);
          }
        } finally {
          r.prompt(true);
        }
      });

      r.once("close", () => {
        console.log("Bye :)");
        process.exit(0);
      });
    });

  program
    .command("run")
    .argument("<filename>", "filename to run interpreter")
    .option("-t, --tree", "tree flag to show AST", false)
    .option("-a, --args <string>", "pass arguments for runtime", "")
    .action(async function (arg) {
      try {
        const options = program.opts();

        const optArgs = options.args.split(",");

        const buffer = await utils.fs.read(arg);
        const lexer = new Lexer(buffer);

        const importer = new Importer(new Eater(lexer.copy()), {
          read: utils.fs.read,
        });
        const [imports, alias] = await importer.imports();

        const symtable = new SymTable("global");
        const parser = new Parser(new Eater(lexer), symtable);
        const tree = await parser.parse();

        const environ = new Environment("global");
        const interpreter = new Interpreter(environ);
        await interpreter.run(
          tree,
          imports,
          alias,
          options.tree as boolean,
          optArgs
        );
      } catch (err) {
        if (err instanceof SyntaxError) {
          console.log(err);
          process.exit(2);
        }
        console.log(err);
        process.exit(1);
      }
    });

  program
    .command("build")
    .argument("<filename>", "filename to run interpreter")
    .option("-t, --tree", "tree flag to show AST", false)
    .action(async function (arg) {
      try {
        const options = program.opts();

        const buffer = await utils.fs.read(arg);
        const builder = new Builder(buffer);
        const ops = await builder.run(options.tree as boolean);
        ops.forEach((op, idx) => {
          console.log(`${idx}: ${op}`);
        });
      } catch (err) {
        if (err instanceof SyntaxError) {
          console.log(err.message);
          process.exit(2);
        }
        console.log((err as Error).message);
        process.exit(1);
      }
    });

  program.parse(process.argv);
}

export default run;
