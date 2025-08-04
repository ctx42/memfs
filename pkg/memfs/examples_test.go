// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs_test

import (
	"fmt"
	"io"
	"io/fs"

	"github.com/ctx42/memfs/pkg/memfs"
)

func ExampleNewFile() {
	fil, _ := memfs.NewFile("file")

	_, _ = fil.Write([]byte{0, 1, 2, 3})
	_, _ = fil.Seek(-2, io.SeekEnd)
	_, _ = fil.Write([]byte{4, 5})
	_, _ = fil.Seek(0, io.SeekStart)

	content, _ := io.ReadAll(fil)
	fmt.Println(content)

	// Output: [0 1 4 5]
}

func ExampleFileWith() {
	fil, _ := memfs.FileWith("file", []byte{0, 1, 2, 3})

	_, _ = fil.Seek(-2, io.SeekEnd)
	_, _ = fil.Write([]byte{4, 5})
	_, _ = fil.Seek(0, io.SeekStart)

	content, _ := io.ReadAll(fil)
	fmt.Println(content)

	// Output: [0 1 4 5]
}

func ExampleFileWith_appendMode() {
	fil, _ := memfs.FileWith("file", []byte{0, 1, 2, 3}, memfs.WithFileAppend)

	_, _ = fil.Seek(-2, io.SeekEnd)
	_, _ = fil.Write([]byte{4, 5})
	_, _ = fil.Seek(0, io.SeekStart)

	content, _ := io.ReadAll(fil)
	fmt.Println(content)

	// Output: [0 1 2 3 4 5]
}

func ExampleNewDirectory() {
	fil0, _ := memfs.FileWith("file0", []byte{0})
	fil1, _ := memfs.FileWith("file1", []byte{1})
	dir, _ := memfs.NewDirectory("dir")
	_ = dir.AddFile(fil0)
	_ = dir.AddFile(fil1)

	entries, _ := dir.ReadDir(-1)
	for _, entry := range entries {
		info, _ := entry.Info()
		fmt.Println(fs.FormatFileInfo(info))
	}

	// Output:
	// -rw------- 1 0001-01-01 00:00:00 file0
	// -rw------- 1 0001-01-01 00:00:00 file1
}
