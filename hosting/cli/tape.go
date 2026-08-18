package cli

// ResolveTapeSize applies the precedence: the flag wins over the manifest, and the
// default applies when neither is set.
func ResolveTapeSize(flag, manifest int) int {
	if flag != 0 {
		return flag
	}
	return manifest
}
