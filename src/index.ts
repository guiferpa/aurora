import {Command} from "commander";

import pkg from "../package.json";

import {Interpreter} from "@/interpreter";
import {Compiler} from "@/compiler";
import {read, write} from "@/fsutil";
import {repl} from "@/repl";

function run() {
  const program = new Command();

  program
    .name(pkg.name)
    .description(pkg.repository)
    .version(pkg.version);

  program
    .option('-t, --tree', 'tree flag to show AST', false)
    .action(function () {
      const options = program.opts();

      const r = repl();
      const interpreter = new Interpreter();

      r.on('line', function (chunk) {
        interpreter.write(Buffer.from(chunk));
        console.log(`= ${interpreter.run(options.tree as boolean)}`);
        r.prompt(true);
      });

      r.once('close', () => {
        console.log('Bye :)');
        process.exit(0);
      });
    });

  program
    .command('run')
    .argument('<filename>', 'filename to run interpreter')
    .option('-t, --tree', 'tree flag to show AST', false)
    .action(async function (arg) {
      const options = program.opts();

      const buffer = await read(arg);
      const interpreter = new Interpreter(buffer);
      console.log(`= ${interpreter.run(options.tree as boolean)}`);
    });

  program
    .command('build')
    .argument('<filename>', 'entrypoint to build source code')
    .option('-t, --tree', 'tree flag to show AST', false)
    .action(async function (arg) {
      const options = program.opts();

      const buffer = await read(arg);
      const compiler = new Compiler(buffer);
      const content = compiler.compile(options.tree as boolean);
      await write("out", content);
    });

  program.parse(process.argv);
}

export default run;

