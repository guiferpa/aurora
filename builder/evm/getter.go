package evm

func GetRuntimeCodeLength(rc *RuntimeCode) int {
	// Every call opens by saying where its frame begins.
	l := FRAME_START_SIZE
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

// GetCalldataArgsOffset answers where the Nth value applied to a scope sits in the calldata,
// counting from zero.
//
// The selector is first and each value has a word after it, so the eighth sits at 256 — which
// is why this answers a number and not a byte. It used to answer a byte, and 256 in a byte is
// zero: the eighth value read the selector, the ninth read the first, and the contract answered
// something rather than failing.
func GetCalldataArgsOffset(index uint64) int {
	return CALLDATA_SLOT_READABLE * int(index+1)
}
