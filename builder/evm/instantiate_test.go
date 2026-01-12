package evm

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/emitter"
)

func TestBuildInstantiateCode(t *testing.T) {
	builder := NewBuilder(make([]emitter.Instruction, 0), NewBuilderOptions{EnableLogging: false})
	bfr, err := builder.buildInstantiateCode(5)
	if err != nil {
		t.Errorf("Error building init code: %v", err)
		return
	}
	got := bfr.Bytes()
	expected := []byte{OpPush1, 5, OpPush1, 0x0c, OpPush1, 0x00, OpCodeCopy, OpPush1, 5, OpPush1, 0x00, OpReturn}
	if !bytes.Equal(got, expected) {
		t.Errorf("Init code: got: %v, expected: %v", ToString(got), ToString(expected))
	}
}
