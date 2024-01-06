import colorize from "json-colorizer";
import { Command } from "commander";

import pkg from "../package.json";

import Lexer from "@/lexer";
import Eater from "@/eater";
import SymTable from "@/symtable";
import Parser from "@/parser";
import Importer, { ImportClaim } from "@/importer";
import Interpreter from "@/interpreter";
import Builder from "@/builder";

import * as utils from "@/utils";
import { repl } from "@/repl";

import * as errors from "./errors";

function run() {
  const reader = { read: utils.fs.read };

  const program = new Command();

  program.name(pkg.name).description(pkg.repository).version(pkg.version);

  program
    .option("-t, --tree", "tree flag to show AST", false)
    .option("-a, --args <string>", "program's arguments", "")
    .action(async function () {
      const options = program.opts();

      const args = options.args.split(",");

      const context = "repl";

      const r = repl();

      const interpreter = new Interpreter();

      r.on("line", async function (chunk) {
        const buffer = Buffer.from(chunk);
        const lexer = new Lexer(buffer);

        const symtable = new SymTable("global");
        const eater = new Eater(context, lexer.copy());
        const parser = new Parser(eater, symtable);

        const importer = new Importer(reader);
        try {
          const claims = await importer.imports(eater);
          const alias = importer.alias(claims);
          const imports = new Map<string, ImportClaim>(
            claims.map((claim) => [claim.context, claim])
          );

          const tree = await parser.parse();
          if (options.tree as boolean)
            console.log(colorize(JSON.stringify(tree, null, 2)));

          const result = await interpreter.run(
            context,
            tree,
            imports,
            alias,
            args
          );
          console.log(`= ${result}`);
        } catch (err) {
          errors.handle(err as Error);
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
    .argument("[args...]", "program's arguments")
    .option("-t, --tree", "tree flag to show AST", false)
    .action(async function (filename, args) {
      try {
        const options = program.opts();

        const buffer = await utils.fs.read(filename);
        const lexer = new Lexer(buffer);
        const eater = new Eater(filename, lexer.copy());

        const importer = new Importer(reader);
        const claims = await importer.imports(eater);
        const alias = importer.alias(claims);
        const imports = new Map<string, ImportClaim>(
          claims.map((claim) => [claim.context, claim])
        );

        const symtable = new SymTable("global");
        const parser = new Parser(new Eater(filename, lexer.copy()), symtable);
        const tree = await parser.parse();
        if (options.tree as boolean)
          console.log(colorize(JSON.stringify(tree, null, 2)));

        const interpreter = new Interpreter();
        await interpreter.run(filename, tree, imports, alias, args);
      } catch (err) {
        errors.handle(err as Error);
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
