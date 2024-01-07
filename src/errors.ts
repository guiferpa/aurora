import { LexerError } from "@/lexer";
import { EaterError } from "@/eater";
import { EnvironError } from "@/environ";
import { EvaluateError, InterpreterError } from "@/interpreter";
import { ParserError } from "@/parser";
import { SymtableError } from "@/symtable";

export function handle(err: Error) {
  if (err instanceof LexerError) {
    console.log(`![LexerError]: ${err.message}`);
    return;
  }
  if (err instanceof SymtableError) {
    console.log(`![SymtableError]: ${err.message}`);
    return;
  }
  if (err instanceof ParserError) {
    console.log(`![ParserError]: ${err.message}`);
    return;
  }
  if (err instanceof EnvironError) {
    console.log(`![EnvironError]: ${err.message}`);
    return;
  }
  if (err instanceof InterpreterError) {
    console.log(`![InterpreterError]: ${err.message}`);
    return;
  }
  if (err instanceof EvaluateError) {
    console.log(`![EvaluateError]: ${err.message}`);
    return;
  }
  if (err instanceof EaterError) {
    console.log(`![EaterError]: ${err.message}`);
    return;
  }
  console.log(`![Error]: ${err.message}`);
}
