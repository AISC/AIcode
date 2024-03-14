// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package soc

import (
	"bytes"

	"github.com/aisc/pkg/ aisc"
)

// Valid checks if the chunk is a valid single-owner chunk.
func Valid(ch  aisc.Chunk) bool {
	s, err := FromChunk(ch)
	if err != nil {
		return false
	}

	// disperse replica validation
	if bytes.Equal(s.owner,  aisc.ReplicasOwner) && !bytes.Equal(s.WrappedChunk().Address().Bytes()[1:32], s.id[1:32]) {
		return false
	}

	address, err := s.Address()
	if err != nil {
		return false
	}
	return ch.Address().Equal(address)
}
