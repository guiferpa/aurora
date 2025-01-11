package byteutil

import "errors"

func Encode(v []byte) (any, error) {
	if len(v) == 8 {
		return ToUint64(v), nil
	}
	if len(v) == 1 {
		return ToBoolean(v), nil
	}
	return nil, errors.New("unknown byte sequence to encode")
}
