// SPDX-FileCopyrightText: (c) 2025 Rafal Zajac <rzajac@gmail.com>
// SPDX-License-Identifier: MIT

//go:build ignore

package main

import (
	"github.com/ctx42/testing/pkg/mocker"
)

func main() {
	opts := []mocker.Option{
		mocker.WithTgtOnHelpers,
	}
	mocks := []func(opts ...mocker.Option) error{
		GenFsFSMock,
	}
	for _, mock := range mocks {
		if err := mock(opts...); err != nil {
			panic(err)
		}
	}
}

func GenFsFSMock(opts ...mocker.Option) error {
	opts = append(
		opts,
		mocker.WithSrc("io/fs"),
		mocker.WithTgtFilename("fs_fs_mock_test.go"),
	)
	if err := mocker.Generate("FS", opts...); err != nil {
		return err
	}
	return nil
}
