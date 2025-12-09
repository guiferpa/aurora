package evm

// https://www.evm.codes/

const (
	OpStop         byte = iota + 0x0 // Halts execution
	OpAdd                            // Addition operation
	OpMul                            // Multiplication operation
	OpSub                            // Subtraction operation
	OpDiv                            // Integer division operation
	OpSignedDiv                      // Signed integer division operation (truncated)
	OpMod                            // Modulo remainder operation
	OpSignedMod                      // Signed modulo remainder operation
	OpAddMod                         // Modulo addition operation
	OpMulMod                         // Modulo multiplication operation
	OpExp                            // Exponential operation
	OpSignedExtend                   // Extend length of two’s complement signed integer
)

const (
	OpLessThan          byte = iota + 0x10 // Less-than comparison
	OpGreaterThan                          // Greater-than comparison
	OpSignedLessThan                       // Signed less-than comparison
	OpSignedGreaterThan                    // Signed greater-than comparison
	OpEqual                                // Equality comparison
	OpIsZero                               // Is-zero comparison
	OpAnd                                  // Bitwise AND operation
	OpOr                                   // Bitwise OR operation
	OpXor                                  // Bitwise XOR operation
	OpNot                                  // Bitwise NOT operation
	OpByte                                 // Retrieve single byte from word
	OpShiftLeft                            // Left shift operation
	OpShiftRight                           // Logical right shift operation
	OpShiftAritRight                       // Arithmetic (signed) right shift operation
)

const OpKECCAK256 byte = 0x20 // Compute Keccak-256 hash

const (
	OpAddress        byte = iota + 0x30 // Get address of currently executing account
	OpBalance                           // Get balance of the given account
	OpOrigin                            // Get execution origination address
	OpCaller                            // Get caller address
	OpCallValue                         // Get deposited value by the instruction/transaction responsible for this execution
	OpCallDataLoad                      // Get input data of current environment
	OpCallDataSize                      // Get size of input data in current environment
	OpCallDataCopy                      // Copy input data in current environment to memory
	OpCodeSize                          // Get size of code running in current environment
	OpCodeCopy                          // Copy code running in current environment to memory
	OpGasPrice                          // Get price of gas in current environment
	OpExtCodeSize                       // Get size of an account’s code
	OpExtCodeCopy                       // Copy an account’s code to memory
	OpReturnDataSize                    // Get size of output data from the previous call from the current environment
	OpReturnDataCopy                    // Copy output data from the previous call to memory
	OpExtCodeHash                       // Get hash of an account’s code
	OpBlockHash                         // Get the hash of one of the 256 most recent complete blocks
	OpCoinBase                          // Get the block’s beneficiary address
	OpTimestamp                         // Get the block’s timestamp
	OpNumber                            // Get the block’s number
	OpPrevRandao                        // Get the block’s difficulty
	OpGasLimit                          // Get the block’s gas limit
	OpChainId                           // Get the chain ID
	OpSelfBalance                       // Get balance of currently executing account
	OpBaseFee                           // Get the base fee
	OpBlobHash                          // Get versioned hashes
	OpBlobBaseFee                       // Returns the value of the blob base-fee of the current block
)

const (
	OpPop            byte = iota + 0x50 // Remove item from stack
	OpMemoryLoad                        // Load word from memory
	OpMemoryStore                       // Save word to memory
	OpMemoryStore8                      // Save byte to memory
	OpStorageLoad                       // Load word from storage
	OpStorageStore                      // Save word to storage
	OpJump                              // Alter the program counter
	OpJumpIf                            // Conditionally alter the program counter
	OpProgramCounter                    // Get the value of the program counter prior to the increment corresponding to this instruction
	OpMemorySize                        // Get the size of active memory in bytes
	OpGas                               // Get the amount of available gas, including the corresponding reduction for the cost of this instruction
	OpJumpDestiny                       // Mark a valid destination for jumps
	OpTransientLoad                     // Load word from transient storage
	OpTransientStore                    // Save word to transient storage
	OpMemoryCopy                        // Copy memory areas
	OpPush0                             // Place value 0 on stack
	OpPush1                             // Place value 1 on stack
	OpPush2                             // Place value 2 on stack
	OpPush3                             // Place value 3 on stack
	OpPush4                             // Place value 4 on stack
	OpPush5                             // Place value 5 on stack
	OpPush6                             // Place value 6 on stack
	OpPush7                             // Place value 7 on stack
	OpPush8                             // Place value 8 on stack
	OpPush9                             // Place value 9 on stack
	OpPush10                            // Place value 10 on stack
	OpPush11                            // Place value 11 on stack
	OpPush12                            // Place value 12 on stack
	OpPush13                            // Place value 13 on stack
	OpPush14                            // Place value 14 on stack
	OpPush15                            // Place value 15 on stack
	OpPush16                            // Place value 16 on stack
	OpPush17                            // Place value 17 on stack
	OpPush18                            // Place value 18 on stack
	OpPush19                            // Place value 19 on stack
	OpPush20                            // Place value 20 on stack
	OpPush21                            // Place value 21 on stack
	OpPush22                            // Place value 22 on stack
	OpPush23                            // Place value 23 on stack
	OpPush24                            // Place value 24 on stack
	OpPush25                            // Place value 25 on stack
	OpPush26                            // Place value 26 on stack
	OpPush27                            // Place value 27 on stack
	OpPush28                            // Place value 28 on stack
	OpPush29                            // Place value 29 on stack
	OpPush30                            // Place value 30 on stack
	OpPush31                            // Place value 31 on stack
	OpPush32                            // Place value 32 on stack
)

const OpReturn byte = 0xf3 // Halt execution returning output data from the last call
