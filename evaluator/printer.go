package evaluator

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

func resolveAny(v any) string {
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

func Print(w io.Writer, debug bool, nth *uint64, op byte, a, b, c any) {
	if debug {
		clr := color.New(color.FgHiCyan)
		lo := fmt.Sprintf("%-12s", emitter.ResolveOpCode(op))
		_, _ = fmt.Fprintf(w, "[%016s] %s %v %v %v\n", clr.Sprintf("%d", *nth), lo, resolveAny(a), resolveAny(b), resolveAny(c))
	}
	*nth++
}
