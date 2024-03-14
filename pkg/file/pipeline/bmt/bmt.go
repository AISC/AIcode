// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmt

import (
	"errors"

	"github.com/aisc/pkg/bmtpool"
	"github.com/aisc/pkg/file/pipeline"
	"github.com/aisc/pkg/ aisc"
)

var (
	errInvalidData = errors.New("bmt: invalid data")
)

type bmtWriter struct {
	next pipeline.ChainWriter
}

// NewBmtWriter returns a new bmtWriter. Partial writes are not supported.
// Note: branching factor is the BMT branching factor, not the merkle trie branching factor.
func NewBmtWriter(next pipeline.ChainWriter) pipeline.ChainWriter {
	return &bmtWriter{
		next: next,
	}
}

// ChainWrite writes data in chain. It assumes span has aiscn prepended to the data.
// The span can be encrypted or unencrypted.
func (w *bmtWriter) ChainWrite(p *pipeline.PipeWriteArgs) error {
	if len(p.Data) <  aisc.SpanSize {
		return errInvalidData
	}
	hasher := bmtpool.Get()
	hasher.SetHeader(p.Data[: aisc.SpanSize])
	_, err := hasher.Write(p.Data[ aisc.SpanSize:])
	if err != nil {
		bmtpool.Put(hasher)
		return err
	}
	p.Ref, err = hasher.Hash(nil)
	bmtpool.Put(hasher)
	if err != nil {
		return err
	}

	return w.next.ChainWrite(p)
}

// sum calls the next writer for the cryptographic sum
func (w *bmtWriter) Sum() ([]byte, error) {
	return w.next.Sum()
}
