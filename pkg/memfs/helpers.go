// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"io/fs"
	"path/filepath"
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

	parts := strings.Split(name, string(filepath.Separator))
	if len(parts) > 1 {
		d, err := open(dir, parts[0])
		if err != nil {
			return nil, err
		}
		return open(d, filepath.Join(parts[1:]...))
	}

	fil, ok := dir.entries[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return fil, nil
}
