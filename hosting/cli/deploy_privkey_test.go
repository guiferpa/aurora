package cli

import (
	"strings"
	"testing"
)

// A key copied out of a wallet has "0x" in front of it more often than not, and the reader
// underneath refuses that with "invalid hex character 'x'" — true, and no help at all.
func TestAKeyWithAPrefixSaysSo(t *testing.T) {
	_, err := readPrivkey("0x3ef32e39895116e1c37123b85aba020e5f38612f0e168b1c25ed564a97ac8720")
	if err == nil {
		t.Fatal("it read a key with a prefix")
	}
	for _, want := range []string{"0x", "drop the prefix"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("it says %q, want it to mention %q", err, want)
		}
	}
}

// And what a key is, it reads — with whatever spaces came along from a file.
func TestAKeyIsReadWithTheSpacesAroundItDropped(t *testing.T) {
	const key = "3ef32e39895116e1c37123b85aba020e5f38612f0e168b1c25ed564a97ac8720"

	read, err := readPrivkey("  " + key + "\n")
	if err != nil {
		t.Fatalf("reading the key: %v", err)
	}
	if read == nil {
		t.Fatal("it read nothing")
	}
}

// Anything else is not a key, and says that rather than repeating what a hex reader thinks.
func TestSomethingThatIsNotAKeySaysSo(t *testing.T) {
	_, err := readPrivkey("not a key")
	if err == nil {
		t.Fatal("it read something that is not a key")
	}
	if !strings.Contains(err.Error(), "privkey is not a key") {
		t.Errorf("it says %q, want it to say what was wrong", err)
	}
}
