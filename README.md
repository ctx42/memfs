[![Go](https://github.com/ctx42/memfs/actions/workflows/go.yml/badge.svg)](https://github.com/ctx42/memfs/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ctx42/memfs?status.svg)](https://pkg.go.dev/github.com/ctx42/memfs/pkg/memfs)
[![Go Report Card](https://goreportcard.com/badge/github.com/ctx42/memfs)](https://goreportcard.com/report/github.com/ctx42/memfs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE.md)

# memfs

In-memory files and directories for Go that behave like `os.File` and plug
into the `io` and `io/fs` APIs.

**memfs** keeps file and directory contents in memory instead of on disk. Its
`memfs.File` type represents both files and directories and implements the
standard `io` and `io/fs` interfaces, so it can stand in wherever code takes
one of them — in unit tests, or for temporary data that never needs to touch
the disk.

The behavior follows the `os` package as closely as possible: the tests in
[compare_test.go](pkg/memfs/compare_test.go) run the same checks against
`os.File` and `memfs.File`.

## Features

- **Files and directories in one type**: create regular files and
  directories, write, append, truncate, and seek, and add, read, and walk
  directory entries.
- **Standard interfaces**: `memfs.File` implements:
  - `fmt.Stringer`
  - `fs.DirEntry`
  - `fs.FS`
  - `fs.File`
  - `fs.FileInfo`
  - `fs.ReadDirFile`
  - `fs.ReadFileFS`
  - `io.ByteReader`
  - `io.ByteWriter`
  - `io.Closer`
  - `io.Reader`
  - `io.ReaderAt`
  - `io.ReaderFrom`
  - `io.Seeker`
  - `io.StringWriter`
  - `io.Writer`
  - `io.WriterAt`
  - `io.WriterTo`
- **A directory as an `fs.FS`**: `File.FS()` returns a file system that also
  implements `fs.StatFS`, `fs.ReadFileFS`, and `fs.ReadDirFS`, so `fs.ReadFile`,
  `fs.Stat`, and `fs.WalkDir` work on it.
- **Buffer control**: start from existing content or a preallocated capacity
  with `FileWith`, and take the buffer back with `Release`. The buffer grows by
  reslicing where possible, and unused capacity is kept zeroed.
- **No dependencies**: the package imports only the standard library.

## Prerequisites

Go 1.26 or later.

## Installation

```shell
go get github.com/ctx42/memfs
```

Then import the package:

```go
import "github.com/ctx42/memfs/pkg/memfs"
```

## Usage

### Creating an Empty File

<!-- gmmce:pkg/memfs/ExampleNewFile -->
```go
fil, _ := memfs.NewFile("file")

_, _ = fil.Write([]byte{0, 1, 2, 3})
_, _ = fil.Seek(-2, io.SeekEnd)
_, _ = fil.Write([]byte{4, 5})
_, _ = fil.Seek(0, io.SeekStart)

content, _ := io.ReadAll(fil)
fmt.Printf("%v\n", content)

// Output: [0 1 4 5]
```

### Creating a File With Content

<!-- gmmce:pkg/memfs/ExampleFileWith -->
```go
fil, _ := memfs.FileWith("file", []byte{0, 1, 2, 3})

_, _ = fil.Seek(-2, io.SeekEnd)
_, _ = fil.Write([]byte{4, 5})
_, _ = fil.Seek(0, io.SeekStart)

content, _ := io.ReadAll(fil)
fmt.Printf("%v\n", content)

// Output: [0 1 4 5]
```

### Append Mode

<!-- gmmce:pkg/memfs/ExampleFileWith_appendMode -->
```go
fil, _ := memfs.FileWith("file", []byte{0, 1, 2, 3}, memfs.WithFileAppend)

_, _ = fil.Seek(-2, io.SeekEnd)
_, _ = fil.Write([]byte{4, 5})
_, _ = fil.Seek(0, io.SeekStart)

content, _ := io.ReadAll(fil)
fmt.Printf("%v\n", content)

// Output: [0 1 2 3 4 5]
```

### Creating a Directory and Adding Files

<!-- gmmce:pkg/memfs/ExampleNewDirectory -->
```go
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
```

### Using a Directory as an fs.FS

<!-- gmmce:pkg/memfs/ExampleFile_FS -->
```go
fil0, _ := memfs.FileWith("file0", []byte("content 0"))
fil1, _ := memfs.FileWith("file1", []byte("content 1"))
sub, _ := memfs.NewDirectory("sub")
_ = sub.AddFile(fil1)

root := memfs.NewRoot()
_ = root.AddFile(fil0)
_ = root.AddFile(sub)

fsys := root.FS()
data, _ := fs.ReadFile(fsys, "sub/file1")
fmt.Println(string(data))

_ = fs.WalkDir(fsys, ".", func(pth string, _ fs.DirEntry, _ error) error {
	fmt.Println(pth)
	return nil
})

// Output:
// content 1
// .
// file0
// sub
// sub/file1
```

## Behavior Notes

- A directory hands out its entries, not copies: `Open` returns the same
  `*memfs.File` every time, so everyone who opens a file shares its offset.
  `File.ReadFile` and `fs.ReadFile` on `File.FS()` return the whole content
  and leave the offset alone.
- `Close` resets the offset and the `ReadDir` position to zero; the content
  stays.
- In append mode (`memfs.WithFileAppend`) every write goes to the end of the
  file, and `WriteAt` returns an error.
- A `memfs.File` is not safe for concurrent use.

## Resources

- More examples: [examples_test.go](pkg/memfs/examples_test.go).
- API reference:
  [pkg.go.dev](https://pkg.go.dev/github.com/ctx42/memfs/pkg/memfs).

## License

This project is licensed under the MIT License — see [LICENSE.md](LICENSE.md)
for details.
