package byteutil

var (
	False = []byte{0}
	True  = []byte{1}
)

func ToBoolean(bs []byte) bool {
	if len(bs) < 1 {
		return false
	}
	v := ToUint64(Padding64Bits(bs))
	return v > 0
}
