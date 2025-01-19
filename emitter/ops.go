package emitter

const (
	OpMultiply    byte = 0x01 // Multiply two numbers with max of 64 bits (uint64)
	OpAdd         byte = 0x02 // Sum two numbers with max of 64 bits (uint64)
	OpSubtract    byte = 0x03 // Subtract two numbers with max of 64 bits (uint64)
	OpDivide      byte = 0x04 // Divide two numbers with max of 64 bits (uint64)
	OpExponential byte = 0x05 // Exponential numbers with max of 64 bits (uint64)
	OpIdent       byte = 0x06 // Identify a definition from scope where evaluate step
	OpSave        byte = 0x07 // Save value with max of 64 bits (uint64) to temporary storage in instructions
	OpLoad        byte = 0x08 // Load value with max of 64 bits (uint64) from temporary storage in instructions
	OpDiff        byte = 0x09 // Operation to compare if two numbers with max of 64 bits (uint64) are different between themself
	OpEquals      byte = 0x0a // Operation to compare if two numbers with max of 64 bits (uint64) are equals between themself
	OpBigger      byte = 0x0b // Operation to decide between two numbers with 64 bits (uint64) which one is bigger than other
	OpSmaller     byte = 0x0c // Operation to decide between two numbers with 64 bits (uint64) which one is smaller than other
	OpAnd         byte = 0x0d // Operation to decide the AND behavior logically between two boolean values (1 bit)
	OpOr          byte = 0x0e // Operation to decide the OR behavior logically between two boolean values (1 bit)
	OpPushArg     byte = 0x0f // Operation to push arguments to next scope
	OpGetArg      byte = 0x10 // Operation to get arguments from higher scopes
	OpBeginScope  byte = 0x11 // Starts a new scope in stack evaluate time
	OpEndScope    byte = 0x12 // Ends the current scope started in stack evalute time
	OpPreCall     byte = 0x13 // It's a pre call operation, main goal is push all scope arguments
	OpCall        byte = 0x14 // Call a scope with parameters, it'll works like function
	OpIf          byte = 0x15 // Operation logical to decide an condition
	OpJump        byte = 0x16 // Operation just for jump to another instruction
	OpReturn      byte = 0x17 // Operation to save an value with max of 64 bits (uint64) to work thought by different scopes
	OpResult      byte = 0x18 // Operation to get the result persisted in return stack
	OpPrint       byte = 0x19 // Operation to print slice of bytes
	OpAppend      byte = 0x1a // Operation to append value in slice of byte
	OpHead        byte = 0x1b // Operation to head values in slice of byte
	OpTail        byte = 0x1c // Operation to tail values in slice of byte
)
