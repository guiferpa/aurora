package byteutil

import (
	"errors"
	"reflect"
	"testing"
)

func TestEncodeTape(t *testing.T) {
	value := []byte{0, 0, 0, 0, 0, 0, 255, 255}
	got, err := Encode(value, DefaultTapeSize)
	if err != nil {
		t.Error(err)
	}
	if got, expected := got.(string), "65535"; got != expected {
		t.Errorf("Unexpected value: got %v, expected: %v", got, expected)
	}
}

func TestEncodeReel(t *testing.T) {
	value := []byte{0, 0, 0, 0, 0, 0, 255, 255, 0, 0, 0, 0, 0, 0, 50, 1}
	got, err := Encode(value, DefaultTapeSize)
	if err != nil {
		t.Error(err)
	}
	if got, expected := got.([]string), []string{"65535", "12801"}; !reflect.DeepEqual(got, expected) {
		t.Errorf("Unexpected values: got %v, expected: %v", got, expected)
	}
}

func TestEncodeSliceGreaterThan64Bits(t *testing.T) {
	value := []byte{0, 0, 0, 0, 0, 0, 0, 255, 255}
	_, got := Encode(value, DefaultTapeSize)
	expected := &ErrEncode{}
	if !errors.Is(got, expected) {
		t.Errorf("Unexpected error: got %v, expected: %v", got, expected)
	}
}

func TestEncodeSliceLessThan64Bits(t *testing.T) {
	value := []byte{0, 0, 0, 0, 0, 255, 255}
	_, got := Encode(value, DefaultTapeSize)
	expected := &ErrEncode{}
	if !errors.Is(got, expected) {
		t.Errorf("Unexpected error: got %v, expected: %v", got, expected)
	}
}
