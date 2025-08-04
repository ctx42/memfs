// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"io/fs"
)

// fsOnly is a wrapper that hides all but the fs.FS methods to avoid an
// infinite recursion when implementing special methods in terms of helpers
// that would use them.
type fsOnly struct{ fs fs.FS }

func (f fsOnly) Open(name string) (fs.File, error) { return f.fs.Open(name) }
