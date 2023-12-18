import { Command } from "commander";

import pkg from "../package.json";

import { Interpreter } from "@/interpreter";
import { read } from "@/fsutil";
import { repl } from "@/repl";
import { Builder } from "./builder";

function run() {
  const program = new Command();

  program.name(pkg.name).description(pkg.repository).version(pkg.version);

  program
    .option("-t, --tree", "tree flag to show AST", false)
    .action(function () {
      const options = program.opts();

      const r = repl();
      const interpreter = new Interpreter();

      r.on("line", function (chunk) {
        interpreter.write(Buffer.from(chunk));
        console.log(`= ${interpreter.run(options.tree as boolean)}`);
        r.prompt(true);
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
    .action(async function (arg) {
      try {
        const options = program.opts();

        const buffer = await read(arg);
        const interpreter = new Interpreter(buffer);
        interpreter.run(options.tree as boolean);
      } catch (err) {
        if (err instanceof SyntaxError) {
          console.log(err.message);
          process.exit(2);
        }
        console.log((err as Error).message);
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

        const buffer = await read(arg);
        const builder = new Builder(buffer);
        const ops = builder.run(options.tree as boolean);
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
