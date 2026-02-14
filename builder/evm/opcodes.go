package evm

import "github.com/guiferpa/aurora/byteutil"

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

const (
	OpReturn byte = 0xf3 // Halt execution returning output data from the last call
	OpSwap1  byte = 0x90 // Swap 1st and 2nd stack items
)

func ToOpByte(op uint32) []byte {
	return byteutil.NoPadding(byteutil.FromUint32(op))
}

func ResolveOpCode(op byte) string {
	switch op {
	case OpStop:
		return "STOP"
	case OpAdd:
		return "ADD"
	case OpMul:
		return "MUL"
	case OpSub:
		return "SUB"
	case OpDiv:
		return "DIV"
	case OpSignedDiv:
		return "SIGNEDDIV"
	case OpMod:
		return "MOD"
	case OpSignedMod:
		return "SIGNEDMOD"
	case OpAddMod:
		return "ADDMOD"
	case OpMulMod:
		return "MULMOD"
	case OpExp:
		return "EXP"
	case OpSignedExtend:
		return "SIGNEXTEND"
	case OpLessThan:
		return "LESSTHAN"
	case OpGreaterThan:
		return "GREATERTHAN"
	case OpSignedLessThan:
		return "SIGNEDLESSTHAN"
	case OpSignedGreaterThan:
		return "SIGNEDGREATERTHAN"
	case OpEqual:
		return "EQUAL"
	case OpIsZero:
		return "ISZERO"
	case OpAnd:
		return "AND"
	case OpOr:
		return "OR"
	case OpXor:
		return "XOR"
	case OpNot:
		return "NOT"
	case OpByte:
		return "BYTE"
	case OpShiftLeft:
		return "SHIFTLEFT"
	case OpShiftRight:
		return "SHIFTRIGHT"
	case OpShiftAritRight:
		return "SHIFTARITRIGHT"
	case OpKECCAK256:
		return "KECCAK256"

	case OpAddress:
		return "ADDRESS"
	case OpBalance:
		return "BALANCE"
	case OpOrigin:
		return "ORIGIN"
	case OpCaller:
		return "CALLER"
	case OpCallValue:
		return "CALLVALUE"
	case OpCallDataLoad:
		return "CALLDATALOAD"
	case OpCallDataSize:
		return "CALLDATASIZE"
	case OpCallDataCopy:
		return "CALLDATACOPY"
	case OpCodeSize:
		return "CODESIZE"
	case OpCodeCopy:
		return "CODECOPY"
	case OpGasPrice:
		return "GASPRICE"
	case OpExtCodeSize:
		return "EXTCODESIZE"
	case OpExtCodeCopy:
		return "EXTCODECOPY"
	case OpReturnDataSize:
		return "RETURNDATASIZE"
	case OpReturnDataCopy:
		return "RETURNDATACOPY"
	case OpExtCodeHash:
		return "EXTHEXHASH"
	case OpBlockHash:
		return "BLOCKHASH"
	case OpCoinBase:
		return "COINBASE"
	case OpTimestamp:
		return "TIMESTAMP"
	case OpNumber:
		return "NUMBER"
	case OpPrevRandao:
		return "PREVRANDAO"
	case OpGasLimit:
		return "GASLIMIT"
	case OpChainId:
		return "CHAINID"
	case OpSelfBalance:
		return "SELFBALANCE"
	case OpBaseFee:
		return "BASEFEE"
	case OpBlobHash:
		return "BLOBHASH"
	case OpBlobBaseFee:
		return "BLOBBASEFEE"
	case OpPop:
		return "POP"
	case OpMemoryLoad:
		return "MEMORYLOAD"
	case OpMemoryStore:
		return "MEMORYSTORE"
	case OpMemoryStore8:
		return "MEMORYSTORE8"
	case OpStorageLoad:
		return "STORAGELOAD"
	case OpStorageStore:
		return "STORAGESTORE"
	case OpJump:
		return "JUMP"
	case OpJumpIf:
		return "JUMPIF"
	case OpProgramCounter:
		return "PROGRAMCOUNTER"
	case OpMemorySize:
		return "MEMORYSIZE"
	case OpGas:
		return "GAS"
	case OpJumpDestiny:
		return "JUMPDESTINY"
	case OpTransientLoad:
		return "TRANSIENTLOAD"
	case OpTransientStore:
		return "TRANSIENTSTORE"
	case OpMemoryCopy:
		return "MEMORYCOPY"
	case OpPush0:
		return "PUSH0"
	case OpPush1:
		return "PUSH1"
	case OpPush2:
		return "PUSH2"
	case OpPush3:
		return "PUSH3"
	case OpPush4:
		return "PUSH4"
	case OpPush5:
		return "PUSH5"
	case OpPush6:
		return "PUSH6"
	case OpPush7:
		return "PUSH7"
	case OpPush8:
		return "PUSH8"
	case OpPush9:
		return "PUSH9"
	case OpPush10:
		return "PUSH10"
	case OpPush11:
		return "PUSH11"
	case OpPush12:
		return "PUSH12"
	case OpPush13:
		return "PUSH13"
	case OpPush14:
		return "PUSH14"
	case OpPush15:
		return "PUSH15"
	case OpPush16:
		return "PUSH16"
	case OpPush17:
		return "PUSH17"
	case OpPush18:
		return "PUSH18"
	case OpPush19:
		return "PUSH19"
	case OpPush20:
		return "PUSH20"
	case OpPush21:
		return "PUSH21"
	case OpPush22:
		return "PUSH22"
	case OpPush23:
		return "PUSH23"
	case OpPush24:
		return "PUSH24"
	case OpPush25:
		return "PUSH25"
	case OpPush26:
		return "PUSH26"
	case OpPush27:
		return "PUSH27"
	case OpPush28:
		return "PUSH28"
	case OpPush29:
		return "PUSH29"
	case OpPush30:
		return "PUSH30"
	case OpPush31:
		return "PUSH31"
	case OpPush32:
		return "PUSH32"
	case OpReturn:
		return "RETURN"
	case OpSwap1:
		return "SWAP1"
	}
	return "Unknown"
}
