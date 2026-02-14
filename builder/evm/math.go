package evm

import "io"

func (t *Builder) writeAdd(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpSwap1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpAdd}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) writeMult(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpSwap1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMul}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) writeSub(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpSwap1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpSub}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) writeDiv(w io.Writer) (int, error) {
	if _, err := w.Write([]byte{OpSwap1}); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpDiv}); err != nil {
		return 0, err
	}
	return 0, nil
}
