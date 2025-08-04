// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"io/fs"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
)

func Test_fsOnly(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		fil := &File{}

		mck := NewFSMock(t)
		mck.OnOpen("name").Return(fil, nil)

		// --- When ---
		have, err := fsOnly{mck}.Open("name")

		// --- Then ---
		assert.NoError(t, err)
		assert.Same(t, fil, have)
	})

	t.Run("error", func(t *testing.T) {
		// --- Given ---
		mck := NewFSMock(t)
		mck.OnOpen("name").Return(nil, fs.ErrNotExist)

		// --- When ---
		have, err := fsOnly{mck}.Open("name")

		// --- Then ---
		assert.ErrorIs(t, fs.ErrNotExist, err)
		assert.Nil(t, have)
	})
}
