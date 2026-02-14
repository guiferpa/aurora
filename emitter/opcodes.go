package emitter

const (
	OpMultiply    byte = iota + 0b1 // Multiply two numbers with max of 64 bits (uint64)
	OpAdd                           // Sum two numbers with max of 64 bits (uint64)
	OpSubtract                      // Subtract two numbers with max of 64 bits (uint64)
	OpDivide                        // Divide two numbers with max of 64 bits (uint64)
	OpExponential                   // Exponential numbers with max of 64 bits (uint64)
	OpIdent                         // Identify a definition from scope where evaluate step
	OpSave                          // Save value with max of 64 bits (uint64) to temporary storage in instructions
	OpLoad                          // Load value with max of 64 bits (uint64) from temporary storage in instructions
	OpDiff                          // Operation to compare if two numbers with max of 64 bits (uint64) are different between themself
	OpEquals                        // Operation to compare if two numbers with max of 64 bits (uint64) are equals between themself
	OpBigger                        // Operation to decide between two numbers with 64 bits (uint64) which one is bigger than other
	OpSmaller                       // Operation to decide between two numbers with 64 bits (uint64) which one is smaller than other
	OpAnd                           // Operation to decide the AND behavior logically between two boolean values (1 bit)
	OpOr                            // Operation to decide the OR behavior logically between two boolean values (1 bit)
	OpPushArg                       // Operation to push arguments to next scope
	OpGetArg                        // Operation to get arguments from higher scopes
	OpBeginScope                    // Starts a new scope in stack evaluate time
	OpDefer                         // Defer: store scope range in temp, skip body (value = pointer to scope)
	OpPreCall                       // It's a pre call operation, main goal is push all scope arguments
	OpCall                          // Call a scope with parameters, it'll works like function
	OpIf                            // Operation logical to decide an condition
	OpJump                          // Operation just for jump to another instruction
	OpReturn                        // Operation to save an value with max of 64 bits (uint64) to work thought by different scopes
	OpResult                        // Operation to get the result persisted in return stack
	OpPrint                         // Operation to print slice of bytes
	OpPull                          // Operation to pull value in slice of byte
	OpHead                          // Operation to head values in slice of byte
	OpTail                          // Operation to tail values in slice of byte
	OpPush                          // Operation to push values in slice of byte
	OpAssert                        // Operation to assert a condition in tests
	OpEcho                          // Operation to echo bytes as text
)
