package evm

import "io"

func (t *Builder) writeBool(w io.Writer, v byte) (int, error) {
	if _, err := w.Write([]byte{OpPush1, v}); err != nil {
		return 0, err
	}
	return 0, nil
}
