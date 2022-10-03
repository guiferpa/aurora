import fs from 'fs';
import util from 'util';

import Lexer from './lexer';

import { Command } from 'commander';

const lexer = new Lexer();

const command = new Command();

command.action(async (str, opts) => {
  try {
    const files: string[] = opts.args.map((arg: string) => arg);
  
    const analized = await Promise.all(files.map(async (file) => {
      const content = await util.promisify(fs.readFile)(file, { encoding: 'utf-8' });
      return [file, lexer.analyze(file, content)];
    }));
    
    console.log(analized.map((item) => JSON.stringify(item)));
  } catch(err) {
    console.error(`${(err as Error).message}`);
  }
});

command.parse();