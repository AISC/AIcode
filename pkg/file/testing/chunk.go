// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testing

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/aisc/pkg/ aisc"
)

// GenerateTestRandomFileChunk generates one single chunk with arbitrary content and address
func GenerateTestRandomFileChunk(address  aisc.Address, spanLength, dataSize int)  aisc.Chunk {
	data := make([]byte, dataSize+8)
	binary.LittleEndian.PutUint64(data, uint64(spanLength))
	_, _ = rand.Read(data[8:])
	key := make([]byte,  aisc.SectionSize)
	if address.IsZero() {
		_, _ = rand.Read(key)
	} else {
		copy(key, address.Bytes())
	}
	return  aisc.NewChunk( aisc.NewAddress(key), data)
}
