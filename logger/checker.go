package logger

import "slices"

func IsLexerLogger(loggers []string) bool {
	return slices.Contains(loggers, "lexer")
}

func IsEmitterLogger(loggers []string) bool {
	return slices.Contains(loggers, "emitter")
}

func IsParserLogger(loggers []string) bool {
	return slices.Contains(loggers, "parser")
}

func IsEvaluatorLogger(loggers []string) bool {
	return slices.Contains(loggers, "evaluator")
}
