// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"io/fs"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
)

func Test_FileInfo_Name(t *testing.T) {
	// --- Given ---
	fi := FileInfo{name: "dir/file"}

	// --- When ---
	have := fi.Name()

	// --- Then ---
	assert.Equal(t, "file", have)
}

func Test_FileInfo_Size(t *testing.T) {
	// --- Given ---
	fi := FileInfo{size: 42}

	// --- When ---
	have := fi.Size()

	// --- Then ---
	assert.Equal(t, int64(42), have)
}

func Test_FileInfo_Mode(t *testing.T) {
	// --- Given ---
	fi := FileInfo{mode: 0444}

	// --- When ---
	have := fi.Mode()

	// --- Then ---
	assert.Equal(t, fs.FileMode(0444), have)
}

func Test_FileInfo_ModTime(t *testing.T) {
	// --- Given ---
	fi := FileInfo{}

	// --- When ---
	have := fi.ModTime()

	// --- Then ---
	assert.Zero(t, have)
}

func Test_FileInfo_IsDir(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		fi := FileInfo{}

		// --- When ---
		have := fi.IsDir()

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		fi := FileInfo{mode: fs.ModeDir}

		// --- When ---
		have := fi.IsDir()

		// --- Then ---
		assert.True(t, have)
	})
}

func Test_FileInfo_Sys(t *testing.T) {
	// --- Given ---
	fi := FileInfo{}

	// --- When ---
	have := fi.Sys()

	// --- Then ---
	assert.Nil(t, have)
}

func Test_FileInfo_Type(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		fi := FileInfo{}

		// --- When ---
		have := fi.Type()

		// --- Then ---
		assert.Equal(t, fs.FileMode(0), have)
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		fi := FileInfo{mode: 0777 | fs.ModeDir}

		// --- When ---
		have := fi.Type()

		// --- Then ---
		assert.Equal(t, fs.ModeDir, have)
	})
}

func Test_FileInfo_Info(t *testing.T) {
	// --- Given ---
	fi := FileInfo{
		name: "file",
		size: 123,
		mode: 0777 | fs.ModeDir,
	}

	// --- When ---
	have, err := fi.Info()

	// --- Then ---
	assert.NoError(t, err)
	assert.Equal(t, "file", have.Name())
	assert.Equal(t, int64(123), have.Size())
	assert.Equal(t, 0777|fs.ModeDir, have.Mode())
	assert.Zero(t, have.ModTime())
	assert.True(t, have.IsDir())
	assert.Nil(t, have.Sys())
}
