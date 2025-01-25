package evm

// https://www.evm.codes/

const (
	OpStop              byte = iota // Halts execution
	OpAdd                           // Addition operation
	OpMul                           // Multiplication operation
	OpSub                           // Subtraction operation
	OpDiv                           // Integer division operation
	OpSDiv                          // Signed integer division operation (truncated)
	OpMod                           // Modulo remainder operation
	OpSMod                          // Signed modulo remainder operation
	OpAddMod                        // Modulo addition operation
	OpMulMod                        // Modulo multiplication operation
	OpExp                           // Exponential operation
	OpSignExtend                    // Extend length of two’s complement signed integer
	OpLessThan                      // Less-than comparison
	OpGreaterThan                   // Greater-than comparison
	OpSignedLessThan                // Signed less-than comparison
	OpSignedGreaterThan             // Signed greater-than comparison
	OpEqual                         // Equality comparison
	OpIsZero                        // Is-zero comparison
	OpAnd                           // Bitwise AND operation
	OpOr                            // Bitwise OR operation
	OpXor                           // Bitwise XOR operation
	OpNot                           // Bitwise NOT operation
	OpByte                          // Retrieve single byte from word
	OpShiftLeft                     // Left shift operation
	OpShiftRight                    // Logical right shift operation
	OpShiftAritRight                // Arithmetic (signed) right shift operation
	OpKECCAK256                     // Compute Keccak-256 hash
	OpAddress                       // Get address of currently executing account
	OpBalance                       // Get balance of the given account
	OpOrigin                        // Get execution origination address
	OpCaller                        // Get caller address
	OpCallValue                     // Get deposited value by the instruction/transaction responsible for this execution
	OpCallDataLoad                  // Get input data of current environment
)
