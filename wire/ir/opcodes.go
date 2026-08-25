package ir

// What each instruction does, beside the name it does it under.
//
// A value in Aurora is a tape: a run of bytes, tape_size wide, between one and thirty-two.
// Arithmetic wraps at that width, which is why nothing here talks about 64 bits — it used
// to, from before the width was something a program chose.
//
// Each line says what the instruction takes and what it leaves. What it takes is written in
// its two operands, and the kind of each says how to read it: Ref is a value another
// instruction left behind, Imm is the bytes themselves, Name is something a scope bound,
// Target is where control goes, Text is written for a person. Every instruction leaves
// exactly one value, because everything in Aurora is an expression.
const (
	// Arithmetic. Two values, and the result wrapped to the tape width: at one byte,
	// 255 + 1 is 0.
	OpMultiply    byte = iota + 0b1 // Ref, Ref -> their product
	OpAdd                           // Ref, Ref -> their sum
	OpSubtract                      // Ref, Ref -> the second taken from the first
	OpDivide                        // Ref, Ref -> the first divided by the second
	OpExponential                   // Ref, Ref -> the first raised to the second

	// Names and values.
	OpIdent // Name, Ref -> binds the name to the value, and leaves the neutral tape
	OpSave  // Imm -> the bytes themselves, which is how a literal reaches the program
	OpLoad  // Name -> what the name is bound to, looked up where the program is running

	// Comparison. What it returns is a tape like any other: false is all zeros.
	OpDiff    // Ref, Ref -> whether the two differ
	OpEquals  // Ref, Ref -> whether the two are the same
	OpBigger  // Ref, Ref -> whether the first is greater
	OpSmaller // Ref, Ref -> whether the first is smaller

	// Logic. Neither short-circuits: both operands are evaluated before the operation runs.
	OpAnd // Ref, Ref -> both hold
	OpOr  // Ref, Ref -> either holds

	// Arguments. Executing is applying a vector of values to a scope, so there is no arity
	// and no parameter list — a position is written, and a position is read.
	OpGetFeed // Imm -> the value at that position of the vector applied to this scope, or a
	//            tape of zeros where nothing was applied

	OpCall // Name -> runs the scope that name reaches, and leaves what it returned

	// Printing. Three readings of one tape, and the whole difference between them is which
	// opcode it is. They are logs, and they produce no bytecode, by decision.
	OpPrintBytes   // Ref -> writes the bytes, and leaves the value
	OpPrintChars   // Ref -> writes those bytes as UTF-8 text, and leaves the value
	OpPrintDecimal // Ref -> writes the number they spell, and leaves the value

	// Tapes, which behave as shift registers. head and tail take their index modulo the
	// width, so it is never out of bounds.
	OpPull // Ref, Ref -> the tape shifted left, the value entering at the right
	OpHead // Ref, Imm -> the first n significant bytes of the tape
	OpTail // Ref, Imm -> the tape with its first n significant bytes dropped
	OpPush // Ref, Ref -> the tape shifted right, the value entering at the left

	// Assertions belong to "aurora test", and are consumed elsewhere.
	OpAssert // Ref, Text -> checks the condition, carrying the message for whoever reads

	// Shapes. A shape is a run of tapes laid end to end, and both of these came with it.
	// Reading past the end gives the neutral value rather than failing.
	OpJoin  // Ref, Ref -> the run with one more tape at its end
	OpField // Ref, Const, Const -> the tape at that index of a run of that many tapes
)
