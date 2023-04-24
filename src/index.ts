import {Command} from "commander";

import pkg from "../package.json";
import {Interpreter, repl, read} from "./v1";

function run() {
  const program = new Command();

  program
    .name(pkg.name)
    .description(pkg.repository)
    .version(pkg.version);

  program
    .option('-d, --debug', 'debug flag to show AST', false)
    .action(function () {
      const options = program.opts();

      const r = repl();

      r.on('line', function (chunk) {
        const interpreter = new Interpreter(Buffer.from(chunk));
        console.log(`= ${interpreter.run(options.debug as boolean)}`);
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
    .option('-d, --debug', 'debug flag to show AST', false)
    .action(async function (arg) {
      const options = program.opts();

      const buffer = await read(arg);
      const interpreter = new Interpreter(buffer);
      console.log(`= ${interpreter.run(options.debug as boolean)}`);
      return
    });

  program.parse(process.argv);
}

export default run;

