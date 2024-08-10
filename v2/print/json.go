package print

import (
	"encoding/json"
	"io"

	"github.com/TylerBrock/colorjson"
)

var f = colorjson.NewFormatter()

func init() {
	f.Indent = 2
}

func JSON(w io.Writer, o interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("print: ", "|  ")
	return enc.Encode(o)
}
