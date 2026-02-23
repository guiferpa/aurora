package fileutil

import (
	"os"
	"path/filepath"
)

// ListFilesByExtension returns paths of files with the given extension in root.
// Only the first level is considered; subdirectories are not traversed.
func ListFilesByExtension(root, ext string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ext {
			files = append(files, filepath.Join(root, e.Name()))
		}
	}
	return files, nil
}
