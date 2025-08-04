// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/must"
)

func Test_WithFileOffset(t *testing.T) {
	// --- Given ---
	fil := &File{}

	// --- When ---
	WithFileOffset(42)(fil)

	// --- Then ---
	assert.Equal(t, 42, fil.off)
}

func Test_WithFileAppend(t *testing.T) {
	// --- Given ---
	fil := &File{buf: make([]byte, 42)}

	// --- When ---
	WithFileAppend(fil)

	// --- Then ---
	assert.Equal(t, fil.flag&os.O_APPEND, os.O_APPEND)
}

func Test_WithFileFlag(t *testing.T) {
	// --- Given ---
	fil := &File{}

	// --- When ---
	WithFileFlag(42)(fil)

	// --- Then ---
	assert.Equal(t, 42, fil.flag)
}

func Test_NewFile(t *testing.T) {
	t.Run("without options", func(t *testing.T) {
		// --- When ---
		have, err := NewFile("file")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 0, have.flag)
		assert.Equal(t, 0, have.off)
		assert.Equal(t, "file", have.info.name)
		assert.Equal(t, fs.FileMode(0600), have.info.mode)
		assert.Cap(t, bytes.MinRead, have.buf)
	})

	t.Run("with options", func(t *testing.T) {
		// --- When ---
		have, err := NewFile("file", WithFileAppend)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, have.flag&os.O_APPEND, os.O_APPEND)
		assert.Equal(t, 0, have.off)
		assert.Equal(t, "file", have.info.name)
		assert.Equal(t, fs.FileMode(0600), have.info.mode)
		assert.Cap(t, bytes.MinRead, have.buf)
	})

	t.Run("panic - out-of-bounds error", func(t *testing.T) {
		// --- Given ---
		fn := func() { _, _ = NewFile("file", WithFileOffset(bytes.MinRead+1)) }

		// --- When ---
		msg := assert.PanicMsg(t, fn)

		// --- Then ---
		assert.Equal(t, "offset out of bounds", *msg)
	})

	t.Run("panic - negative offset", func(t *testing.T) {
		// --- Given ---
		fn := func() { _, _ = NewFile("file", WithFileOffset(-1)) }

		// --- When ---
		msg := assert.PanicMsg(t, fn)

		// --- Then ---
		assert.Equal(t, "offset out of bounds", *msg)
	})
}

func Test_FileWith(t *testing.T) {
	t.Run("without options", func(t *testing.T) {
		// --- Given ---
		content := make([]byte, 3, 42)
		copy(content, []byte{1, 2, 3})

		// --- When ---
		have, err := FileWith("file", content)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 0, have.flag)
		assert.Equal(t, 0, have.off)
		assert.Cap(t, 42, have.buf)
		assert.Equal(t, "file", have.info.name)
		assert.Equal(t, fs.FileMode(0600), have.info.mode)
		assert.Equal(t, []byte{1, 2, 3}, have.buf)
	})

	t.Run("with options", func(t *testing.T) {
		// --- Given ---
		content := make([]byte, 42, 44)

		// --- When ---
		have, err := FileWith("file", content, WithFileAppend)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, have.flag&os.O_APPEND, os.O_APPEND)
		assert.Equal(t, 0, have.off)
		assert.Equal(t, "file", have.info.name)
		assert.Equal(t, fs.FileMode(0600), have.info.mode)
		assert.Cap(t, 44, have.buf)
	})

	t.Run("error - name has separators", func(t *testing.T) {
		// --- When ---
		fil, err := FileWith("a/file", nil)

		// --- Then ---
		assert.ErrorIs(t, fs.ErrInvalid, err)
		assert.Nil(t, fil)
	})

	t.Run("panic - out-of-bounds error", func(t *testing.T) {
		// --- Given ---
		content := make([]byte, 42, 44)

		// --- Given ---
		fn := func() {
			_, _ = FileWith("file", content, WithFileOffset(43))
		}

		// --- When ---
		msg := assert.PanicMsg(t, fn)

		// --- Then ---
		assert.Equal(t, "offset out of bounds", *msg)
	})

	t.Run("panic - negative offset", func(t *testing.T) {
		// --- Given ---
		content := make([]byte, 42, 44)

		// --- Given ---
		fn := func() {
			_, _ = FileWith("file", content, WithFileOffset(-1))
		}

		// --- When ---
		msg := assert.PanicMsg(t, fn)

		// --- Then ---
		assert.Equal(t, "offset out of bounds", *msg)
	})
}

func Test_NewDirectory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- When ---
		have, err := NewDirectory("dir")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "dir", have.info.name)
		assert.Equal(t, int64(4096), have.info.size)
		assert.Equal(t, 0700|fs.ModeDir, have.info.mode)
		assert.Nil(t, have.entries)
	})

	t.Run("error - invalid name", func(t *testing.T) {
		// --- When ---
		have, err := NewDirectory("/dir")

		// --- Then ---
		assert.ErrorIs(t, fs.ErrInvalid, err)
		assert.Nil(t, have)
	})
}

func Test_NewRoot(t *testing.T) {
	// --- When ---
	have := NewRoot()

	// --- Then ---
	assert.Equal(t, "", have.info.name)
	assert.Equal(t, int64(4096), have.info.size)
	assert.Equal(t, 0700|fs.ModeDir, have.info.mode)
	assert.Nil(t, have.entries)
}

func Test_File_AddFile(t *testing.T) {
	t.Run("add a file", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		fil := MustFile("file")

		// --- When ---
		err := dir.AddFile(fil)

		// --- Then ---
		assert.NoError(t, err)
		have, _ := assert.HasKey(t, "file", dir.entries)
		assert.Same(t, fil, have)
		assert.Same(t, fil.parent, dir)
		assert.Equal(t, "file", fil.info.name)
	})

	t.Run("add a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		sub := MustDirectory("sub")

		// --- When ---
		err := dir.AddFile(sub)

		// --- Then ---
		assert.NoError(t, err)
		have, _ := assert.HasKey(t, "sub", dir.entries)
		assert.Same(t, sub, have)
		assert.Same(t, sub.parent, dir)
		assert.Equal(t, "sub", sub.info.name)
	})

	t.Run("add a directory with files", func(t *testing.T) {
		// --- Given ---
		dir := NewRoot()
		must.Nil(dir.AddFile(MustFile("file0")))
		must.Nil(dir.AddFile(MustFile("file1")))

		sub := MustDirectory("sub")
		must.Nil(sub.AddFile(MustFile("file2")))
		must.Nil(sub.AddFile(MustFile("file3")))

		sub2 := MustDirectory("sub2")
		must.Nil(sub2.AddFile(MustFile("file4")))
		must.Nil(sub2.AddFile(MustFile("file5")))
		must.Nil(sub.AddFile(sub2))

		// --- When ---
		err := dir.AddFile(sub)

		// --- Then ---
		assert.NoError(t, err)

		var have []string
		fn := func(path string, d fs.DirEntry, err error) error {
			have = append(have, path)
			return nil
		}
		assert.NoError(t, fs.WalkDir(dir, ".", fn))

		for i, pth := range have {
			f := must.Value(open(dir, pth))
			have[i] = f.path()
		}

		want := []string{
			".",
			"file0",
			"file1",
			"sub",
			"sub/file2",
			"sub/file3",
			"sub/sub2",
			"sub/sub2/file4",
			"sub/sub2/file5",
		}
		assert.Equal(t, want, have)
	})

	t.Run("error - file already exists", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		fil := MustFile("file")
		must.Nil(dir.AddFile(fil))

		// --- When ---
		err := dir.AddFile(MustFile("file"))

		// --- Then ---
		assert.ErrorIs(t, fs.ErrExist, err)
		have, _ := assert.HasKey(t, "file", dir.entries)
		assert.Same(t, fil, have)
	})

	t.Run("error - unsupported type", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		fil := MustFile("file")
		fil.info.mode = fs.ModeSymlink

		// --- When ---
		err := dir.AddFile(fil)

		// --- Then ---
		assert.ErrorIs(t, fs.ErrInvalid, err)
		assert.HasNoKey(t, "file", dir.entries)
	})

	t.Run("error - cannot add file to file", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")

		// --- When ---
		err := fil.AddFile(MustFile("txt"))

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "open", e.Op)
		assert.Equal(t, "file/txt", e.Path)
		assert.Equal(t, syscall.ENOTDIR, e.Err)
	})

	t.Run("error - file belongs to another directory", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")
		dir0 := MustDirectory("dir0")
		must.Nil(dir0.AddFile(fil))
		dir1 := MustDirectory("dir1")

		// --- When ---
		err := dir1.AddFile(fil)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "AddFile", e.Op)
		assert.Equal(t, "dir0/file", e.Path)
		assert.Equal(t, ErrHasParent, e.Err)
	})
}

func Test_File_ReadDir(t *testing.T) {
	t.Run("success - arg negative returns all", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustFileWith("file2", []byte("file2"))))
		must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
		must.Nil(dir.AddFile(MustFileWith("file1", []byte("file1"))))

		// --- When ---
		have, err := dir.ReadDir(-1)

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 3, have)
		assert.Equal(t, "file0", have[0].Name())
		assert.Equal(t, "file1", have[1].Name())
		assert.Equal(t, "file2", have[2].Name())

		// --- When ---
		have, err = dir.ReadDir(-1)

		// --- Then ---
		assert.ErrorIs(t, io.EOF, err)
		assert.Nil(t, have)
	})

	t.Run("success - arg zero returns all", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustFileWith("file2", []byte("file2"))))
		must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
		must.Nil(dir.AddFile(MustFileWith("file1", []byte("file1"))))

		// --- When ---
		have, err := dir.ReadDir(0)

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 3, have)
		assert.Equal(t, "file0", have[0].Name())
		assert.Equal(t, "file1", have[1].Name())
		assert.Equal(t, "file2", have[2].Name())

		// --- When ---
		have, err = dir.ReadDir(-1)

		// --- Then ---
		assert.ErrorIs(t, io.EOF, err)
		assert.Nil(t, have)
	})

	t.Run("success - arg less than the number of entries", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustFileWith("file2", []byte("file2"))))
		must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
		must.Nil(dir.AddFile(MustFileWith("file1", []byte("file1"))))

		// --- When ---
		have, err := dir.ReadDir(2)

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 2, have)
		assert.Equal(t, "file0", have[0].Name())
		assert.Equal(t, "file1", have[1].Name())

		// --- When ---
		have, err = dir.ReadDir(2)

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 1, have)
		assert.Equal(t, "file2", have[0].Name())

		// --- When ---
		have, err = dir.ReadDir(2)

		// --- Then ---
		assert.ErrorIs(t, io.EOF, err)
		assert.Nil(t, have)
	})

	t.Run("success - arg equal the number of entries", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustFileWith("file2", []byte("file2"))))
		must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
		must.Nil(dir.AddFile(MustFileWith("file1", []byte("file1"))))

		// --- When ---
		have, err := dir.ReadDir(3)

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 3, have)
		assert.Equal(t, "file0", have[0].Name())
		assert.Equal(t, "file1", have[1].Name())
		assert.Equal(t, "file2", have[2].Name())

		// --- When ---
		have, err = dir.ReadDir(3)

		// --- Then ---
		assert.ErrorIs(t, io.EOF, err)
		assert.Nil(t, have)
	})

	t.Run("success some", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustFileWith("file2", []byte("file2"))))
		must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
		must.Nil(dir.AddFile(MustFileWith("file1", []byte("file1"))))

		// --- When ---
		have, err := dir.ReadDir(2)

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 2, have)
		assert.Equal(t, "file0", have[0].Name())
		assert.Equal(t, "file1", have[1].Name())

		// --- When ---
		have, err = dir.ReadDir(2)

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 1, have)
		assert.Equal(t, "file2", have[0].Name())

		// --- When ---
		have, err = dir.ReadDir(2)

		// --- Then ---
		assert.ErrorIs(t, io.EOF, err)
		assert.Nil(t, have)
	})

	t.Run("error - not a directory", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")

		// --- When ---
		have, err := fil.ReadDir(-1)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "ReadDir", e.Op)
		assert.Equal(t, "file", e.Path)
		assert.Equal(t, syscall.ENOTDIR, e.Err)
		assert.Nil(t, have)
	})
}

func Test_Directory_ReadFile(t *testing.T) {
	t.Run("read an existing file", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
		must.Nil(dir.AddFile(MustFileWith("file1", []byte("file1"))))

		// --- When ---
		have, err := dir.ReadFile("file1")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, []byte("file1"), have)
	})

	t.Run("error - not existing file", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.ReadFile("not-existing")

		// --- Then ---
		assert.ErrorIs(t, fs.ErrNotExist, err)
		assert.Nil(t, have)
	})

	t.Run("error - read sub directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustDirectory("sub")))

		// --- When ---
		have, err := dir.ReadFile("sub")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "read", e.Op)
		assert.Equal(t, "dir/sub", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
		assert.NotNil(t, have)
		assert.Len(t, 0, have)
	})

	t.Run("error - can be called only on directories", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")

		// --- When ---
		have, err := fil.ReadFile("other")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "ReadFile", e.Op)
		assert.Equal(t, "file", e.Path)
		assert.Equal(t, syscall.ENOTDIR, e.Err)
		assert.Nil(t, have)
	})
}

func Test_File_Name(t *testing.T) {
	// --- Given ---
	fil := &File{info: FileInfo{name: "dir/file"}}

	// --- When ---
	have := fil.Name()

	// --- Then ---
	assert.Equal(t, "file", have)
}

func Test_File_IsDir(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")

		// --- When ---
		have := fil.IsDir()

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have := dir.IsDir()

		// --- Then ---
		assert.True(t, have)
	})
}

func Test_File_Type(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")

		// --- When ---
		have := fil.Type()

		// --- Then ---
		assert.Equal(t, fs.FileMode(0), have)
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		fil := MustDirectory("dir")

		// --- When ---
		have := fil.Type()

		// --- Then ---
		assert.Equal(t, fs.ModeDir, have)
	})
}

func Test_File_Stat(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")

		// --- When ---
		have, err := fil.Stat()

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "file", have.Name())
		assert.Equal(t, int64(0), have.Size())
		assert.Equal(t, fs.FileMode(0600), have.Mode())
		assert.Zero(t, have.ModTime())
		assert.False(t, have.IsDir())
		assert.Nil(t, have.Sys())
	})

	t.Run("file with contents", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 4})

		// --- When ---
		have, err := fil.Stat()

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, int64(4), have.Size())
	})

	t.Run("file size updates after writing", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3})
		must.Value(fil.WriteAt([]byte{4, 5}, 4))

		// --- When ---
		have, err := fil.Stat()

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, int64(6), have.Size())
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.Stat()

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "dir", have.Name())
		assert.Equal(t, int64(4096), have.Size())
		assert.Equal(t, fs.FileMode(0700)|fs.ModeDir, have.Mode())
		assert.Zero(t, have.ModTime())
		assert.True(t, have.IsDir())
		assert.Nil(t, have.Sys())
	})
}

func Test_File_Info(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3})

		// --- When ---
		have, err := fil.Info()

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "file", have.Name())
		assert.Equal(t, int64(4), have.Size())
		assert.Equal(t, fs.FileMode(0600), have.Mode())
		assert.Zero(t, have.ModTime())
		assert.False(t, have.IsDir())
		assert.Nil(t, have.Sys())
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.Info()

		// --- Then ---
		assert.NoError(t, err)

		assert.Equal(t, "dir", have.Name())
		assert.Equal(t, int64(4096), have.Size())
		assert.Equal(t, fs.FileMode(0700)|fs.ModeDir, have.Mode())
		assert.Zero(t, have.ModTime())
		assert.True(t, have.IsDir())
		assert.Nil(t, have.Sys())
	})
}

func Test_File_Size(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3})

		// --- When ---
		have := fil.Size()

		// --- Then ---
		assert.Equal(t, int64(4), have)
	})

	t.Run("file size updates after writing", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3})

		// --- When ---
		must.Value(fil.WriteAt([]byte{4, 5}, 4))
		have := fil.Size()

		// --- Then ---
		assert.Equal(t, int64(6), have)
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have := dir.Size()

		// --- Then ---
		assert.Equal(t, int64(4096), have)
	})
}

func Test_File_Mode(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")

		// --- When ---
		have := fil.Mode()

		// --- Then ---
		assert.Equal(t, fs.FileMode(0600), have)
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have := dir.Mode()

		// --- Then ---
		assert.Equal(t, fs.FileMode(0700)|fs.ModeDir, have)
	})
}

func Test_File_ModTime(t *testing.T) {
	// --- Given ---
	fil := MustFile("file")

	// --- When ---
	have := fil.ModTime()

	// --- Then ---
	assert.Zero(t, have)
}

func Test_File_Sys(t *testing.T) {
	// --- Given ---
	fil := MustFileWith("file", []byte{0, 1, 2, 3})

	// --- When ---
	have := fil.Sys()

	// --- Then ---
	assert.Nil(t, have)
}

func Test_File_Open(t *testing.T) {
	t.Run("open file", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
		must.Nil(dir.AddFile(MustFileWith("file1", []byte("file1"))))

		// --- When ---
		have, err := dir.Open("file0")

		// --- Then ---
		assert.NoError(t, err)
		assert.False(t, must.Value(have.Stat()).IsDir())
		assert.Equal(t, "file0", must.Value(have.Stat()).Name())
		assert.NoError(t, have.Close())
	})

	t.Run("open directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
		must.Nil(dir.AddFile(MustDirectory("sub")))

		// --- When ---
		have, err := dir.Open("sub")

		// --- Then ---
		assert.NoError(t, err)
		assert.True(t, must.Value(have.Stat()).IsDir())
		assert.Equal(t, "sub", must.Value(have.Stat()).Name())
		assert.NoError(t, have.Close())
	})

	t.Run("error - not existing", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.Open("not-existing")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "open", e.Op)
		assert.Equal(t, "not-existing", e.Path)
		assert.Equal(t, fs.ErrNotExist, e.Err)
		assert.Nil(t, have)
	})
}

func Test_File_FS(t *testing.T) {
	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		dir := tstDirMem()

		// --- When ---
		have := dir.FS()

		// --- Then ---
		assert.NotNil(t, have)
		fil := must.Value(have.Open("sub/sub2/file6"))
		assert.Equal(t, "file6", string(must.Value(io.ReadAll(fil))))
	})

	t.Run("file", func(t *testing.T) {
		// --- Given ---
		dir := MustFile("file")

		// --- When ---
		have := dir.FS()

		// --- Then ---
		assert.Nil(t, have)
	})
}

func Test_File_Release(t *testing.T) {
	// --- Given ---
	fil := MustFileWith("file", []byte{0, 1, 2, 3}, WithFileOffset(1))

	// --- When ---
	have := fil.Release()

	// --- Then ---
	assert.Equal(t, []byte{0, 1, 2, 3}, have)
	assert.Equal(t, 0, fil.off)
	assert.Nil(t, fil.buf)
}

func Test_File_Write(t *testing.T) {
	t.Run("error - cannot write to a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.Write([]byte{0, 1, 2, 3})

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "write", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
		assert.Equal(t, 0, have)
	})
}

func Test_File_Write_tabular(t *testing.T) {
	tt := []struct {
		testN string

		content []byte
		opts    []func(*File)
		src     []byte
		wN      int
		wOff    int
		wLen    int
		wCap    int
		wBuf    []byte
	}{
		{
			testN:   "zero value",
			content: nil,
			opts:    nil,
			src:     []byte{0, 1, 2},
			wN:      3,
			wOff:    3,
			wLen:    3,
			wCap:    64,
			wBuf:    []byte{0, 1, 2},
		},
		{
			testN:   "empty with capacity",
			content: make([]byte, 0, 5),
			opts:    nil,
			src:     []byte{0, 1, 2},
			wN:      3,
			wOff:    3,
			wLen:    3,
			wCap:    5,
			wBuf:    []byte{0, 1, 2},
		},
		{
			testN:   "empty with capacity write more then cap",
			content: make([]byte, 0, 5),
			opts:    nil,
			src:     []byte{0, 1, 2, 3, 4, 5},
			wN:      6,
			wOff:    6,
			wLen:    6,
			wCap:    16,
			wBuf:    []byte{0, 1, 2, 3, 4, 5},
		},
		{
			testN:   "offset at len",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileOffset(3)},
			src:     []byte{3, 4, 5},
			wN:      3,
			wOff:    6,
			wLen:    6,
			wCap:    9,
			wBuf:    []byte{0, 1, 2, 3, 4, 5},
		},
		{
			testN:   "append",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileAppend},
			src:     []byte{3, 4, 5},
			wN:      3,
			wOff:    6,
			wLen:    6,
			wCap:    9,
			wBuf:    []byte{0, 1, 2, 3, 4, 5},
		},
		{
			testN:   "override and extend",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileOffset(1)},
			src:     []byte{3, 4, 5},
			wN:      3,
			wOff:    4,
			wLen:    4,
			wCap:    9,
			wBuf:    []byte{0, 3, 4, 5},
		},
		// {
		// 	testN:   "override and extend big",
		// 	content: []byte{0, 1, 2},
		// 	opts:    []func(*File){WithFileOffset(1)},
		// 	src:     bytes.Repeat([]byte{0, 1}, 1<<20),
		// 	wN:      2 * 1 << 20,
		// 	wOff:    2*1<<20 + 1,
		// 	wLen:    2*1<<20 + 1,
		// 	wCap:    2*1<<20 + 6,
		// 	wBuf:    append([]byte{0}, bytes.Repeat([]byte{0, 1}, 1<<20)...),
		// },
		{
			testN:   "override tail",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileOffset(1)},
			src:     []byte{3, 4},
			wN:      2,
			wOff:    3,
			wLen:    3,
			wCap:    3,
			wBuf:    []byte{0, 3, 4},
		},
		{
			testN:   "override middle",
			content: []byte{0, 1, 2, 3},
			opts:    []func(*File){WithFileOffset(1)},
			src:     []byte{4, 5},
			wN:      2,
			wOff:    3,
			wLen:    4,
			wCap:    4,
			wBuf:    []byte{0, 4, 5, 3},
		},
		{
			testN:   "override all no extend",
			content: []byte{0, 1, 2, 3},
			opts:    nil,
			src:     []byte{4, 5, 6, 7},
			wN:      4,
			wOff:    4,
			wLen:    4,
			wCap:    4,
			wBuf:    []byte{4, 5, 6, 7},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			var fil *File
			if tc.content == nil {
				fil = &File{} // Test for zero value.
			} else {
				fil = MustFileWith("file", tc.content, tc.opts...)
			}

			// --- When ---
			n, err := fil.Write(tc.src)

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.wN, n)

			assert.Equal(t, tc.wOff, fil.Offset())
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
			assert.Equal(t, tc.wBuf, fil.buf)
			assert.NoError(t, fil.Close())
		})
	}
}
func Test_File_WriteByte(t *testing.T) {
	t.Run("error - cannot write to a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		err := dir.WriteByte(0)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "write", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
	})
}

func Test_File_WriteByte_tabular(t *testing.T) {
	tt := []struct {
		testN string

		init []byte
		opts []func(*File)
		wOff int
		wLen int
		wCap int
		wBuf []byte
	}{
		{
			testN: "zero value",
			init:  nil,
			opts:  nil,
			wOff:  1,
			wLen:  1,
			wCap:  64,
			wBuf:  []byte{0xFF},
		},
		{
			testN: "empty with capacity",
			init:  make([]byte, 0, 5),
			opts:  nil,
			wOff:  1,
			wLen:  1,
			wCap:  5,
			wBuf:  []byte{0xFF},
		},
		{
			testN: "offset at len",
			init:  []byte{0, 1, 2},
			opts:  []func(*File){WithFileOffset(3)},
			wOff:  4,
			wLen:  4,
			wCap:  7,
			wBuf:  []byte{0, 1, 2, 0xFF},
		},
		{
			testN: "append",
			init:  []byte{0, 1, 2},
			opts:  []func(*File){WithFileAppend},
			wOff:  4,
			wLen:  4,
			wCap:  7,
			wBuf:  []byte{0, 1, 2, 0xFF},
		},
		{
			testN: "override tail",
			init:  []byte{0, 1, 2},
			opts:  []func(*File){WithFileOffset(2)},
			wOff:  3,
			wLen:  3,
			wCap:  3,
			wBuf:  []byte{0, 1, 0xFF},
		},
		{
			testN: "override middle",
			init:  []byte{0, 1, 2, 3},
			opts:  []func(*File){WithFileOffset(1)},
			wOff:  2,
			wLen:  4,
			wCap:  4,
			wBuf:  []byte{0, 0xFF, 2, 3},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			var fil *File
			if tc.init == nil {
				fil = &File{} // Test for zero value.
			} else {
				fil = MustFileWith("file", tc.init, tc.opts...)
			}

			// --- When ---
			err := fil.WriteByte(0xFF)

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.wOff, fil.Offset())
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
			assert.Equal(t, tc.wBuf, fil.buf)
			assert.NoError(t, fil.Close())
		})
	}
}

func Test_File_WriteAt(t *testing.T) {
	t.Run("using zero value", func(t *testing.T) {
		// --- Given ---
		fil := &File{}

		// --- When ---
		n, err := fil.WriteAt([]byte{0, 1, 2}, 0)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, 0, fil.Offset())
		assert.Equal(t, 3, fil.Len())
		assert.Equal(t, 64, fil.Cap())
		want := []byte{0, 1, 2}
		assert.Equal(t, want, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("override and extend", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2})
		data := bytes.Repeat([]byte{0, 1}, 500)

		// --- When ---
		n, err := fil.WriteAt(data, 1)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 1000, n)
		assert.Equal(t, 0, fil.Offset())
		assert.Equal(t, 1001, fil.Len())
		assert.Equal(t, 1004, fil.Cap())
		want := append([]byte{0}, bytes.Repeat([]byte{0, 1}, 500)...)
		assert.Equal(t, want, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("beyond capacity", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2})

		// --- When ---
		n, err := fil.WriteAt([]byte{3, 4, 5}, 1000)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, 0, fil.Offset())
		assert.Equal(t, 1003, fil.Len())
		assert.Equal(t, 1006, fil.Cap())
		want := append([]byte{0, 1, 2}, bytes.Repeat([]byte{0}, 997)...)
		want = append(want, []byte{3, 4, 5}...)
		assert.Equal(t, want, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("error - used with O_APPEND", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2}, WithFileAppend)

		// --- When ---
		n, err := fil.WriteAt([]byte{3}, 1)

		// --- Then ---
		assert.ErrorIs(t, errWriteAtInAppendMode, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, []byte{0, 1, 2}, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("error - cannot write to a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.WriteAt([]byte{0, 1, 2}, 3)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "write", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
		assert.Equal(t, 0, have)
	})
}

func Test_File_WriteAt_tabular(t *testing.T) {
	tt := []struct {
		testN string

		init []byte
		opts []func(*File)
		src  []byte
		off  int64
		wN   int
		wOff int
		wLen int
		wCap int
		wBuf []byte
	}{
		{
			testN: "zero value - write at zero offset",
			init:  nil,
			opts:  nil,
			src:   []byte{0, 1, 2},
			off:   0,
			wN:    3,
			wOff:  0,
			wLen:  3,
			wCap:  64,
			wBuf:  []byte{0, 1, 2},
		},
		{
			testN: "write at zero offset - override",
			init:  []byte{0, 1, 2},
			opts:  nil,
			src:   []byte{3, 4, 5},
			off:   0,
			wN:    3,
			wOff:  0,
			wLen:  3,
			wCap:  3,
			wBuf:  []byte{3, 4, 5},
		},
		{
			testN: "write at offset middle - no extend",
			init:  []byte{0, 1, 2},
			opts:  nil,
			src:   []byte{3, 4},
			off:   1,
			wN:    2,
			wOff:  0,
			wLen:  3,
			wCap:  3,
			wBuf:  []byte{0, 3, 4},
		},
		{
			testN: "write at offset middle - extend",
			init:  []byte{0, 1, 2},
			opts:  nil,
			src:   []byte{3, 4, 5},
			off:   1,
			wN:    3,
			wOff:  0,
			wLen:  4,
			wCap:  7,
			wBuf:  []byte{0, 3, 4, 5},
		},
		{
			testN: "append",
			init:  []byte{0, 1, 2},
			opts:  nil,
			src:   []byte{3, 4, 5},
			off:   3,
			wN:    3,
			wOff:  0,
			wLen:  6,
			wCap:  9,
			wBuf:  []byte{0, 1, 2, 3, 4, 5},
		},
		{
			testN: "write at offset beyond len - within cap",
			init:  make([]byte, 3, 6),
			opts:  nil,
			src:   []byte{1, 2},
			off:   4,
			wN:    2,
			wOff:  0,
			wLen:  6,
			wCap:  6,
			wBuf:  []byte{0, 0, 0, 0, 1, 2},
		},
		{
			testN: "write at offset beyond len - beyond cap",
			init:  make([]byte, 3, 6),
			opts:  nil,
			src:   []byte{1, 2},
			off:   5,
			wN:    2,
			wOff:  0,
			wLen:  7,
			wCap:  16,
			wBuf:  []byte{0, 0, 0, 0, 0, 1, 2},
		},
		{
			testN: "write at offset beyond cap",
			init:  make([]byte, 3, 6),
			opts:  nil,
			src:   []byte{1, 2},
			off:   8,
			wN:    2,
			wOff:  0,
			wLen:  10,
			wCap:  19,
			wBuf:  []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2},
		},
		{
			testN: "write at offset beyond cap - offset close to len",
			init:  make([]byte, 5, 7),
			opts:  []func(*File){WithFileOffset(4)},
			src:   []byte{1, 2},
			off:   8,
			wN:    2,
			wOff:  4,
			wLen:  10,
			wCap:  19,
			wBuf:  []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			var fil *File

			if tc.init == nil {
				fil = &File{} // Test for zero value.
			} else {
				fil = MustFileWith("file", tc.init, tc.opts...)
			}

			// --- When ---
			n, err := fil.WriteAt(tc.src, tc.off)

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.wN, n)
			assert.Equal(t, tc.wOff, fil.Offset())
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
			assert.Equal(t, tc.wBuf, fil.buf)
			assert.NoError(t, fil.Close())
		})
	}
}

func Test_File_WriteTo(t *testing.T) {
	t.Run("without offset", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3})
		dst := &bytes.Buffer{}

		// --- When ---
		have, err := fil.WriteTo(dst)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, int64(4), have)
		assert.Equal(t, []byte{0, 1, 2, 3}, dst.Bytes())
		assert.Equal(t, 4, fil.Offset())
	})

	t.Run("with offset", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3}, WithFileOffset(1))
		dst := &bytes.Buffer{}

		// --- When ---
		have, err := fil.WriteTo(dst)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, int64(3), have)
		assert.Equal(t, []byte{1, 2, 3}, dst.Bytes())
		assert.Equal(t, 4, fil.Offset())
	})

	t.Run("beyond capacity", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3}, WithFileOffset(4))
		dst := &bytes.Buffer{}

		// --- When ---
		have, err := fil.WriteTo(dst)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, int64(0), have)
		assert.Equal(t, []byte(nil), dst.Bytes())
		assert.Equal(t, 4, fil.Offset())
	})

	t.Run("error - cannot write to when a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.WriteTo(nil)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "write", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
		assert.Equal(t, int64(0), have)
	})
}

func Test_File_WriteString(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2}, WithFileOffset(1))

		// --- When ---
		n, err := fil.WriteString("abc")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte{0, 0x61, 0x62, 0x63}, fil.buf)
		assert.Equal(t, 4, fil.Offset())
	})

	t.Run("error - cannot write to a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.WriteString("abc")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "write", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
		assert.Equal(t, 0, have)
	})
}

func Test_File_Read(t *testing.T) {
	t.Run("read zero value", func(t *testing.T) {
		// --- Given ---
		fil := &File{}
		dst := make([]byte, 3)

		// --- When ---
		n, err := fil.Read(dst)

		// --- Then ---
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 0, n)
		assert.Equal(t, 0, fil.Offset())
		assert.Equal(t, 0, fil.Len())
		assert.Equal(t, 0, fil.Cap())
		want := []byte{0, 0, 0}
		assert.Equal(t, want, dst)
		assert.NoError(t, fil.Close())
	})

	t.Run("with a small buffer", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3, 4})
		dst := make([]byte, 3)

		// --- Then ---

		// First read.
		n, err := fil.Read(dst)

		assert.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, 3, fil.Offset())
		assert.Equal(t, 5, fil.Len())
		assert.Equal(t, 5, fil.Cap())
		assert.Equal(t, []byte{0, 1, 2}, dst)

		// Second read.
		n, err = fil.Read(dst)

		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, 5, fil.Offset())
		assert.Equal(t, 5, fil.Len())
		assert.Equal(t, 5, fil.Cap())
		assert.Equal(t, []byte{3, 4, 2}, dst)

		// Third read.
		n, err = fil.Read(dst)

		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 0, n)
		assert.Equal(t, 5, fil.Offset())
		assert.Equal(t, 5, fil.Len())
		assert.Equal(t, 5, fil.Cap())
		assert.Equal(t, []byte{3, 4, 2}, dst)
		assert.NoError(t, fil.Close())
	})

	t.Run("read big buffer", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2})
		dst := make([]byte, 6)

		// --- When ---
		n, err := fil.Read(dst)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, 3, fil.Offset())
		assert.Equal(t, 3, fil.Len())
		assert.Equal(t, 3, fil.Cap())
		assert.Equal(t, []byte{0, 1, 2, 0, 0, 0}, dst)
		assert.NoError(t, fil.Close())
	})

	t.Run("error - cannot read beyond length", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2})
		must.Value(fil.Seek(5, io.SeekStart))
		dst := make([]byte, 3)

		// --- When ---
		n, err := fil.Read(dst)

		// --- Then ---
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 0, n)
		assert.Equal(t, 5, fil.Offset())
		assert.Equal(t, 3, fil.Len())
		assert.Equal(t, 3, fil.Cap())
		assert.Equal(t, []byte{0, 0, 0}, dst)
		assert.NoError(t, fil.Close())
	})

	t.Run("error - cannot read a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		buf := make([]byte, 3)

		// --- When ---
		have, err := dir.Read(buf)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "read", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)

		assert.Equal(t, 0, have)
		assert.Equal(t, []byte{0, 0, 0}, buf)
	})
}

func Test_File_Read_tabular(t *testing.T) {
	tt := []struct {
		testN string

		content []byte
		opts    []func(*File)
		dst     []byte
		wN      int
		wOff    int
		wLen    int
		wCap    int
		wDst    []byte
	}{
		{
			testN:   "read all",
			content: []byte{0, 1, 2},
			opts:    nil,
			dst:     make([]byte, 3, 3),
			wN:      3,
			wOff:    3,
			wLen:    3,
			wCap:    3,
			wDst:    []byte{0, 1, 2},
		},
		{
			testN:   "read head",
			content: []byte{0, 1, 2},
			opts:    nil,
			dst:     make([]byte, 2, 3),
			wN:      2,
			wOff:    2,
			wLen:    3,
			wCap:    3,
			wDst:    []byte{0, 1},
		},
		{
			testN:   "read tail",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileOffset(1)},
			dst:     make([]byte, 2, 3),
			wN:      2,
			wOff:    3,
			wLen:    3,
			wCap:    3,
			wDst:    []byte{1, 2},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			fil := MustFileWith("file", tc.content, tc.opts...)

			// --- When ---
			n, err := fil.Read(tc.dst)

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.wN, n)
			assert.Equal(t, tc.wOff, fil.Offset())
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
			assert.Equal(t, tc.wDst, tc.dst)
			assert.NoError(t, fil.Close())
		})
	}
}

func Test_File_ReadByte(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2}, WithFileOffset(2))

		// --- When ---
		have, err := fil.ReadByte()

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 3, fil.Offset())
		assert.Equal(t, 3, fil.Len())
		assert.Equal(t, 3, fil.Cap())
		assert.Equal(t, byte(2), have)
		assert.NoError(t, fil.Close())
	})

	t.Run("eof", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2}, WithFileOffset(3))

		// --- When ---
		have, err := fil.ReadByte()

		// --- Then ---
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 3, fil.Offset())
		assert.Equal(t, 3, fil.Len())
		assert.Equal(t, 3, fil.Cap())
		assert.Equal(t, byte(0), have)
		assert.NoError(t, fil.Close())
	})

	t.Run("error - cannot read a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.ReadByte()

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "read", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
		assert.Equal(t, byte(0), have)
	})
}

func Test_File_ReadAt(t *testing.T) {
	t.Run("beyond length", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2})
		dst := make([]byte, 4)

		// --- When ---
		have, err := fil.ReadAt(dst, 6)

		// --- Then ---
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 0, have)
		assert.Equal(t, 0, fil.Offset())
		assert.Equal(t, 3, fil.Len())
		assert.Equal(t, 3, fil.Cap())
		assert.Equal(t, []byte{0, 0, 0, 0}, dst)
		assert.NoError(t, fil.Close())
	})

	t.Run("big buffer", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2}, WithFileOffset(1))
		dst := make([]byte, 4)

		// --- When ---
		have, err := fil.ReadAt(dst, 0)

		// --- Then ---
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 3, have)
		assert.Equal(t, 1, fil.Offset())
		assert.Equal(t, 3, fil.Len())
		assert.Equal(t, 3, fil.Cap())
		assert.Equal(t, []byte{0, 1, 2, 0}, dst)
		assert.NoError(t, fil.Close())
	})

	t.Run("error - cannot read a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		buf := make([]byte, 3)

		// --- When ---
		have, err := dir.ReadAt(buf, 1)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "read", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)

		assert.Equal(t, 0, have)
		assert.Equal(t, []byte{0, 0, 0}, buf)
	})
}

func Test_File_ReadAt_tabular(t *testing.T) {
	tt := []struct {
		testN string

		content []byte
		opts    []func(*File)
		dst     []byte
		off     int64
		wN      int
		wOff    int
		wLen    int
		wCap    int
		wDst    []byte
	}{
		{
			testN:   "read all",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileOffset(1)},
			dst:     make([]byte, 3),
			off:     0,
			wN:      3,
			wOff:    1,
			wLen:    3,
			wCap:    3,
			wDst:    []byte{0, 1, 2},
		},
		{
			testN:   "read head",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileOffset(1)},
			dst:     make([]byte, 2, 3),
			off:     0,
			wN:      2,
			wOff:    1,
			wLen:    3,
			wCap:    3,
			wDst:    []byte{0, 1},
		},
		{
			testN:   "read tail",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileOffset(2)},
			dst:     make([]byte, 2, 3),
			off:     1,
			wN:      2,
			wOff:    2,
			wLen:    3,
			wCap:    3,
			wDst:    []byte{1, 2},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			fil := MustFileWith("file", tc.content, tc.opts...)

			// --- When ---
			have, err := fil.ReadAt(tc.dst, tc.off)

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.wN, have)
			assert.Equal(t, tc.wOff, fil.Offset())
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
			assert.Equal(t, tc.wDst, tc.dst)
			assert.NoError(t, fil.Close())
		})
	}
}

func Test_File_ReadFrom(t *testing.T) {
	t.Run("error - cannot read a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")
		buf := &bytes.Buffer{}

		// --- When ---
		have, err := dir.ReadFrom(buf)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "write", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
		assert.Equal(t, int64(0), have)
	})
}

func Test_File_ReadFrom_tabular(t *testing.T) {
	tt := []struct {
		testN string

		content []byte
		opts    []func(*File)
		src     []byte
		wN      int64
		wOff    int
		wLen    int
		wCap    int
		wBuf    []byte
	}{
		{
			testN:   "zero value",
			content: nil,
			opts:    nil,
			src:     bytes.Repeat([]byte{1, 2, 3}, 1<<9),
			wN:      3 * 1 << 9,
			wOff:    3 * 1 << 9,
			wLen:    3 * 1 << 9,
			wCap:    3584,
			wBuf:    bytes.Repeat([]byte{1, 2, 3}, 1<<9),
		},
		{
			testN:   "append",
			content: []byte{0, 1, 2},
			opts:    []func(*File){WithFileAppend},
			src:     []byte{3, 4, 5},
			wN:      3,
			wOff:    6,
			wLen:    6,
			wCap:    518,
			wBuf:    []byte{0, 1, 2, 3, 4, 5},
		},
		{
			testN:   "read up to len",
			content: make([]byte, 3, 6),
			opts:    nil,
			src:     []byte{0, 1, 2},
			wN:      3,
			wOff:    3,
			wLen:    3,
			wCap:    524,
			wBuf:    []byte{0, 1, 2},
		},
		{
			testN:   "read up to cap",
			content: make([]byte, 3, 6),
			opts:    []func(*File){WithFileAppend},
			src:     []byte{3, 4, 5},
			wN:      3,
			wOff:    6,
			wLen:    6,
			wCap:    524,
			wBuf:    []byte{0, 0, 0, 3, 4, 5},
		},
		{
			testN:   "use of tmp space",
			content: bytes.Repeat([]byte{0}, 50),
			opts:    []func(*File){WithFileOffset(25)},
			src:     bytes.Repeat([]byte{1, 2, 3}, 1<<9),
			wN:      3 * 1 << 9,
			wOff:    3*1<<9 + 25,
			wLen:    3*1<<9 + 25,
			wCap:    3984,
			wBuf:    append(bytes.Repeat([]byte{0}, 25), bytes.Repeat([]byte{1, 2, 3}, 1<<9)...),
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			var fil *File
			if tc.content == nil {
				fil = &File{} // Test for zero value.
			} else {
				fil = MustFileWith("file", tc.content, tc.opts...)
			}

			// --- When ---
			have, err := fil.ReadFrom(bytes.NewReader(tc.src))

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.wN, have)
			assert.Equal(t, tc.wOff, fil.Offset())
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
			assert.Equal(t, tc.wBuf, fil.buf)
			assert.NoError(t, fil.Close())
		})
	}
}

func Test_File_String(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{'A', 'B', 'C'}, WithFileOffset(1))

		// --- When ---
		have := fil.String()

		// --- Then ---
		assert.Equal(t, "BC", have)
		assert.Equal(t, 3, fil.Offset())
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have := dir.String()

		// --- Then ---
		assert.Equal(t, "", have)
	})
}

func Test_File_Seek(t *testing.T) {
	t.Run("error - negative final offset", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2})

		// --- When ---
		have, err := fil.Seek(-4, io.SeekEnd)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "file", e.Path)
		assert.Equal(t, int64(0), have)
	})

	t.Run("error - negative final offset with file name", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2})

		// --- When ---
		have, err := fil.Seek(-4, io.SeekEnd)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "file", e.Path)
		assert.Equal(t, int64(0), have)
	})

	t.Run("seek beyond length", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2})

		// --- When ---
		have, err := fil.Seek(5, io.SeekStart)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, int64(5), have)
	})

	t.Run("error - cannot seek a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		have, err := dir.Seek(0, 0)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "seek", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
		assert.Equal(t, int64(0), have)
	})
}

func Test_File_Seek_tabular(t *testing.T) {
	// --- Given ---
	tt := []struct {
		testN string

		seek   int64
		whence int
		wantN  int64
		wantD  []byte
	}{
		{"1", 0, io.SeekCurrent, 1, []byte{1, 2, 3}},
		{"2", 0, io.SeekEnd, 4, []byte{}},
		{"3", -1, io.SeekEnd, 3, []byte{3}},
		{"4", -3, io.SeekEnd, 1, []byte{1, 2, 3}},
		{"5", 0, io.SeekStart, 0, []byte{0, 1, 2, 3}},
		{"6", 2, io.SeekStart, 2, []byte{2, 3}},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			fil := MustFileWith("file", []byte{0, 1, 2, 3}, WithFileOffset(1))

			// --- When ---
			have, err := fil.Seek(tc.seek, tc.whence)

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.wantN, have)
			assert.Equal(t, tc.wantD, must.Value(io.ReadAll(fil)))
			assert.NoError(t, fil.Close())
		})
	}
}

func Test_File_SeekStart(t *testing.T) {
	// --- Given ---
	fil := MustFileWith("file", []byte{0, 1, 2}, WithFileOffset(2))

	// --- When ---
	n := fil.SeekStart()

	// --- Then ---
	assert.Equal(t, int64(2), n)
	assert.Equal(t, 0, fil.off)
}

func Test_File_SeekEnd(t *testing.T) {
	// --- Given ---
	fil := MustFileWith("file", []byte{0, 1, 2}, WithFileOffset(1))

	// --- When ---
	n := fil.SeekEnd()

	// --- Then ---
	assert.Equal(t, int64(1), n)
	assert.Equal(t, len(fil.buf), fil.off)
}

func Test_File_Truncate(t *testing.T) {
	t.Run("truncate to zero and write", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3})
		must.Nil(fil.Truncate(0))

		// --- When ---
		have, err := fil.Write([]byte{4, 5})

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 2, have)
		assert.Equal(t, 2, fil.Offset())
		assert.Equal(t, 2, fil.Len())
		assert.Equal(t, 4, fil.Cap())
		assert.Equal(t, []byte{4, 5}, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("truncate beyond length and write", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3}, WithFileAppend)
		must.Value(fil.Seek(1, io.SeekStart))
		must.Nil(fil.Truncate(8))

		// --- When ---
		have, err := fil.Write([]byte{4, 5})

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 2, have)
		assert.Equal(t, 10, fil.Offset())
		assert.Equal(t, 10, fil.Len())
		assert.Equal(t, 12, fil.Cap())
		assert.Equal(t, []byte{0, 1, 2, 3, 0, 0, 0, 0, 4, 5}, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("truncate beyond capacity and write", func(t *testing.T) {
		// --- Given ---
		content := make([]byte, 4, 8)
		content[0] = 0
		content[1] = 1
		content[2] = 2
		content[3] = 3
		fil := MustFileWith("file", content, WithFileAppend)
		must.Nil(fil.Truncate(10))

		// --- When ---
		have, err := fil.Write([]byte{4, 5})

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 2, have)
		assert.Equal(t, 12, fil.Offset())
		assert.Equal(t, 12, fil.Len())
		assert.Equal(t, 22, fil.Cap())
		assert.Equal(t, []byte{0, 1, 2, 3, 0, 0, 0, 0, 0, 0, 4, 5}, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("extend beyond length, then reset and write", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3}, WithFileAppend)
		must.Nil(fil.Truncate(8))
		must.Nil(fil.Truncate(0))

		// --- When ---
		have, err := fil.Write([]byte{4, 5})

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 2, have)
		assert.Equal(t, 2, fil.Offset())
		assert.Equal(t, 2, fil.Len())
		assert.Equal(t, 12, fil.Cap())
		assert.Equal(t, []byte{4, 5}, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("edge case when size equals length", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2, 3}, WithFileAppend)
		must.Nil(fil.Truncate(4))

		// --- When ---
		n, err := fil.Write([]byte{4, 5})

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, 6, fil.Offset())
		assert.Equal(t, 6, fil.Len())
		assert.Equal(t, 10, fil.Cap())
		assert.Equal(t, []byte{0, 1, 2, 3, 4, 5}, fil.buf)
		assert.NoError(t, fil.Close())
	})

	t.Run("error - invalid truncate value", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2}, WithFileAppend)

		// --- When ---
		err := fil.Truncate(-1)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
	})

	t.Run("error - invalid truncate value with name", func(t *testing.T) {
		// --- Given ---
		fil := MustFileWith("file", []byte{0, 1, 2}, WithFileAppend)

		// --- When ---
		err := fil.Truncate(-1)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "file", e.Path)
	})

	t.Run("error - cannot truncate a directory", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		err := dir.Truncate(0)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "truncate", e.Op)
		assert.Equal(t, "dir", e.Path)
		assert.Equal(t, syscall.EISDIR, e.Err)
	})
}

func Test_File_Truncate_tabular(t *testing.T) {
	tt := []struct {
		testN string

		content []byte
		opts    []func(*File)
		off     int64
		wOff    int
		wLen    int
		wCap    int
		wBuf    []byte
	}{
		{
			testN:   "truncate to zero",
			content: []byte{0, 1, 2, 3},
			opts:    nil,
			off:     0,
			wOff:    0,
			wLen:    0,
			wCap:    4,
			wBuf:    []byte{},
		},
		{
			testN:   "truncate to one",
			content: []byte{0, 1, 2, 3},
			opts:    nil,
			off:     1,
			wOff:    0,
			wLen:    1,
			wCap:    4,
			wBuf:    []byte{0},
		},
		{
			testN:   "truncate beyond len, less then cap",
			content: make([]byte, 3, 5),
			opts:    nil,
			off:     4,
			wOff:    0,
			wLen:    4,
			wCap:    5,
			wBuf:    []byte{0, 0, 0, 0},
		},
		{
			testN:   "truncate beyond cap",
			content: make([]byte, 3, 5),
			opts:    nil,
			off:     6,
			wOff:    0,
			wLen:    6,
			wCap:    13,
			wBuf:    []byte{0, 0, 0, 0, 0, 0},
		},
		{
			testN:   "truncate at len",
			content: make([]byte, 3, 5),
			opts:    nil,
			off:     3,
			wOff:    0,
			wLen:    3,
			wCap:    5,
			wBuf:    []byte{0, 0, 0},
		},
		{
			testN:   "truncate at cap",
			content: make([]byte, 3, 5),
			opts:    nil,
			off:     5,
			wOff:    0,
			wLen:    5,
			wCap:    5,
			wBuf:    []byte{0, 0, 0, 0, 0},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			var fil *File
			if tc.content == nil {
				fil = &File{} // Test for zero value.
			} else {
				fil = MustFileWith("file", tc.content, tc.opts...)
			}

			// --- When ---
			err := fil.Truncate(tc.off)

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.wOff, fil.Offset())
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
			assert.Equal(t, tc.wBuf, fil.buf)
			assert.NoError(t, fil.Close())
		})
	}
}

func Test_File_Grow(t *testing.T) {
	t.Run("grow", func(t *testing.T) {
		// --- Given ---
		content := make([]byte, 10, 15)
		fil := MustFileWith("file", content, WithFileOffset(5))

		// --- When ---
		fil.Grow(20)

		// --- Then ---
		assert.Equal(t, 10, fil.Len())
		assert.Equal(t, 30, fil.Cap())
		assert.Equal(t, 5, fil.Offset())
	})

	t.Run("already enough space", func(t *testing.T) {
		// --- Given ---
		content := make([]byte, 10, 15)
		fil := MustFileWith("file", content, WithFileOffset(5))

		// --- When ---
		fil.Grow(5)

		// --- Then ---
		assert.Equal(t, 10, fil.Len())
		assert.Equal(t, 15, fil.Cap())
		assert.Equal(t, 5, fil.Offset())
	})

	t.Run("grow directory has no effect", func(t *testing.T) {
		// --- Given ---
		dir := MustDirectory("dir")

		// --- When ---
		dir.Grow(20)

		// --- Then ---
		assert.Nil(t, dir.buf)
	})

	t.Run("panic - negative argument", func(t *testing.T) {
		// --- Given ---
		buf := &File{}

		// --- Then ---
		assert.Panic(t, func() { buf.Grow(-1) })
	})
}

func Test_File_grow_tabular(t *testing.T) {
	tt := []struct {
		testN string

		len  int
		cap  int
		off  int
		grow int
		wLen int
		wCap int
	}{
		{
			testN: "1",
			len:   0,
			cap:   100,
			off:   0,
			grow:  50,
			wLen:  50,
			wCap:  100,
		},
		{
			testN: "2",
			len:   10,
			cap:   100,
			off:   10,
			grow:  50,
			wLen:  60,
			wCap:  100,
		},
		{
			testN: "3",
			len:   0,
			cap:   100,
			off:   0,
			grow:  100,
			wLen:  100,
			wCap:  100,
		},
		{
			testN: "4",
			len:   10,
			cap:   100,
			off:   10,
			grow:  90,
			wLen:  100,
			wCap:  100,
		},
		{
			testN: "5",
			len:   10,
			cap:   100,
			off:   5,
			grow:  150,
			wLen:  350,
			wCap:  350,
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			content := make([]byte, tc.len, tc.cap)
			fil := MustFileWith("file", content, WithFileOffset(tc.off))

			// --- When ---
			fil.grow(tc.grow)

			// --- Then ---
			assert.Equal(t, tc.off, fil.off)
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
		})
	}
}

func Test_File_tryGrowByReslice_tabular(t *testing.T) {
	tt := []struct {
		testN string

		len  int
		cap  int
		off  int
		grow int
		wOK  bool
		wLen int
		wCap int
	}{
		{
			testN: "1",
			len:   0,
			cap:   100,
			off:   0,
			grow:  50,
			wOK:   true,
			wLen:  50,
			wCap:  100,
		},
		{
			testN: "2",
			len:   10,
			cap:   100,
			off:   10,
			grow:  50,
			wOK:   true,
			wLen:  60,
			wCap:  100,
		},
		{
			testN: "3",
			len:   0,
			cap:   100,
			off:   0,
			grow:  100,
			wOK:   true,
			wLen:  100,
			wCap:  100,
		},
		{
			testN: "4",
			len:   10,
			cap:   100,
			off:   10,
			grow:  90,
			wOK:   true,
			wLen:  100,
			wCap:  100,
		},
		{
			testN: "5",
			len:   10,
			cap:   100,
			off:   10,
			grow:  150,
			wOK:   false,
			wLen:  10,
			wCap:  100,
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- Given ---
			content := make([]byte, tc.len, tc.cap)
			fil := MustFileWith("file", content, WithFileOffset(tc.off))

			// --- When ---
			have := fil.tryGrowByReslice(tc.grow)

			// --- Then ---
			assert.Equal(t, tc.wOK, have)
			assert.Equal(t, tc.off, fil.off)
			assert.Equal(t, tc.wLen, fil.Len())
			assert.Equal(t, tc.wCap, fil.Cap())
		})
	}
}

func Test_File_Offset(t *testing.T) {
	// --- Given ---
	fil := &File{off: 42}

	// --- When ---
	have := fil.Offset()

	// --- Then ---
	assert.Equal(t, 42, have)
}

func Test_File_Len(t *testing.T) {
	// --- Given ---
	fil := &File{buf: make([]byte, 42, 44)}

	// --- When ---
	have := fil.Len()

	// --- Then ---
	assert.Equal(t, 42, have)
}

func Test_File_Cap(t *testing.T) {
	// --- Given ---
	fil := &File{buf: make([]byte, 42, 44)}

	// --- When ---
	have := fil.Cap()

	// --- Then ---
	assert.Equal(t, 44, have)
}

func Test_File_path(t *testing.T) {
	t.Run("level 1 file", func(t *testing.T) {
		// --- Given ---
		root := tstDirMem()
		fil := must.Value(open(root, "file0"))

		// --- When ---
		have := fil.path()

		// --- Then ---
		assert.Equal(t, "file0", have)
	})

	t.Run("level 1 directory", func(t *testing.T) {
		// --- Given ---
		root := tstDirMem()
		fil := must.Value(open(root, "sub"))

		// --- When ---
		have := fil.path()

		// --- Then ---
		assert.Equal(t, "sub", have)
	})

	t.Run("level 3 file", func(t *testing.T) {
		// --- Given ---
		root := tstDirMem()
		fil := must.Value(open(root, "sub/sub2/file5"))

		// --- When ---
		have := fil.path()

		// --- Then ---
		assert.Equal(t, "sub/sub2/file5", have)
	})

	t.Run("level 2 directory", func(t *testing.T) {
		// --- Given ---
		root := tstDirMem()
		fil := must.Value(open(root, "sub/sub2"))

		// --- When ---
		have := fil.path()

		// --- Then ---
		assert.Equal(t, "sub/sub2", have)
	})
}

func Test_File_Close(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		// --- When ---
		fil := &File{}

		// --- Then ---
		assert.NoError(t, fil.Close())
	})

	t.Run("nil file instance", func(t *testing.T) {
		// --- When ---
		var fil *File

		// --- Then ---
		assert.NoError(t, fil.Close())
	})

	t.Run("zero out underlying buffer", func(t *testing.T) {
		// --- Given ---
		content := make([]byte, 4, 10)
		copy(content, []byte{0, 1, 2, 3})
		fil := &File{off: 4, buf: content}

		// --- When ---
		err := fil.Close()

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 4, fil.buf)
		assert.Cap(t, 10, fil.buf)
		assert.Equal(t, []byte{0, 1, 2, 3}, fil.buf)
	})
}

func Test_File_List(t *testing.T) {
	wantList := "" +
		".\n" +
		"file0\n" +
		"file1\n" +
		"file2\n" +
		"sub\n" +
		"sub/file3\n" +
		"sub/file4\n" +
		"sub/sub2\n" +
		"sub/sub2/file5\n" +
		"sub/sub2/file6\n"

	t.Run("memfs directory", func(t *testing.T) {
		// --- Given ---
		dir := tstDirMem()

		// --- When ---
		have, err := dir.List()

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, wantList, have)
	})

	t.Run("os directory", func(t *testing.T) {
		// --- Given ---
		dir := tstDirOS(t)
		root := must.Value(os.OpenRoot(dir))

		// --- When ---
		have, err := list(root.FS())

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, wantList, have)
	})

	t.Run("error - cannot list a file", func(t *testing.T) {
		// --- Given ---
		fil := MustFile("file")

		// --- When ---
		have, err := fil.List()

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "list", e.Op)
		assert.Equal(t, "file", e.Path)
		assert.Equal(t, syscall.ENOTDIR, e.Err)
		assert.Empty(t, have)
	})
}

func Test_zeroOutSlice(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		// --- Given ---
		data := []byte{0, 1, 2, 3}

		// --- When ---
		zeroOutSlice(data)

		// --- Then ---
		assert.Equal(t, []byte{0, 0, 0, 0}, data)
	})

	t.Run("nil slice", func(t *testing.T) {
		// --- Given ---
		var data []byte

		// --- When ---
		zeroOutSlice(data)
	})
}

func Test_fsDir_ReadDir(t *testing.T) {
	t.Run("reading directory", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.ReadDir("sub")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 3, len(have))
		assert.Equal(t, "file3", have[0].Name())
		assert.Equal(t, "file4", have[1].Name())
		assert.Equal(t, "sub2", have[2].Name())
	})

	t.Run("nested directory path", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.ReadDir("sub/sub2")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 2, len(have))
		assert.Equal(t, "file5", have[0].Name())
		assert.Equal(t, "file6", have[1].Name())
	})

	t.Run("error - reading a file", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.ReadDir("file0")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "readdirent", e.Op)
		assert.Equal(t, "file0", e.Path)
		assert.Equal(t, syscall.ENOTDIR, e.Err)
		assert.Nil(t, have)
	})

	t.Run("error - invalid path", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{MustDirectory("dir")}

		// --- When ---
		have, err := dir.ReadDir("/root")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "open", e.Op)
		assert.Equal(t, "/root", e.Path)
		assert.Equal(t, fs.ErrInvalid, e.Err)
		assert.Nil(t, have)
	})
}

func Test_fsDir_Open(t *testing.T) {
	t.Run("open file", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.Open("file0")

		// --- Then ---
		assert.NoError(t, err)
		assert.False(t, must.Value(have.Stat()).IsDir())
		assert.Equal(t, "file0", must.Value(have.Stat()).Name())
		assert.NoError(t, have.Close())
	})

	t.Run("open directory", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.Open("sub")

		// --- Then ---
		assert.NoError(t, err)
		assert.True(t, must.Value(have.Stat()).IsDir())
		assert.Equal(t, "sub", must.Value(have.Stat()).Name())
		assert.NoError(t, have.Close())
	})

	t.Run("error - not existing", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.Open("not-existing")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "openat", e.Op)
		assert.Equal(t, "not-existing", e.Path)
		assert.Equal(t, syscall.ENOENT, e.Err)
		assert.Nil(t, have)
	})
}

func Test_fdDir_Stat(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.Stat("file0")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "file0", have.Name())
		assert.Equal(t, int64(5), have.Size())
		assert.Equal(t, fs.FileMode(0600), have.Mode())
		assert.Zero(t, have.ModTime())
		assert.False(t, have.IsDir())
		assert.Nil(t, have.Sys())
	})

	t.Run("directory", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.Stat("sub")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "sub", have.Name())
		assert.Equal(t, int64(4096), have.Size())
		assert.Equal(t, fs.FileMode(0700)|fs.ModeDir, have.Mode())
		assert.Zero(t, have.ModTime())
		assert.True(t, have.IsDir())
		assert.Nil(t, have.Sys())
	})

	t.Run("error - not existing", func(t *testing.T) {
		// --- Given ---
		dir := fsDir{tstDirMem()}

		// --- When ---
		have, err := dir.Stat("not-existing")

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "statat", e.Op)
		assert.Equal(t, "not-existing", e.Path)
		assert.Equal(t, syscall.ENOENT, e.Err)
		assert.Nil(t, have)
	})
}
