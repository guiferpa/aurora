package print

import (
	"encoding/json"
	"io"
)

func JSON(w io.Writer, o interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("print: ", "|  ")
	return enc.Encode(o)
}
