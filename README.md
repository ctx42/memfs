# memfs

[![GoDoc](https://pkg.go.dev/badge/github.com/ctx42/memfs?status.svg)](https://pkg.go.dev/github.com/ctx42/memfs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Go Report Card](https://goreportcard.com/badge/github.com/ctx42/memfs)](https://goreportcard.com/report/github.com/ctx42/memfs)

**memfs** is a lightweight, in-memory file system implementation in Go,
designed for efficient file and directory operations without disk I/O. It
provides a flexible and performant alternative for scenarios like unit testing,
or temporary data manipulation in memory. The package implements key interfaces
from the `io` and `io/fs` standard library packages, making it seamlessly
integrable with existing Go file system APIs.

The file and directory behavior mimic the `os` package as closely as possible -
see tests in [compare_test.go](pkg/memfs/compare_test.go) file.

The workhorse of the package is the `memfs.File` type which is used to represent 
files or directories and implements a lot of useful interfaces.

## Key Features

**Comprehensive Interface Support**: File implements a wide range of standard
library interfaces, including:

- `fmt.Stringer`
- `fs.DirEntry`
- `fs.File`
- `fs.FileInfo`
- `fs.FS`
- `fs.ReadDirFile`
- `fs.ReadFileFS`
- `io.ByteReader`
- `io.ByteWriter`
- `io.Closer`
- `io.Reader`
- `io.ReaderAt`
- `io.Seeker`
- `io.StringWriter`
- `io.ReaderFrom`
- `io.Writer`
- `io.WriterAt`
- `io.WriterTo`

**In-Memory Operations**: All data is stored in RAM, enabling ultra-fast
read/write access without the overhead of disk operations. Ideal for
performance-critical applications where filesystem is needed or environments
where persistence is not required.

**File and Directory Management**:

- Supports creating regular files and directories.
- Handles file appending, truncation, seeking.
- Directory entries can be added, read, and traversed recursively.

**Efficiency and Optimization**:

- Uses reslicing for buffer growth where possible to minimize allocations.
- Zeroes out unused buffer space to prevent data leaks.
- Supports initial buffer capacities for optimized reads/writes.

**Portability and Simplicity**: Pure Go implementation with no external
  dependencies beyond the standard library. Easy to embed in any Go project.

This package excels in testing frameworks (e.g., mocking file systems), 
embedded systems, or applications requiring ephemeral storage, offering a 
robust alternative to disk-based file systems like `os.DirFS`.

## Installation

```shell
go get github.com/ctx42/memfs
```

## Examples

### Creating an Empty File

```go
fil, _ := memfs.NewFile("file")

_, _ = fil.Write([]byte{0, 1, 2, 3})
_, _ = fil.Seek(-2, io.SeekEnd)
_, _ = fil.Write([]byte{4, 5})
_, _ = fil.Seek(0, io.SeekStart)

content, _ := io.ReadAll(fil)
fmt.Println(content)

// Output: [0 1 4 5]
```

### Creating a File With Content

```go
fil, _ := memfs.FileWith("file", []byte{0, 1, 2, 3})

_, _ = fil.Seek(-2, io.SeekEnd)
_, _ = fil.Write([]byte{4, 5})
_, _ = fil.Seek(0, io.SeekStart)

content, _ := io.ReadAll(fil)
fmt.Println(content)

// Output: [0 1 4 5]
```

### Append Mode

```go
fil, _ := memfs.FileWith("file", []byte{0, 1, 2, 3}, memfs.WithFileAppend)

_, _ = fil.Seek(-2, io.SeekEnd)
_, _ = fil.Write([]byte{4, 5})
_, _ = fil.Seek(0, io.SeekStart)

content, _ := io.ReadAll(fil)
fmt.Println(content)

// Output: [0 1 2 3 4 5]
```

### Creating a Directory and Adding Files

```go
fil0, _ := memfs.FileWith("file0", []byte{0})
fil1, _ := memfs.FileWith("file1", []byte{1})
dir := memfs.NewDirectory("dir")
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

### Using as fs.FS interface.

```go
fsys := dir.FS()
data, _ := fs.ReadFile(fsys, "file1.txt")
fmt.Println(string(data)) // Output: Content 1
```

See more examples in [examples_test.go](pkg/memfs/examples_test.go)

For more advanced usage, refer to
the [GoDoc documentation](https://pkg.go.dev/github.com/ctx42/memfs).

## Contributing

Contributions are welcome! Please submit issues or pull requests on
the [GitHub repository](https://github.com/ctx42/memfs). Ensure the code adheres
to Go conventions and includes tests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md)
file for details.
