package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// A transaction carries the same calldata a question does.
//
// `aurora call read 1 2` and `aurora tx keep 1 2` build the selector and the arguments the same
// way, because they are the same thing said to the same contract — one asked and one sent. This
// is what says so, without a network: both are shown rather than sent.
func TestATransactionCarriesTheSameCalldataAsAQuestion(t *testing.T) {
	asked := &strings.Builder{}
	sent := &strings.Builder{}
	const contract = "0x0000000000000000000000000000000000000001"

	if err := Call(context.Background(), CallInput{
		Function: "keep", ContractAddress: contract, Args: []string{"1", "2"}, Pretend: true, Out: asked,
	}); err != nil {
		t.Fatalf("asking: %v", err)
	}
	if err := Tx(context.Background(), TxInput{
		Function: "keep", ContractAddress: contract, Args: []string{"1", "2"}, Pretend: true, Out: sent,
	}); err != nil {
		t.Fatalf("sending: %v", err)
	}

	if dataOf(asked.String()) == "" {
		t.Fatal("the question showed no data")
	}
	if dataOf(asked.String()) != dataOf(sent.String()) {
		t.Errorf("a question carries %q and a transaction %q", dataOf(asked.String()), dataOf(sent.String()))
	}
}

// And what it carries in wei is read as a whole number, because ether is a decimal and a
// decimal is a rounding waiting to happen. This is money.
func TestWhatATransactionCarriesIsWholeWei(t *testing.T) {
	for _, tc := range []struct {
		name    string
		written string
		refused bool
	}{
		{name: "nothing", written: "0"},
		{name: "a thousandth of an ether", written: "1000000000000000"},
		{name: "more than a word would hold as a decimal", written: "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
		{name: "a fraction", written: "0.001", refused: true},
		{name: "less than nothing", written: "-1", refused: true},
		{name: "not a number", written: "one ether", refused: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCallValue(tc.written)
			if tc.refused && err == nil {
				t.Errorf("%q was read as an amount", tc.written)
			}
			if !tc.refused && err != nil {
				t.Errorf("%q was refused: %v", tc.written, err)
			}
			if tc.refused && err != nil && !strings.Contains(err.Error(), "wei") {
				t.Errorf("it says %q, and never says what it wanted", err)
			}
		})
	}
}

// dataOf reads the calldata line out of what a pretended command printed.
func dataOf(printed string) string {
	for _, line := range strings.Split(printed, "\n") {
		if after, found := strings.CutPrefix(line, "Data:"); found {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// What a question answers is the bytes that came back, and nothing else.
//
// It used to be a line saying "Result:" with the value as decimals inside brackets — readable,
// and not the answer. Anybody piping it somewhere had to undo it first.
func TestWhatAQuestionAnswers(t *testing.T) {
	answer := []byte{0, 0, 0, 0, 0, 0, 0, 42}

	t.Run("the bytes themselves", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeAnswer(&out, answer, false); err != nil {
			t.Fatalf("writing: %v", err)
		}
		if !bytes.Equal(out.Bytes(), answer) {
			t.Errorf("it wrote %v, want the eight bytes that came back", out.Bytes())
		}
	})

	t.Run("and nothing around them", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeAnswer(&out, answer, false); err != nil {
			t.Fatalf("writing: %v", err)
		}
		// Not even a newline: a byte nobody sent is not part of the answer, and one more at
		// the end is one more to strip.
		if out.Len() != len(answer) {
			t.Errorf("it wrote %d bytes for an answer of %d", out.Len(), len(answer))
		}
	})

	t.Run("as hex, for a person", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeAnswer(&out, answer, true); err != nil {
			t.Fatalf("writing: %v", err)
		}
		if got := out.String(); got != "0x000000000000002a\n" {
			t.Errorf("it wrote %q, want the hex and a newline", got)
		}
	})

	t.Run("an answer of nothing", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeAnswer(&out, nil, false); err != nil {
			t.Fatalf("writing: %v", err)
		}
		if out.Len() != 0 {
			t.Errorf("it wrote %q for a contract that answered nothing", out.String())
		}
	})
}
