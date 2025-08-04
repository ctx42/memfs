// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/ctx42/testing/pkg/must"
	"github.com/ctx42/testing/pkg/tester"
)

// file is an interface common to [os.File] and [File].
type file interface {
	io.Seeker
	io.Reader
	io.ReaderAt
	io.Closer
	io.ReaderFrom
	io.Writer
	io.WriterAt
	io.StringWriter
	io.WriterTo

	Stat() (os.FileInfo, error)
	Truncate(size int64) error
}

// MustFile is a helper calling [NewFile] which panics on error.
func MustFile(name string, opts ...func(*File)) *File {
	return must.Value(NewFile(name, opts...))
}

// MustFileWith is a helper calling [FileWith] which panics on error.
func MustFileWith(name string, content []byte, opts ...func(*File)) *File {
	return must.Value(FileWith(name, content, opts...))
}

// MustDirectory is a helper calling [NewDirectory] which panics on error.
func MustDirectory(name string) *File {
	return must.Value(NewDirectory(name))
}

// fileCreator creates a new file instance with given content (if it's not nil)
// and the seek offset at its beginning. The returned file will automatically
// be closed at the test end.
type fileCreator func(t tester.T, flag int, content []byte) file

// createOSFile creates a new [os.File] instance with given content (if it is
// not nil) and the seek offset at its beginning. The returned file will
// automatically be closed at the test end.
func createOSFile(t tester.T, flag int, content []byte) file {
	t.Helper()

	umask := syscall.Umask(0)
	defer syscall.Umask(umask)

	pth := filepath.Join(t.TempDir(), "file.txt")
	fil := must.Value(os.OpenFile(pth, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600))
	if content != nil {
		must.Value(fil.Write(content))
	}
	must.Nil(fil.Close())
	fil = must.Value(os.OpenFile(pth, flag, 0666))
	t.Cleanup(func() { t.Helper(); must.Nil(fil.Close()) })
	return fil
}

// createKitFile creates a [File] instance with given content and the seek
// offset at its beginning. The returned file will automatically be closed at
// the test end.
func createKitFile(t tester.T, flag int, content []byte) file {
	t.Helper()
	fil := must.Value(FileWith("file.txt", content, WithFileFlag(flag)))
	t.Cleanup(func() { t.Helper(); must.Nil(fil.Close()) })
	return fil
}

// tstDirOS returns an absolute path to a test directory structure.
//
// Structure with file contents in square brackets:
//
//	dir/
//	  sub/
//	    file3 [file3]
//	    file4 [file4]
//	    sub2/
//	      file5 [file5]
//	      file6 [file6]
//	  file0 [file0]
//	  file1 [file1]
//	  file2 [file2]
func tstDirOS(t *testing.T) string {
	dir := filepath.Join(t.TempDir(), "dir")
	sub := filepath.Join(dir, "sub")
	sub2 := filepath.Join(sub, "sub2")
	must.Nil(os.Mkdir(dir, 0700))
	must.Nil(os.Mkdir(sub, 0700))
	must.Nil(os.Mkdir(sub2, 0700))
	must.Nil(os.WriteFile(filepath.Join(dir, "file0"), []byte("file0"), 0600))
	must.Nil(os.WriteFile(filepath.Join(dir, "file1"), []byte("file1"), 0600))
	must.Nil(os.WriteFile(filepath.Join(dir, "file2"), []byte("file2"), 0600))
	must.Nil(os.WriteFile(filepath.Join(sub, "file3"), []byte("file3"), 0600))
	must.Nil(os.WriteFile(filepath.Join(sub, "file4"), []byte("file4"), 0600))
	must.Nil(os.WriteFile(filepath.Join(sub2, "file5"), []byte("file5"), 0600))
	must.Nil(os.WriteFile(filepath.Join(sub2, "file6"), []byte("file6"), 0600))
	return dir
}

// tstDirMem returns directory structure matching one created by the tstDirOS.
func tstDirMem() *File {
	sub := must.Value(NewDirectory("sub"))
	must.Nil(sub.AddFile(MustFileWith("file3", []byte("file3"))))
	must.Nil(sub.AddFile(MustFileWith("file4", []byte("file4"))))

	sub2 := MustDirectory("sub2")
	must.Nil(sub2.AddFile(MustFileWith("file5", []byte("file5"))))
	must.Nil(sub2.AddFile(MustFileWith("file6", []byte("file6"))))
	must.Nil(sub.AddFile(sub2))

	dir := NewRoot()
	must.Nil(dir.AddFile(sub))
	must.Nil(dir.AddFile(MustFileWith("file0", []byte("file0"))))
	must.Nil(dir.AddFile(MustFileWith("file1", []byte("file1"))))
	must.Nil(dir.AddFile(MustFileWith("file2", []byte("file2"))))

	return dir
}
