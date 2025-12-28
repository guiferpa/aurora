package evaluator

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
)

func ResolveAny(v any) string {
	if cv, ok := v.([]byte); ok {
		if len(cv) == 0 {
			return "-"
		}
		pcv := byteutil.Padding64Bits(cv)
		return fmt.Sprintf("%v", byteutil.ToUint64(pcv))
	}
	if cv, ok := v.(string); ok {
		return cv
	}
	if cv, ok := v.(uint64); ok {
		return fmt.Sprintf("%v", cv)
	}
	if cv, ok := v.(bool); ok {
		return fmt.Sprintf("%v", cv)
	}
	return "-"
}
