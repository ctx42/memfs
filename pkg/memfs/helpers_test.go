// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"io"
	"io/fs"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/must"
)

func Test_open(t *testing.T) {
	root := tstDirMem()

	t.Run("open an existing file", func(t *testing.T) {
		// --- When ---
		have, err := open(root, "file1")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "file1", string(must.Value(io.ReadAll(have))))
	})

	t.Run("open dot", func(t *testing.T) {
		// --- When ---
		have, err := open(root, ".")

		// --- Then ---
		assert.NoError(t, err)
		assert.Same(t, root, have)
	})

	t.Run("open an existing directory", func(t *testing.T) {
		// --- When ---
		have, err := open(root, "sub")

		// --- Then ---
		assert.NoError(t, err)
		assert.True(t, have.IsDir())
		assert.Equal(t, "sub", have.Name())
	})

	t.Run("open a deep file", func(t *testing.T) {
		// --- When ---
		have, err := open(root, "sub/file3")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "file3", string(must.Value(io.ReadAll(have))))
	})

	t.Run("open a deep directory", func(t *testing.T) {
		// --- When ---
		have, err := open(root, "sub/sub2")

		// --- Then ---
		assert.NoError(t, err)
		assert.True(t, have.IsDir())
		assert.Equal(t, "sub2", have.Name())
	})

	t.Run("error - open not existing", func(t *testing.T) {
		// --- When ---
		have, err := open(root, "not-existing")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "open", e.Op)
		assert.Equal(t, "not-existing", e.Path)
		assert.Equal(t, fs.ErrNotExist, e.Err)
		assert.Nil(t, have)
	})

	t.Run("error - open deep not existing", func(t *testing.T) {
		// --- When ---
		have, err := open(root, "sub/not-existing")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "open", e.Op)
		assert.Equal(t, "not-existing", e.Path)
		assert.Equal(t, fs.ErrNotExist, e.Err)
		assert.Nil(t, have)
	})

	t.Run("error - open deep", func(t *testing.T) {
		// --- When ---
		have, err := open(root, "not-existing1/not-existing2")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "open", e.Op)
		assert.Equal(t, "not-existing1", e.Path)
		assert.Equal(t, fs.ErrNotExist, e.Err)
		assert.Nil(t, have)
	})

	t.Run("error - a rooted path is invalid", func(t *testing.T) {
		// --- When ---
		have, err := open(root, "/sub/sub2")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "open", e.Op)
		assert.Equal(t, "/sub/sub2", e.Path)
		assert.Equal(t, fs.ErrInvalid, e.Err)
		assert.Nil(t, have)
	})
}
