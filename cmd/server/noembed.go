//go:build noembed

package main

import (
	"fmt"
	"io/fs"
)

var _ fs.FS = (*EmptyFS)(nil)

type EmptyFS struct {
}

func (e *EmptyFS) Open(name string) (fs.File, error) {
	return nil, fmt.Errorf("empty filesystem being used")
}

var httpStaticContent fs.FS = &EmptyFS{}
