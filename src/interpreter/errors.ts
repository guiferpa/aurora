export class InterpreterError extends Error {
  constructor(message: string) {
    super(message);
  }
}

export class EvaluateError extends InterpreterError {
  constructor(message: string) {
    super(message);
  }
}
