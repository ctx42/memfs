// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

// Package memfs provides in-memory files and directories. A [File] works like
// [os.File] or [bytes.Buffer], and a directory tree built from [File] values
// serves the [io/fs] interfaces. A [File] is not safe for concurrent use.
package memfs

import (
	"errors"
)

// ErrOutOfBounds is returned for invalid offsets.
var ErrOutOfBounds = errors.New("offset out of bounds")

// ErrHasParent is returned when an instance of [File] which is already a
// child of another instance is added to a directory.
var ErrHasParent = errors.New("entry already has a parent")
