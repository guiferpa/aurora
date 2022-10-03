function buildPadding(value: number): string {
  return [...Array(value).keys()].map(() => " ").join("");
}

export class LexerError extends Error {
  constructor (
    public readonly message: string
  ) {
    super(message);
  }
}

export class MissingSemicolonLexerError extends LexerError {
  constructor(
    public readonly filename: string,
    public readonly numberOfLine: number,
    public readonly contentOfLine: string,
  ) {
    const numberOfColumn: number = contentOfLine.length;
    const identifier: string = `${filename}:${numberOfLine}:${numberOfColumn} - `;
    const error: string = 'Missing semicolon: '
    super(`${identifier}${error}${contentOfLine}\n${buildPadding(identifier.length + error.length + contentOfLine.length)}^`);
  }
}

export type Token = string[];

export default class Lexer {
  public analyze(filename: string, content: string): Token[] {
    return content.split('\n').map((line, numberOfLine) => {
      const indexOfSemicolon: number = line.indexOf(';');
      if (indexOfSemicolon < 0) {
        throw new MissingSemicolonLexerError(filename, numberOfLine + 1, line);
      }

      if (indexOfSemicolon !== (line.length - 1)) {
        throw new LexerError("");
      }

      return [`${numberOfLine + 1}`, "", line];
    });
  }
}