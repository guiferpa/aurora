import fs, { ReadStream } from 'fs';

import Lexer from './lexer';

const lexer = new Lexer();

describe('Statment variable', () => {
  test('Declare variable', () => {
    const content = "";
    lexer.analyze("", content);
  });
});