package emitter

const (
	OpMultiply    byte = 0x01 // Multiply two numbers with max of 64 bits (uint64)
	OpAdd              = 0x02 // Sum two numbers with max of 64 bits (uint64)
	OpSubstract        = 0x03 // Substract two numbers with max of 64 bits (uint64)
	OpDivide           = 0x04 // Divide two numbers with max of 64 bits (uint64)
	OpExponential      = 0x05 // Exponential numbers with max of 64 bits (uint64)
	OpIdent            = 0x06 // Identify a definition from scope where evaluate step
	OpSave             = 0x07 // Save value with max of 64 bits (uint64) to temporary storage in instructions
	OpLoad             = 0x08 // Load value with max of 64 bits (uint64) from temporary storage in instructions
	OpDiff             = 0x09 // Operation to compare if two numbers with max of 64 bits (uint64) are different between themself
	OpEquals           = 0x0a // Operation to compare if two numbers with max of 64 bits (uint64) are equals between themself
	OpBigger           = 0x0b // Operation to decide between two numbers with 64 bits (uint64) which one is bigger than other
	OpSmaller          = 0x0c // Operation to decide between two numbers with 64 bits (uint64) which one is smaller than other
	OpAnd              = 0x0d // Operation to decide the AND behavior logically between two boolean values (1 bit)
	OpOr               = 0x0e // Operation to decide the OR behavior logically between two boolean values (1 bit)
	OpPushArg          = 0x0f // Operation to push arguments to next scope
	OpGetArg           = 0x10 // Operation to get arguments from higher scopes
	OpBeginScope       = 0x11 // Starts a new scope in stack evaluate time
	OpEndScope         = 0x12 // Ends the current scope started in stack evalute time
	OpPreCall          = 0x13 // It's a pre call operation, main goal is push all scope arguments
	OpCall             = 0x14 // Call a scope with paramters, it'll works like function
	OpIfNot            = 0x15 // Operation logical to decide an negative condition
	OpJump             = 0x16 // Operation just for jump to another instruction
	OpReturn           = 0x17 // Operation to save an value with max of 64 bits (uint64) to work throught by different scopes
	OpResult           = 0x18 // Operation to get the result persisted in return stack
	OpPrint            = 0x19 // Operation to print slice of bytes
)
