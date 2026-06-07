This file provides guidance to AI agents (and human contributors) when working 
with code in this repository.

## Commands

```bash
# Run all tests with race detection (matches CI)
go test -v -race ./...

# Run tests for the single package
go test -v -race ./pkg/memfs/...

# Run a single test
go test -v -race -run TestFunctionName ./pkg/memfs/...

# Run benchmarks
go test -bench=. -benchmem ./pkg/memfs/...

# Regenerate mocks (fs_fs_mock_test.go)
cd pkg/memfs && go generate

# Lint (config is in tmp/.golangci.yml)
golangci-lint run --config tmp/.golangci.yml ./...
```

## Architecture

The entire library lives in `pkg/memfs` — a single package with no subpackages.

**Core type: `File`** (`file.go`)  
`File` is the central abstraction for both files and directories. It holds a
`[]byte` buffer, an offset, OS-style flags, a `FileInfo`, a parent pointer, and
a `[]*File` entries slice (used only for directories). The same type satisfies ~
15 standard library interfaces simultaneously (compile-time checks are at the
top of `file.go`).

**`FileInfo`** (`file_info.go`)  
A lightweight value type embedded in `File` that implements `fs.FileInfo` and
`fs.DirEntry`. `ModTime` always returns zero; `Sys` always returns nil.

**`fsOnly` / `fsDir`** (`memfs.go`, `file.go`)  
Two thin wrappers around `*File` that expose restricted interface subsets.
`fsOnly` prevents infinite recursion when `fs` helpers call back into `Open`.
`fsDir` is what `File.FS()` returns — it implements `fs.StatFS`,
`fs.ReadFileFS`, and `fs.ReadDirFS`.

**`helpers.go`**  
Contains the `open` function that walks the directory tree recursively,
resolving path segments against `entries`.

**Buffer growth strategy**  
Writes first try to reslice (`tryGrowByReslice`), then double the capacity plus
the required size (`grow`). `ReadFrom` uses a scratch area at the tail of the
backing slice so the caller's buffer is never exposed to the `io.Reader` scratch
contract.

**Test design**  
`compare_test.go` runs the same test suite (`TstFile`, `TstDirectory`) against
both `os.File` / `os.DirFS` and the in-memory `File` / `File.FS()` to ensure
behavioral parity. The helpers `createOSFile` / `createKitFile` and `tstDirOS` /
`tstDirMem` build equivalent real and in-memory structures. `fs_fs_mock_test.go`
is generated via `go generate`.

## Key design rules

- `File.Close()` resets the offset to zero (it does not free the buffer).
- `File.Release()` transfers ownership of the buffer to the caller; the instance
  must not be used afterwards.
- `NewRoot()` creates a nameless directory for use as a filesystem root (no path
  validation applied).
- A `File` added to a directory via `AddFile` cannot be added to a second
  directory (`ErrHasParent`).
- Only `os.O_APPEND` is supported as a flag; `WriteAt` is rejected when append
  mode is active.
- All SPDX license headers must be preserved on every source file.
