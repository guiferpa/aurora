import {Interpreter, repl, read} from "./v1";

export async function run(args: string[]) {
  if (args.length > 0) {
    const buffer = await read(args);
    const interpreter = new Interpreter(buffer);
    console.log(`= ${interpreter.run()}`);
    return
  }

  const r = repl();

  r.on('line', function (chunk) {
    const interpreter = new Interpreter(Buffer.from(chunk));
    console.log(`= ${interpreter.run()}`);
    r.prompt(true);
  });

  r.once('close', () => {
    console.log('Bye :)');
    process.exit(0);
  });
}

