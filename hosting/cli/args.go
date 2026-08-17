package cli

import (
	"math/big"
	"strings"

	"github.com/guiferpa/aurora/byteutil"
)

// ParseArgs encodes each argument as a 32-byte ABI word, inferring type from the string:
// - bool: "true" / "false" (case-insensitive) → 0 or 1 right-padded to 32 bytes
// - number: decimal ("42") or hex ("0x2a") → uint256 big-endian
// - string: anything else; use "" for empty string; quoted strings have quotes stripped
func ParseArgs(args []string) []byte {
	data := make([]byte, 0)
	for _, arg := range args {
		data = append(data, parseArg(arg)...)
	}
	return data
}

func parseArg(arg string) []byte {
	// bool
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "true":
		return byteutil.Padding32Bytes([]byte{1})
	case "false":
		return byteutil.Padding32Bytes([]byte{0})
	}
	// number (decimal or 0x-prefixed hex)
	if n := parseNumber(arg); n != nil {
		b := make([]byte, 32)
		nb := n.Bytes()
		if len(nb) > 32 {
			copy(b, nb[len(nb)-32:])
		} else {
			copy(b[32-len(nb):], nb)
		}
		return b
	}
	// string (strip surrounding double quotes; "" → empty)
	s := strings.TrimSpace(arg)
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && len(s) >= 2 {
		s = s[1 : len(s)-1]
	}
	return byteutil.Padding32Bytes([]byte(s))
}

func parseNumber(s string) *big.Int {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n := new(big.Int)
		if _, ok := n.SetString(s[2:], 16); !ok {
			return nil
		}
		return n
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return nil
	}
	return n
}
