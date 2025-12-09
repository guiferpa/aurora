package evm

import (
	"encoding/hex"
	"strings"
)

func ToString(bs []byte) string {
	return strings.ToUpper(hex.EncodeToString(bs))
}
