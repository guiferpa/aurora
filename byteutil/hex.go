package byteutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
)

func ToHex(bs []byte) string {
	return hex.EncodeToString(bs)
}

func ToUpperHex(bs []byte) string {
	return strings.ToUpper(hex.EncodeToString(bs))
}

func ToHexBloom(bs []byte) string {
	nbs := bytes.NewBufferString("")
	for i := 0; i < len(bs); i++ {
		if i > 0 {
			nbs.WriteString(" ")
		}
		nbs.WriteString(fmt.Sprintf("%02X", bs[i]))
	}
	return nbs.String()
}
