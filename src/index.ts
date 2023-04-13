import fs from "fs";
import rl from "readline";

import Lexer from "./lexer";
import OperatorEvaluator from "./operator-evaluator";
import OperatorParser from "./operator-parser";

export async function read(args: string[]): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    let buffer = Buffer.from("");

    const reader = fs.createReadStream(args[0]);

    reader.on('data', (chunk: Buffer) => {
      buffer = Buffer.concat([buffer, chunk]);
    });

    reader.on('error', (err) => {
      reject(err);
    });

    reader.on('close', () => {
      resolve(buffer);
    });
  });
}

export function runInterpret(buffer: Buffer): string {
  const lexer = new Lexer(buffer); // Tokenizer
  const parser = new OperatorParser(lexer);
  const ast = parser.parse();
  const result = OperatorEvaluator.eval(ast);
  return `${result}`;
}

export async function run(args: string[]) {
  if (args.length > 0) {
    const buffer = await read(process.argv.slice(2));
    const result = runInterpret(buffer);
    console.log(result);
    return
  }

  const repl = rl.createInterface({
    input: process.stdin,
    output: process.stdout
  });

  repl.on('line', (chunk) => {
    const out = runInterpret(Buffer.from(chunk));
    console.log(`= ${out}`);
  });

  repl.once('close', () => {
    console.log('Bye :)');
    process.exit(0);
  });
}

