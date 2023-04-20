import fs from "fs";
import rl from "readline";

import {Evaluator, Lexer} from "./v1";
import {Parser} from "./v3";
import {BlockStatmentNode, ParserNode} from "./v3/parser/node";

const DEFAULT_PROMPT = ">> ";

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

export function runInterpret(buffer: Buffer): string[] {
  const lexer = new Lexer(buffer); // Tokenizer
  const parser = new Parser(lexer);
  const tree = parser.parse();

  function evaluate(block: ParserNode[]): any {
    return block.map((stmt) => {
      if (stmt instanceof BlockStatmentNode) {
        return evaluate(stmt.block);
      }
      return `${Evaluator.evaluate(stmt)}`;
    });
  }

  return evaluate(tree.block);
}

export async function run(args: string[]) {
  if (args.length > 0) {
    const buffer = await read(process.argv.slice(2));
    const out = runInterpret(buffer);
    console.log(`= ${out}`);
    return
  }

  const repl = rl.createInterface({
    input: process.stdin,
    output: process.stdout
  });

  repl.setPrompt(DEFAULT_PROMPT);
  repl.prompt(true);

  repl.on('line', (chunk) => {
    const out = runInterpret(Buffer.from(chunk));
    console.log(`= ${out}`);
    repl.prompt(true);
  });

  repl.once('close', () => {
    console.log('Bye :)');
    process.exit(0);
  });
}

