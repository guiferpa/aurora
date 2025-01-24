package byteutil

type ErrEncode struct{}

func (err *ErrEncode) Error() string {
	return "unknown byte sequence to encode"
}

func Encode(v []byte) (any, error) {
	if len(v) == 8 {
		return ToUint64(v), nil
	}
	if len(v)%8 == 0 {
		r := make([]uint64, 0)
		i := 0
		for i < len(v) {
			r = append(r, ToUint64(v[i:i+8]))
			i += 8
		}
		return r, nil
	}
	if len(v) == 1 {
		return ToBoolean(v), nil
	}
	return nil, &ErrEncode{}
}
