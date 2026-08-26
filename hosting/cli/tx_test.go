package cli

import (
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
