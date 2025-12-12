package evm

import "io"

func (t *Builder) buildAdd(w io.Writer) (int, error) {
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpAdd}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) buildMult(w io.Writer) (int, error) {
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpMul}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) buildSub(w io.Writer) (int, error) {
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpSub}); err != nil {
		return 0, err
	}
	return 0, nil
}

func (t *Builder) buildDiv(w io.Writer) (int, error) {
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := t.buildPush8SafeFromOperands(w); err != nil {
		return 0, err
	}
	if _, err := w.Write([]byte{OpDiv}); err != nil {
		return 0, err
	}
	return 0, nil
}
