package byteutil

import (
	"errors"
	"reflect"
	"testing"
)

func TestEncodeBooleanTrue(t *testing.T) {
	value := []byte{1}
	got, err := Encode(value)
	if err != nil {
		t.Error(err)
	}
	if got, expected := reflect.ValueOf(got).Bool(), true; got != expected {
		t.Errorf("Unexpected boolean: got %v, expected: %v", got, expected)
	}
}

func TestEncodeBooleanFalse(t *testing.T) {
	value := []byte{0}
	got, err := Encode(value)
	if err != nil {
		t.Error(err)
	}
	if got, expected := reflect.ValueOf(got).Bool(), false; got != expected {
		t.Errorf("Unexpected boolean: got %v, expected: %v", got, expected)
	}
}

func TestEncodeUint64(t *testing.T) {
	value := []byte{0, 0, 0, 0, 0, 0, 255, 255}
	got, err := Encode(value)
	if err != nil {
		t.Error(err)
	}
	if got, expected := reflect.ValueOf(got).Uint(), uint64(65535); got != expected {
		t.Errorf("Unexpected uint64: got %v, expected: %v", got, expected)
	}
}

func TestEncodeTapeUint64(t *testing.T) {
	value := []byte{0, 0, 0, 0, 0, 0, 255, 255, 0, 0, 0, 0, 0, 0, 50, 1}
	got, err := Encode(value)
	if err != nil {
		t.Error(err)
	}
	if got, expected := got.([]uint64), []uint64{65535, 12801}; !reflect.DeepEqual(got, expected) {
		t.Errorf("Unexpected uint64: got %v, expected: %v", got, expected)
	}
}

func TestEncodeSliceGreaterThan64Bits(t *testing.T) {
	value := []byte{0, 0, 0, 0, 0, 0, 0, 255, 255}
	_, got := Encode(value)
	expected := &ErrEncode{}
	if !errors.Is(got, expected) {
		t.Errorf("Unexpected error: got %v, expected: %v", got, expected)
	}
}

func TestEncodeSliceLessThan64Bits(t *testing.T) {
	value := []byte{0, 0, 0, 0, 0, 255, 255}
	_, got := Encode(value)
	expected := &ErrEncode{}
	if !errors.Is(got, expected) {
		t.Errorf("Unexpected error: got %v, expected: %v", got, expected)
	}
}
