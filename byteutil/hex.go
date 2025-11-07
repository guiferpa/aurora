package byteutil

import (
	"encoding/hex"
)

func ToHex(bs []byte) string {
	return hex.EncodeToString(bs)
}

