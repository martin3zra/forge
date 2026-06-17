package store

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

type Query map[string]string

func (qs Query) Q(name string) string {
	return qs[name]
}

func NewQueryStore(filesystem fs.FS, dir string) (Query, error) {
	qs := make(Query)
	pattern := dir + "*.sql"

	matches, err := fs.Glob(filesystem, pattern)
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		bytes, err := fs.ReadFile(filesystem, match)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", match, err)
		}

		// remove the dir and extension from the name
		name := filepath.Base(match)
		name = name[:len(name)-4]
		qs[name] = string(bytes)
	}

	return qs, nil
}
