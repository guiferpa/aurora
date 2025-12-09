package builtin

import (
	"fmt"
)

func PrintFunction(bs []byte) {
	fmt.Println(string(bs))
}

// EchoFunction converts bytes to text and prints it
// If bytes represent a reel (array of tapes), it prints each tape in sequence
// If bytes represent a tape (8 bytes), it prints the tape as a character
// If bytes represent a number, it encodes it as text (ASCII character)
func EchoFunction(bs []byte) {
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
			fmt.Println(result)
		} else {
			fmt.Println()
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
		fmt.Println(string(char))
	} else {
		// Non-printable, print as-is
		fmt.Println(string(significant))
	}
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
