// Copyright 2021 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cac

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/aisc/pkg/bmtpool"
	"github.com/aisc/pkg/ aisc"
)

var (
	ErrChunkSpanShort = fmt.Errorf("chunk span must have exactly length of %d",  aisc.SpanSize)
	ErrChunkDataLarge = fmt.Errorf("chunk data exceeds maximum allowed length")
)

// New creates a new content address chunk by initializing a span and appending the data to it.
func New(data []byte) ( aisc.Chunk, error) {
	dataLength := len(data)

	if err := validateDataLength(dataLength); err != nil {
		return nil, err
	}

	span := make([]byte,  aisc.SpanSize)
	binary.LittleEndian.PutUint64(span, uint64(dataLength))

	return newWithSpan(data, span)
}

// NewWithDataSpan creates a new chunk assuming that the span precedes the actual data.
func NewWithDataSpan(data []byte) ( aisc.Chunk, error) {
	dataLength := len(data)

	if err := validateDataLength(dataLength -  aisc.SpanSize); err != nil {
		return nil, err
	}

	return newWithSpan(data[ aisc.SpanSize:], data[: aisc.SpanSize])
}

// validateDataLength validates if data length (without span) is correct.
func validateDataLength(dataLength int) error {
	if dataLength < 0 { // dataLength could be negative when span size is subtracted
		spanLength :=  aisc.SpanSize + dataLength
		return fmt.Errorf("invalid CAC span length %d: %w", spanLength, ErrChunkSpanShort)
	}
	if dataLength >  aisc.ChunkSize {
		return fmt.Errorf("invalid CAC data length %d: %w", dataLength, ErrChunkDataLarge)
	}
	return nil
}

// newWithSpan creates a new chunk prepending the given span to the data.
func newWithSpan(data, span []byte) ( aisc.Chunk, error) {
	hash, err := DoHash(data, span)
	if err != nil {
		return nil, err
	}

	cacData := make([]byte, len(data)+len(span))
	copy(cacData, span)
	copy(cacData[ aisc.SpanSize:], data)

	return  aisc.NewChunk( aisc.NewAddress(hash), cacData), nil
}

// Valid checks whether the given chunk is a valid content-addressed chunk.
func Valid(c  aisc.Chunk) bool {
	data := c.Data()

	if validateDataLength(len(data)- aisc.SpanSize) != nil {
		return false
	}

	hash, _ := DoHash(data[ aisc.SpanSize:], data[: aisc.SpanSize])

	return bytes.Equal(hash, c.Address().Bytes())
}

func DoHash(data, span []byte) ([]byte, error) {
	hasher := bmtpool.Get()
	defer bmtpool.Put(hasher)

	hasher.SetHeader(span)
	if _, err := hasher.Write(data); err != nil {
		return nil, err
	}

	return hasher.Hash(nil)
}
