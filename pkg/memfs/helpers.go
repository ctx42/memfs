// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"io/fs"
	"strings"
)

// open opens files in a given directory or its subdirectories.
func open(dir *File, name string) (*File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	if name == "." {
		return dir, nil
	}

	// Paths in the io/fs package always use forward slashes.
	first, rest, nested := strings.Cut(name, "/")
	if nested {
		d, err := open(dir, first)
		if err != nil {
			return nil, err
		}
		return open(d, rest)
	}

	for _, fil := range dir.entries {
		if fil.Name() == name {
			return fil, nil
		}
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
