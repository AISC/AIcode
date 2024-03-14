// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package splitter provides implementations of the file.Splitter interface
package splitter

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aisc/pkg/file"
	"github.com/aisc/pkg/file/splitter/internal"
	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
)

// simpleSplitter wraps a non-optimized implementation of file.Splitter
type simpleSplitter struct {
	putter storage.Putter
}

// NewSimpleSplitter creates a new SimpleSplitter
func NewSimpleSplitter(storePutter storage.Putter) file.Splitter {
	return &simpleSplitter{
		putter: storePutter,
	}
}

// Split implements the file.Splitter interface
//
// It uses a non-optimized internal component that blocks when performing
// multiple levels of hashing when building the file hash tree.
//
// It returns the Aischash of the data.
func (s *simpleSplitter) Split(ctx context.Context, r io.ReadCloser, dataLength int64, toEncrypt bool) (addr  aisc.Address, err error) {
	j := internal.NewSimpleSplitterJob(ctx, s.putter, dataLength, toEncrypt)
	var total int64
	data := make([]byte,  aisc.ChunkSize)
	var eof bool
	for !eof {
		c, err := r.Read(data)
		total += int64(c)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if total < dataLength {
					return  aisc.ZeroAddress, fmt.Errorf("splitter only received %d bytes of data, expected %d bytes", total+int64(c), dataLength)
				}
				eof = true
				continue
			} else {
				return  aisc.ZeroAddress, err
			}
		}
		cc, err := j.Write(data[:c])
		if err != nil {
			return  aisc.ZeroAddress, err
		}
		if cc < c {
			return  aisc.ZeroAddress, fmt.Errorf("write count to file hasher component %d does not match read count %d", cc, c)
		}
	}

	sum := j.Sum(nil)
	newAddress :=  aisc.NewAddress(sum)
	return newAddress, nil
}
