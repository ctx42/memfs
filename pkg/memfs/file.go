// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

// smallBufferSize is an initial allocation minimal capacity.
const smallBufferSize = 64

// ErrOutOfBounds is returned for invalid offsets.
var ErrOutOfBounds = errors.New("offset out of bounds")

// ErrHasParent is returned when an instance of [File] which is already a
// child of another instance is added to a directory.
var ErrHasParent = errors.New("entry already has a parent")

// errWriteAtInAppendMode returned when [File.WriteAt] is used with a
// [os.O_APPEND] flag. It has the same message as the error with the same name
// in the os package.
var errWriteAtInAppendMode = errors.New("os: invalid use of WriteAt on file " +
	"opened with O_APPEND")

// WithFileOffset is a [File] constructor function option setting the offset.
func WithFileOffset(off int) func(*File) {
	return func(fil *File) { fil.off = off }
}

// WithFileAppend is a [File] constructor function option setting the offset to
// the end of the file. This option must be the last option on the option list.
//
// When [File.Truncate] is used, the offset will be set to the end of the
// [File].
func WithFileAppend(fil *File) { fil.flag |= os.O_APPEND }

// WithFileFlag is a [File] constructor function option setting flags. Flags
// are the same as for [os.OpenFile].
//
// Currently, only the [os.O_APPEND] flag is supported.
func WithFileFlag(flag int) func(*File) {
	return func(fil *File) { fil.flag = flag }
}

// Compile time checks.
var (
	_ io.Closer       = &File{}
	_ io.Reader       = &File{}
	_ io.ReaderAt     = &File{}
	_ io.Seeker       = &File{}
	_ io.StringWriter = &File{}
	_ io.ReaderFrom   = &File{}
	_ io.Writer       = &File{}
	_ io.WriterAt     = &File{}
	_ io.WriterTo     = &File{}

	_ fs.DirEntry    = &File{}
	_ fs.File        = &File{}
	_ fs.FileInfo    = &File{}
	_ fs.FS          = &File{}
	_ fs.ReadDirFile = &File{}
	_ fs.ReadFileFS  = &File{}
)

// A File is a variable-sized buffer of bytes representing a file or directory.
type File struct {
	off     int              // Current offset for read and write operations.
	buf     []byte           // Underlying buffer.
	flag    int              // Instance flags.
	info    FileInfo         // The file or directory information.
	parent  *File            // Parent directory (nil for the root directory).
	cursor  int              // Used as [File.ReadDir] cursor.
	entries map[string]*File // Entries when the file represents a directory.
}

// NewFile returns a new instance of [File] with an initial capacity of
// [bytes.MinRead]. It will panic with [ErrOutOfBounds] if the [WithFileOffset]
// option sets offset to a negative number or greater than [bytes.MinRead].
func NewFile(name string, opts ...func(*File)) (*File, error) {
	return FileWith(name, make([]byte, 0, bytes.MinRead), opts...)
}

// FileWith creates a new instance of [File] initialized with content. The
// created instance takes ownership of the content slice, and the caller must
// not use it after passing it to this function. FileWith is intended to
// prepare the [File] instance to read existing data. It can also be used to
// set the initial size of the internal buffer for writing. To do that, the
// content slice should have the desired capacity but a length of zero. It will
// panic with [ErrOutOfBounds] if the [WithFileOffset] option sets the offset
// to a negative number or beyond sata slice length.
func FileWith(name string, content []byte, opts ...func(*File)) (*File, error) {
	if !fs.ValidPath(name) || strings.Contains(name, "/") {
		return nil, fs.ErrInvalid
	}
	fil := &File{
		buf: content,
		info: FileInfo{
			name: name,
			size: int64(len(content)),
			mode: 0600,
		},
	}
	for _, opt := range opts {
		opt(fil)
	}
	if fil.off < 0 || fil.off > len(fil.buf) {
		panic(ErrOutOfBounds)
	}
	return fil, nil
}

// NewDirectory returns a new instance of [File] representing a directory.
func NewDirectory(name string) (*File, error) {
	dir, err := FileWith(name, nil)
	if err != nil {
		return nil, err
	}
	dir.info.size = 4096
	dir.info.mode = 0700 | os.ModeDir
	return dir, nil
}

// NewRoot returns a new instance of [File] representing the root directory.
// The root directory is a nameless special directory that contains all other
// files and directories.
func NewRoot() *File {
	return &File{info: FileInfo{size: 4096, mode: 0700 | os.ModeDir}}
}

// AddFile adds a file to the directory. Returns [fs.ErrExist] if the file by
// that name already exists, [fs.ErrInvalid] if the file is not a regular file
// or a directory. Returns [fs.ErrInvalid] if the file name is a path.
func (fil *File) AddFile(file *File) error {
	if !fil.IsDir() {
		return &fs.PathError{
			Op:   "open",
			Path: filepath.Join(fil.info.name, file.info.Name()),
			Err:  syscall.ENOTDIR,
		}
	}

	if file.parent != nil {
		return &fs.PathError{
			Op:   "AddFile",
			Path: file.path(),
			Err:  ErrHasParent,
		}
	}

	switch file.Type() {
	case fs.ModeDir, fs.FileMode(0):
	default:
		return fs.ErrInvalid
	}

	name := file.Name()
	if _, ok := fil.entries[name]; ok {
		return fs.ErrExist
	}
	if fil.entries == nil {
		fil.entries = make(map[string]*File)
	}
	fil.entries[name] = file
	file.parent = fil
	return nil
}

// ReadDir implements [fs.ReadDirFile] interface.
func (fil *File) ReadDir(n int) ([]fs.DirEntry, error) {
	if !fil.IsDir() {
		return nil, &fs.PathError{
			Op:   "ReadDir",
			Path: fil.path(),
			Err:  syscall.ENOTDIR,
		}
	}

	var names []string
	for _, name := range slices.Sorted(maps.Keys(fil.entries)) {
		names = append(names, name)
	}

	// If n <= 0, return all remaining entries.
	if n <= 0 {
		n = len(fil.entries) - fil.cursor
	}

	// Check if we've reached the end.
	if fil.cursor >= len(fil.entries) {
		return nil, io.EOF
	}

	// Calculate how many entries to return.
	end := fil.cursor + n
	if end > len(fil.entries) {
		end = len(fil.entries)
	}

	ets := make([]fs.DirEntry, 0, end-fil.cursor)
	for i := fil.cursor; i < end; i++ {
		file := fil.entries[names[i]]
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		ets = append(ets, fs.FileInfoToDirEntry(info))
	}
	fil.cursor = end
	return ets, nil
}

// ReadFile implements [fs.ReadFileFS] interface.
func (fil *File) ReadFile(name string) ([]byte, error) {
	if !fil.IsDir() {
		return nil, &fs.PathError{
			Op:   "ReadFile",
			Path: fil.path(),
			Err:  syscall.ENOTDIR,
		}
	}
	return fs.ReadFile(fsOnly{fil}, name)
}

// Name implements [fs.DirEntry].
func (fil *File) Name() string { return fil.info.Name() }

// IsDir implements [fs.DirEntry] and always returns false
func (fil *File) IsDir() bool { return fil.info.IsDir() }

// Type implements [fs.DirEntry] and always returns 0 for the regular file.
func (fil *File) Type() fs.FileMode { return fil.info.Type() }

// Stat returns information about the in-memory file, the size is the length of
// the underlying buffer, modification time is always zero value time and
// [fs.FileInfo.Sys] always returns nil.
func (fil *File) Stat() (fs.FileInfo, error) {
	info := fil.info
	info.size = fil.Size()
	return info, nil
}

// Info returns information about the in-memory file, the size is the length of
// the underlying buffer, modification time is always zero value time and
// [fs.FileInfo.Sys] always returns nil.
func (fil *File) Info() (fs.FileInfo, error) { return fil.Stat() }

// Size implements [fs.FileInfo] interface. Always returns 4096 for directories.
func (fil *File) Size() int64 {
	if !fil.IsDir() {
		return int64(len(fil.buf))
	}
	return fil.info.size
}

// Mode implements [fs.FileInfo] interface.
func (fil *File) Mode() fs.FileMode { return fil.info.mode }

// ModTime implements [fs.FileInfo] interface - always returns zero value time.
func (fil *File) ModTime() time.Time { return fil.info.ModTime() }

// Sys implements [fs.FileInfo] interface - always returns nil.
func (fil *File) Sys() any { return fil.info.Sys() }

// Open implements [fs.FS] interface.
func (fil *File) Open(name string) (fs.File, error) { return open(fil, name) }

// FS returns a file system [fs.FS] for the list of files in the directory.
// Returns nil if the file is not a directory.
//
// The result implements:
//   - [io/fs.StatFS],
//   - [io/fs.ReadFileFS],
//   - [io/fs.ReadDirFS].
func (fil *File) FS() fs.FS {
	if fil.IsDir() {
		return fsDir{dir: fil}
	}
	return nil
}

// Release releases ownership of the underlying buffer, the caller should not
// use this instance after this call.
func (fil *File) Release() []byte {
	buf := fil.buf
	fil.off = 0
	fil.buf = nil
	return buf
}

// Write writes the contents of p to the underlying buffer at the current
// offset, growing the buffer as needed. The return value n is the length of p;
// returns an error when the file represents a directory.
func (fil *File) Write(p []byte) (n int, err error) {
	if fil.IsDir() {
		return 0, &fs.PathError{
			Op:   "write",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}
	return fil.write(p), nil
}

// WriteByte writes a byte b to the underlying buffer at the current offset.
// Returns an error when the file represents a directory.
func (fil *File) WriteByte(b byte) error {
	if fil.IsDir() {
		return &fs.PathError{
			Op:   "write",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}
	fil.write([]byte{b})
	return nil
}

// WriteAt writes len(p) bytes to the underlying buffer starting at the current
// offset. It returns the number of bytes written; err is returned only when
// the file was opened with an [os.O_APPEND] flag or the file represents a
// directory. It does not change the offset.
func (fil *File) WriteAt(p []byte, off int64) (n int, err error) {
	if fil.IsDir() {
		return 0, &fs.PathError{
			Op:   "write",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}

	if fil.flag&os.O_APPEND != 0 {
		return 0, errWriteAtInAppendMode
	}

	prev := fil.off
	c := cap(fil.buf)
	pl := len(p)

	// Handle writing beyond capacity.
	if int(off)+pl > c {
		fil.off = c // So tryGrowByReslice returns false.
		fil.grow(int(off) + pl - len(fil.buf))
		fil.buf = fil.buf[:int(off)+pl]
	}

	fil.off = int(off)
	n = fil.write(p)
	fil.off = prev
	return n, nil
}

// WriteTo writes data to w starting at the current offset until there are no
// more bytes to write or when an error occurs. The int64 return value is the
// number of bytes written. When an error occurred during the operation, it is
// also returned.
func (fil *File) WriteTo(w io.Writer) (int64, error) {
	if fil.IsDir() {
		return 0, &fs.PathError{
			Op:   "write",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}
	n, err := w.Write(fil.buf[fil.off:])
	fil.off += n
	return int64(n), err
}

// WriteString writes string s to the buffer at the current offset.
func (fil *File) WriteString(s string) (int, error) {
	return fil.Write([]byte(s)) // nolint: gocritic
}

// write writes p at the current offset.
func (fil *File) write(p []byte) int {
	if fil.flag&os.O_APPEND != 0 {
		fil.off = len(fil.buf)
	}
	l := len(fil.buf)
	fil.grow(len(p))
	n := copy(fil.buf[fil.off:], p)
	fil.off += n
	if fil.off > l {
		l = fil.off
	}
	fil.buf = fil.buf[:l]
	return n
}

// Read reads the next len(p) bytes from the buffer at the current offset or
// until the buffer is drained. The return value is the number of bytes read.
// If the buffer has no data to return, err is [io.EOF] (unless len(p) is zero)
// or if the file represents a directory; otherwise it is nil.
func (fil *File) Read(p []byte) (int, error) {
	if fil.IsDir() {
		return 0, &fs.PathError{
			Op:   "read",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}
	// Nothing more to read.
	if len(p) > 0 && fil.off >= len(fil.buf) {
		return 0, io.EOF
	}
	n := copy(p, fil.buf[fil.off:])
	fil.off += n
	return n, nil
}

// ReadByte reads and returns the next byte from the buffer at the current
// offset or returns an error. If ReadByte returns an error, no input byte was
// consumed, and the returned byte value is undefined.
func (fil *File) ReadByte() (byte, error) {
	if fil.IsDir() {
		return 0, &fs.PathError{
			Op:   "read",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}
	// Nothing more to read.
	if fil.off >= len(fil.buf) {
		return 0, io.EOF
	}
	v := fil.buf[fil.off]
	fil.off++
	return v, nil
}

// ReadAt reads len(p) bytes from the buffer at the current offset. It returns
// the number of bytes read and the error, if any. ReadAt always returns a
// non-nil error when n < len(p) or when the file represents a directory. It
// does not change the offset.
func (fil *File) ReadAt(p []byte, off int64) (int, error) {
	if fil.IsDir() {
		return 0, &fs.PathError{
			Op:   "read",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}
	prev := fil.off
	defer func() { fil.off = prev }()
	fil.off = int(off)
	n, err := fil.Read(p)
	if err != nil {
		return n, err
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// ReadFrom reads data from r until EOF and appends it to the buffer at the
// current offset, growing the buffer as needed. The return value is the number
// of bytes read. Any error except [io.EOF] encountered during the read is also
// returned. If the buffer becomes too large, ReadFrom will panic with
// [bytes.ErrTooLarge].
func (fil *File) ReadFrom(r io.Reader) (int64, error) {
	var err error
	var n, total int

	if fil.IsDir() {
		return 0, &fs.PathError{
			Op:   "write",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}
	if fil.flag&os.O_APPEND != 0 {
		fil.off = len(fil.buf)
	}

	for {

		// Length before growing the buffer.
		l := len(fil.buf)

		// Make sure we can fit [bytes.MinRead] between the current offset and
		// the new buffer length.
		fil.grow(bytes.MinRead)

		// We will use bytes between l and cap(fil.buf) as a temporary scratch
		// space for reading from r and then slide read bytes to place. We have
		// to do it this way because [io.Read] documentation says that: "Even
		// if Read returns n < len(p), it may use all of p as scratch space
		// during the call." so we can't pass our buffer to Read because it
		// might change parts of it not involved in the read operation.
		tmp := fil.buf[l:cap(fil.buf)]
		n, err = r.Read(tmp)

		if l != fil.off {
			// Move bytes from temporary area to correct place.
			copy(fil.buf[fil.off:], tmp[:n])
			if n < len(tmp) {
				// Clean up any garbage the reader might put in there. We want
				// to keep all bytes between len and cap as zeros.
				zeroOutSlice(tmp[n:])
			}
		}

		fil.off += n
		total += n

		if fil.off > l {
			l = fil.off
		}

		// Set proper buffer length.
		fil.buf = fil.buf[:l]

		if err != nil {
			break
		}
	}

	// The [io.EOF] is not an error.
	if err == io.EOF {
		err = nil
	}

	return int64(total), err
}

// String returns string representation of the buffer starting at the current
// offset. Calling this method is considered as reading the buffer and advances
// offset to the end of the buffer. When the file represents a directory, it
// returns an empty string.
func (fil *File) String() string {
	s := string(fil.buf[fil.off:])
	fil.off = len(fil.buf)
	return s
}

// Seek sets the offset for the next Read or Write on the buffer to the offset,
// interpreted according to whence: 0 means relative to the origin of the file,
// 1 means relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error (only if calculated offset < 0).
// Returns a non-nil error of the [fs.PathError] type.
func (fil *File) Seek(offset int64, whence int) (int64, error) {
	if fil.IsDir() {
		return 0, &fs.PathError{
			Op:   "seek",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}

	var off int
	switch whence {
	case io.SeekStart:
		off = int(offset)
	case io.SeekCurrent:
		off = fil.off + int(offset)
	case io.SeekEnd:
		off = len(fil.buf) + int(offset)
	}

	if off < 0 {
		return 0, &fs.PathError{
			Op:   "seek",
			Path: fil.path(),
			Err:  syscall.EINVAL,
		}
	}
	fil.off = off

	return int64(fil.off), nil
}

// SeekStart is a convenience method setting the buffer's offset to zero and
// returning the value it had before the method was called.
func (fil *File) SeekStart() int64 {
	prev := fil.off
	fil.off = 0
	return int64(prev)
}

// SeekEnd is a convenience method setting the buffer's offset to the buffer
// length and returning the value it had before the method was called.
func (fil *File) SeekEnd() int64 {
	prev := fil.off
	fil.off = len(fil.buf)
	return int64(prev)
}

// Truncate changes the size of the buffer discarding bytes at the offsets
// greater than size. It does not change the offset unless a [WithFileAppend]
// option was used, then it sets the offset to the end of the buffer. Returns
// an error only when the size is negative. The error is of the [fs.PathError]
// type.
func (fil *File) Truncate(size int64) error {
	if fil.IsDir() {
		return &fs.PathError{
			Op:   "truncate",
			Path: fil.path(),
			Err:  syscall.EISDIR,
		}
	}

	if size < 0 {
		return &os.PathError{
			Op:   "truncate",
			Path: fil.path(),
			Err:  syscall.EINVAL,
		}
	}

	prev := fil.off
	l := len(fil.buf)
	c := cap(fil.buf)

	switch {
	case int(size) == l:
		// Nothing to do.

	case int(size) == c:
		// Reslice.
		fil.buf = fil.buf[:size]

	case int(size) > l && int(size) < c:
		// Truncate between len and cap.
		fil.buf = fil.buf[:size]

	case int(size) > c:
		// Truncate beyond the cap.
		fil.off = c // So tryGrowByReslice returns false.
		fil.grow(int(size) - l)
		fil.buf = fil.buf[:int(size)]

	default:
		// Reduce the size of the buffer.
		zeroOutSlice(fil.buf[size:])
		fil.buf = fil.buf[:size]
	}

	fil.off = prev

	return nil
}

// Grow grows the buffer's capacity, if necessary, to guarantee space for
// another n bytes. After Grow(n), at least n bytes can be written to the
// buffer without another allocation. If n is negative, Grow will panic. If the
// buffer can't grow, it will panic with [bytes.ErrTooLarge].
func (fil *File) Grow(n int) {
	if n < 0 {
		panic("memfs.File.Grow: negative count")
	}
	if fil.IsDir() {
		return
	}

	l := len(fil.buf)
	if l+n <= cap(fil.buf) {
		return
	}

	// Allocate bigger buffer.
	tmp := makeSlice(l + n)
	copy(tmp, fil.buf)
	fil.buf = tmp
	fil.buf = fil.buf[:l]
}

// grow grows the buffer's capacity to guarantee space for n more bytes. In
// other words, it makes sure there are n bytes between the current offset and
// the buffer capacity. It's worth noting that after calling this method the
// len(b.buf) changes. If the buffer can't grow, it will panic with
// [bytes.ErrTooLarge].
func (fil *File) grow(n int) {
	// Try to grow by a reslice.
	if ok := fil.tryGrowByReslice(n); ok {
		return
	}
	if fil.buf == nil && n <= smallBufferSize {
		fil.buf = make([]byte, n, smallBufferSize)
		return
	}
	// Allocate bigger buffer.
	tmp := makeSlice(cap(fil.buf)*2 + n) // cap(b.buf) may be zero.
	copy(tmp, fil.buf)
	fil.buf = tmp
}

// tryGrowByReslice is an inlineable version of [File.grow] for the fast-case
// where the internal buffer only needs to be resliced. It returns whether it
// succeeded.
func (fil *File) tryGrowByReslice(n int) bool {
	// No need to do anything if there is enough space between the current
	// offset and the length of the buffer.
	if n <= len(fil.buf)-fil.off {
		return true
	}

	if n <= cap(fil.buf)-fil.off {
		fil.buf = fil.buf[:fil.off+n]
		return true
	}
	return false
}

// Offset returns the current offset.
func (fil *File) Offset() int { return fil.off }

// Len returns the buffer length.
func (fil *File) Len() int { return len(fil.buf) }

// Cap returns the buffer capacity, that is, the total space allocated for the
// buffer's data.
func (fil *File) Cap() int { return cap(fil.buf) }

// path returns the full path of the instance, including the parent's path if
// it's not the root.
func (fil *File) path() string {
	pth := fil.Name()
	if fil.parent != nil {
		if par := fil.parent.path(); par != "" {
			return filepath.Join(par, pth)
		}
	}
	return pth
}

// Close sets offset to zero. It always returns nil error.
func (fil *File) Close() error {
	if fil == nil {
		return nil
	}
	fil.off = 0
	return nil
}

// List recursively lists the directory and returns a string with one entry per
// line. If the instance is not a directory, it returns an error.
func (fil *File) List() (string, error) {
	if !fil.IsDir() {
		return "", &fs.PathError{
			Op:   "list",
			Path: fil.path(),
			Err:  syscall.ENOTDIR,
		}
	}
	return list(fil)
}

func list(root fs.FS) (string, error) {
	out := ""
	fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		out += path + "\n"
		return nil
	}
	return out, fs.WalkDir(root, ".", fn)
}

// zeroOutSlice zeroes out the byte slice.
func zeroOutSlice(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// makeSlice allocates a slice of size n. If the allocation fails, it panics
// with [bytes.ErrTooLarge].
func makeSlice(n int) []byte {
	// If the make fails, give a known error.
	defer func() {
		if recover() != nil {
			panic(bytes.ErrTooLarge)
		}
	}()
	return make([]byte, n)
}

// fsDir wraps instance of [Directory] and implements the following interfaces:
//
//   - [io/fs.StatFS],
//   - [io/fs.ReadFileFS],
//   - [io/fs.ReadDirFS].
type fsDir struct{ dir *File }

// ReadDir implements [fs.ReadDirFS] interface.
func (f fsDir) ReadDir(name string) ([]fs.DirEntry, error) {
	fil, err := f.open(name)
	if err != nil {
		return nil, err
	}

	if !fil.IsDir() {
		return nil, &fs.PathError{
			Op:   "readdirent",
			Path: filepath.Join(f.dir.info.name, name),
			Err:  syscall.ENOTDIR,
		}
	}
	return fil.ReadDir(-1)
}

// Open implements [fs.FS] interface.
func (f fsDir) Open(name string) (fs.File, error) { return f.open(name) }

// open opens the file with the given name and handles errors in a way that
// matches the behavior of [os.Open] and [os.OpenFile].
func (f fsDir) open(name string) (*File, error) {
	fil, err := open(f.dir, name)
	if err != nil {
		var e *fs.PathError
		if errors.As(err, &e) {
			switch {
			case errors.Is(err, fs.ErrNotExist):
				e.Op = "openat"
				e.Err = syscall.ENOENT
			default:
				return nil, e
			}
		}
		return nil, err
	}
	return fil, nil
}

// Stat implements [fs.StatFS] interface.
func (f fsDir) Stat(name string) (fs.FileInfo, error) {
	fil, ok := f.dir.entries[name]
	if !ok {
		return nil, &fs.PathError{Op: "statat", Path: name, Err: syscall.ENOENT}
	}
	return fil.Stat()
}
