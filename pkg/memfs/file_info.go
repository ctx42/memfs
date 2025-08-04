// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs

import (
	"io/fs"
	"path/filepath"
	"time"
)

// Compile time checks.
var (
	_ fs.FileInfo = &FileInfo{}
	_ fs.DirEntry = &FileInfo{}
)

// FileInfo implements [fs.FileInfo] interface.
type FileInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (fi FileInfo) Name() string               { return filepath.Base(fi.name) }
func (fi FileInfo) Size() int64                { return fi.size }
func (fi FileInfo) Mode() fs.FileMode          { return fi.mode }
func (fi FileInfo) ModTime() time.Time         { return time.Time{} }
func (fi FileInfo) IsDir() bool                { return fi.mode&fs.ModeDir != 0 }
func (fi FileInfo) Sys() any                   { return nil }
func (fi FileInfo) Type() fs.FileMode          { return fi.Mode().Type() }
func (fi FileInfo) Info() (fs.FileInfo, error) { return fi, nil }
