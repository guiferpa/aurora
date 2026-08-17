package cli

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
)

// ValidateTapeSize rejects a size outside the supported range. Validation lives at the
// command boundary, where a useful message can be given; deeper layers just normalize.
func ValidateTapeSize(size int) error {
	if size == 0 {
		return nil // unset: the default applies
	}
	if size < byteutil.MinTapeSize || size > byteutil.MaxTapeSize {
		return fmt.Errorf("tape size must be between %d and %d bytes, got %d",
			byteutil.MinTapeSize, byteutil.MaxTapeSize, size)
	}
	return nil
}

// ResolveTapeSize applies the precedence: the flag wins over the manifest, and the
// default applies when neither is set.
func ResolveTapeSize(flag, manifest int) int {
	if flag != 0 {
		return flag
	}
	return manifest
}
