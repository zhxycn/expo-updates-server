package storage

import (
	"os"
	"path/filepath"
	"strings"
)

func safeJoin(base string, parts ...string) (string, error) {
	for _, p := range parts {
		if filepath.IsAbs(p) {
			return "", os.ErrNotExist
		}
		for _, seg := range strings.FieldsFunc(p, isSeparator) {
			if seg == ".." {
				return "", os.ErrNotExist
			}
		}
	}

	cleanBase := filepath.Clean(base)

	full := filepath.Clean(filepath.Join(append([]string{cleanBase}, parts...)...))

	rel, err := filepath.Rel(cleanBase, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", os.ErrNotExist
	}

	return full, nil
}

func isSeparator(r rune) bool {
	return r == '/' || r == '\\'
}
