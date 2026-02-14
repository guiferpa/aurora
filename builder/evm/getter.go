package evm

func GetRuntimeCodeLength(rc *RuntimeCode) int {
	l := 0
	// Dispatcher block (selector checks + no-match STOP) is written before body code.
	if len(rc.Dispatchers) > 0 {
		l += DISPATCHER_BYTES_SIZE*len(rc.Dispatchers) + NO_MATCH_DISPATCHER_SIZE
	}
	for _, r := range rc.Dispatchers {
		l += r.Code.Len()
	}
	if rc.Root != nil {
		l += rc.Root.Len()
	}
	return l
}

// GetCalldataArgsOffset returns the calldata byte offset for the Nth argument (0-based).
// ABI layout: selector at 0, then each arg in a 32-byte slot: arg0 at 0x20, arg1 at 0x40, arg2 at 0x60, ...
func GetCalldataArgsOffset(index uint64) byte {
	return byte(CALLDATA_SLOT_READABLE * (index + 1))
}
