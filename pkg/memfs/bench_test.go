// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

package memfs_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ctx42/memfs/pkg/memfs"
)

//goland:noinspection GoUnusedGlobalVariable
var bufferWrite int

func BenchmarkFileWrite(b *testing.B) {
	data := make([]byte, 1<<15)

	b.Run("memfs", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		var n int

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			buf := &memfs.File{}
			n, _ = buf.Write(data)
		}
		bufferWrite = n
	})

	b.Run("bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		var n int

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			buf := &bytes.Buffer{}
			n, _ = buf.Write(data)
		}
		bufferWrite = n
	})
}

//goland:noinspection GoUnusedGlobalVariable
var bufferWriteByte error

func BenchmarkFileWriteByte(b *testing.B) {
	b.Run("memfs", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		var err error

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			buf := &memfs.File{}
			err = buf.WriteByte(1)
		}
		bufferWriteByte = err
	})

	b.Run("bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		var err error

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			buf := &bytes.Buffer{}
			err = buf.WriteByte(1)
		}
		bufferWriteByte = err
	})
}

//goland:noinspection GoUnusedGlobalVariable
var bufferWriteString int

func BenchmarkFileWriteString(b *testing.B) {
	b.Run("memfs", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		var n int

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			buf := &memfs.File{}
			n, _ = buf.WriteString("abcdefghijkl")
		}
		bufferWriteString = n
	})

	b.Run("bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		var n int

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			buf := &bytes.Buffer{}
			n, _ = buf.WriteString("abcdefghijkl")
		}
		bufferWriteString = n
	})
}

//goland:noinspection GoUnusedGlobalVariable
var bufferReadFrom int64

func BenchmarkFileReadFrom(b *testing.B) {
	data := make([]byte, 1<<15)

	b.Run("memfs", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		var n int64
		src := bytes.NewReader(data)

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			buf := &memfs.File{}
			n, _ = buf.ReadFrom(src)
			src.Reset(data)
		}
		bufferReadFrom = n
	})

	b.Run("bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		var n int64
		src := bytes.NewReader(data)

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			buf := &bytes.Buffer{}
			n, _ = buf.ReadFrom(src)
			src.Reset(data)
		}
		bufferReadFrom = n
	})
}

// BenchmarkFileWriteByte_loop measures WriteByte in a tight loop on a
// pre-allocated file, isolating per-call overhead from setup cost.
func BenchmarkFileWriteByte_loop(b *testing.B) {
	const N = 1000
	b.Run("memfs", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf, _ := memfs.NewFile("f")
			for j := 0; j < N; j++ {
				_ = buf.WriteByte(byte(j))
			}
		}
	})
	b.Run("bytes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := &bytes.Buffer{}
			for j := 0; j < N; j++ {
				_ = buf.WriteByte(byte(j))
			}
		}
	})
}

//goland:noinspection GoUnusedGlobalVariable
var benchReadDir int

// BenchmarkFileReadDir_single measures a single ReadDir(-1) call at various
// directory sizes. Setup is excluded from the timer.
func BenchmarkFileReadDir_single(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.StopTimer()
			for i := 0; i < b.N; i++ {
				dir := memfs.NewRoot()
				for j := 0; j < n; j++ {
					f, _ := memfs.NewFile(fmt.Sprintf("file%04d", j))
					_ = dir.AddFile(f)
				}
				b.StartTimer()
				entries, _ := dir.ReadDir(-1)
				b.StopTimer()
				benchReadDir = len(entries)
			}
		})
	}
}

// BenchmarkFileReadDir_paginated measures paginated ReadDir(10) calls on a
// pre-built directory. This exercises the sorted-flag optimization: only the
// first call pays the sort cost; subsequent calls skip it.
func BenchmarkFileReadDir_paginated(b *testing.B) {
	for _, n := range []int{100, 1000} {
		n := n
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.StopTimer()
			for i := 0; i < b.N; i++ {
				dir := memfs.NewRoot()
				for j := 0; j < n; j++ {
					f, _ := memfs.NewFile(fmt.Sprintf("file%04d", j))
					_ = dir.AddFile(f)
				}
				b.StartTimer()
				var total int
				for {
					entries, err := dir.ReadDir(10)
					total += len(entries)
					if err != nil {
						break
					}
				}
				b.StopTimer()
				benchReadDir = total
			}
		})
	}
}
