package byteutil

type ErrEncode struct{}

func (err *ErrEncode) Error() string {
	return "unknown byte sequence to encode"
}

// Encode renders bytes for display: a single tape becomes its decimal value, and a run of
// tapes (a reel) becomes one value per tape.
//
// There is no boolean case any more. Every value is a tape, so true is indistinguishable
// from 1 and showing it otherwise would be inventing a type the language does not have.
func Encode(v []byte, size int) (any, error) {
	size = TapeSize(size)
	if len(v) == 0 || len(v)%size != 0 {
		return nil, &ErrEncode{}
	}
	if len(v) == size {
		return ToUint256(v, size).Dec(), nil
	}
	values := make([]string, 0, len(v)/size)
	for i := 0; i < len(v); i += size {
		values = append(values, ToUint256(v[i:i+size], size).Dec())
	}
	return values, nil
}
