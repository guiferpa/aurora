package builtin

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/byteutil"
)

func PrintFunction(w io.Writer, bs []byte) {
	_, _ = w.Write(bs)
}

// EchoFunction converts bytes to text and prints it
// If bytes represent a reel (array of tapes), it prints each tape in sequence
// If bytes represent a tape (8 bytes), it prints the tape as a character
// If bytes represent a number, it encodes it as text (ASCII character)
func EchoFunction(w io.Writer, bs []byte) {
	// Check if this is a reel (array of tapes)
	// A reel is a concatenation of multiple 8-byte tapes
	// If the length is a multiple of 8 and greater than 8, it's likely a reel
	if len(bs) > 8 && len(bs)%8 == 0 {
		// This is a reel - iterate over each tape (8 bytes each)
		result := ""
		for i := 0; i < len(bs); i += 8 {
			tape := bs[i : i+8]
			// Extract significant bytes from this tape (right-aligned)
			significant := extractSignificantBytes(tape)
			if len(significant) > 0 {
				// Convert significant bytes to character
				char := rune(significant[len(significant)-1]) // Use last byte as character
				if char >= 32 && char <= 126 {
					result += string(char)
				}
			}
		}
		if result != "" {
			_, _ = w.Write([]byte(result))
		} else {
			_, _ = w.Write([]byte("\n"))
		}
		return
	}

	// This is a tape (8 bytes or less) - extract significant bytes
	significant := extractSignificantBytes(bs)

	// If no significant bytes, print empty line
	if len(significant) == 0 {
		fmt.Println()
		return
	}

	// Convert bytes to text
	// Use the last significant byte as the character
	char := rune(significant[len(significant)-1])
	if char >= 32 && char <= 126 {
		// Printable ASCII character
		_, _ = w.Write([]byte(string(char)))
	} else {
		// Non-printable, print as-is
		_, _ = w.Write([]byte(string(significant)))
	}
}

// AssertFunction evaluates an assert: condition (bytes as boolean) and message (reel bytes for error display).
// Returns (passed, errMessage). When passed is false, errMessage is the decoded message to show.
func AssertFunction(cond, msg []byte) (bool, error) {
	passed := byteutil.ToBoolean(cond)
	if passed {
		return true, nil
	}
	message := reelBytesToString(msg)
	return false, fmt.Errorf("assertion failed: %s", message)
}

// reelBytesToString decodes reel bytes (concatenated 8-byte tapes) to a Go string for display.
func reelBytesToString(bs []byte) string {
	if len(bs) == 0 {
		return ""
	}
	if len(bs) > 8 && len(bs)%8 == 0 {
		result := ""
		for i := 0; i < len(bs); i += 8 {
			tape := bs[i : i+8]
			significant := extractSignificantBytes(tape)
			if len(significant) > 0 {
				char := rune(significant[len(significant)-1])
				if char >= 32 && char <= 126 {
					result += string(char)
				}
			}
		}
		return result
	}
	significant := extractSignificantBytes(bs)
	if len(significant) == 0 {
		return ""
	}
	return string(significant)
}

// extractSignificantBytes extracts bytes from the first non-zero byte to the end (right-aligned)
func extractSignificantBytes(bs []byte) []byte {
	if len(bs) == 0 {
		return []byte{}
	}
	// Start from the right and collect non-zero bytes
	significant := make([]byte, 0)
	for i := len(bs) - 1; i >= 0; i-- {
		if bs[i] != 0 || len(significant) > 0 {
			significant = append([]byte{bs[i]}, significant...)
		}
	}
	return significant
}
